package sqlstore

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func (s *Store) CreateGuild(guildName string, leaderCharacterID string) (platform.Guild, error) {
	leader, err := s.GetCharacterByID(leaderCharacterID)
	if err != nil {
		return platform.Guild{}, err
	}
	if _, err := s.GetGuildForCharacter(leaderCharacterID); err == nil {
		return platform.Guild{}, filestore.ErrGuildMemberExists
	} else if !errors.Is(err, filestore.ErrGuildMissing) {
		return platform.Guild{}, err
	}

	now := s.now().Unix()
	guild := platform.Guild{
		ID:                   randomID("guild"),
		RealmID:              leader.RealmID,
		GuildName:            strings.TrimSpace(guildName),
		CreatedAt:            now,
		UpdatedAt:            now,
		CreatedByCharacterID: leader.ID,
		LeaderCharacterID:    leader.ID,
		Ranks:                platform.DefaultGuildRanks(),
		Members:              []platform.GuildMember{buildGuildMember(*leader, platform.GuildRankLeader, now)},
	}
	if guild.GuildName == "" {
		return platform.Guild{}, fmt.Errorf("guild name is required")
	}
	if err := s.WithTransaction("sqlstore.guild_create", func(tx *Tx) error {
		return tx.saveGuild(guild)
	}); err != nil {
		if isConstraintError(err) {
			return platform.Guild{}, filestore.ErrGuildNameExists
		}
		return platform.Guild{}, err
	}
	return cloneGuild(guild), nil
}

func (s *Store) SaveGuild(guild platform.Guild) (platform.Guild, error) {
	guild.GuildName = strings.TrimSpace(guild.GuildName)
	guild.Members = normalizeGuildMembers(guild.Members)
	if len(guild.Members) == 0 {
		if err := s.DeleteGuild(guild.ID); err != nil {
			return platform.Guild{}, err
		}
		return cloneGuild(guild), nil
	}
	if !guildHasMember(guild.Members, guild.LeaderCharacterID) {
		guild.LeaderCharacterID = guild.Members[0].CharacterID
		guild.Members[0].RankID = platform.GuildRankLeader
	}
	guild.UpdatedAt = s.now().Unix()
	if err := s.WithTransaction("sqlstore.guild_save", func(tx *Tx) error {
		return tx.saveGuild(guild)
	}); err != nil {
		return platform.Guild{}, err
	}
	loaded, err := s.GetGuildByID(guild.ID)
	if err != nil {
		return platform.Guild{}, err
	}
	return *loaded, nil
}

func (tx *Tx) saveGuild(guild platform.Guild) error {
	now := tx.store.now().Unix()
	if guild.ID == "" {
		guild.ID = randomID("guild")
	}
	if guild.CreatedAt == 0 {
		guild.CreatedAt = now
	}
	if guild.UpdatedAt == 0 {
		guild.UpdatedAt = now
	}
	if len(guild.Ranks) == 0 {
		guild.Ranks = platform.DefaultGuildRanks()
	}
	ranksJSON, err := encodeJSON(guild.Ranks)
	if err != nil {
		return err
	}
	_, err = tx.tx.Exec(
		`INSERT INTO ac_guilds (
			guild_id, realm_id, guild_name, normalized_guild_name, leader_character_id,
			created_by_character_id, motd, ranks_json, created_at, updated_at, version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(guild_id) DO UPDATE SET
			guild_name = excluded.guild_name,
			normalized_guild_name = excluded.normalized_guild_name,
			leader_character_id = excluded.leader_character_id,
			motd = excluded.motd,
			ranks_json = excluded.ranks_json,
			updated_at = excluded.updated_at,
			disbanded_at = 0,
			version = ac_guilds.version + 1`,
		guild.ID,
		guild.RealmID,
		guild.GuildName,
		normalize(guild.GuildName),
		guild.LeaderCharacterID,
		guild.CreatedByCharacterID,
		guild.MessageOfTheDay,
		ranksJSON,
		guild.CreatedAt,
		guild.UpdatedAt)
	if err != nil {
		return err
	}
	_, err = tx.tx.Exec(`UPDATE ac_guild_members SET left_at = ? WHERE guild_id = ? AND left_at = 0`, now, guild.ID)
	if err != nil {
		return err
	}
	for _, member := range normalizeGuildMembers(guild.Members) {
		character, _, err := tx.loadCharacterForMutation(member.CharacterID)
		if err != nil {
			return err
		}
		member.DisplayName = character.DisplayName
		member.RaceID = character.RaceID
		member.ClassID = character.ClassID
		member.Level = character.Level
		member.LastOnlineAt = character.LastSeenAt
		if member.JoinedAt == 0 {
			member.JoinedAt = now
		}
		if _, err := tx.tx.Exec(
			`INSERT INTO ac_guild_members (
				guild_id, character_id, display_name, race_id, class_id, level, rank_id, joined_at, last_online_at, left_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
			ON CONFLICT(guild_id, character_id) DO UPDATE SET
				display_name = excluded.display_name,
				race_id = excluded.race_id,
				class_id = excluded.class_id,
				level = excluded.level,
				rank_id = excluded.rank_id,
				last_online_at = excluded.last_online_at,
				left_at = 0`,
			guild.ID,
			member.CharacterID,
			member.DisplayName,
			member.RaceID,
			member.ClassID,
			member.Level,
			member.RankID,
			member.JoinedAt,
			member.LastOnlineAt); err != nil {
			if isConstraintError(err) {
				return filestore.ErrGuildMemberExists
			}
			return err
		}
	}
	return nil
}

func (s *Store) DeleteGuild(guildID string) error {
	now := s.now().Unix()
	result, err := s.db.Exec(
		`UPDATE ac_guilds SET disbanded_at = ?, updated_at = ?, version = version + 1 WHERE guild_id = ? AND disbanded_at = 0`,
		now,
		now,
		guildID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return filestore.ErrGuildMissing
	}
	_, err = s.db.Exec(`UPDATE ac_guild_members SET left_at = ? WHERE guild_id = ? AND left_at = 0`, now, guildID)
	return err
}

func (s *Store) GetGuildByID(guildID string) (*platform.Guild, error) {
	row := s.db.QueryRow(guildSelectSQL()+` WHERE guild_id = ? AND disbanded_at = 0`, guildID)
	return s.scanGuild(row)
}

func (s *Store) GetGuildForCharacter(characterID string) (*platform.Guild, error) {
	row := s.db.QueryRow(
		guildSelectSQL()+`
		WHERE guild_id = (
			SELECT guild_id FROM ac_guild_members WHERE character_id = ? AND left_at = 0 LIMIT 1
		) AND disbanded_at = 0`,
		characterID)
	return s.scanGuild(row)
}

func guildSelectSQL() string {
	return `SELECT guild_id, realm_id, guild_name, created_at, updated_at, created_by_character_id,
		leader_character_id, motd, ranks_json FROM ac_guilds`
}

func (s *Store) scanGuild(row rowScanner) (*platform.Guild, error) {
	var guild platform.Guild
	var ranksJSON string
	if err := row.Scan(
		&guild.ID,
		&guild.RealmID,
		&guild.GuildName,
		&guild.CreatedAt,
		&guild.UpdatedAt,
		&guild.CreatedByCharacterID,
		&guild.LeaderCharacterID,
		&guild.MessageOfTheDay,
		&ranksJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, filestore.ErrGuildMissing
		}
		return nil, err
	}
	if err := decodeJSON(ranksJSON, &guild.Ranks); err != nil {
		return nil, err
	}
	members, err := s.loadGuildMembers(guild.ID)
	if err != nil {
		return nil, err
	}
	guild.Members = members
	return &guild, nil
}

func (s *Store) loadGuildMembers(guildID string) ([]platform.GuildMember, error) {
	rows, err := s.db.Query(
		`SELECT character_id, display_name, race_id, class_id, level, rank_id, joined_at, last_online_at
		FROM ac_guild_members
		WHERE guild_id = ? AND left_at = 0
		ORDER BY joined_at, character_id`,
		guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []platform.GuildMember
	for rows.Next() {
		var member platform.GuildMember
		if err := rows.Scan(
			&member.CharacterID,
			&member.DisplayName,
			&member.RaceID,
			&member.ClassID,
			&member.Level,
			&member.RankID,
			&member.JoinedAt,
			&member.LastOnlineAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (s *Store) CreateGuildInvite(guildID string, inviterCharacterID string, targetCharacterID string, expiresAt int64) (platform.GuildInvite, error) {
	guild, err := s.GetGuildByID(guildID)
	if err != nil {
		return platform.GuildInvite{}, err
	}
	target, err := s.GetCharacterByID(targetCharacterID)
	if err != nil {
		return platform.GuildInvite{}, err
	}
	if target.RealmID != guild.RealmID {
		return platform.GuildInvite{}, fmt.Errorf("target player is not on this realm")
	}
	if _, err := s.GetGuildForCharacter(targetCharacterID); err == nil {
		return platform.GuildInvite{}, filestore.ErrGuildMemberExists
	} else if !errors.Is(err, filestore.ErrGuildMissing) {
		return platform.GuildInvite{}, err
	}
	now := s.now().Unix()
	if expiresAt == 0 {
		expiresAt = now + 900
	}
	invite := platform.GuildInvite{
		InviteID:           randomID("ginvite"),
		GuildID:            guild.ID,
		GuildName:          guild.GuildName,
		InviterCharacterID: inviterCharacterID,
		TargetCharacterID:  targetCharacterID,
		CreatedAt:          now,
		ExpiresAt:          expiresAt,
	}
	_, err = s.db.Exec(
		`INSERT INTO ac_guild_invites (
			invite_id, guild_id, guild_name, inviter_character_id, target_character_id, state, created_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		invite.InviteID,
		invite.GuildID,
		invite.GuildName,
		invite.InviterCharacterID,
		invite.TargetCharacterID,
		filestore.InviteStatePending,
		invite.CreatedAt,
		invite.ExpiresAt)
	if err != nil {
		if isConstraintError(err) {
			return platform.GuildInvite{}, filestore.ErrDuplicateMutation
		}
		return platform.GuildInvite{}, err
	}
	return invite, nil
}

func (s *Store) GetGuildInvite(inviteID string) (*platform.GuildInvite, error) {
	row := s.db.QueryRow(
		`SELECT invite_id, guild_id, guild_name, inviter_character_id, target_character_id, created_at, expires_at
		FROM ac_guild_invites WHERE invite_id = ?`,
		inviteID)
	var invite platform.GuildInvite
	if err := row.Scan(
		&invite.InviteID,
		&invite.GuildID,
		&invite.GuildName,
		&invite.InviterCharacterID,
		&invite.TargetCharacterID,
		&invite.CreatedAt,
		&invite.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, filestore.ErrGuildInviteMissing
		}
		return nil, err
	}
	return &invite, nil
}

func (s *Store) ListGuildInvitesForCharacter(characterID string) ([]platform.GuildInvite, error) {
	rows, err := s.db.Query(
		`SELECT invite_id, guild_id, guild_name, inviter_character_id, target_character_id, created_at, expires_at
		FROM ac_guild_invites
		WHERE target_character_id = ? AND state = ?
		ORDER BY created_at`,
		characterID,
		filestore.InviteStatePending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []platform.GuildInvite
	for rows.Next() {
		var invite platform.GuildInvite
		if err := rows.Scan(
			&invite.InviteID,
			&invite.GuildID,
			&invite.GuildName,
			&invite.InviterCharacterID,
			&invite.TargetCharacterID,
			&invite.CreatedAt,
			&invite.ExpiresAt); err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}
	return invites, rows.Err()
}

func (s *Store) DeleteGuildInvite(inviteID string) error {
	result, err := s.db.Exec(`DELETE FROM ac_guild_invites WHERE invite_id = ?`, inviteID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return filestore.ErrGuildInviteMissing
	}
	return nil
}

func (s *Store) CleanupExpiredGuildInvites(nowUnix int64) error {
	_, err := s.db.Exec(
		`UPDATE ac_guild_invites SET state = ?, responded_at = ? WHERE state = ? AND expires_at <= ?`,
		filestore.InviteStateExpired,
		nowUnix,
		filestore.InviteStatePending,
		nowUnix)
	return err
}

func (s *Store) AcceptGuildInvite(inviteID string, targetCharacterID string, options filestore.MutationOptions) (platform.Guild, error) {
	var guild platform.Guild
	err := s.WithTransaction("sqlstore.guild_invite_accept", func(tx *Tx) error {
		if options.MutationKey != "" {
			if err := tx.replaySocialMutation(targetCharacterID, "guild.accept", options.MutationKey, &guild); err == nil && guild.ID != "" {
				return nil
			} else if err != nil {
				return err
			}
		}
		invite, state, err := tx.loadGuildInviteForMutation(inviteID)
		if err != nil {
			return err
		}
		now := tx.store.now().Unix()
		if invite.TargetCharacterID != targetCharacterID {
			return filestore.ErrGuildInviteMissing
		}
		if state != filestore.InviteStatePending || invite.ExpiresAt <= now {
			return filestore.ErrGuildInviteConsumed
		}
		if existing, err := tx.getGuildForCharacter(targetCharacterID); err == nil && existing.ID != "" {
			return filestore.ErrGuildMemberExists
		} else if err != nil && !errors.Is(err, filestore.ErrGuildMissing) {
			return err
		}
		loaded, err := tx.getGuildByID(invite.GuildID)
		if err != nil {
			return err
		}
		target, _, err := tx.loadCharacterForMutation(targetCharacterID)
		if err != nil {
			return err
		}
		loaded.Members = append(loaded.Members, buildGuildMember(target, platform.GuildRankRecruit, now))
		loaded.UpdatedAt = now
		if err := tx.saveGuild(loaded); err != nil {
			return err
		}
		if _, err := tx.tx.Exec(`UPDATE ac_guild_invites SET state = ?, responded_at = ?, mutation_key = ? WHERE invite_id = ?`,
			filestore.InviteStateAccepted, now, options.MutationKey, inviteID); err != nil {
			return err
		}
		guild = loaded
		if options.MutationKey != "" {
			return tx.recordSocialMutation(targetCharacterID, "guild.accept", options.MutationKey, guild)
		}
		return nil
	})
	return cloneGuild(guild), err
}

func (s *Store) DeclineGuildInvite(inviteID string, targetCharacterID string, options filestore.MutationOptions) error {
	now := s.now().Unix()
	result, err := s.db.Exec(
		`UPDATE ac_guild_invites SET state = ?, responded_at = ?, mutation_key = ?
		WHERE invite_id = ? AND target_character_id = ? AND state = ?`,
		filestore.InviteStateDeclined,
		now,
		options.MutationKey,
		inviteID,
		targetCharacterID,
		filestore.InviteStatePending)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return filestore.ErrGuildInviteConsumed
	}
	return nil
}

func (tx *Tx) loadGuildInviteForMutation(inviteID string) (platform.GuildInvite, string, error) {
	row := tx.tx.QueryRow(
		`SELECT invite_id, guild_id, guild_name, inviter_character_id, target_character_id, created_at, expires_at, state
		FROM ac_guild_invites WHERE invite_id = ?`,
		inviteID)
	var invite platform.GuildInvite
	var state string
	if err := row.Scan(
		&invite.InviteID,
		&invite.GuildID,
		&invite.GuildName,
		&invite.InviterCharacterID,
		&invite.TargetCharacterID,
		&invite.CreatedAt,
		&invite.ExpiresAt,
		&state); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return platform.GuildInvite{}, "", filestore.ErrGuildInviteMissing
		}
		return platform.GuildInvite{}, "", err
	}
	return invite, state, nil
}

func (tx *Tx) getGuildByID(guildID string) (platform.Guild, error) {
	row := tx.tx.QueryRow(guildSelectSQL()+` WHERE guild_id = ? AND disbanded_at = 0`, guildID)
	return tx.scanGuild(row)
}

func (tx *Tx) getGuildForCharacter(characterID string) (platform.Guild, error) {
	row := tx.tx.QueryRow(
		guildSelectSQL()+`
		WHERE guild_id = (
			SELECT guild_id FROM ac_guild_members WHERE character_id = ? AND left_at = 0 LIMIT 1
		) AND disbanded_at = 0`,
		characterID)
	return tx.scanGuild(row)
}

func (tx *Tx) scanGuild(row rowScanner) (platform.Guild, error) {
	var guild platform.Guild
	var ranksJSON string
	if err := row.Scan(
		&guild.ID,
		&guild.RealmID,
		&guild.GuildName,
		&guild.CreatedAt,
		&guild.UpdatedAt,
		&guild.CreatedByCharacterID,
		&guild.LeaderCharacterID,
		&guild.MessageOfTheDay,
		&ranksJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return platform.Guild{}, filestore.ErrGuildMissing
		}
		return platform.Guild{}, err
	}
	if err := decodeJSON(ranksJSON, &guild.Ranks); err != nil {
		return platform.Guild{}, err
	}
	members, err := tx.loadGuildMembers(guild.ID)
	if err != nil {
		return platform.Guild{}, err
	}
	guild.Members = members
	return guild, nil
}

func (tx *Tx) loadGuildMembers(guildID string) ([]platform.GuildMember, error) {
	rows, err := tx.tx.Query(
		`SELECT character_id, display_name, race_id, class_id, level, rank_id, joined_at, last_online_at
		FROM ac_guild_members
		WHERE guild_id = ? AND left_at = 0
		ORDER BY joined_at, character_id`,
		guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []platform.GuildMember
	for rows.Next() {
		var member platform.GuildMember
		if err := rows.Scan(
			&member.CharacterID,
			&member.DisplayName,
			&member.RaceID,
			&member.ClassID,
			&member.Level,
			&member.RankID,
			&member.JoinedAt,
			&member.LastOnlineAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func buildGuildMember(character platform.Character, rankID string, joinedAt int64) platform.GuildMember {
	if rankID == "" {
		rankID = platform.GuildRankMember
	}
	return platform.GuildMember{
		CharacterID:  character.ID,
		DisplayName:  character.DisplayName,
		RaceID:       character.RaceID,
		ClassID:      character.ClassID,
		Level:        character.Level,
		RankID:       rankID,
		JoinedAt:     joinedAt,
		LastOnlineAt: character.LastSeenAt,
	}
}

func guildHasMember(members []platform.GuildMember, characterID string) bool {
	for _, member := range members {
		if member.CharacterID == characterID {
			return true
		}
	}
	return false
}
