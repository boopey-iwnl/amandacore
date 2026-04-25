package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

type banRequest struct {
	AccountID string `json:"accountId"`
	Banned    bool   `json:"banned"`
	Reason    string `json:"reason"`
}

type roleRequest struct {
	AccountID string        `json:"accountId"`
	Role      platform.Role `json:"role"`
	Reason    string        `json:"reason"`
}

type supportTicketCreateRequest struct {
	CharacterID           string `json:"characterId"`
	Category              string `json:"category"`
	Subject               string `json:"subject"`
	Body                  string `json:"body"`
	AttachedDiagnosticsID string `json:"attachedDiagnosticsId"`
	BuildID               string `json:"buildId"`
	ClientVersion         string `json:"clientVersion"`
}

type supportTicketUpdateRequest struct {
	Status            platform.SupportTicketStatus `json:"status"`
	AssignedToAdminID string                       `json:"assignedToAdminId"`
	ResolutionNote    string                       `json:"resolutionNote"`
	Note              string                       `json:"note"`
	Reason            string                       `json:"reason"`
}

type teleportRequest struct {
	Destination string `json:"destination"`
	Reason      string `json:"reason"`
	Confirm     bool   `json:"confirm"`
}

type repairRequest struct {
	Normalize               bool     `json:"normalize"`
	RestoreStarterAbilities bool     `json:"restoreStarterAbilities"`
	RebuildActionBar        bool     `json:"rebuildActionBar"`
	ForceLogout             bool     `json:"forceLogout"`
	Actions                 []string `json:"actions"`
	Reason                  string   `json:"reason"`
	Confirm                 bool     `json:"confirm"`
}

type itemRequest struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
	Reason   string `json:"reason"`
	Confirm  bool   `json:"confirm"`
}

type currencyRequest struct {
	Copper  int    `json:"copper"`
	Reason  string `json:"reason"`
	Confirm bool   `json:"confirm"`
}

type questRequest struct {
	QuestID string `json:"questId"`
	Reason  string `json:"reason"`
	Confirm bool   `json:"confirm"`
}

type muteRequest struct {
	CharacterID     string `json:"characterId"`
	DurationSeconds int64  `json:"durationSeconds"`
	Reason          string `json:"reason"`
	Confirm         bool   `json:"confirm"`
}

type accountModerationRequest struct {
	AccountID       string `json:"accountId"`
	DurationSeconds int64  `json:"durationSeconds"`
	Reason          string `json:"reason"`
	Confirm         bool   `json:"confirm"`
}

type kickRequest struct {
	AccountID   string `json:"accountId"`
	CharacterID string `json:"characterId"`
	Reason      string `json:"reason"`
	Confirm     bool   `json:"confirm"`
}

func RegisterRoutes(mux *http.ServeMux, fileStore *store.FileStore) {
	mux.Handle("GET /v1/admin/authz/me", httpapi.RequireSession(fileStore, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		account, err := fileStore.GetAccountByID(session.AccountID)
		if err != nil {
			httpapi.Error(w, http.StatusUnauthorized, "invalid_account", err.Error())
			return
		}

		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"account":     sanitizeAccount(*account),
			"permissions": platform.PermissionsForRoles(account.Roles),
		})
	}))

	mux.Handle("GET /v1/admin/accounts", httpapi.RequirePermission(fileStore, platform.PermissionViewAccount, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		accounts, err := fileStore.ListAccounts()
		if err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "account_list_failed", err.Error())
			return
		}

		query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
		results := make([]map[string]any, 0, len(accounts))
		for _, account := range accounts {
			if query != "" &&
				!strings.Contains(strings.ToLower(account.Username), query) &&
				!strings.Contains(strings.ToLower(account.ID), query) {
				continue
			}
			results = append(results, sanitizeAccount(account))
		}

		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"accounts": results})
	}))

	mux.Handle("GET /v1/admin/accounts/{accountId}", httpapi.RequirePermission(fileStore, platform.PermissionViewAccount, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		accountID := r.PathValue("accountId")
		account, err := fileStore.GetAccountByID(accountID)
		if err != nil {
			httpapi.Error(w, http.StatusNotFound, "account_missing", err.Error())
			return
		}
		characters, err := fileStore.SearchCharacters("", account.ID, "")
		if err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "character_lookup_failed", err.Error())
			return
		}
		_ = audit(fileStore, *actor, "admin.account_viewed", account.ID, "", "", nil, nil, nil)
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"account":    sanitizeAccount(*account),
			"characters": sanitizeCharacterList(characters),
		})
	}))

	mux.Handle("POST /v1/admin/accounts/ban", httpapi.RequirePermission(fileStore, platform.PermissionSuspendAccount, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request banRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		before, after, err := fileStore.SetAccountSuspension(request.AccountID, request.Banned, request.Reason, 0)
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "ban_failed", err.Error())
			return
		}
		action := "moderation.suspension_removed"
		if request.Banned {
			action = "moderation.suspension_applied"
			_ = fileStore.RevokeAccountSessions(request.AccountID)
		}
		if err := audit(fileStore, *actor, action, request.AccountID, "", request.Reason, accountModerationSummary(before), accountModerationSummary(after), nil); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "account": sanitizeAccount(after)})
	}))

	mux.Handle("POST /v1/admin/accounts/role", httpapi.RequirePermission(fileStore, platform.PermissionSuspendAccount, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request roleRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		before, _ := fileStore.GetAccountByID(request.AccountID)
		if err := fileStore.SetAccountRole(request.AccountID, request.Role); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "role_failed", err.Error())
			return
		}
		after, _ := fileStore.GetAccountByID(request.AccountID)
		if err := audit(fileStore, *actor, "admin.account_role_updated", request.AccountID, "", request.Reason, roleSummary(before), roleSummary(after), map[string]any{"role": request.Role}); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "account": sanitizeAccount(*after)})
	}))

	mux.Handle("GET /v1/admin/characters", httpapi.RequirePermission(fileStore, platform.PermissionViewCharacter, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		characters, err := fileStore.SearchCharacters(r.URL.Query().Get("query"), r.URL.Query().Get("accountId"), r.URL.Query().Get("realmId"))
		if err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "character_search_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"characters": sanitizeCharacterList(characters)})
	}))

	mux.Handle("GET /v1/admin/characters/{characterId}", httpapi.RequirePermission(fileStore, platform.PermissionViewCharacter, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		character, err := fileStore.GetCharacterByID(r.PathValue("characterId"))
		if err != nil {
			httpapi.Error(w, http.StatusNotFound, "character_missing", err.Error())
			return
		}
		response, err := buildCharacterDetails(fileStore, *character)
		if err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "character_details_failed", err.Error())
			return
		}
		_ = audit(fileStore, *actor, "admin.character_viewed", character.AccountID, character.ID, "", nil, nil, nil)
		httpapi.WriteJSON(w, http.StatusOK, response)
	}))

	mux.Handle("POST /v1/admin/characters/{characterId}/teleport", httpapi.RequirePermission(fileStore, platform.PermissionTeleportCharacter, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request teleportRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
		if !ok {
			return
		}
		character, err := fileStore.GetCharacterByID(r.PathValue("characterId"))
		if err != nil {
			httpapi.Error(w, http.StatusNotFound, "character_missing", err.Error())
			return
		}
		destination := teleportDestination(request.Destination, character.ZoneID)
		updated, err := fileStore.UpdateCharacterState(character.ID, destination.ZoneID, destination.X, destination.Y, destination.Z)
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "teleport_failed", err.Error())
			return
		}
		if err := fileStore.RevokeCharacterJoinTickets(character.ID); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "session_invalidate_failed", err.Error())
			return
		}
		if err := audit(fileStore, *actor, "admin.character_teleported", character.AccountID, character.ID, reason, positionSummary(*character), positionSummary(*updated), map[string]any{"destination": request.Destination}); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"character": sanitizeCharacter(*updated)})
	}))

	mux.Handle("POST /v1/admin/characters/{characterId}/repair", httpapi.RequirePermission(fileStore, platform.PermissionRepairCharacter, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request repairRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
		if !ok {
			return
		}
		characterID := r.PathValue("characterId")
		before, after, err := fileStore.NormalizeCharacterForAdmin(characterID, request.RestoreStarterAbilities || hasAction(request.Actions, "restore_starter_abilities"), request.RebuildActionBar || hasAction(request.Actions, "rebuild_action_bar"))
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "repair_failed", err.Error())
			return
		}
		if request.ForceLogout || hasAction(request.Actions, "force_logout") || hasAction(request.Actions, "clear_invalid_session_state") {
			_ = fileStore.RevokeCharacterJoinTickets(characterID)
		}
		if err := audit(fileStore, *actor, "admin.character_repaired", after.AccountID, after.ID, reason, repairSummary(before), repairSummary(after), map[string]any{"actions": request.Actions}); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		if request.ForceLogout || hasAction(request.Actions, "force_logout") {
			_ = audit(fileStore, *actor, "admin.session_invalidated", after.AccountID, after.ID, reason, nil, nil, map[string]any{"scope": "join_tickets"})
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"character": sanitizeCharacter(after)})
	}))

	mux.Handle("GET /v1/admin/characters/{characterId}/quests", httpapi.RequirePermission(fileStore, platform.PermissionViewCharacter, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		character, err := fileStore.GetCharacterByID(r.PathValue("characterId"))
		if err != nil {
			httpapi.Error(w, http.StatusNotFound, "character_missing", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"quests": character.Quests, "trackedQuestIds": character.TrackedQuestIDs})
	}))

	mux.Handle("POST /v1/admin/characters/{characterId}/quests/reset", httpapi.RequirePermission(fileStore, platform.PermissionModifyQuestState, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request questRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
		if !ok {
			return
		}
		targetCount, found := worlds.AdminQuestTargetCount(request.QuestID)
		if !found {
			httpapi.Error(w, http.StatusBadRequest, "quest_unknown", "Quest ID is not defined.")
			return
		}
		before, after, err := fileStore.ResetCharacterQuest(r.PathValue("characterId"), request.QuestID, targetCount)
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "quest_reset_failed", err.Error())
			return
		}
		if err := audit(fileStore, *actor, "admin.quest_modified", after.AccountID, after.ID, reason, questSummary(before, request.QuestID), questSummary(after, request.QuestID), map[string]any{"operation": "reset", "questId": request.QuestID}); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"quest": after.Quests[request.QuestID]})
	}))

	mux.Handle("POST /v1/admin/characters/{characterId}/quests/complete-objective", httpapi.RequirePermission(fileStore, platform.PermissionModifyQuestState, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request questRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
		if !ok {
			return
		}
		targetCount, found := worlds.AdminQuestTargetCount(request.QuestID)
		if !found {
			httpapi.Error(w, http.StatusBadRequest, "quest_unknown", "Quest ID is not defined.")
			return
		}
		before, after, err := fileStore.CompleteCharacterQuestObjective(r.PathValue("characterId"), request.QuestID, targetCount)
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "quest_complete_failed", err.Error())
			return
		}
		if err := audit(fileStore, *actor, "admin.quest_modified", after.AccountID, after.ID, reason, questSummary(before, request.QuestID), questSummary(after, request.QuestID), map[string]any{"operation": "complete_objective", "questId": request.QuestID}); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"quest": after.Quests[request.QuestID]})
	}))

	mux.Handle("POST /v1/admin/characters/{characterId}/items/grant", httpapi.RequirePermission(fileStore, platform.PermissionGrantItem, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request itemRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
		if !ok {
			return
		}
		item, found := worlds.FindAdminItemDefinition(request.ItemID)
		if !found {
			httpapi.Error(w, http.StatusBadRequest, "item_unknown", "Item ID is not defined.")
			return
		}
		before, after, err := fileStore.GrantCharacterItem(r.PathValue("characterId"), item.ItemID, item.DisplayName, request.Quantity, item.MaxStack, item.Stackable)
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "item_grant_failed", err.Error())
			return
		}
		if err := audit(fileStore, *actor, "admin.item_granted", after.AccountID, after.ID, reason, inventoryItemSummary(before, item.ItemID), inventoryItemSummary(after, item.ItemID), map[string]any{"itemId": item.ItemID, "quantity": request.Quantity}); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"character": sanitizeCharacter(after)})
	}))

	mux.Handle("POST /v1/admin/characters/{characterId}/items/remove", httpapi.RequirePermission(fileStore, platform.PermissionGrantItem, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request itemRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
		if !ok {
			return
		}
		if _, found := worlds.FindAdminItemDefinition(request.ItemID); !found {
			httpapi.Error(w, http.StatusBadRequest, "item_unknown", "Item ID is not defined.")
			return
		}
		before, after, err := fileStore.RemoveCharacterItem(r.PathValue("characterId"), request.ItemID, request.Quantity)
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "item_remove_failed", err.Error())
			return
		}
		if err := audit(fileStore, *actor, "admin.item_removed", after.AccountID, after.ID, reason, inventoryItemSummary(before, request.ItemID), inventoryItemSummary(after, request.ItemID), map[string]any{"itemId": request.ItemID, "quantity": request.Quantity}); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"character": sanitizeCharacter(after)})
	}))

	mux.Handle("POST /v1/admin/characters/{characterId}/currency/grant", httpapi.RequirePermission(fileStore, platform.PermissionGrantCurrency, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		handleCurrencyChange(w, r, fileStore, *actor, true)
	}))
	mux.Handle("POST /v1/admin/characters/{characterId}/currency/remove", httpapi.RequirePermission(fileStore, platform.PermissionGrantCurrency, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		handleCurrencyChange(w, r, fileStore, *actor, false)
	}))

	mux.Handle("POST /v1/support/tickets", httpapi.RequireSession(fileStore, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		var request supportTicketCreateRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		ticket, err := fileStore.CreateSupportTicket(session.AccountID, request.CharacterID, request.Category, request.Subject, request.Body, request.AttachedDiagnosticsID, request.BuildID, request.ClientVersion)
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "ticket_create_failed", err.Error())
			return
		}
		_ = auditWithAccountID(fileStore, session.AccountID, "support.ticket_created", ticket.CreatedByAccountID, ticket.CreatedByCharacterID, "player submitted support ticket", nil, nil, map[string]any{"ticketId": ticket.TicketID, "category": ticket.Category})
		httpapi.WriteJSON(w, http.StatusCreated, map[string]any{"ticket": ticket})
	}))

	mux.Handle("GET /v1/admin/support/tickets", httpapi.RequirePermission(fileStore, platform.PermissionManageSupport, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		tickets, err := fileStore.ListSupportTickets(r.URL.Query().Get("status"))
		if err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "ticket_list_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"tickets": tickets})
	}))

	mux.Handle("GET /v1/admin/support/tickets/{ticketId}", httpapi.RequirePermission(fileStore, platform.PermissionManageSupport, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		ticket, err := fileStore.GetSupportTicket(r.PathValue("ticketId"))
		if err != nil {
			httpapi.Error(w, http.StatusNotFound, "ticket_missing", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"ticket": ticket})
	}))

	mux.Handle("POST /v1/admin/support/tickets/{ticketId}/update", httpapi.RequirePermission(fileStore, platform.PermissionManageSupport, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request supportTicketUpdateRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		ticket, err := fileStore.UpdateSupportTicket(r.PathValue("ticketId"), actor.ID, request.Status, request.AssignedToAdminID, request.ResolutionNote, request.Note)
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "ticket_update_failed", err.Error())
			return
		}
		_ = audit(fileStore, *actor, "support.ticket_updated", ticket.CreatedByAccountID, ticket.CreatedByCharacterID, request.Reason, nil, map[string]any{"status": ticket.Status}, map[string]any{"ticketId": ticket.TicketID})
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"ticket": ticket})
	}))

	mux.Handle("POST /v1/admin/moderation/mute", httpapi.RequirePermission(fileStore, platform.PermissionModerateChat, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request muteRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
		if !ok {
			return
		}
		record, err := fileStore.SetCharacterMute(request.CharacterID, actor.ID, reason, request.DurationSeconds)
		if err != nil {
			httpapi.Error(w, http.StatusBadRequest, "mute_failed", err.Error())
			return
		}
		if err := audit(fileStore, *actor, "moderation.mute_applied", record.AccountID, record.CharacterID, reason, nil, map[string]any{"expiresAt": record.ExpiresAt}, nil); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"mute": record})
	}))

	mux.Handle("POST /v1/admin/moderation/unmute", httpapi.RequirePermission(fileStore, platform.PermissionModerateChat, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request muteRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
		if !ok {
			return
		}
		character, err := fileStore.GetCharacterByID(request.CharacterID)
		if err != nil {
			httpapi.Error(w, http.StatusNotFound, "character_missing", err.Error())
			return
		}
		if err := fileStore.ClearCharacterMute(request.CharacterID); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "unmute_failed", err.Error())
			return
		}
		if err := audit(fileStore, *actor, "moderation.mute_removed", character.AccountID, character.ID, reason, nil, nil, nil); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.Handle("POST /v1/admin/moderation/suspend", httpapi.RequirePermission(fileStore, platform.PermissionSuspendAccount, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		handleSuspension(w, r, fileStore, *actor, true)
	}))
	mux.Handle("POST /v1/admin/moderation/unsuspend", httpapi.RequirePermission(fileStore, platform.PermissionSuspendAccount, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		handleSuspension(w, r, fileStore, *actor, false)
	}))
	mux.Handle("POST /v1/admin/moderation/kick", httpapi.RequirePermission(fileStore, platform.PermissionRepairCharacter, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		var request kickRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
		if !ok {
			return
		}
		targetAccountID := request.AccountID
		targetCharacterID := request.CharacterID
		if targetCharacterID != "" {
			character, err := fileStore.GetCharacterByID(targetCharacterID)
			if err != nil {
				httpapi.Error(w, http.StatusNotFound, "character_missing", err.Error())
				return
			}
			targetAccountID = character.AccountID
			_ = fileStore.RevokeCharacterJoinTickets(targetCharacterID)
		}
		if targetAccountID == "" {
			httpapi.Error(w, http.StatusBadRequest, "target_required", "accountId or characterId is required.")
			return
		}
		if err := fileStore.RevokeAccountSessions(targetAccountID); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "kick_failed", err.Error())
			return
		}
		if err := audit(fileStore, *actor, "admin.session_invalidated", targetAccountID, targetCharacterID, reason, nil, nil, map[string]any{"scope": "auth_sessions_and_join_tickets"}); err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.Handle("GET /v1/admin/audit", httpapi.RequirePermission(fileStore, platform.PermissionViewAuditLog, func(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
		query := store.AuditQuery{
			ActorAccountID:    r.URL.Query().Get("actorAccountId"),
			TargetAccountID:   r.URL.Query().Get("targetAccountId"),
			TargetCharacterID: r.URL.Query().Get("targetCharacterId"),
			Action:            r.URL.Query().Get("action"),
			From:              parseInt64(r.URL.Query().Get("from")),
			To:                parseInt64(r.URL.Query().Get("to")),
			Limit:             int(parseInt64(r.URL.Query().Get("limit"))),
		}
		events, err := fileStore.QueryAuditEvents(query)
		if err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "audit_query_failed", err.Error())
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"events": events})
	}))
}

func handleCurrencyChange(w http.ResponseWriter, r *http.Request, fileStore *store.FileStore, actor platform.Account, grant bool) {
	var request currencyRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
	if !ok {
		return
	}
	if request.Copper <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "invalid_currency", "Copper amount must be positive.")
		return
	}
	delta := request.Copper
	action := "admin.currency_granted"
	if !grant {
		delta = -delta
		action = "admin.currency_removed"
	}
	before, after, err := fileStore.ChangeCharacterCurrency(r.PathValue("characterId"), delta)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "currency_change_failed", err.Error())
		return
	}
	if err := audit(fileStore, actor, action, after.AccountID, after.ID, reason, currencySummary(before), currencySummary(after), map[string]any{"copper": request.Copper}); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"character": sanitizeCharacter(after)})
}

func handleSuspension(w http.ResponseWriter, r *http.Request, fileStore *store.FileStore, actor platform.Account, suspend bool) {
	var request accountModerationRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	reason, ok := requiredMutationReason(w, request.Reason, request.Confirm)
	if !ok {
		return
	}
	before, after, err := fileStore.SetAccountSuspension(request.AccountID, suspend, reason, request.DurationSeconds)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "suspension_failed", err.Error())
		return
	}
	if suspend {
		_ = fileStore.RevokeAccountSessions(request.AccountID)
	}
	action := "moderation.suspension_removed"
	if suspend {
		action = "moderation.suspension_applied"
	}
	if err := audit(fileStore, actor, action, request.AccountID, "", reason, accountModerationSummary(before), accountModerationSummary(after), nil); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"account": sanitizeAccount(after)})
}

func buildCharacterDetails(fileStore *store.FileStore, character platform.Character) (map[string]any, error) {
	account, err := fileStore.GetAccountByID(character.AccountID)
	if err != nil {
		return nil, err
	}
	guild, _ := fileStore.GetGuildForCharacter(character.ID)
	mail, _ := fileStore.ListMailForCharacter(character.ID)
	auctions, _ := fileStore.ListAuctionsForSeller(character.ID)
	entitlement, housingSpace, housingStorage, decorations, _ := fileStore.GetHousingForCharacter(character.ID)
	events, _ := fileStore.QueryAuditEvents(store.AuditQuery{TargetCharacterID: character.ID, Limit: 20})
	mute, _ := fileStore.ActiveMuteForCharacter(character.ID)

	return map[string]any{
		"account":           sanitizeAccount(*account),
		"character":         sanitizeCharacter(character),
		"inventory":         character.Inventory,
		"equipment":         character.Equipment,
		"currencyCopper":    character.CurrencyCopper,
		"learnedAbilityIds": character.LearnedAbilityIDs,
		"actionBarSlots":    character.ActionBarSlots,
		"talents":           character.Talents,
		"quests":            character.Quests,
		"trackedQuestIds":   character.TrackedQuestIDs,
		"professions":       character.Professions,
		"guild":             guildSummary(guild),
		"mailSummary":       mailSummary(mail),
		"tradeSummary":      emptyOwnershipSummary("trades"),
		"auctionSummary":    auctionSummary(auctions),
		"housingSummary":    housingSummary(entitlement, housingSpace, housingStorage, decorations),
		"pvpSummary":        pvpSummary(character.PvPStats),
		"moderation":        map[string]any{"activeMute": mute},
		"recentAuditEvents": events,
	}, nil
}

func sanitizeAccount(account platform.Account) map[string]any {
	status := "active"
	if account.Banned {
		status = "suspended"
	} else if account.SuspendedUntil > 0 && account.SuspendedUntil > time.Now().Unix() {
		status = "suspended"
	}
	return map[string]any{
		"id":               account.ID,
		"username":         account.Username,
		"roles":            account.Roles,
		"permissions":      platform.PermissionsForRoles(account.Roles),
		"status":           status,
		"banned":           account.Banned,
		"suspendedUntil":   account.SuspendedUntil,
		"suspensionReason": account.SuspensionReason,
		"createdAt":        account.CreatedAt,
		"updatedAt":        account.UpdatedAt,
		"lastLoginAt":      account.LastLoginAt,
		"lastSessionId":    account.LastSessionID,
	}
}

func sanitizeCharacter(character platform.Character) map[string]any {
	return map[string]any{
		"id":             character.ID,
		"accountId":      character.AccountID,
		"realmId":        character.RealmID,
		"displayName":    character.DisplayName,
		"raceId":         character.RaceID,
		"classId":        character.ClassID,
		"archetypeId":    character.ArchetypeID,
		"level":          character.Level,
		"experience":     character.Experience,
		"currencyCopper": character.CurrencyCopper,
		"zoneId":         character.ZoneID,
		"position": map[string]float64{
			"x": character.PositionX,
			"y": character.PositionY,
			"z": character.PositionZ,
		},
		"lastSeenAt": character.LastSeenAt,
	}
}

func sanitizeCharacterList(characters []platform.Character) []map[string]any {
	results := make([]map[string]any, 0, len(characters))
	for _, character := range characters {
		results = append(results, sanitizeCharacter(character))
	}
	return results
}

func audit(fileStore *store.FileStore, actor platform.Account, action string, targetAccountID string, targetCharacterID string, reason string, before map[string]any, after map[string]any, metadata map[string]any) error {
	return auditWithAccountID(fileStore, actor.ID, action, targetAccountID, targetCharacterID, reason, before, after, metadata)
}

func auditWithAccountID(fileStore *store.FileStore, actorAccountID string, action string, targetAccountID string, targetCharacterID string, reason string, before map[string]any, after map[string]any, metadata map[string]any) error {
	event, err := fileStore.RecordAuditEvent(platform.AuditEvent{
		Action:            action,
		ActorAccountID:    actorAccountID,
		TargetAccountID:   targetAccountID,
		TargetCharacterID: targetCharacterID,
		Reason:            strings.TrimSpace(reason),
		BeforeSummary:     before,
		AfterSummary:      after,
		Metadata:          metadata,
	})
	if err != nil {
		return err
	}
	observability.LogEvent("admin-service", action, map[string]any{
		"auditEventId":      event.ID,
		"actorAccountId":    actorAccountID,
		"targetAccountId":   targetAccountID,
		"targetCharacterId": targetCharacterID,
		"reasonProvided":    strings.TrimSpace(reason) != "",
	})
	return nil
}

func requiredMutationReason(w http.ResponseWriter, reason string, confirmed bool) (string, bool) {
	reason = strings.TrimSpace(reason)
	if !confirmed {
		httpapi.Error(w, http.StatusBadRequest, "confirmation_required", "Admin mutation requires confirm=true.")
		return "", false
	}
	if reason == "" {
		httpapi.Error(w, http.StatusBadRequest, "reason_required", "Admin mutation requires a reason.")
		return "", false
	}
	return reason, true
}

func teleportDestination(destination string, currentZoneID string) worlds.AdminSafePosition {
	switch strings.TrimSpace(destination) {
	case "current_zone_safe":
		return worlds.CurrentZoneAdminSafePosition(currentZoneID)
	default:
		return worlds.StonewakeAdminSafePosition()
	}
}

func hasAction(actions []string, target string) bool {
	for _, action := range actions {
		if action == target {
			return true
		}
	}
	return false
}

func positionSummary(character platform.Character) map[string]any {
	return map[string]any{
		"zoneId": character.ZoneID,
		"x":      character.PositionX,
		"y":      character.PositionY,
		"z":      character.PositionZ,
	}
}

func repairSummary(character platform.Character) map[string]any {
	return map[string]any{
		"zoneId":            character.ZoneID,
		"learnedAbilityIds": character.LearnedAbilityIDs,
		"actionBarSlots":    len(character.ActionBarSlots),
		"trackedQuestIds":   character.TrackedQuestIDs,
	}
}

func inventoryItemSummary(character platform.Character, itemID string) map[string]any {
	count := 0
	for _, slot := range character.Inventory {
		if slot.ItemID == itemID {
			count += slot.StackCount
		}
	}
	return map[string]any{"itemId": itemID, "count": count}
}

func currencySummary(character platform.Character) map[string]any {
	return map[string]any{"currencyCopper": character.CurrencyCopper}
}

func questSummary(character platform.Character, questID string) map[string]any {
	return map[string]any{"questId": questID, "progress": character.Quests[questID]}
}

func accountModerationSummary(account platform.Account) map[string]any {
	return map[string]any{
		"banned":           account.Banned,
		"suspendedUntil":   account.SuspendedUntil,
		"suspensionReason": account.SuspensionReason,
	}
}

func roleSummary(account *platform.Account) map[string]any {
	if account == nil {
		return map[string]any{}
	}
	return map[string]any{"roles": account.Roles}
}

func guildSummary(guild *platform.Guild) map[string]any {
	if guild == nil {
		return map[string]any{"guildId": "", "guildName": "", "memberCount": 0}
	}
	return map[string]any{
		"guildId":           guild.ID,
		"guildName":         guild.GuildName,
		"leaderCharacterId": guild.LeaderCharacterID,
		"memberCount":       len(guild.Members),
	}
}

func mailSummary(mail []platform.MailEnvelope) map[string]any {
	attachments := 0
	currencyCopper := 0
	for _, envelope := range mail {
		attachments += len(envelope.ItemAttachments)
		currencyCopper += envelope.CurrencyCopper
	}
	return map[string]any{
		"count":          len(mail),
		"attachments":    attachments,
		"currencyCopper": currencyCopper,
		"mail":           mail,
	}
}

func auctionSummary(auctions []platform.AuctionListing) map[string]any {
	active := 0
	for _, auction := range auctions {
		if auction.State == platform.AuctionStateActive {
			active++
		}
	}
	return map[string]any{
		"count":    len(auctions),
		"active":   active,
		"auctions": auctions,
	}
}

func housingSummary(entitlement *platform.HousingEntitlement, space *platform.HousingSpace, storage []platform.HousingStorageSlot, decorations []platform.DecorationPlacement) map[string]any {
	return map[string]any{
		"entitlement":      entitlement,
		"space":            space,
		"storageSlotCount": len(storage),
		"storage":          storage,
		"decorations":      decorations,
	}
}

func pvpSummary(stats platform.CharacterPvPStats) map[string]any {
	return map[string]any{
		"duelsWon":      stats.DuelsWon,
		"duelsLost":     stats.DuelsLost,
		"lastDuelWonAt": stats.LastDuelWonAt,
	}
}

func emptyOwnershipSummary(kind string) map[string]any {
	return map[string]any{"kind": kind, "count": 0, "items": []any{}}
}

func parseInt64(value string) int64 {
	parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return parsed
}
