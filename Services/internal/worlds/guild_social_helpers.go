package worlds

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/platform"
)

func validateGuildName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) < guildNameMinLength || len(trimmed) > guildNameMaxLength {
		return "", fmt.Errorf("guild name must be %d-%d characters", guildNameMinLength, guildNameMaxLength)
	}
	for _, r := range trimmed {
		if r == ' ' || r == '-' || r == '\'' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			continue
		}
		return "", fmt.Errorf("guild name can only use letters, numbers, spaces, hyphens, and apostrophes")
	}
	return trimmed, nil
}

func guildRankByID(guild platform.Guild, rankID string) platform.GuildRank {
	ranks := guild.Ranks
	if len(ranks) == 0 {
		ranks = platform.DefaultGuildRanks()
	}
	for _, rank := range ranks {
		if rank.RankID == rankID {
			return rank
		}
	}
	for _, rank := range ranks {
		if rank.RankID == platform.GuildRankMember {
			return rank
		}
	}
	return platform.GuildRank{RankID: platform.GuildRankMember, DisplayName: "Member", Priority: 2}
}

func guildMemberHasPermission(guild platform.Guild, characterID string, permission string) bool {
	for _, member := range guild.Members {
		if member.CharacterID != characterID {
			continue
		}
		rank := guildRankByID(guild, member.RankID)
		for _, candidate := range rank.Permissions {
			if candidate == permission {
				return true
			}
		}
		return false
	}
	return false
}

func guildCanActOnMember(guild platform.Guild, actorCharacterID string, targetCharacterID string) bool {
	actorRank, actorFound := guildMemberRank(guild, actorCharacterID)
	targetRank, targetFound := guildMemberRank(guild, targetCharacterID)
	if !actorFound || !targetFound || actorCharacterID == targetCharacterID {
		return false
	}
	return actorRank.Priority < targetRank.Priority
}

func guildMemberRank(guild platform.Guild, characterID string) (platform.GuildRank, bool) {
	for _, member := range guild.Members {
		if member.CharacterID == characterID {
			return guildRankByID(guild, member.RankID), true
		}
	}
	return platform.GuildRank{}, false
}

func guildRecipientSet(guild platform.Guild) map[string]struct{} {
	recipients := map[string]struct{}{}
	for _, member := range guild.Members {
		if member.CharacterID != "" {
			recipients[member.CharacterID] = struct{}{}
		}
	}
	return recipients
}

func removeGuildMember(members []platform.GuildMember, characterID string) []platform.GuildMember {
	next := make([]platform.GuildMember, 0, len(members))
	for _, member := range members {
		if member.CharacterID != characterID {
			next = append(next, member)
		}
	}
	return next
}

func (s *worldServer) acceptGuildInviteLocked(session *worldSessionState, inviteID string) (*platform.Guild, map[string]struct{}, error) {
	if s.store == nil {
		return nil, nil, fmt.Errorf("social persistence is unavailable")
	}
	if err := s.store.CleanupExpiredGuildInvites(time.Now().Unix()); err != nil {
		return nil, nil, err
	}
	invite, err := s.store.GetGuildInvite(inviteID)
	if err != nil || invite.TargetCharacterID != session.CharacterID {
		return nil, nil, fmt.Errorf("guild invite was not found")
	}
	if existing, err := s.store.GetGuildForCharacter(session.CharacterID); err == nil && existing != nil {
		return nil, nil, fmt.Errorf("you are already in a guild")
	}
	guild, err := s.store.GetGuildByID(invite.GuildID)
	if err != nil {
		return nil, nil, err
	}
	character, err := s.store.GetCharacterByID(session.CharacterID)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now().Unix()
	guild.Members = append(guild.Members, platform.GuildMember{
		CharacterID:  character.ID,
		DisplayName:  character.DisplayName,
		RaceID:       character.RaceID,
		ClassID:      character.ClassID,
		Level:        character.Level,
		RankID:       platform.GuildRankRecruit,
		JoinedAt:     now,
		LastOnlineAt: now,
	})
	saved, err := s.store.SaveGuild(*guild)
	if err != nil {
		return nil, nil, err
	}
	_ = s.store.DeleteGuildInvite(invite.InviteID)
	return &saved, guildRecipientSet(saved), nil
}

func (s *worldServer) resolveGuildTargetLocked(session *worldSessionState, targetName string) (*platform.Guild, *platform.GuildMember, error) {
	if s.store == nil {
		return nil, nil, fmt.Errorf("social persistence is unavailable")
	}
	guild, err := s.store.GetGuildForCharacter(session.CharacterID)
	if err != nil {
		return nil, nil, fmt.Errorf("you are not in a guild")
	}
	target, err := s.store.GetCharacterByName(session.RealmID, targetName)
	if err != nil {
		return nil, nil, fmt.Errorf("target character not found")
	}
	for index := range guild.Members {
		if guild.Members[index].CharacterID == target.ID {
			return guild, &guild.Members[index], nil
		}
	}
	return nil, nil, fmt.Errorf("target character is not in your guild")
}

func (s *worldServer) handleGuildRankChange(w http.ResponseWriter, r *http.Request, promote bool) {
	_ = promote
	httpapi.Error(w, http.StatusBadRequest, "guild_unavailable", "Guild rank changes are not available.")
}
