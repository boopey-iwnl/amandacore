package worlds

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
	storepkg "amandacore/services/internal/store"
)

const (
	chatChannelSystem  = "system"
	chatChannelSay     = "say"
	chatChannelWhisper = "whisper"
	chatChannelParty   = "party"

	maxChatMessageLength = 256
	chatRingLimit        = 200
	sayChatRadius        = 40.0
	partyInviteTTL       = 60 * time.Second
	partySizeLimit       = 5
)

type chatEnvelope struct {
	Message               platform.ChatMessage
	Sequence              int64
	RecipientCharacterIDs map[string]struct{}
}

type partyInviteState struct {
	InviteID           string
	InviterCharacterID string
	TargetCharacterID  string
	PartyID            string
	CreatedAt          int64
	ExpiresAt          int64
}

type chatSendRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	Channel           string `json:"channel"`
	TargetName        string `json:"targetName"`
	MessageText       string `json:"messageText"`
}

type friendNameRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	Name              string `json:"name"`
}

type partyInviteRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	TargetName        string `json:"targetName"`
	TargetCharacterID string `json:"targetCharacterId"`
}

type partyInviteActionRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	InviteID          string `json:"inviteId"`
}

type partyActionRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
}

type socialStateResponse struct {
	ChatMessages []platform.ChatMessage `json:"chatMessages"`
	Friends      []friendResponse       `json:"friends"`
	Party        *partyResponse         `json:"party"`
	PartyInvites []partyInviteResponse  `json:"partyInvites"`
}

type friendResponse struct {
	CharacterID string `json:"characterId"`
	DisplayName string `json:"displayName"`
	Level       int    `json:"level"`
	ClassID     string `json:"classId"`
	ZoneID      string `json:"zoneId"`
	Online      bool   `json:"online"`
}

type partyResponse struct {
	PartyID           string                `json:"partyId"`
	LeaderCharacterID string                `json:"leaderCharacterId"`
	Members           []partyMemberResponse `json:"members"`
}

type partyMemberResponse struct {
	CharacterID  string  `json:"characterId"`
	DisplayName  string  `json:"displayName"`
	Level        int     `json:"level"`
	ClassID      string  `json:"classId"`
	ZoneID       string  `json:"zoneId"`
	Online       bool    `json:"online"`
	Leader       bool    `json:"leader"`
	Health       float64 `json:"health"`
	MaxHealth    float64 `json:"maxHealth"`
	Resource     float64 `json:"resource"`
	MaxResource  float64 `json:"maxResource"`
	Disconnected bool    `json:"disconnected"`
}

type partyInviteResponse struct {
	InviteID           string `json:"inviteId"`
	PartyID            string `json:"partyId"`
	InviterCharacterID string `json:"inviterCharacterId"`
	InviterDisplayName string `json:"inviterDisplayName"`
	ExpiresAt          int64  `json:"expiresAt"`
}

func (s *worldServer) handleSocialState(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("worldSessionToken")
	if token == "" {
		httpapi.Error(w, http.StatusBadRequest, "missing_token", "worldSessionToken query parameter is required.")
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[token]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	s.cleanupExpiredPartyInvitesLocked(time.Now())
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, r.URL.Query().Get("afterMessageId")))
}

func (s *worldServer) handleChatSend(w http.ResponseWriter, r *http.Request) {
	var request chatSendRequest
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

	if err := s.sendChatMessageLocked(session, request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "chat_send_failed", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handleFriendAdd(w http.ResponseWriter, r *http.Request) {
	var request friendNameRequest
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
		httpapi.Error(w, http.StatusServiceUnavailable, "store_unavailable", "Social persistence is unavailable.")
		return
	}

	target, err := s.store.GetCharacterByName(session.RealmID, request.Name)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "friend_target_missing", err.Error())
		return
	}
	if _, err := s.store.AddFriend(session.CharacterID, target.ID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "friend_add_failed", err.Error())
		return
	}

	s.sendSystemMessageLocked(
		fmt.Sprintf("%s added to friends.", target.DisplayName),
		recipientSet(session.CharacterID))
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handleFriendRemove(w http.ResponseWriter, r *http.Request) {
	var request friendNameRequest
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
		httpapi.Error(w, http.StatusServiceUnavailable, "store_unavailable", "Social persistence is unavailable.")
		return
	}

	target, err := s.store.GetCharacterByName(session.RealmID, request.Name)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "friend_target_missing", err.Error())
		return
	}
	if err := s.store.RemoveFriend(session.CharacterID, target.ID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "friend_remove_failed", err.Error())
		return
	}

	s.sendSystemMessageLocked(
		fmt.Sprintf("%s removed from friends.", target.DisplayName),
		recipientSet(session.CharacterID))
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handlePartyInvite(w http.ResponseWriter, r *http.Request) {
	var request partyInviteRequest
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
	target, err := s.resolvePartyInviteTargetLocked(session, request)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "party_invite_failed", err.Error())
		return
	}

	targetSession := s.findConnectedSessionByCharacterLocked(target.ID)
	if targetSession == nil {
		httpapi.Error(w, http.StatusBadRequest, "party_target_offline", "Target player is not online.")
		return
	}

	inviterParty, err := s.store.GetPartyForCharacter(session.CharacterID)
	if err != nil && !errors.Is(err, storepkg.ErrPartyMissing) {
		httpapi.Error(w, http.StatusBadRequest, "party_lookup_failed", err.Error())
		return
	}
	if inviterParty != nil && inviterParty.LeaderCharacterID != session.CharacterID {
		httpapi.Error(w, http.StatusBadRequest, "party_not_leader", "Only the party leader can invite players.")
		return
	}
	if inviterParty != nil && len(inviterParty.MemberCharacterIDs) >= partySizeLimit {
		httpapi.Error(w, http.StatusBadRequest, "party_full", "Party is full.")
		return
	}
	if targetParty, err := s.store.GetPartyForCharacter(target.ID); err == nil && targetParty != nil {
		httpapi.Error(w, http.StatusBadRequest, "party_target_grouped", "Target player is already in a party.")
		return
	}

	s.cleanupExpiredPartyInvitesLocked(time.Now())
	for _, invite := range s.partyInvites {
		if invite.InviterCharacterID == session.CharacterID && invite.TargetCharacterID == target.ID {
			httpapi.Error(w, http.StatusBadRequest, "party_invite_duplicate", "An invite is already pending for that player.")
			return
		}
	}

	now := time.Now()
	s.partyInviteCounter++
	invite := partyInviteState{
		InviteID:           fmt.Sprintf("invite_%06d", s.partyInviteCounter),
		InviterCharacterID: session.CharacterID,
		TargetCharacterID:  target.ID,
		CreatedAt:          now.Unix(),
		ExpiresAt:          now.Add(partyInviteTTL).Unix(),
	}
	if inviterParty != nil {
		invite.PartyID = inviterParty.ID
	}
	s.partyInvites[invite.InviteID] = invite

	s.sendSystemMessageLocked(
		fmt.Sprintf("Party invite sent to %s.", target.DisplayName),
		recipientSet(session.CharacterID))
	s.sendSystemMessageLocked(
		fmt.Sprintf("%s invited you to a party.", session.DisplayName),
		recipientSet(target.ID))

	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handlePartyAccept(w http.ResponseWriter, r *http.Request) {
	var request partyInviteActionRequest
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

	party, err := s.acceptPartyInviteLocked(session, request.InviteID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "party_accept_failed", err.Error())
		return
	}

	recipients := recipientSet(party.MemberCharacterIDs...)
	s.sendSystemMessageLocked(fmt.Sprintf("%s joined the party.", session.DisplayName), recipients)
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handlePartyDecline(w http.ResponseWriter, r *http.Request) {
	var request partyInviteActionRequest
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

	s.cleanupExpiredPartyInvitesLocked(time.Now())
	invite, ok := s.partyInvites[request.InviteID]
	if !ok || invite.TargetCharacterID != session.CharacterID {
		httpapi.Error(w, http.StatusBadRequest, "party_invite_missing", "Party invite was not found.")
		return
	}
	delete(s.partyInvites, request.InviteID)

	s.sendSystemMessageLocked("Party invite declined.", recipientSet(session.CharacterID, invite.InviterCharacterID))
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handlePartyLeave(w http.ResponseWriter, r *http.Request) {
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

	party, err := s.store.GetPartyForCharacter(session.CharacterID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "party_missing", "You are not in a party.")
		return
	}

	remaining := make([]string, 0, len(party.MemberCharacterIDs))
	for _, memberID := range party.MemberCharacterIDs {
		if memberID != session.CharacterID {
			remaining = append(remaining, memberID)
		}
	}

	recipients := recipientSet(party.MemberCharacterIDs...)
	if len(remaining) <= 1 {
		if err := s.store.DeleteParty(party.ID); err != nil && !errors.Is(err, storepkg.ErrPartyMissing) {
			httpapi.Error(w, http.StatusBadRequest, "party_leave_failed", err.Error())
			return
		}
	} else {
		party.MemberCharacterIDs = remaining
		if party.LeaderCharacterID == session.CharacterID {
			party.LeaderCharacterID = remaining[0]
		}
		if _, err := s.store.SaveParty(*party); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "party_leave_failed", err.Error())
			return
		}
	}

	s.sendSystemMessageLocked(fmt.Sprintf("%s left the party.", session.DisplayName), recipients)
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) handlePartyDisband(w http.ResponseWriter, r *http.Request) {
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

	party, err := s.store.GetPartyForCharacter(session.CharacterID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "party_missing", "You are not in a party.")
		return
	}
	if party.LeaderCharacterID != session.CharacterID {
		httpapi.Error(w, http.StatusBadRequest, "party_not_leader", "Only the party leader can disband the party.")
		return
	}

	recipients := recipientSet(party.MemberCharacterIDs...)
	if err := s.store.DeleteParty(party.ID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "party_disband_failed", err.Error())
		return
	}

	s.sendSystemMessageLocked("Party disbanded.", recipients)
	httpapi.WriteJSON(w, http.StatusOK, s.buildSocialStateLocked(session, ""))
}

func (s *worldServer) sendChatMessageLocked(session *worldSessionState, request chatSendRequest) error {
	channel := strings.ToLower(strings.TrimSpace(request.Channel))
	messageText := strings.TrimSpace(request.MessageText)
	if messageText == "" {
		return fmt.Errorf("message cannot be empty")
	}
	if len(messageText) > maxChatMessageLength {
		return fmt.Errorf("message cannot exceed %d characters", maxChatMessageLength)
	}

	switch channel {
	case "", chatChannelSay:
		channel = chatChannelSay
		recipients := map[string]struct{}{}
		for _, candidate := range s.sessionsByToken {
			if candidate == nil || !candidate.Connected || candidate.ZoneID != session.ZoneID {
				continue
			}
			if distance2D(session.X, session.Y, candidate.X, candidate.Y) <= sayChatRadius {
				recipients[candidate.CharacterID] = struct{}{}
			}
		}
		s.appendChatMessageLocked(platform.ChatMessage{
			Channel:           channel,
			SenderCharacterID: session.CharacterID,
			SenderDisplayName: session.DisplayName,
			ZoneID:            session.ZoneID,
			MessageText:       messageText,
		}, recipients)
		return nil

	case chatChannelWhisper:
		if s.store == nil {
			return fmt.Errorf("social persistence is unavailable")
		}
		target, err := s.store.GetCharacterByName(session.RealmID, request.TargetName)
		if err != nil {
			return fmt.Errorf("target character not found")
		}
		targetSession := s.findConnectedSessionByCharacterLocked(target.ID)
		if targetSession == nil {
			return fmt.Errorf("target player is offline")
		}
		s.appendChatMessageLocked(platform.ChatMessage{
			Channel:           channel,
			SenderCharacterID: session.CharacterID,
			SenderDisplayName: session.DisplayName,
			TargetCharacterID: target.ID,
			MessageText:       messageText,
		}, recipientSet(session.CharacterID, target.ID))
		return nil

	case chatChannelParty:
		if s.store == nil {
			return fmt.Errorf("social persistence is unavailable")
		}
		party, err := s.store.GetPartyForCharacter(session.CharacterID)
		if err != nil {
			return fmt.Errorf("you are not in a party")
		}
		s.appendChatMessageLocked(platform.ChatMessage{
			Channel:           channel,
			SenderCharacterID: session.CharacterID,
			SenderDisplayName: session.DisplayName,
			PartyID:           party.ID,
			MessageText:       messageText,
		}, recipientSet(party.MemberCharacterIDs...))
		return nil
	case chatChannelGuild:
		if s.store == nil {
			return fmt.Errorf("social persistence is unavailable")
		}
		guild, err := s.store.GetGuildForCharacter(session.CharacterID)
		if err != nil {
			return fmt.Errorf("you are not in a guild")
		}
		s.appendChatMessageLocked(platform.ChatMessage{
			Channel:           channel,
			SenderCharacterID: session.CharacterID,
			SenderDisplayName: session.DisplayName,
			GuildID:           guild.ID,
			MessageText:       messageText,
		}, s.onlineGuildRecipientSetLocked(*guild))
		return nil
	default:
		return fmt.Errorf("unsupported chat channel")
	}
}

func (s *worldServer) appendChatMessageLocked(message platform.ChatMessage, recipients map[string]struct{}) platform.ChatMessage {
	if message.SenderDisplayName == "" {
		message.SenderDisplayName = "System"
	}
	message.MessageText = strings.TrimSpace(message.MessageText)
	message.Timestamp = time.Now().UnixMilli()
	s.chatSequence++
	message.MessageID = fmt.Sprintf("chat_%06d", s.chatSequence)

	s.chatMessages = append(s.chatMessages, chatEnvelope{
		Message:               message,
		Sequence:              s.chatSequence,
		RecipientCharacterIDs: recipients,
	})
	if len(s.chatMessages) > chatRingLimit {
		s.chatMessages = s.chatMessages[len(s.chatMessages)-chatRingLimit:]
	}

	observability.LogEvent("world-service", "social.chat_message", map[string]any{
		"messageId": message.MessageID,
		"channel":   message.Channel,
		"sender":    message.SenderCharacterID,
		"target":    message.TargetCharacterID,
		"partyId":   message.PartyID,
		"guildId":   message.GuildID,
	})
	return message
}

func (s *worldServer) sendSystemMessageLocked(messageText string, recipients map[string]struct{}) {
	if strings.TrimSpace(messageText) == "" || len(recipients) == 0 {
		return
	}
	s.appendChatMessageLocked(platform.ChatMessage{
		Channel:           chatChannelSystem,
		SenderDisplayName: "System",
		MessageText:       messageText,
	}, recipients)
}

func (s *worldServer) buildSocialStateLocked(session *worldSessionState, afterMessageID string) socialStateResponse {
	if s.store != nil {
		_ = s.store.CleanupExpiredGuildInvites(time.Now().Unix())
	}
	return socialStateResponse{
		ChatMessages: s.visibleChatMessagesLocked(session.CharacterID, afterMessageID),
		Friends:      s.buildFriendResponsesLocked(session),
		Party:        s.buildPartyResponseLocked(session.CharacterID),
		PartyInvites: s.buildPartyInviteResponsesLocked(session.CharacterID),
		Guild:        s.buildGuildResponseLocked(session.CharacterID),
		GuildInvites: s.buildGuildInviteResponsesLocked(session.CharacterID),
	}
}

func (s *worldServer) visibleChatMessagesLocked(characterID string, afterMessageID string) []platform.ChatMessage {
	afterSequence := chatMessageSequence(afterMessageID)
	messages := make([]platform.ChatMessage, 0)
	for _, envelope := range s.chatMessages {
		if afterSequence > 0 && envelope.Sequence <= afterSequence {
			continue
		}
		if _, visible := envelope.RecipientCharacterIDs[characterID]; !visible {
			continue
		}
		messages = append(messages, envelope.Message)
	}
	return messages
}

func (s *worldServer) buildFriendResponsesLocked(session *worldSessionState) []friendResponse {
	if s.store == nil {
		return []friendResponse{}
	}

	relationships, err := s.store.ListFriends(session.CharacterID)
	if err != nil {
		return []friendResponse{}
	}

	friends := make([]friendResponse, 0, len(relationships))
	for _, relationship := range relationships {
		character, err := s.store.GetCharacterByID(relationship.FriendCharacterID)
		if err != nil {
			continue
		}
		response := friendResponse{
			CharacterID: character.ID,
			DisplayName: character.DisplayName,
			Level:       character.Level,
			ClassID:     character.ClassID,
			ZoneID:      character.ZoneID,
		}
		if friendSession := s.findConnectedSessionByCharacterLocked(character.ID); friendSession != nil {
			response.Online = true
			response.Level = friendSession.Level
			response.ClassID = friendSession.ClassID
			response.ZoneID = friendSession.ZoneID
		}
		friends = append(friends, response)
	}

	sort.Slice(friends, func(left int, right int) bool {
		return strings.ToLower(friends[left].DisplayName) < strings.ToLower(friends[right].DisplayName)
	})
	return friends
}

func (s *worldServer) buildPartyResponseLocked(characterID string) *partyResponse {
	if s.store == nil {
		return nil
	}

	party, err := s.store.GetPartyForCharacter(characterID)
	if err != nil {
		return nil
	}

	response := &partyResponse{
		PartyID:           party.ID,
		LeaderCharacterID: party.LeaderCharacterID,
		Members:           make([]partyMemberResponse, 0, len(party.MemberCharacterIDs)),
	}
	for _, memberID := range party.MemberCharacterIDs {
		character, err := s.store.GetCharacterByID(memberID)
		if err != nil {
			continue
		}
		member := partyMemberResponse{
			CharacterID:  character.ID,
			DisplayName:  character.DisplayName,
			Level:        character.Level,
			ClassID:      character.ClassID,
			ZoneID:       character.ZoneID,
			Leader:       character.ID == party.LeaderCharacterID,
			Disconnected: true,
		}
		if memberSession := s.findConnectedSessionByCharacterLocked(character.ID); memberSession != nil {
			member.Online = true
			member.Disconnected = false
			member.Level = memberSession.Level
			member.ClassID = memberSession.ClassID
			member.ZoneID = memberSession.ZoneID
			member.Health = memberSession.Health
			member.MaxHealth = memberSession.MaxHealth
			member.Resource = memberSession.Resource
			member.MaxResource = memberSession.MaxResource
		}
		response.Members = append(response.Members, member)
	}
	return response
}

func (s *worldServer) buildPartyInviteResponsesLocked(characterID string) []partyInviteResponse {
	invites := make([]partyInviteResponse, 0)
	for _, invite := range s.partyInvites {
		if invite.TargetCharacterID != characterID {
			continue
		}
		inviter, err := s.store.GetCharacterByID(invite.InviterCharacterID)
		if err != nil {
			continue
		}
		invites = append(invites, partyInviteResponse{
			InviteID:           invite.InviteID,
			PartyID:            invite.PartyID,
			InviterCharacterID: invite.InviterCharacterID,
			InviterDisplayName: inviter.DisplayName,
			ExpiresAt:          invite.ExpiresAt,
		})
	}
	return invites
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

func (s *worldServer) resolvePartyInviteTargetLocked(session *worldSessionState, request partyInviteRequest) (*platform.Character, error) {
	if s.store == nil {
		return nil, fmt.Errorf("social persistence is unavailable")
	}
	if request.TargetCharacterID != "" {
		target, err := s.store.GetCharacterByID(request.TargetCharacterID)
		if err != nil {
			return nil, err
		}
		if target.RealmID != session.RealmID {
			return nil, fmt.Errorf("target player is not on this realm")
		}
		if target.ID == session.CharacterID {
			return nil, fmt.Errorf("cannot invite yourself")
		}
		return target, nil
	}

	target, err := s.store.GetCharacterByName(session.RealmID, request.TargetName)
	if err != nil {
		return nil, fmt.Errorf("target character not found")
	}
	if target.ID == session.CharacterID {
		return nil, fmt.Errorf("cannot invite yourself")
	}
	return target, nil
}

func (s *worldServer) acceptPartyInviteLocked(session *worldSessionState, inviteID string) (*platform.Party, error) {
	if s.store == nil {
		return nil, fmt.Errorf("social persistence is unavailable")
	}
	s.cleanupExpiredPartyInvitesLocked(time.Now())

	invite, ok := s.partyInvites[inviteID]
	if !ok || invite.TargetCharacterID != session.CharacterID {
		return nil, fmt.Errorf("party invite was not found")
	}

	if _, err := s.store.GetPartyForCharacter(session.CharacterID); err == nil {
		return nil, fmt.Errorf("you are already in a party")
	}

	inviterSession := s.findConnectedSessionByCharacterLocked(invite.InviterCharacterID)
	if inviterSession == nil {
		delete(s.partyInvites, inviteID)
		return nil, fmt.Errorf("inviter is no longer online")
	}

	party, err := s.store.GetPartyForCharacter(invite.InviterCharacterID)
	if errors.Is(err, storepkg.ErrPartyMissing) {
		created, err := s.store.CreateParty(invite.InviterCharacterID, []string{invite.InviterCharacterID, session.CharacterID})
		if err != nil {
			return nil, err
		}
		delete(s.partyInvites, inviteID)
		return &created, nil
	}
	if err != nil {
		return nil, err
	}
	if party.LeaderCharacterID != invite.InviterCharacterID {
		return nil, fmt.Errorf("inviter is not the party leader")
	}
	if len(party.MemberCharacterIDs) >= partySizeLimit {
		delete(s.partyInvites, inviteID)
		return nil, fmt.Errorf("party is full")
	}
	party.MemberCharacterIDs = append(party.MemberCharacterIDs, session.CharacterID)
	saved, err := s.store.SaveParty(*party)
	if err != nil {
		return nil, err
	}
	delete(s.partyInvites, inviteID)
	return &saved, nil
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

func (s *worldServer) cleanupExpiredPartyInvitesLocked(now time.Time) {
	if len(s.partyInvites) == 0 {
		return
	}
	nowUnix := now.Unix()
	for inviteID, invite := range s.partyInvites {
		if invite.ExpiresAt <= nowUnix {
			delete(s.partyInvites, inviteID)
		}
	}
}

func recipientSet(characterIDs ...string) map[string]struct{} {
	recipients := map[string]struct{}{}
	for _, characterID := range characterIDs {
		characterID = strings.TrimSpace(characterID)
		if characterID == "" {
			continue
		}
		recipients[characterID] = struct{}{}
	}
	return recipients
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
	for _, rank := range platform.DefaultGuildRanks() {
		if rank.RankID == rankID {
			return rank
		}
	}
	return platform.DefaultGuildRanks()[len(platform.DefaultGuildRanks())-1]
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

func chatMessageSequence(messageID string) int64 {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return 0
	}
	messageID = strings.TrimPrefix(messageID, "chat_")
	sequence, err := strconv.ParseInt(messageID, 10, 64)
	if err != nil {
		return 0
	}
	return sequence
}
