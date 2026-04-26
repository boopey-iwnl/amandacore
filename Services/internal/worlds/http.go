package worlds

import (
	"crypto/rand"
	"encoding/hex"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

func RegisterRoutes(mux *http.ServeMux, fileStore *store.FileStore) {
	RegisterRoutesWithAdmin(mux, fileStore, true)
}

func RegisterRoutesWithAdmin(mux *http.ServeMux, fileStore *store.FileStore, adminEnabled bool) {
	server := newConfiguredWorldServer(fileStore)

	joinTicketHandler := httpapi.RequireSession(fileStore, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		var request joinTicketRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		persistStartedAt := time.Now()
		ticket, err := fileStore.IssueWorldJoinTicket(session.AccountID, session.ID, request.CharacterID, request.RealmID)
		server.recordPersistenceDuration("world_join_ticket", persistStartedAt, err)
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "join_ticket_failed", err.Error())
			return
		}

		observability.LogEvent("world-service", "character.selected", map[string]any{
			"accountId":   session.AccountID,
			"sessionId":   session.ID,
			"characterId": request.CharacterID,
			"realmId":     request.RealmID,
		})
		observability.LogEvent("world-service", "world.join_ticket_issued", map[string]any{
			"accountId":   ticket.AccountID,
			"sessionId":   ticket.SessionID,
			"ticketId":    ticket.TicketID,
			"characterId": ticket.CharacterID,
			"realmId":     ticket.RealmID,
			"expiresAt":   ticket.ExpiresAt,
		})

		httpapi.WriteJSON(w, http.StatusCreated, ticket)
	})
	mux.Handle("POST /v1/world/join-ticket", server.instrumentEndpoint("join_ticket", joinTicketHandler))

	mux.HandleFunc("POST /v1/world/connect", server.instrumentEndpointFunc("connect", server.handleConnect))
	mux.HandleFunc("POST /v1/world/reconnect", server.instrumentEndpointFunc("reconnect", server.handleReconnect))
	mux.HandleFunc("POST /v1/world/move", server.instrumentEndpointFunc("move", server.handleMove))
	mux.HandleFunc("POST /v1/world/disconnect", server.instrumentEndpointFunc("disconnect", server.handleDisconnect))
	mux.HandleFunc("POST /v1/world/target", server.instrumentEndpointFunc("target", server.handleTarget))
	mux.HandleFunc("GET /v1/world/travel/state", server.instrumentEndpointFunc("travel_state", server.handleTravelState))
	mux.HandleFunc("POST /v1/world/bind/set", server.instrumentEndpointFunc("bind_set", server.handleSetBindPoint))
	mux.HandleFunc("POST /v1/world/recall/use", server.instrumentEndpointFunc("recall_use", server.handleUseRecall))
	mux.HandleFunc("POST /v1/world/travel/discover", server.instrumentEndpointFunc("travel_discover", server.handleDiscoverTravelPoint))
	mux.HandleFunc("GET /v1/world/travel/destinations", server.instrumentEndpointFunc("travel_destinations", server.handleTravelDestinations))
	mux.HandleFunc("POST /v1/world/travel/route", server.instrumentEndpointFunc("travel_route", server.handleTravelRoute))
	mux.HandleFunc("GET /v1/world/mounts", server.instrumentEndpointFunc("mount_collection", server.handleMountCollection))
	mux.HandleFunc("POST /v1/world/mount/unlock", server.instrumentEndpointFunc("mount_unlock", server.handleUnlockMount))
	mux.HandleFunc("POST /v1/world/mount/select", server.instrumentEndpointFunc("mount_select", server.handleSelectMount))
	mux.HandleFunc("POST /v1/world/mount/summon", server.instrumentEndpointFunc("mount_summon", server.handleSummonMount))
	mux.HandleFunc("POST /v1/world/mount/dismiss", server.instrumentEndpointFunc("mount_dismiss", server.handleDismissMount))
	mux.HandleFunc("POST /v1/world/quest/accept", server.instrumentEndpointFunc("quest_accept", server.handleQuestAccept))
	mux.HandleFunc("POST /v1/world/quest/complete", server.instrumentEndpointFunc("quest_complete", server.handleQuestComplete))
	mux.HandleFunc("POST /v1/world/quest/track", server.instrumentEndpointFunc("quest_track", server.handleQuestTrack))
	mux.HandleFunc("POST /v1/world/dungeon/enter", server.instrumentEndpointFunc("dungeon_enter", server.handleDungeonEnter))
	mux.HandleFunc("POST /v1/world/dungeon/exit", server.instrumentEndpointFunc("dungeon_exit", server.handleDungeonExit))
	mux.HandleFunc("POST /v1/world/dungeon/reset", server.instrumentEndpointFunc("dungeon_reset", server.handleDungeonReset))
	mux.HandleFunc("POST /v1/world/trainer/learn", server.instrumentEndpointFunc("trainer_learn", server.handleTrainerLearn))
	mux.HandleFunc("POST /v1/world/profession/learn", server.instrumentEndpointFunc("profession_learn", server.handleProfessionLearn))
	mux.HandleFunc("POST /v1/world/gather", server.instrumentEndpointFunc("gather", server.handleGather))
	mux.HandleFunc("POST /v1/world/craft", server.instrumentEndpointFunc("craft", server.handleCraft))
	mux.HandleFunc("POST /v1/world/talent/select", server.instrumentEndpointFunc("talent_select", server.handleTalentSelect))
	mux.HandleFunc("POST /v1/world/action-bar/assign", server.instrumentEndpointFunc("action_bar_assign", server.handleActionBarAssign))
	mux.HandleFunc("POST /v1/world/action-bar/move", server.instrumentEndpointFunc("action_bar_move", server.handleActionBarMove))
	mux.HandleFunc("POST /v1/world/action-bar/clear", server.instrumentEndpointFunc("action_bar_clear", server.handleActionBarClear))
	mux.HandleFunc("POST /v1/world/inventory/move", server.instrumentEndpointFunc("inventory_move", server.handleInventoryMove))
	mux.HandleFunc("POST /v1/world/inventory/equip", server.instrumentEndpointFunc("inventory_equip", server.handleInventoryEquip))
	mux.HandleFunc("POST /v1/world/loot/inspect", server.instrumentEndpointFunc("loot_inspect", server.handleLootInspect))
	mux.HandleFunc("POST /v1/world/loot/claim", server.instrumentEndpointFunc("loot_claim", server.handleLootClaim))
	mux.HandleFunc("GET /v1/world/housing/status", server.instrumentEndpointFunc("housing_status", server.handleHousingStatus))
	mux.HandleFunc("POST /v1/world/housing/enter", server.instrumentEndpointFunc("housing_enter", server.handleHousingEnter))
	mux.HandleFunc("POST /v1/world/housing/leave", server.instrumentEndpointFunc("housing_leave", server.handleHousingLeave))
	mux.HandleFunc("GET /v1/world/housing/storage", server.instrumentEndpointFunc("housing_storage", server.handleHousingStorageList))
	mux.HandleFunc("POST /v1/world/housing/storage/deposit", server.instrumentEndpointFunc("housing_storage_deposit", server.handleHousingStorageDeposit))
	mux.HandleFunc("POST /v1/world/housing/storage/withdraw", server.instrumentEndpointFunc("housing_storage_withdraw", server.handleHousingStorageWithdraw))
	mux.HandleFunc("POST /v1/world/housing/storage/move", server.instrumentEndpointFunc("housing_storage_move", server.handleHousingStorageMove))
	mux.HandleFunc("GET /v1/world/housing/decorations", server.instrumentEndpointFunc("housing_decorations", server.handleHousingDecorationsList))
	mux.HandleFunc("POST /v1/world/housing/decorations/place", server.instrumentEndpointFunc("housing_decoration_place", server.handleHousingDecorationPlace))
	mux.HandleFunc("POST /v1/world/housing/decorations/remove", server.instrumentEndpointFunc("housing_decoration_remove", server.handleHousingDecorationRemove))
	mux.HandleFunc("POST /v1/world/vendor/buy", server.instrumentEndpointFunc("vendor_buy", server.handleVendorBuy))
	mux.HandleFunc("POST /v1/world/vendor/sell", server.instrumentEndpointFunc("vendor_sell", server.handleVendorSell))
	mux.HandleFunc("GET /v1/world/auction/listings", server.instrumentEndpointFunc("auction_browse", server.handleAuctionBrowse))
	mux.HandleFunc("GET /v1/world/auction/mine", server.instrumentEndpointFunc("auction_mine", server.handleAuctionMine))
	mux.HandleFunc("POST /v1/world/auction/list", server.instrumentEndpointFunc("auction_list", server.handleAuctionList))
	mux.HandleFunc("POST /v1/world/auction/buyout", server.instrumentEndpointFunc("auction_buyout", server.handleAuctionBuyout))
	mux.HandleFunc("POST /v1/world/auction/cancel", server.instrumentEndpointFunc("auction_cancel", server.handleAuctionCancel))
	mux.HandleFunc("POST /v1/world/attack/auto", server.instrumentEndpointFunc("attack_auto", server.handleAutoAttack))
	mux.HandleFunc("POST /v1/world/attack/ability", server.instrumentEndpointFunc("attack_ability", server.handleAbility))
	mux.HandleFunc("POST /v1/world/duel/request", server.instrumentEndpointFunc("duel_request", server.handleDuelRequest))
	mux.HandleFunc("POST /v1/world/duel/accept", server.instrumentEndpointFunc("duel_accept", server.handleDuelAccept))
	mux.HandleFunc("POST /v1/world/duel/decline", server.instrumentEndpointFunc("duel_decline", server.handleDuelDecline))
	mux.HandleFunc("POST /v1/world/duel/cancel", server.instrumentEndpointFunc("duel_cancel", server.handleDuelCancel))
	mux.HandleFunc("POST /v1/world/duel/surrender", server.instrumentEndpointFunc("duel_surrender", server.handleDuelSurrender))
	mux.HandleFunc("GET /v1/world/duel/state", server.instrumentEndpointFunc("duel_state", server.handleDuelState))
	mux.HandleFunc("GET /v1/world/pvp/stats", server.instrumentEndpointFunc("pvp_stats", server.handlePvPStats))
	mux.HandleFunc("GET /v1/world/social/state", server.instrumentEndpointFunc("social_state", server.handleSocialState))
	mux.HandleFunc("POST /v1/world/chat/send", server.instrumentEndpointFunc("chat_send", server.handleChatSend))
	mux.HandleFunc("POST /v1/world/friends/add", server.instrumentEndpointFunc("friends_add", server.handleFriendAdd))
	mux.HandleFunc("POST /v1/world/friends/remove", server.instrumentEndpointFunc("friends_remove", server.handleFriendRemove))
	mux.HandleFunc("POST /v1/world/guild/create", server.instrumentEndpointFunc("guild_create", server.handleGuildCreate))
	mux.HandleFunc("POST /v1/world/guild/invite", server.instrumentEndpointFunc("guild_invite", server.handleGuildInvite))
	mux.HandleFunc("POST /v1/world/guild/accept", server.instrumentEndpointFunc("guild_accept", server.handleGuildAccept))
	mux.HandleFunc("POST /v1/world/guild/decline", server.instrumentEndpointFunc("guild_decline", server.handleGuildDecline))
	mux.HandleFunc("POST /v1/world/guild/leave", server.instrumentEndpointFunc("guild_leave", server.handleGuildLeave))
	mux.HandleFunc("POST /v1/world/guild/disband", server.instrumentEndpointFunc("guild_disband", server.handleGuildDisband))
	mux.HandleFunc("POST /v1/world/guild/promote", server.instrumentEndpointFunc("guild_promote", server.handleGuildPromote))
	mux.HandleFunc("POST /v1/world/guild/demote", server.instrumentEndpointFunc("guild_demote", server.handleGuildDemote))
	mux.HandleFunc("POST /v1/world/guild/remove", server.instrumentEndpointFunc("guild_remove", server.handleGuildRemove))
	mux.HandleFunc("POST /v1/world/guild/motd", server.instrumentEndpointFunc("guild_motd", server.handleGuildMOTD))
	mux.HandleFunc("POST /v1/world/party/invite", server.instrumentEndpointFunc("party_invite", server.handlePartyInvite))
	mux.HandleFunc("POST /v1/world/party/accept", server.instrumentEndpointFunc("party_accept", server.handlePartyAccept))
	mux.HandleFunc("POST /v1/world/party/decline", server.instrumentEndpointFunc("party_decline", server.handlePartyDecline))
	mux.HandleFunc("POST /v1/world/party/leave", server.instrumentEndpointFunc("party_leave", server.handlePartyLeave))
	mux.HandleFunc("POST /v1/world/party/disband", server.instrumentEndpointFunc("party_disband", server.handlePartyDisband))
	mux.HandleFunc("GET /v1/world/state", server.instrumentEndpointFunc("state", server.handleState))
	mux.HandleFunc("GET /v1/world/metrics", server.instrumentEndpointFunc("metrics", server.handleMetrics))
	if adminEnabled {
		registerAdminRoutes(mux, server, fileStore)
	}
	mux.HandleFunc("GET /v1/world/bootstrap", server.instrumentEndpointFunc("bootstrap", func(w http.ResponseWriter, r *http.Request) {
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"zoneId":   defaultZoneID,
			"cellId":   defaultZoneID,
			"motd":     "Stonewake Vale is active. Muster at Hearthwatch Yard, train with Armsmaster Corin, and follow the westward road.",
			"revision": "alpha-0.1-stonewake-starter-zone",
		})
	}))
}

func newConfiguredWorldServer(fileStore *store.FileStore) *worldServer {
	if contentPackagePath := strings.TrimSpace(os.Getenv("AMANDACORE_CONTENT_PACKAGE")); contentPackagePath != "" {
		return newWorldServerWithContentPackage(fileStore, contentPackagePath)
	}
	return newWorldServer(fileStore)
}

func registerAdminRoutes(mux *http.ServeMux, server *worldServer, fileStore *store.FileStore) {
	mux.Handle("POST /v1/world/admin/characters/{characterId}/teleport", server.instrumentEndpoint("admin_teleport", httpapi.RequirePermission(fileStore, platform.PermissionTeleportCharacter, server.handleAdminTeleport)))
	mux.Handle("POST /v1/world/admin/characters/{characterId}/repair", server.instrumentEndpoint("admin_repair", httpapi.RequirePermission(fileStore, platform.PermissionRepairCharacter, server.handleAdminRepair)))
	mux.Handle("POST /v1/world/admin/characters/{characterId}/session/invalidate", server.instrumentEndpoint("admin_session_invalidate", httpapi.RequirePermission(fileStore, platform.PermissionRepairCharacter, server.handleAdminSessionInvalidate)))
	mux.Handle("POST /v1/world/admin/characters/{characterId}/items/grant", server.instrumentEndpoint("admin_item_grant", httpapi.RequirePermission(fileStore, platform.PermissionGrantItem, server.handleAdminItemGrant)))
	mux.Handle("POST /v1/world/admin/characters/{characterId}/items/remove", server.instrumentEndpoint("admin_item_remove", httpapi.RequirePermission(fileStore, platform.PermissionGrantItem, server.handleAdminItemRemove)))
	mux.Handle("POST /v1/world/admin/characters/{characterId}/currency/grant", server.instrumentEndpoint("admin_currency_grant", httpapi.RequirePermission(fileStore, platform.PermissionGrantCurrency, server.handleAdminCurrencyGrant)))
	mux.Handle("POST /v1/world/admin/characters/{characterId}/currency/remove", server.instrumentEndpoint("admin_currency_remove", httpapi.RequirePermission(fileStore, platform.PermissionGrantCurrency, server.handleAdminCurrencyRemove)))
}

func (s *worldServer) handleConnect(w http.ResponseWriter, r *http.Request) {
	var request connectRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	persistStartedAt := time.Now()
	ticket, err := s.store.ConsumeWorldJoinTicket(request.TicketID)
	s.recordPersistenceDuration("world_join_ticket_consume", persistStartedAt, err)
	if err != nil {
		s.metrics.recordSessionEvent("connect_ticket_failed")
		httpapi.Error(w, http.StatusUnauthorized, "invalid_ticket", err.Error())
		return
	}

	observability.LogEvent("world-service", "world.join_ticket_consumed", map[string]any{
		"ticketId":    ticket.TicketID,
		"accountId":   ticket.AccountID,
		"sessionId":   ticket.SessionID,
		"characterId": ticket.CharacterID,
		"realmId":     ticket.RealmID,
		"consumedAt":  ticket.ConsumedAt,
	})

	character, err := s.store.GetCharacterByID(ticket.CharacterID)
	if err != nil {
		httpapi.Error(w, http.StatusNotFound, "character_missing", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ensureMobsLocked()
	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	if existingToken, ok := s.sessionTokenByChar[character.ID]; ok {
		if session, found := s.sessionsByToken[existingToken]; found {
			session.DisplayName = character.DisplayName
			session.Connected = true
			session.LastSeenAt = time.Now().Unix()
			if session.HousingSpaceID != "" {
				if err := s.returnSessionFromHousingLocked(session, "connect_resume"); err != nil {
					httpapi.Error(w, http.StatusInternalServerError, "housing_recover_failed", err.Error())
					return
				}
			} else if session.ZoneID == "" {
				session.ZoneID = character.ZoneID
			}
			s.cancelDuelForCharacterLocked(session.CharacterID, duelReasonDisconnect)
			s.resetSessionCombatStateLocked(session, "reconnect")
			s.clearMobAggroForCharacterLocked(session.CharacterID)
			if !session.Alive || session.Health <= 0.0 {
				s.reviveSessionLocked(session)
			}
			s.applyCharacterProgressionLocked(session, character)
			observability.LogEvent("world-service", "world.player_spawned", map[string]any{
				"worldSessionToken": session.Token,
				"accountId":         session.AccountID,
				"characterId":       session.CharacterID,
				"realmId":           session.RealmID,
				"zoneId":            session.ZoneID,
				"x":                 session.X,
				"y":                 session.Y,
				"z":                 session.Z,
				"resumeExisting":    true,
			})
			s.sendSystemMessageLocked("World session reconnected.", recipientSet(session.CharacterID))
			s.metrics.recordSessionEvent("connect_resumed")
			httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
			return
		}
	}

	session := &worldSessionState{
		Token:       "world_" + randomWorldToken(),
		AccountID:   ticket.AccountID,
		CharacterID: ticket.CharacterID,
		DisplayName: character.DisplayName,
		ClassID:     character.ClassID,
		RealmID:     ticket.RealmID,
		ZoneID:      character.ZoneID,
		X:           character.PositionX,
		Y:           character.PositionY,
		Z:           character.PositionZ,
		Connected:   true,
		LastSeenAt:  time.Now().Unix(),
		Health:      playerMaxHealth,
		MaxHealth:   playerMaxHealth,
		Resource:    0,
		MaxResource: playerMaxResource,
		Alive:       true,
	}
	s.applyCharacterProgressionLocked(session, character)

	s.sessionsByToken[session.Token] = session
	s.sessionTokenByChar[session.CharacterID] = session.Token
	observability.LogEvent("world-service", "world.player_spawned", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"realmId":           session.RealmID,
		"zoneId":            session.ZoneID,
		"x":                 session.X,
		"y":                 session.Y,
		"z":                 session.Z,
		"resumeExisting":    false,
	})
	s.sendSystemMessageLocked("World session linked.", recipientSet(session.CharacterID))
	s.metrics.recordSessionEvent("connect_created")
	httpapi.WriteJSON(w, http.StatusCreated, s.buildResponse(session))
}

func (s *worldServer) handleReconnect(w http.ResponseWriter, r *http.Request) {
	var request reconnectRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		s.metrics.recordSessionEvent("reconnect_missing")
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	character, err := s.store.GetCharacterByID(session.CharacterID)
	if err != nil {
		httpapi.Error(w, http.StatusNotFound, "character_missing", err.Error())
		return
	}

	session.DisplayName = character.DisplayName
	session.Connected = true
	session.LastSeenAt = time.Now().Unix()
	if session.InstanceID != "" && s.dungeonInstanceActiveForSessionLocked(session) {
		if instance := s.dungeonInstances[session.InstanceID]; instance != nil {
			instance.PlayersInside[session.CharacterID] = true
			instance.State = dungeonStateActive
			instance.LastPlayerLeftAtMs = 0
		}
	} else if session.InstanceID != "" {
		s.recoverExpiredDungeonSessionLocked(session)
	} else if session.HousingSpaceID != "" {
		session.HousingSpaceID = ""
		session.HousingInstanceID = ""
		session.ReturnZoneID = ""
		session.ReturnX = 0
		session.ReturnY = 0
		session.ReturnZ = 0
		session.ZoneID = character.ZoneID
		session.X = character.PositionX
		session.Y = character.PositionY
		session.Z = character.PositionZ
	} else {
		session.ZoneID = character.ZoneID
		session.X = character.PositionX
		session.Y = character.PositionY
		session.Z = character.PositionZ
	}
	s.cancelDuelForCharacterLocked(session.CharacterID, duelReasonDisconnect)
	s.resetSessionCombatStateLocked(session, "reconnect")
	s.clearMobAggroForCharacterLocked(session.CharacterID)
	if !session.Alive || session.Health <= 0.0 {
		s.reviveSessionLocked(session)
	}
	s.applyCharacterProgressionLocked(session, character)

	observability.LogEvent("world-service", "world.reconnected", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"realmId":           session.RealmID,
		"zoneId":            session.ZoneID,
		"x":                 session.X,
		"y":                 session.Y,
		"z":                 session.Z,
		"health":            session.Health,
		"resource":          session.Resource,
	})
	s.sendSystemMessageLocked("World session reconnected.", recipientSet(session.CharacterID))
	s.metrics.recordSessionEvent("reconnect_succeeded")
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleMove(w http.ResponseWriter, r *http.Request) {
	var request moveRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if !session.Alive {
		httpapi.Error(w, http.StatusConflict, "player_dead", "Dead players cannot move.")
		return
	}

	nextX, nextY := s.resolveMovementLocked(session, request.DeltaX, request.DeltaY)
	session.X = nextX
	session.Y = nextY
	session.LastSeenAt = time.Now().Unix()
	if _, err := s.applyContentZoneTransitionsLocked(session); err != nil {
		httpapi.Error(w, http.StatusConflict, "zone_transition_failed", err.Error())
		return
	}

	if session.HousingSpaceID != "" || session.InstanceID != "" {
		httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
		return
	}

	persistStartedAt := time.Now()
	character, err := s.store.UpdateCharacterState(session.CharacterID, session.ZoneID, session.X, session.Y, session.Z)
	s.recordPersistenceDuration("character_state_move", persistStartedAt, err)
	if err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "character_save_failed", err.Error())
		return
	}

	observability.LogEvent("world-service", "world.character_saved", map[string]any{
		"reason":            "move",
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"zoneId":            character.ZoneID,
		"x":                 character.PositionX,
		"y":                 character.PositionY,
		"z":                 character.PositionZ,
	})

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	var request disconnectRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	session.Connected = false
	session.LastSeenAt = time.Now().Unix()
	s.cancelDuelForCharacterLocked(session.CharacterID, duelReasonDisconnect)
	s.resetSessionCombatStateLocked(session, "disconnect")
	s.clearMobAggroForCharacterLocked(session.CharacterID)

	if session.HousingSpaceID != "" {
		if err := s.returnSessionFromHousingLocked(session, "disconnect"); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "housing_recover_failed", err.Error())
			return
		}
		s.metrics.recordSessionEvent("disconnect_succeeded")
		httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	persistStartedAt := time.Now()
	persistZoneID, persistX, persistY, persistZ := session.ZoneID, session.X, session.Y, session.Z
	if session.InstanceID != "" {
		if instance := s.dungeonInstances[session.InstanceID]; instance != nil {
			delete(instance.PlayersInside, session.CharacterID)
			s.markDungeonEmptyIfNeededLocked(instance)
		}
		persistZoneID, persistX, persistY, persistZ = session.ReturnZoneID, session.ReturnX, session.ReturnY, session.ReturnZ
		if persistZoneID == "" {
			persistZoneID, persistX, persistY, persistZ = secondZoneID, 590, 342, 0
		}
	}
	character, err := s.store.UpdateCharacterState(session.CharacterID, persistZoneID, persistX, persistY, persistZ)
	s.recordPersistenceDuration("character_state_disconnect", persistStartedAt, err)
	if err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "character_save_failed", err.Error())
		return
	}

	observability.LogEvent("world-service", "world.character_saved", map[string]any{
		"reason":            "disconnect",
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"zoneId":            character.ZoneID,
		"x":                 character.PositionX,
		"y":                 character.PositionY,
		"z":                 character.PositionZ,
	})

	s.metrics.recordSessionEvent("disconnect_succeeded")
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *worldServer) handleActionBarMove(w http.ResponseWriter, r *http.Request) {
	var request actionBarMoveRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.moveActionBarSlotLocked(session, request.FromSlotIndex, request.ToSlotIndex); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "action_bar_move_failed", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleInventoryMove(w http.ResponseWriter, r *http.Request) {
	var request inventoryMoveRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.moveInventorySlotLocked(session, request.FromSlotIndex, request.ToSlotIndex); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "inventory_move_failed", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleInventoryEquip(w http.ResponseWriter, r *http.Request) {
	var request inventoryEquipRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.equipInventorySlotLocked(session, request.SlotIndex); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "inventory_equip_failed", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleLootInspect(w http.ResponseWriter, r *http.Request) {
	var request lootInspectRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if _, err := s.inspectLootLocked(session, request.LootContainerID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "loot_inspect_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleLootClaim(w http.ResponseWriter, r *http.Request) {
	var request lootClaimRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if err := s.claimLootLocked(session, request.LootContainerID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "loot_claim_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleVendorBuy(w http.ResponseWriter, r *http.Request) {
	var request vendorBuyRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.buyVendorItemLocked(session, request.VendorID, request.ItemID, request.StackCount); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "vendor_buy_failed", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleVendorSell(w http.ResponseWriter, r *http.Request) {
	var request vendorSellRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.sellVendorItemLocked(session, request.VendorID, request.SlotIndex, request.StackCount); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "vendor_sell_failed", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleTarget(w http.ResponseWriter, r *http.Request) {
	var request targetRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.setTargetLocked(session, request.TargetID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "target_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleQuestAccept(w http.ResponseWriter, r *http.Request) {
	var request questAcceptRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.acceptQuestLocked(session, request.QuestID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "quest_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleQuestComplete(w http.ResponseWriter, r *http.Request) {
	var request questCompleteRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.completeQuestLocked(session, request.QuestID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "quest_complete_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleQuestTrack(w http.ResponseWriter, r *http.Request) {
	var request questTrackRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	quest, found := s.quests[request.QuestID]
	if !found {
		httpapi.Error(w, http.StatusBadRequest, "quest_invalid", "Quest is not available.")
		return
	}
	progress := s.normalizeQuestProgress(quest, session.QuestProgress[quest.ID])
	if request.Tracked && progress.State != questStateActive && progress.State != questStateCompleted {
		httpapi.Error(w, http.StatusBadRequest, "quest_tracking_invalid", "Only active or ready-to-turn-in quests can be tracked.")
		return
	}

	if request.Tracked {
		s.trackQuestLocked(session, quest.ID)
	} else {
		s.untrackQuestLocked(session, quest.ID)
	}

	character, err := s.store.UpdateCharacterTrackedQuests(session.CharacterID, session.TrackedQuestIDs)
	if err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "quest_tracking_save_failed", err.Error())
		return
	}
	session.TrackedQuestIDs = s.normalizeTrackedQuestIDsLocked(character.TrackedQuestIDs, session.QuestProgress)

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleAutoAttack(w http.ResponseWriter, r *http.Request) {
	var request autoAttackRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.setAutoAttackLocked(session, request.Enabled); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "auto_attack_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleTrainerLearn(w http.ResponseWriter, r *http.Request) {
	var request trainerLearnRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.learnTrainerAbilityLocked(session, request.TrainerID, request.AbilityID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "trainer_learn_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleProfessionLearn(w http.ResponseWriter, r *http.Request) {
	var request professionLearnRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.learnProfessionLocked(session, request.TrainerID, request.ProfessionID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "profession_learn_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleGather(w http.ResponseWriter, r *http.Request) {
	var request gatherRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.gatherNodeLocked(session, request.NodeID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "gather_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleCraft(w http.ResponseWriter, r *http.Request) {
	var request craftRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.craftRecipeLocked(session, request.RecipeID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "craft_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleTalentSelect(w http.ResponseWriter, r *http.Request) {
	var request talentSelectRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.selectTalentLocked(session, request.TalentID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "talent_select_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleActionBarAssign(w http.ResponseWriter, r *http.Request) {
	var request actionBarAssignRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.assignActionBarSlotLocked(session, request.SlotIndex, request.AbilityID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "action_bar_assign_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleActionBarClear(w http.ResponseWriter, r *http.Request) {
	var request actionBarClearRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.clearActionBarSlotLocked(session, request.SlotIndex); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "action_bar_clear_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleAbility(w http.ResponseWriter, r *http.Request) {
	var request abilityRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.activateAbilityLocked(session, request.AbilityID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "ability_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleState(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("worldSessionToken")
	if token == "" {
		httpapi.Error(w, http.StatusBadRequest, "missing_token", "worldSessionToken query parameter is required.")
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[token]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) buildResponse(session *worldSessionState) map[string]any {
	s.ensureMobsLocked()
	s.ensureGatheringNodesLocked()
	s.touchSessionLocked(session)

	if session.QuestProgress == nil {
		session.QuestProgress = map[string]platform.CharacterQuestProgress{}
	}

	entities := make([]sessionEntity, 0, len(s.sessionsByToken)+len(s.mobOrder)+len(s.friendlyNPCOrder)+len(s.gatheringNodeOrder))
	for _, npcID := range s.friendlyNPCOrder {
		npc := s.friendlyNPCs[npcID]
		npcZoneID := npc.ZoneID
		if npcZoneID == "" {
			npcZoneID = defaultZoneID
		}
		if npcZoneID != session.ZoneID {
			continue
		}
		entities = append(entities, sessionEntity{
			ID:          npc.ID,
			DisplayName: npc.DisplayName,
			Kind:        npc.Kind,
			X:           npc.X,
			Y:           npc.Y,
			Z:           npc.Z,
			Health:      1,
			MaxHealth:   1,
			Alive:       true,
			Targetable:  true,
			AIState:     npc.AIState,
			Services:    npc.Services,
		})
	}

	nowMs := nowMillis()
	for _, nodeID := range s.gatheringNodeOrder {
		node := s.gatheringNodes[nodeID]
		if node == nil {
			continue
		}
		nodeZoneID := node.Definition.ZoneID
		if nodeZoneID == "" {
			nodeZoneID = defaultZoneID
		}
		if nodeZoneID != session.ZoneID {
			continue
		}
		entities = append(entities, buildGatheringNodeEntity(node, nowMs))
	}

	if session.HousingSpaceID != "" {
		entities = append(entities, s.buildHousingEntitiesLocked(session)...)
	}

	for _, mob := range s.hostileMobsForSessionLocked(session) {
		if mob == nil || mob.ZoneID != session.ZoneID {
			continue
		}

		entities = append(entities, sessionEntity{
			ID:              mob.ID,
			ArchetypeID:     mob.ArchetypeID,
			SpawnPointID:    mob.SpawnPointID,
			DisplayName:     mob.DisplayName,
			Kind:            mob.Kind,
			MobTypeID:       mob.MobTypeID,
			Disposition:     mob.Disposition,
			Classification:  mob.Classification,
			Elite:           mob.Elite,
			X:               mob.X,
			Y:               mob.Y,
			Z:               mob.Z,
			Health:          mob.Health,
			MaxHealth:       mob.MaxHealth,
			Alive:           mob.Alive,
			Targetable:      mob.Targetable,
			IsInCombat:      mob.CurrentTargetCharacter != "",
			CurrentTargetID: mob.CurrentTargetCharacter,
			LastDamagedByID: mob.LastDamagedByCharacter,
			RespawnDelayMs:  mob.RespawnDelayMs,
			DeathTick:       mob.DeathTick,
			RespawnTick:     mob.RespawnTick,
			AIState:         mob.AIState,
		})
	}

	for _, candidate := range s.sessionsByToken {
		if candidate.Token == session.Token || !candidate.Connected || candidate.ZoneID != session.ZoneID {
			continue
		}
		if candidate.InstanceID != session.InstanceID || candidate.HousingSpaceID != session.HousingSpaceID {
			continue
		}
		pvpState, duelOpponent := s.pvpStateForVisiblePlayerLocked(session, candidate)

		entities = append(entities, sessionEntity{
			ID:           candidate.CharacterID,
			DisplayName:  candidate.DisplayName,
			Kind:         "player",
			X:            candidate.X,
			Y:            candidate.Y,
			Z:            candidate.Z,
			Health:       candidate.Health,
			MaxHealth:    candidate.MaxHealth,
			Alive:        candidate.Alive,
			Targetable:   true,
			PvPState:     pvpState,
			DuelOpponent: duelOpponent,
		})
	}

	return map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"realmId":           session.RealmID,
		"zoneId":            session.ZoneID,
		"displayName":       session.DisplayName,
		"level":             session.Level,
		"position": map[string]float64{
			"x": session.X,
			"y": session.Y,
			"z": session.Z,
		},
		"health":         session.Health,
		"maxHealth":      session.MaxHealth,
		"resource":       session.Resource,
		"maxResource":    session.MaxResource,
		"resourceName":   "Grit",
		"alive":          session.Alive,
		"experience":     session.Experience,
		"currencyCopper": session.CurrencyCopper,
		"currency":       breakdownCurrency(session.CurrencyCopper),
		"inventory": inventoryResponse{
			SlotCount: platform.InventorySlotCount,
			Slots:     buildInventorySlotsResponse(session.Inventory),
		},
		"equipment": equipmentResponse{
			Slots: platform.NormalizeEquipmentSlots(session.Equipment),
		},
		"stats":                s.buildStatsResponse(session),
		"talents":              s.buildTalentsResponse(session),
		"learnedAbilityIds":    platform.NormalizeLearnedAbilityIDs(session.LearnedAbilityIDs),
		"spellbook":            s.buildSpellbookResponse(session),
		"actionBar":            s.buildActionBarResponse(session),
		"trainer":              s.buildTrainerResponse(session),
		"professions":          s.buildProfessionsResponse(session),
		"professionTrainer":    s.buildProfessionTrainerResponse(session),
		"vendor":               s.buildVendorResponse(session),
		"quest":                s.buildQuestResponse(session),
		"quests":               s.buildQuestListResponse(session),
		"lootContainers":       s.buildLootContainersResponseLocked(session),
		"domainEvents":         cloneDomainEvents(s.domainEvents),
		"stateDiffs":           cloneStateDiffs(s.stateDiffs),
		"trackedQuestIds":      s.normalizeTrackedQuestIDsLocked(session.TrackedQuestIDs, session.QuestProgress),
		"zoneMap":              s.buildZoneMapResponse(session),
		"navigationAreas":      s.buildNavigationAreasResponse(session),
		"mapMarkers":           s.buildMapMarkersResponse(session),
		"housing":              s.buildHousingResponseLocked(session),
		"instance":             s.buildDungeonInstanceResponse(session),
		"travel":               s.buildTravelStateResponseLocked(session),
		"mounts":               s.buildMountCollectionResponseLocked(session),
		"pvp":                  s.buildPvPResponseLocked(session),
		"currentTargetId":      session.CurrentTargetID,
		"autoAttackActive":     session.AutoAttackActive,
		"globalCooldownEndsAt": session.GlobalCooldownEnds,
		"castEndsAt":           session.CastEndsAtMs,
		"castingAbilityId":     session.CastingAbilityID,
		"entities":             entities,
	}
}

func (s *worldServer) findConnectedSessionByCharacterLocked(characterID string) *worldSessionState {
	if characterID == "" {
		return nil
	}

	for _, session := range s.sessionsByToken {
		if session.CharacterID == characterID && session.Connected {
			return session
		}
	}

	return nil
}

func distance2D(x1 float64, y1 float64, x2 float64, y2 float64) float64 {
	// Gameplay range and targeting stay planar so client-local grounded Z never affects authoritative logic.
	return math.Hypot(x2-x1, y2-y1)
}

func randomWorldToken() string {
	buffer := make([]byte, 16)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}

func clamp(value float64, minimum float64, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func (s *worldServer) resolveMovementLocked(session *worldSessionState, deltaX float64, deltaY float64) (float64, float64) {
	maxX := starterZoneMaxX
	maxY := starterZoneMaxY
	if session != nil {
		if zone, ok := s.zones[session.ZoneID]; ok {
			maxX = zone.Bounds.MaxX
			maxY = zone.Bounds.MaxY
		}
	}
	candidateX := clamp(session.X+deltaX, 0.0, maxX)
	candidateY := clamp(session.Y+deltaY, 0.0, maxY)

	if session != nil && session.ZoneID == defaultZoneID &&
		candidateX >= 72.0 && candidateX <= 80.0 && candidateY >= 28.0 && candidateY <= 46.0 {
		return session.X, session.Y
	}

	return candidateX, candidateY
}
