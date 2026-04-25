package worlds

import (
	"net/http"
	"strings"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

type worldAdminTeleportRequest struct {
	Destination string `json:"destination"`
	Reason      string `json:"reason"`
	Confirm     bool   `json:"confirm"`
}

type worldAdminRepairRequest struct {
	Normalize               bool     `json:"normalize"`
	RestoreStarterAbilities bool     `json:"restoreStarterAbilities"`
	RebuildActionBar        bool     `json:"rebuildActionBar"`
	Revive                  bool     `json:"revive"`
	ForceLogout             bool     `json:"forceLogout"`
	Actions                 []string `json:"actions"`
	Reason                  string   `json:"reason"`
	Confirm                 bool     `json:"confirm"`
}

type worldAdminItemRequest struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
	Reason   string `json:"reason"`
	Confirm  bool   `json:"confirm"`
}

type worldAdminCurrencyRequest struct {
	Copper  int    `json:"copper"`
	Reason  string `json:"reason"`
	Confirm bool   `json:"confirm"`
}

type worldAdminInvalidateRequest struct {
	Reason  string `json:"reason"`
	Confirm bool   `json:"confirm"`
}

func (s *worldServer) handleAdminTeleport(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
	var request worldAdminTeleportRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	reason, ok := worldRequiredMutationReason(w, request.Reason, request.Confirm)
	if !ok {
		return
	}

	characterID := r.PathValue("characterId")
	character, err := s.store.GetCharacterByID(characterID)
	if err != nil {
		httpapi.Error(w, http.StatusNotFound, "character_missing", err.Error())
		return
	}
	destination := StonewakeAdminSafePosition()
	if strings.TrimSpace(request.Destination) == "current_zone_safe" {
		destination = CurrentZoneAdminSafePosition(character.ZoneID)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	before := worldCharacterPositionSummary(*character)
	liveSession := s.findSessionByCharacterIDLocked(characterID)
	if liveSession != nil {
		liveSession.ZoneID = destination.ZoneID
		liveSession.X = destination.X
		liveSession.Y = destination.Y
		liveSession.Z = destination.Z
		liveSession.Connected = true
		s.resetSessionCombatStateLocked(liveSession, "admin_teleport")
		s.clearTargetLocked(liveSession, "admin_teleport")
		s.clearMobAggroForCharacterLocked(liveSession.CharacterID)
		s.cancelDuelForCharacterLocked(liveSession.CharacterID, duelReasonDisconnect)
	}
	updated, err := s.store.UpdateCharacterState(characterID, destination.ZoneID, destination.X, destination.Y, destination.Z)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "teleport_failed", err.Error())
		return
	}
	if liveSession != nil {
		liveSession.ZoneID = updated.ZoneID
		liveSession.X = updated.PositionX
		liveSession.Y = updated.PositionY
		liveSession.Z = updated.PositionZ
		_ = s.auditAdminAction(actor.ID, "admin.character_teleported", updated.AccountID, updated.ID, reason, before, worldCharacterPositionSummary(*updated), map[string]any{"liveSession": true, "destination": request.Destination})
		httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(liveSession))
		return
	}
	_ = s.auditAdminAction(actor.ID, "admin.character_teleported", updated.AccountID, updated.ID, reason, before, worldCharacterPositionSummary(*updated), map[string]any{"liveSession": false, "destination": request.Destination})
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"character": updated})
}

func (s *worldServer) handleAdminRepair(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
	var request worldAdminRepairRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	reason, ok := worldRequiredMutationReason(w, request.Reason, request.Confirm)
	if !ok {
		return
	}

	characterID := r.PathValue("characterId")
	before, after, err := s.store.NormalizeCharacterForAdmin(
		characterID,
		request.RestoreStarterAbilities || worldHasAction(request.Actions, "restore_starter_abilities"),
		request.RebuildActionBar || worldHasAction(request.Actions, "rebuild_action_bar"))
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "repair_failed", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	liveSession := s.findSessionByCharacterIDLocked(characterID)
	if liveSession != nil {
		s.applyCharacterProgressionLocked(liveSession, &after)
		s.resetSessionCombatStateLocked(liveSession, "admin_repair")
		s.clearTargetLocked(liveSession, "admin_repair")
		s.clearMobAggroForCharacterLocked(liveSession.CharacterID)
		s.cancelDuelForCharacterLocked(liveSession.CharacterID, duelReasonDisconnect)
		if request.Revive || worldHasAction(request.Actions, "revive") {
			s.reviveSessionLocked(liveSession)
		}
	}
	if request.ForceLogout || worldHasAction(request.Actions, "force_logout") {
		s.invalidateWorldSessionForCharacterLocked(characterID)
		_ = s.store.RevokeCharacterJoinTickets(characterID)
		_ = s.auditAdminAction(actor.ID, "admin.session_invalidated", after.AccountID, after.ID, reason, nil, nil, map[string]any{"scope": "world_session"})
	}

	if err := s.auditAdminAction(actor.ID, "admin.character_repaired", after.AccountID, after.ID, reason, worldRepairSummary(before), worldRepairSummary(after), map[string]any{"actions": request.Actions}); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
		return
	}
	if liveSession != nil && !request.ForceLogout && !worldHasAction(request.Actions, "force_logout") {
		httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(liveSession))
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"character": after})
}

func (s *worldServer) handleAdminSessionInvalidate(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
	var request worldAdminInvalidateRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	reason, ok := worldRequiredMutationReason(w, request.Reason, request.Confirm)
	if !ok {
		return
	}

	characterID := r.PathValue("characterId")
	character, err := s.store.GetCharacterByID(characterID)
	if err != nil {
		httpapi.Error(w, http.StatusNotFound, "character_missing", err.Error())
		return
	}
	s.mutex.Lock()
	s.invalidateWorldSessionForCharacterLocked(characterID)
	s.mutex.Unlock()
	_ = s.store.RevokeCharacterJoinTickets(characterID)
	if err := s.auditAdminAction(actor.ID, "admin.session_invalidated", character.AccountID, character.ID, reason, nil, nil, map[string]any{"scope": "world_session"}); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *worldServer) handleAdminItemGrant(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
	s.handleAdminItemChange(w, r, actor, true)
}

func (s *worldServer) handleAdminItemRemove(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
	s.handleAdminItemChange(w, r, actor, false)
}

func (s *worldServer) handleAdminCurrencyGrant(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
	s.handleAdminCurrencyChange(w, r, actor, true)
}

func (s *worldServer) handleAdminCurrencyRemove(w http.ResponseWriter, r *http.Request, session *platform.Session, actor *platform.Account) {
	s.handleAdminCurrencyChange(w, r, actor, false)
}

func (s *worldServer) handleAdminItemChange(w http.ResponseWriter, r *http.Request, actor *platform.Account, grant bool) {
	var request worldAdminItemRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	reason, ok := worldRequiredMutationReason(w, request.Reason, request.Confirm)
	if !ok {
		return
	}
	item, found := FindAdminItemDefinition(request.ItemID)
	if !found {
		httpapi.Error(w, http.StatusBadRequest, "item_unknown", "Item ID is not defined.")
		return
	}

	var before platform.Character
	var after platform.Character
	var err error
	action := "admin.item_granted"
	if grant {
		before, after, err = s.store.GrantCharacterItem(r.PathValue("characterId"), item.ItemID, item.DisplayName, request.Quantity, item.MaxStack, item.Stackable)
	} else {
		action = "admin.item_removed"
		before, after, err = s.store.RemoveCharacterItem(r.PathValue("characterId"), item.ItemID, request.Quantity)
	}
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "item_change_failed", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	liveSession := s.findSessionByCharacterIDLocked(after.ID)
	if liveSession != nil {
		s.applyCharacterProgressionLocked(liveSession, &after)
	}
	if err := s.auditAdminAction(actor.ID, action, after.AccountID, after.ID, reason, worldInventoryItemSummary(before, item.ItemID), worldInventoryItemSummary(after, item.ItemID), map[string]any{"itemId": item.ItemID, "quantity": request.Quantity, "liveSession": liveSession != nil}); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
		return
	}
	if liveSession != nil {
		httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(liveSession))
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"character": after})
}

func (s *worldServer) handleAdminCurrencyChange(w http.ResponseWriter, r *http.Request, actor *platform.Account, grant bool) {
	var request worldAdminCurrencyRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	reason, ok := worldRequiredMutationReason(w, request.Reason, request.Confirm)
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
	before, after, err := s.store.ChangeCharacterCurrency(r.PathValue("characterId"), delta)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "currency_change_failed", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	liveSession := s.findSessionByCharacterIDLocked(after.ID)
	if liveSession != nil {
		s.applyCharacterProgressionLocked(liveSession, &after)
	}
	if err := s.auditAdminAction(actor.ID, action, after.AccountID, after.ID, reason, map[string]any{"currencyCopper": before.CurrencyCopper}, map[string]any{"currencyCopper": after.CurrencyCopper}, map[string]any{"copper": request.Copper, "liveSession": liveSession != nil}); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "audit_failed", err.Error())
		return
	}
	if liveSession != nil {
		httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(liveSession))
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"character": after})
}

func (s *worldServer) findSessionByCharacterIDLocked(characterID string) *worldSessionState {
	token := s.sessionTokenByChar[characterID]
	if token == "" {
		return nil
	}
	return s.sessionsByToken[token]
}

func (s *worldServer) invalidateWorldSessionForCharacterLocked(characterID string) {
	token := s.sessionTokenByChar[characterID]
	if token == "" {
		return
	}
	if session := s.sessionsByToken[token]; session != nil {
		s.cancelDuelForCharacterLocked(session.CharacterID, duelReasonDisconnect)
		s.clearMobAggroForCharacterLocked(session.CharacterID)
	}
	delete(s.sessionsByToken, token)
	delete(s.sessionTokenByChar, characterID)
	s.metrics.recordSessionEvent("admin_invalidated")
}

func (s *worldServer) auditAdminAction(actorAccountID string, action string, targetAccountID string, targetCharacterID string, reason string, before map[string]any, after map[string]any, metadata map[string]any) error {
	event, err := s.store.RecordAuditEvent(platform.AuditEvent{
		Action:            action,
		ActorAccountID:    actorAccountID,
		TargetAccountID:   targetAccountID,
		TargetCharacterID: targetCharacterID,
		Reason:            reason,
		BeforeSummary:     before,
		AfterSummary:      after,
		Metadata:          metadata,
	})
	if err != nil {
		return err
	}
	observability.LogEvent("world-service", action, map[string]any{
		"auditEventId":      event.ID,
		"actorAccountId":    actorAccountID,
		"targetAccountId":   targetAccountID,
		"targetCharacterId": targetCharacterID,
		"reasonProvided":    strings.TrimSpace(reason) != "",
	})
	return nil
}

func worldRequiredMutationReason(w http.ResponseWriter, reason string, confirmed bool) (string, bool) {
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

func worldHasAction(actions []string, target string) bool {
	for _, action := range actions {
		if action == target {
			return true
		}
	}
	return false
}

func worldCharacterPositionSummary(character platform.Character) map[string]any {
	return map[string]any{
		"zoneId": character.ZoneID,
		"x":      character.PositionX,
		"y":      character.PositionY,
		"z":      character.PositionZ,
	}
}

func worldRepairSummary(character platform.Character) map[string]any {
	return map[string]any{
		"zoneId":            character.ZoneID,
		"learnedAbilityIds": character.LearnedAbilityIDs,
		"actionBarSlots":    len(character.ActionBarSlots),
		"trackedQuestIds":   character.TrackedQuestIDs,
	}
}

func worldInventoryItemSummary(character platform.Character, itemID string) map[string]any {
	count := 0
	for _, slot := range character.Inventory {
		if slot.ItemID == itemID {
			count += slot.StackCount
		}
	}
	return map[string]any{"itemId": itemID, "count": count}
}
