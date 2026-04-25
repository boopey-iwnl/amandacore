package worlds

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/platform"
)

func (s *worldServer) handleGuildCreate(w http.ResponseWriter, r *http.Request) {
	var request guildCreateRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if s.store == nil {
		httpapi.Error(w, http.StatusServiceUnavailable, "store_unavailable", "Guild persistence is unavailable.")
		return
	}
	guildName, err := validateGuildName(request.GuildName)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_name_invalid", err.Error())
		return
	}
	guild, err := s.store.CreateGuild(guildName, session.CharacterID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_create_failed", err.Error())
		return
	}
	s.sendSystemMessageLocked(fmt.Sprintf("Guild %s created.", guild.GuildName), recipientSet(session.CharacterID))
	httpapi.WriteJSON(w, http.StatusCreated, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handleGuildInvite(w http.ResponseWriter, r *http.Request) {
	var request guildInviteRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if s.store == nil {
		httpapi.Error(w, http.StatusServiceUnavailable, "store_unavailable", "Guild persistence is unavailable.")
		return
	}
	_ = s.store.CleanupExpiredGuildInvites(time.Now().Unix())
	guild, err := s.store.GetGuildForCharacter(session.CharacterID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_missing", "You are not in a guild.")
		return
	}
	if !guildMemberHasPermission(*guild, session.CharacterID, platform.GuildPermissionInviteMember) {
		httpapi.Error(w, http.StatusBadRequest, "guild_permission_denied", "You do not have permission to invite guild members.")
		return
	}
	target, err := s.store.GetCharacterByName(session.RealmID, request.TargetName)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_invite_target_missing", "Target character not found.")
		return
	}
	if target.ID == session.CharacterID {
		httpapi.Error(w, http.StatusBadRequest, "guild_invite_self", "Cannot invite yourself.")
		return
	}
	if targetGuild, err := s.store.GetGuildForCharacter(target.ID); err == nil && targetGuild != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_target_already_member", "Target player is already in a guild.")
		return
	}
	invite, err := s.store.CreateGuildInvite(guild.ID, session.CharacterID, target.ID, time.Now().Add(guildInviteTTL).Unix())
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_invite_failed", err.Error())
		return
	}
	s.sendSystemMessageLocked(fmt.Sprintf("Guild invite sent to %s.", target.DisplayName), recipientSet(session.CharacterID))
	s.sendSystemMessageLocked(fmt.Sprintf("%s invited you to join %s.", session.DisplayName, invite.GuildName), recipientSet(target.ID))
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handleGuildAccept(w http.ResponseWriter, r *http.Request) {
	var request guildInviteActionRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	guild, recipients, err := s.acceptGuildInviteLocked(session, request.InviteID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_accept_failed", err.Error())
		return
	}
	s.sendSystemMessageLocked(fmt.Sprintf("%s joined %s.", session.DisplayName, guild.GuildName), recipients)
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handleGuildDecline(w http.ResponseWriter, r *http.Request) {
	var request guildInviteActionRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if s.store == nil {
		httpapi.Error(w, http.StatusServiceUnavailable, "store_unavailable", "Guild persistence is unavailable.")
		return
	}
	_ = s.store.CleanupExpiredGuildInvites(time.Now().Unix())
	invite, err := s.store.GetGuildInvite(request.InviteID)
	if err != nil || invite.TargetCharacterID != session.CharacterID {
		httpapi.Error(w, http.StatusBadRequest, "guild_invite_missing", "Guild invite was not found.")
		return
	}
	if err := s.store.DeleteGuildInvite(invite.InviteID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_decline_failed", err.Error())
		return
	}
	s.sendSystemMessageLocked("Guild invite declined.", recipientSet(session.CharacterID, invite.InviterCharacterID))
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handleGuildLeave(w http.ResponseWriter, r *http.Request) {
	var request partyActionRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	guild, err := s.store.GetGuildForCharacter(session.CharacterID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_missing", "You are not in a guild.")
		return
	}
	if guild.LeaderCharacterID == session.CharacterID {
		httpapi.Error(w, http.StatusBadRequest, "guild_leader_leave_blocked", "Guild leaders must disband the guild.")
		return
	}
	recipients := guildRecipientSet(*guild)
	guild.Members = removeGuildMember(guild.Members, session.CharacterID)
	if _, err := s.store.SaveGuild(*guild); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_leave_failed", err.Error())
		return
	}
	s.sendSystemMessageLocked(fmt.Sprintf("%s left the guild.", session.DisplayName), recipients)
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handleGuildDisband(w http.ResponseWriter, r *http.Request) {
	var request partyActionRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	guild, err := s.store.GetGuildForCharacter(session.CharacterID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_missing", "You are not in a guild.")
		return
	}
	if !guildMemberHasPermission(*guild, session.CharacterID, platform.GuildPermissionDisbandGuild) {
		httpapi.Error(w, http.StatusBadRequest, "guild_permission_denied", "You do not have permission to disband the guild.")
		return
	}
	recipients := guildRecipientSet(*guild)
	if err := s.store.DeleteGuild(guild.ID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_disband_failed", err.Error())
		return
	}
	s.sendSystemMessageLocked(fmt.Sprintf("%s disbanded.", guild.GuildName), recipients)
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handleGuildPromote(w http.ResponseWriter, r *http.Request) {
	s.handleGuildRankChange(w, r, true)
}

func (s *worldServer) handleGuildDemote(w http.ResponseWriter, r *http.Request) {
	s.handleGuildRankChange(w, r, false)
}

func (s *worldServer) handleGuildRemove(w http.ResponseWriter, r *http.Request) {
	var request guildMemberActionRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	guild, target, err := s.resolveGuildTargetLocked(session, request.TargetName)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_remove_failed", err.Error())
		return
	}
	if target.CharacterID == session.CharacterID {
		httpapi.Error(w, http.StatusBadRequest, "guild_remove_self", "Use leave guild instead.")
		return
	}
	if !guildMemberHasPermission(*guild, session.CharacterID, platform.GuildPermissionRemoveMember) ||
		!guildCanActOnMember(*guild, session.CharacterID, target.CharacterID) {
		httpapi.Error(w, http.StatusBadRequest, "guild_permission_denied", "You do not have permission to remove that guild member.")
		return
	}
	recipients := guildRecipientSet(*guild)
	guild.Members = removeGuildMember(guild.Members, target.CharacterID)
	if _, err := s.store.SaveGuild(*guild); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_remove_failed", err.Error())
		return
	}
	s.sendSystemMessageLocked(fmt.Sprintf("%s was removed from the guild.", target.DisplayName), recipients)
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handleGuildMOTD(w http.ResponseWriter, r *http.Request) {
	var request guildMOTDRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	guild, err := s.store.GetGuildForCharacter(session.CharacterID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_missing", "You are not in a guild.")
		return
	}
	if !guildMemberHasPermission(*guild, session.CharacterID, platform.GuildPermissionEditMOTD) {
		httpapi.Error(w, http.StatusBadRequest, "guild_permission_denied", "You do not have permission to edit the guild message.")
		return
	}
	guild.MessageOfTheDay = strings.TrimSpace(request.MessageOfTheDay)
	if len(guild.MessageOfTheDay) > 160 {
		httpapi.Error(w, http.StatusBadRequest, "guild_motd_too_long", "Guild message of the day cannot exceed 160 characters.")
		return
	}
	if _, err := s.store.SaveGuild(*guild); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_motd_failed", err.Error())
		return
	}
	s.sendSystemMessageLocked("Guild message of the day updated.", guildRecipientSet(*guild))
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) buildGuildResponseLocked(characterID string) *guildResponse {
	if s.store == nil {
		return nil
	}
	guild, err := s.store.GetGuildForCharacter(characterID)
	if err != nil {
		return nil
	}

	currentMember, _ := guildMemberByID(*guild, characterID)
	currentRank := guildRankByID(*guild, currentMember.RankID)
	response := &guildResponse{
		GuildID:              guild.ID,
		GuildName:            guild.GuildName,
		LeaderCharacterID:    guild.LeaderCharacterID,
		MessageOfTheDay:      guild.MessageOfTheDay,
		CurrentRankID:        currentMember.RankID,
		CurrentPermissions:   append([]string(nil), currentRank.Permissions...),
		Ranks:                append([]platform.GuildRank(nil), guild.Ranks...),
		Members:              make([]guildMemberResponse, 0, len(guild.Members)),
		CreatedAt:            guild.CreatedAt,
		CreatedByCharacterID: guild.CreatedByCharacterID,
	}
	for _, member := range guild.Members {
		character, err := s.store.GetCharacterByID(member.CharacterID)
		rank := guildRankByID(*guild, member.RankID)
		memberResponse := guildMemberResponse{
			CharacterID:  member.CharacterID,
			DisplayName:  member.DisplayName,
			RaceID:       member.RaceID,
			ClassID:      member.ClassID,
			Level:        member.Level,
			RankID:       member.RankID,
			RankName:     rank.DisplayName,
			JoinedAt:     member.JoinedAt,
			LastOnlineAt: member.LastOnlineAt,
		}
		if err == nil {
			memberResponse.DisplayName = character.DisplayName
			memberResponse.RaceID = character.RaceID
			memberResponse.ClassID = character.ClassID
			memberResponse.Level = character.Level
			memberResponse.LastOnlineAt = character.LastSeenAt
			memberResponse.CurrentZoneID = character.ZoneID
		}
		if memberSession := s.findConnectedSessionByCharacterLocked(member.CharacterID); memberSession != nil {
			memberResponse.Online = true
			memberResponse.DisplayName = memberSession.DisplayName
			memberResponse.ClassID = memberSession.ClassID
			memberResponse.Level = memberSession.Level
			memberResponse.CurrentZoneID = memberSession.ZoneID
			memberResponse.LastOnlineAt = memberSession.LastSeenAt
		}
		response.Members = append(response.Members, memberResponse)
	}
	sort.Slice(response.Members, func(left int, right int) bool {
		leftRank := guildRankByID(*guild, response.Members[left].RankID)
		rightRank := guildRankByID(*guild, response.Members[right].RankID)
		if leftRank.Priority != rightRank.Priority {
			return leftRank.Priority < rightRank.Priority
		}
		return strings.ToLower(response.Members[left].DisplayName) < strings.ToLower(response.Members[right].DisplayName)
	})
	return response
}

func (s *worldServer) buildGuildInviteResponsesLocked(characterID string) []guildInviteResponse {
	if s.store == nil {
		return []guildInviteResponse{}
	}
	invites, err := s.store.ListGuildInvitesForCharacter(characterID)
	if err != nil {
		return []guildInviteResponse{}
	}
	responses := make([]guildInviteResponse, 0, len(invites))
	for _, invite := range invites {
		inviter, err := s.store.GetCharacterByID(invite.InviterCharacterID)
		if err != nil {
			continue
		}
		responses = append(responses, guildInviteResponse{
			InviteID:           invite.InviteID,
			GuildID:            invite.GuildID,
			GuildName:          invite.GuildName,
			InviterCharacterID: invite.InviterCharacterID,
			InviterDisplayName: inviter.DisplayName,
			ExpiresAt:          invite.ExpiresAt,
		})
	}
	return responses
}

func (s *worldServer) acceptGuildInviteLocked(session *worldSessionState, inviteID string) (*platform.Guild, map[string]struct{}, error) {
	if s.store == nil {
		return nil, nil, fmt.Errorf("guild persistence is unavailable")
	}
	if err := s.store.CleanupExpiredGuildInvites(time.Now().Unix()); err != nil {
		return nil, nil, err
	}
	invite, err := s.store.GetGuildInvite(inviteID)
	if err != nil || invite.TargetCharacterID != session.CharacterID {
		return nil, nil, fmt.Errorf("guild invite was not found")
	}
	if _, err := s.store.GetGuildForCharacter(session.CharacterID); err == nil {
		return nil, nil, fmt.Errorf("you are already in a guild")
	}
	guild, err := s.store.GetGuildByID(invite.GuildID)
	if err != nil {
		_ = s.store.DeleteGuildInvite(invite.InviteID)
		return nil, nil, fmt.Errorf("guild is no longer available")
	}
	character, err := s.store.GetCharacterByID(session.CharacterID)
	if err != nil {
		return nil, nil, err
	}
	if character.RealmID != guild.RealmID {
		return nil, nil, fmt.Errorf("target player is not on this realm")
	}

	guild.Members = append(guild.Members, platform.GuildMember{
		CharacterID:  character.ID,
		DisplayName:  character.DisplayName,
		RaceID:       character.RaceID,
		ClassID:      character.ClassID,
		Level:        character.Level,
		RankID:       platform.GuildRankRecruit,
		JoinedAt:     time.Now().Unix(),
		LastOnlineAt: character.LastSeenAt,
	})
	saved, err := s.store.SaveGuild(*guild)
	if err != nil {
		return nil, nil, err
	}
	_ = s.store.DeleteGuildInvite(invite.InviteID)
	recipients := guildRecipientSet(saved)
	return &saved, recipients, nil
}

func (s *worldServer) handleGuildRankChange(w http.ResponseWriter, r *http.Request, promote bool) {
	var request guildMemberActionRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	guild, target, err := s.resolveGuildTargetLocked(session, request.TargetName)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_rank_failed", err.Error())
		return
	}
	if target.CharacterID == session.CharacterID {
		httpapi.Error(w, http.StatusBadRequest, "guild_rank_self", "You cannot change your own guild rank.")
		return
	}

	permission := platform.GuildPermissionDemoteMember
	if promote {
		permission = platform.GuildPermissionPromoteMember
	}
	if !guildMemberHasPermission(*guild, session.CharacterID, permission) ||
		!guildCanActOnMember(*guild, session.CharacterID, target.CharacterID) {
		httpapi.Error(w, http.StatusBadRequest, "guild_permission_denied", "You do not have permission to change that guild member's rank.")
		return
	}

	nextRank, ok := nextGuildRank(*guild, target.RankID, promote)
	if !ok || nextRank.RankID == platform.GuildRankLeader || !guildCanAssignRank(*guild, session.CharacterID, nextRank.RankID) {
		httpapi.Error(w, http.StatusBadRequest, "guild_rank_invalid", "That rank change is not allowed.")
		return
	}
	for index, member := range guild.Members {
		if member.CharacterID == target.CharacterID {
			guild.Members[index].RankID = nextRank.RankID
			break
		}
	}
	if _, err := s.store.SaveGuild(*guild); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "guild_rank_failed", err.Error())
		return
	}
	action := "promoted"
	if !promote {
		action = "demoted"
	}
	s.sendSystemMessageLocked(fmt.Sprintf("%s was %s to %s.", target.DisplayName, action, nextRank.DisplayName), guildRecipientSet(*guild))
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) resolveGuildTargetLocked(session *worldSessionState, targetName string) (*platform.Guild, platform.GuildMember, error) {
	if s.store == nil {
		return nil, platform.GuildMember{}, fmt.Errorf("guild persistence is unavailable")
	}
	guild, err := s.store.GetGuildForCharacter(session.CharacterID)
	if err != nil {
		return nil, platform.GuildMember{}, fmt.Errorf("you are not in a guild")
	}
	targetCharacter, err := s.store.GetCharacterByName(session.RealmID, targetName)
	if err != nil {
		return nil, platform.GuildMember{}, fmt.Errorf("target character not found")
	}
	target, ok := guildMemberByID(*guild, targetCharacter.ID)
	if !ok {
		return nil, platform.GuildMember{}, fmt.Errorf("target is not in your guild")
	}
	return guild, target, nil
}

func guildRecipientSet(guild platform.Guild) map[string]struct{} {
	recipients := map[string]struct{}{}
	for _, member := range guild.Members {
		if strings.TrimSpace(member.CharacterID) == "" {
			continue
		}
		recipients[member.CharacterID] = struct{}{}
	}
	return recipients
}

func (s *worldServer) onlineGuildRecipientSetLocked(guild platform.Guild) map[string]struct{} {
	recipients := map[string]struct{}{}
	for _, member := range guild.Members {
		if s.findConnectedSessionByCharacterLocked(member.CharacterID) == nil {
			continue
		}
		recipients[member.CharacterID] = struct{}{}
	}
	return recipients
}

func validateGuildName(name string) (string, error) {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	if len(trimmed) < guildNameMinLength {
		return "", fmt.Errorf("guild name must be at least %d characters", guildNameMinLength)
	}
	if len(trimmed) > guildNameMaxLength {
		return "", fmt.Errorf("guild name cannot exceed %d characters", guildNameMaxLength)
	}
	for _, r := range trimmed {
		if r < 32 || r == 127 {
			return "", fmt.Errorf("guild name contains unsupported characters")
		}
	}
	return trimmed, nil
}

func guildMemberByID(guild platform.Guild, characterID string) (platform.GuildMember, bool) {
	for _, member := range guild.Members {
		if member.CharacterID == characterID {
			return member, true
		}
	}
	return platform.GuildMember{}, false
}

func guildRankByID(guild platform.Guild, rankID string) platform.GuildRank {
	for _, rank := range guild.Ranks {
		if rank.RankID == rankID {
			return rank
		}
	}
	defaultRanks := platform.DefaultGuildRanks()
	for _, rank := range defaultRanks {
		if rank.RankID == rankID {
			return rank
		}
	}
	return defaultRanks[len(defaultRanks)-1]
}

func guildMemberHasPermission(guild platform.Guild, characterID string, permission string) bool {
	member, ok := guildMemberByID(guild, characterID)
	if !ok {
		return false
	}
	rank := guildRankByID(guild, member.RankID)
	for _, allowed := range rank.Permissions {
		if allowed == permission {
			return true
		}
	}
	return false
}

func guildCanActOnMember(guild platform.Guild, actorCharacterID string, targetCharacterID string) bool {
	actor, ok := guildMemberByID(guild, actorCharacterID)
	if !ok {
		return false
	}
	target, ok := guildMemberByID(guild, targetCharacterID)
	if !ok {
		return false
	}
	if target.RankID == platform.GuildRankLeader {
		return false
	}
	return guildRankByID(guild, actor.RankID).Priority < guildRankByID(guild, target.RankID).Priority
}

func guildCanAssignRank(guild platform.Guild, actorCharacterID string, rankID string) bool {
	actor, ok := guildMemberByID(guild, actorCharacterID)
	if !ok {
		return false
	}
	if rankID == platform.GuildRankLeader {
		return false
	}
	return guildRankByID(guild, actor.RankID).Priority < guildRankByID(guild, rankID).Priority
}

func nextGuildRank(guild platform.Guild, currentRankID string, promote bool) (platform.GuildRank, bool) {
	current := guildRankByID(guild, currentRankID)
	ranks := append([]platform.GuildRank(nil), guild.Ranks...)
	sort.Slice(ranks, func(left int, right int) bool {
		return ranks[left].Priority < ranks[right].Priority
	})
	for index, rank := range ranks {
		if rank.RankID != current.RankID {
			continue
		}
		if promote {
			if index <= 0 {
				return platform.GuildRank{}, false
			}
			return ranks[index-1], true
		}
		if index >= len(ranks)-1 {
			return platform.GuildRank{}, false
		}
		return ranks[index+1], true
	}
	return platform.GuildRank{}, false
}

func removeGuildMember(members []platform.GuildMember, characterID string) []platform.GuildMember {
	remaining := make([]platform.GuildMember, 0, len(members))
	for _, member := range members {
		if member.CharacterID == characterID {
			continue
		}
		remaining = append(remaining, member)
	}
	return remaining
}
