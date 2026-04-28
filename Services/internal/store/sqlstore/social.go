package sqlstore

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func (s *Store) AddFriend(ownerCharacterID string, friendCharacterID string) (platform.FriendRelationship, error) {
	owner, err := s.GetCharacterByID(ownerCharacterID)
	if err != nil {
		return platform.FriendRelationship{}, err
	}
	friend, err := s.GetCharacterByID(friendCharacterID)
	if err != nil {
		return platform.FriendRelationship{}, err
	}
	if owner.ID == friend.ID {
		return platform.FriendRelationship{}, filestore.ErrFriendSelf
	}
	if owner.RealmID != friend.RealmID {
		return platform.FriendRelationship{}, fmt.Errorf("friend is not on this realm")
	}

	relationship := platform.FriendRelationship{
		OwnerCharacterID:  owner.ID,
		FriendCharacterID: friend.ID,
		FriendDisplayName: friend.DisplayName,
		CreatedAt:         s.now().Unix(),
	}
	_, err = s.db.Exec(
		`INSERT INTO ac_friend_links (owner_character_id, friend_character_id, friend_display_name, created_at)
		VALUES (?, ?, ?, ?)`,
		relationship.OwnerCharacterID,
		relationship.FriendCharacterID,
		relationship.FriendDisplayName,
		relationship.CreatedAt)
	if err != nil {
		if isConstraintError(err) {
			return platform.FriendRelationship{}, filestore.ErrFriendExists
		}
		return platform.FriendRelationship{}, err
	}
	return relationship, nil
}

func (s *Store) RemoveFriend(ownerCharacterID string, friendCharacterID string) error {
	result, err := s.db.Exec(
		`DELETE FROM ac_friend_links WHERE owner_character_id = ? AND friend_character_id = ?`,
		ownerCharacterID,
		friendCharacterID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return filestore.ErrFriendMissing
	}
	return nil
}

func (s *Store) ListFriends(ownerCharacterID string) ([]platform.FriendRelationship, error) {
	rows, err := s.db.Query(
		`SELECT f.owner_character_id, f.friend_character_id, c.display_name, f.created_at
		FROM ac_friend_links f
		JOIN ac_characters c ON c.id = f.friend_character_id
		WHERE f.owner_character_id = ?
		ORDER BY LOWER(c.display_name)`,
		ownerCharacterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []platform.FriendRelationship
	for rows.Next() {
		var relationship platform.FriendRelationship
		if err := rows.Scan(
			&relationship.OwnerCharacterID,
			&relationship.FriendCharacterID,
			&relationship.FriendDisplayName,
			&relationship.CreatedAt); err != nil {
			return nil, err
		}
		relationships = append(relationships, relationship)
	}
	return relationships, rows.Err()
}

func (s *Store) AddIgnore(ownerCharacterID string, ignoredCharacterID string) (filestore.IgnoreRelationship, error) {
	if ownerCharacterID == ignoredCharacterID {
		return filestore.IgnoreRelationship{}, filestore.ErrFriendSelf
	}
	if _, err := s.GetCharacterByID(ownerCharacterID); err != nil {
		return filestore.IgnoreRelationship{}, err
	}
	if _, err := s.GetCharacterByID(ignoredCharacterID); err != nil {
		return filestore.IgnoreRelationship{}, err
	}

	relationship := filestore.IgnoreRelationship{
		OwnerCharacterID:   ownerCharacterID,
		IgnoredCharacterID: ignoredCharacterID,
		CreatedAt:          s.now().Unix(),
	}
	_, err := s.db.Exec(
		`INSERT INTO ac_ignore_links (owner_character_id, ignored_character_id, created_at)
		VALUES (?, ?, ?)`,
		relationship.OwnerCharacterID,
		relationship.IgnoredCharacterID,
		relationship.CreatedAt)
	if err != nil {
		if isConstraintError(err) {
			return filestore.IgnoreRelationship{}, filestore.ErrIgnoreExists
		}
		return filestore.IgnoreRelationship{}, err
	}
	return relationship, nil
}

func (s *Store) RemoveIgnore(ownerCharacterID string, ignoredCharacterID string) error {
	result, err := s.db.Exec(
		`DELETE FROM ac_ignore_links WHERE owner_character_id = ? AND ignored_character_id = ?`,
		ownerCharacterID,
		ignoredCharacterID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return filestore.ErrIgnoreMissing
	}
	return nil
}

func (s *Store) ListIgnores(ownerCharacterID string) ([]filestore.IgnoreRelationship, error) {
	rows, err := s.db.Query(
		`SELECT owner_character_id, ignored_character_id, created_at
		FROM ac_ignore_links
		WHERE owner_character_id = ?
		ORDER BY ignored_character_id`,
		ownerCharacterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []filestore.IgnoreRelationship
	for rows.Next() {
		var relationship filestore.IgnoreRelationship
		if err := rows.Scan(&relationship.OwnerCharacterID, &relationship.IgnoredCharacterID, &relationship.CreatedAt); err != nil {
			return nil, err
		}
		relationships = append(relationships, relationship)
	}
	return relationships, rows.Err()
}

func (s *Store) CreateParty(leaderCharacterID string, memberCharacterIDs []string) (platform.Party, error) {
	members := normalizePartyMembers(leaderCharacterID, memberCharacterIDs)
	now := s.now().Unix()
	party := platform.Party{
		ID:                 randomID("party"),
		LeaderCharacterID:  leaderCharacterID,
		MemberCharacterIDs: members,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.WithTransaction("sqlstore.party_create", func(tx *Tx) error {
		if _, _, err := tx.loadCharacterForMutation(leaderCharacterID); err != nil {
			return err
		}
		return tx.saveParty(party)
	}); err != nil {
		return platform.Party{}, err
	}
	return party, nil
}

func (s *Store) SaveParty(party platform.Party) (platform.Party, error) {
	party.MemberCharacterIDs = normalizePartyMembers(party.LeaderCharacterID, party.MemberCharacterIDs)
	if party.UpdatedAt == 0 {
		party.UpdatedAt = s.now().Unix()
	}
	if len(party.MemberCharacterIDs) == 0 {
		if err := s.DeleteParty(party.ID); err != nil {
			return platform.Party{}, err
		}
		return party, nil
	}
	if !containsStoreString(party.MemberCharacterIDs, party.LeaderCharacterID) {
		party.LeaderCharacterID = party.MemberCharacterIDs[0]
	}
	if err := s.WithTransaction("sqlstore.party_save", func(tx *Tx) error {
		return tx.saveParty(party)
	}); err != nil {
		return platform.Party{}, err
	}
	loaded, err := s.GetPartyByID(party.ID)
	if err != nil {
		return platform.Party{}, err
	}
	return *loaded, nil
}

func (tx *Tx) saveParty(party platform.Party) error {
	now := tx.store.now().Unix()
	if party.ID == "" {
		party.ID = randomID("party")
	}
	if party.CreatedAt == 0 {
		party.CreatedAt = now
	}
	if party.UpdatedAt == 0 {
		party.UpdatedAt = now
	}
	_, err := tx.tx.Exec(
		`INSERT INTO ac_parties (party_id, leader_character_id, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, 1)
		ON CONFLICT(party_id) DO UPDATE SET
			leader_character_id = excluded.leader_character_id,
			updated_at = excluded.updated_at,
			disbanded_at = 0,
			version = ac_parties.version + 1`,
		party.ID,
		party.LeaderCharacterID,
		party.CreatedAt,
		party.UpdatedAt)
	if err != nil {
		return err
	}
	_, err = tx.tx.Exec(`UPDATE ac_party_members SET left_at = ? WHERE party_id = ? AND left_at = 0`, now, party.ID)
	if err != nil {
		return err
	}
	for _, memberID := range party.MemberCharacterIDs {
		if _, _, err := tx.loadCharacterForMutation(memberID); err != nil {
			return err
		}
		roleID := "member"
		if memberID == party.LeaderCharacterID {
			roleID = "leader"
		}
		if _, err := tx.tx.Exec(
			`INSERT INTO ac_party_members (party_id, character_id, role_id, joined_at, left_at)
			VALUES (?, ?, ?, ?, 0)
			ON CONFLICT(party_id, character_id) DO UPDATE SET role_id = excluded.role_id, left_at = 0`,
			party.ID,
			memberID,
			roleID,
			now); err != nil {
			if isConstraintError(err) {
				return filestore.ErrPartyMemberExists
			}
			return err
		}
	}
	return nil
}

func (s *Store) DeleteParty(partyID string) error {
	result, err := s.db.Exec(`UPDATE ac_parties SET disbanded_at = ?, updated_at = ?, version = version + 1 WHERE party_id = ? AND disbanded_at = 0`,
		s.now().Unix(),
		s.now().Unix(),
		partyID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return filestore.ErrPartyMissing
	}
	_, err = s.db.Exec(`UPDATE ac_party_members SET left_at = ? WHERE party_id = ? AND left_at = 0`, s.now().Unix(), partyID)
	return err
}

func (s *Store) GetPartyByID(partyID string) (*platform.Party, error) {
	row := s.db.QueryRow(
		`SELECT party_id, leader_character_id, created_at, updated_at
		FROM ac_parties
		WHERE party_id = ? AND disbanded_at = 0`,
		partyID)
	return s.scanParty(row)
}

func (s *Store) GetPartyForCharacter(characterID string) (*platform.Party, error) {
	row := s.db.QueryRow(
		`SELECT p.party_id, p.leader_character_id, p.created_at, p.updated_at
		FROM ac_parties p
		JOIN ac_party_members m ON m.party_id = p.party_id
		WHERE m.character_id = ? AND m.left_at = 0 AND p.disbanded_at = 0`,
		characterID)
	return s.scanParty(row)
}

func (s *Store) scanParty(row rowScanner) (*platform.Party, error) {
	var party platform.Party
	if err := row.Scan(&party.ID, &party.LeaderCharacterID, &party.CreatedAt, &party.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, filestore.ErrPartyMissing
		}
		return nil, err
	}
	members, err := s.loadPartyMembers(party.ID)
	if err != nil {
		return nil, err
	}
	party.MemberCharacterIDs = members
	return &party, nil
}

func (s *Store) loadPartyMembers(partyID string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT character_id FROM ac_party_members WHERE party_id = ? AND left_at = 0 ORDER BY joined_at, character_id`,
		partyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []string
	for rows.Next() {
		var memberID string
		if err := rows.Scan(&memberID); err != nil {
			return nil, err
		}
		members = append(members, memberID)
	}
	return members, rows.Err()
}

func (s *Store) CreatePartyInvite(invite filestore.PartyInvite) (filestore.PartyInvite, error) {
	if invite.InviteID == "" {
		invite.InviteID = randomID("pinvite")
	}
	if invite.State == "" {
		invite.State = filestore.InviteStatePending
	}
	if invite.CreatedAt == 0 {
		invite.CreatedAt = s.now().Unix()
	}
	if invite.ExpiresAt == 0 {
		invite.ExpiresAt = invite.CreatedAt + 300
	}
	_, err := s.db.Exec(
		`INSERT INTO ac_party_invites (
			invite_id, party_id, inviter_character_id, target_character_id, state, created_at, expires_at, responded_at, mutation_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invite.InviteID,
		invite.PartyID,
		invite.InviterCharacterID,
		invite.TargetCharacterID,
		invite.State,
		invite.CreatedAt,
		invite.ExpiresAt,
		invite.RespondedAt,
		"")
	if err != nil {
		if isConstraintError(err) {
			return filestore.PartyInvite{}, filestore.ErrDuplicateMutation
		}
		return filestore.PartyInvite{}, err
	}
	return invite, nil
}

func (s *Store) AcceptPartyInvite(inviteID string, targetCharacterID string, options filestore.MutationOptions) (platform.Party, error) {
	var party platform.Party
	err := s.WithTransaction("sqlstore.party_invite_accept", func(tx *Tx) error {
		if options.MutationKey != "" {
			if err := tx.replaySocialMutation(targetCharacterID, "party.accept", options.MutationKey, &party); err == nil && party.ID != "" {
				return nil
			} else if err != nil {
				return err
			}
		}

		invite, err := tx.loadPartyInvite(inviteID)
		if err != nil {
			return err
		}
		if invite.TargetCharacterID != targetCharacterID {
			return filestore.ErrPartyInviteMissing
		}
		now := tx.store.now().Unix()
		if invite.State != filestore.InviteStatePending || invite.ExpiresAt <= now {
			return filestore.ErrPartyInviteConsumed
		}
		if existing, err := tx.getPartyForCharacter(targetCharacterID); err == nil && existing.ID != "" {
			return filestore.ErrPartyMemberExists
		} else if err != nil && !errors.Is(err, filestore.ErrPartyMissing) {
			return err
		}

		if invite.PartyID != "" {
			existing, err := tx.getPartyByID(invite.PartyID)
			if err == nil {
				party = existing
			} else if !errors.Is(err, filestore.ErrPartyMissing) {
				return err
			}
		}
		if party.ID == "" {
			existing, err := tx.getPartyForCharacter(invite.InviterCharacterID)
			if err == nil {
				party = existing
			} else if errors.Is(err, filestore.ErrPartyMissing) {
				party = platform.Party{
					ID:                 randomID("party"),
					LeaderCharacterID:  invite.InviterCharacterID,
					MemberCharacterIDs: []string{invite.InviterCharacterID},
					CreatedAt:          now,
					UpdatedAt:          now,
				}
			} else {
				return err
			}
		}
		party.MemberCharacterIDs = normalizePartyMembers(party.LeaderCharacterID, append(party.MemberCharacterIDs, targetCharacterID))
		party.UpdatedAt = now
		if err := tx.saveParty(party); err != nil {
			return err
		}
		if _, err := tx.tx.Exec(`UPDATE ac_party_invites SET state = ?, responded_at = ?, mutation_key = ? WHERE invite_id = ?`,
			filestore.InviteStateAccepted, now, options.MutationKey, inviteID); err != nil {
			return err
		}
		if options.MutationKey != "" {
			return tx.recordSocialMutation(targetCharacterID, "party.accept", options.MutationKey, party)
		}
		return nil
	})
	return party, err
}

func (s *Store) DeclinePartyInvite(inviteID string, targetCharacterID string, options filestore.MutationOptions) error {
	now := s.now().Unix()
	result, err := s.db.Exec(
		`UPDATE ac_party_invites SET state = ?, responded_at = ?, mutation_key = ?
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
		return filestore.ErrPartyInviteConsumed
	}
	return nil
}

func (tx *Tx) loadPartyInvite(inviteID string) (filestore.PartyInvite, error) {
	row := tx.tx.QueryRow(
		`SELECT invite_id, party_id, inviter_character_id, target_character_id, state, created_at, expires_at, responded_at
		FROM ac_party_invites WHERE invite_id = ?`,
		inviteID)
	var invite filestore.PartyInvite
	if err := row.Scan(
		&invite.InviteID,
		&invite.PartyID,
		&invite.InviterCharacterID,
		&invite.TargetCharacterID,
		&invite.State,
		&invite.CreatedAt,
		&invite.ExpiresAt,
		&invite.RespondedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return filestore.PartyInvite{}, filestore.ErrPartyInviteMissing
		}
		return filestore.PartyInvite{}, err
	}
	return invite, nil
}

func (tx *Tx) getPartyByID(partyID string) (platform.Party, error) {
	row := tx.tx.QueryRow(
		`SELECT party_id, leader_character_id, created_at, updated_at FROM ac_parties WHERE party_id = ? AND disbanded_at = 0`,
		partyID)
	return tx.scanParty(row)
}

func (tx *Tx) getPartyForCharacter(characterID string) (platform.Party, error) {
	row := tx.tx.QueryRow(
		`SELECT p.party_id, p.leader_character_id, p.created_at, p.updated_at
		FROM ac_parties p
		JOIN ac_party_members m ON m.party_id = p.party_id
		WHERE m.character_id = ? AND m.left_at = 0 AND p.disbanded_at = 0`,
		characterID)
	return tx.scanParty(row)
}

func (tx *Tx) scanParty(row rowScanner) (platform.Party, error) {
	var party platform.Party
	if err := row.Scan(&party.ID, &party.LeaderCharacterID, &party.CreatedAt, &party.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return platform.Party{}, filestore.ErrPartyMissing
		}
		return platform.Party{}, err
	}
	members, err := tx.loadPartyMembers(party.ID)
	if err != nil {
		return platform.Party{}, err
	}
	party.MemberCharacterIDs = members
	return party, nil
}

func (tx *Tx) loadPartyMembers(partyID string) ([]string, error) {
	rows, err := tx.tx.Query(
		`SELECT character_id FROM ac_party_members WHERE party_id = ? AND left_at = 0 ORDER BY joined_at, character_id`,
		partyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []string
	for rows.Next() {
		var memberID string
		if err := rows.Scan(&memberID); err != nil {
			return nil, err
		}
		members = append(members, memberID)
	}
	return members, rows.Err()
}

func (s *Store) AppendChatMessage(message platform.ChatMessage) (platform.ChatMessage, error) {
	if message.MessageID == "" {
		message.MessageID = randomID("chat")
	}
	if message.Timestamp == 0 {
		message.Timestamp = s.now().Unix()
	}
	scopeID := chatScopeID(message)
	err := s.WithTransaction("sqlstore.chat_append", func(tx *Tx) error {
		var sequence int64
		if err := tx.tx.QueryRow(
			`SELECT COALESCE(MAX(sequence), 0) + 1 FROM ac_chat_messages WHERE channel = ? AND scope_id = ?`,
			message.Channel,
			scopeID).Scan(&sequence); err != nil {
			return err
		}
		_, err := tx.tx.Exec(
			`INSERT INTO ac_chat_messages (
				message_id, channel, scope_id, sender_character_id, sender_display_name, target_character_id,
				party_id, guild_id, zone_id, message_text, sequence, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			message.MessageID,
			message.Channel,
			scopeID,
			message.SenderCharacterID,
			message.SenderDisplayName,
			message.TargetCharacterID,
			message.PartyID,
			message.GuildID,
			message.ZoneID,
			message.MessageText,
			sequence,
			message.Timestamp)
		return err
	})
	return message, err
}

func (s *Store) ListRecentChatMessages(channel string, scopeID string, limit int) ([]platform.ChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT message_id, channel, sender_character_id, sender_display_name, target_character_id, party_id, guild_id, zone_id, message_text, created_at
		FROM ac_chat_messages
		WHERE channel = ? AND scope_id = ?
		ORDER BY sequence DESC
		LIMIT ?`,
		channel,
		scopeID,
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []platform.ChatMessage
	for rows.Next() {
		var message platform.ChatMessage
		if err := rows.Scan(
			&message.MessageID,
			&message.Channel,
			&message.SenderCharacterID,
			&message.SenderDisplayName,
			&message.TargetCharacterID,
			&message.PartyID,
			&message.GuildID,
			&message.ZoneID,
			&message.MessageText,
			&message.Timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, nil
}

func chatScopeID(message platform.ChatMessage) string {
	switch {
	case message.TargetCharacterID != "":
		return "target:" + message.TargetCharacterID
	case message.PartyID != "":
		return "party:" + message.PartyID
	case message.GuildID != "":
		return "guild:" + message.GuildID
	case message.ZoneID != "":
		return "zone:" + message.ZoneID
	default:
		return "global"
	}
}

func normalizePartyMembers(leaderCharacterID string, memberCharacterIDs []string) []string {
	seen := map[string]struct{}{}
	members := make([]string, 0, len(memberCharacterIDs)+1)
	if leaderCharacterID != "" {
		seen[leaderCharacterID] = struct{}{}
		members = append(members, leaderCharacterID)
	}
	for _, memberID := range memberCharacterIDs {
		memberID = strings.TrimSpace(memberID)
		if memberID == "" {
			continue
		}
		if _, ok := seen[memberID]; ok {
			continue
		}
		seen[memberID] = struct{}{}
		members = append(members, memberID)
	}
	return members
}

func containsStoreString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func cloneGuild(source platform.Guild) platform.Guild {
	source.Ranks = append([]platform.GuildRank(nil), source.Ranks...)
	for index := range source.Ranks {
		source.Ranks[index].Permissions = append([]string(nil), source.Ranks[index].Permissions...)
	}
	source.Members = append([]platform.GuildMember(nil), source.Members...)
	return source
}

func normalizeGuildMembers(members []platform.GuildMember) []platform.GuildMember {
	seen := map[string]struct{}{}
	normalized := make([]platform.GuildMember, 0, len(members))
	for _, member := range members {
		if member.CharacterID == "" {
			continue
		}
		if _, exists := seen[member.CharacterID]; exists {
			continue
		}
		seen[member.CharacterID] = struct{}{}
		if member.RankID == "" {
			member.RankID = platform.GuildRankMember
		}
		normalized = append(normalized, member)
	}
	sort.SliceStable(normalized, func(left int, right int) bool {
		return normalized[left].JoinedAt < normalized[right].JoinedAt
	})
	return normalized
}

func (tx *Tx) replaySocialMutation(actorCharacterID string, operation string, mutationKey string, out any) error {
	row := tx.tx.QueryRow(
		`SELECT response_json FROM ac_social_mutations WHERE actor_character_id = ? AND operation = ? AND mutation_key = ?`,
		actorCharacterID,
		operation,
		mutationKey)
	var responseJSON string
	if err := row.Scan(&responseJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return decodeJSON(responseJSON, out)
}

func (tx *Tx) recordSocialMutation(actorCharacterID string, operation string, mutationKey string, response any) error {
	payload, err := encodeJSON(response)
	if err != nil {
		return err
	}
	_, err = tx.tx.Exec(
		`INSERT INTO ac_social_mutations (mutation_id, actor_character_id, operation, mutation_key, response_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		randomID("smut"),
		actorCharacterID,
		operation,
		mutationKey,
		payload,
		tx.store.now().Unix())
	if err != nil && isConstraintError(err) {
		return filestore.ErrDuplicateMutation
	}
	return err
}
