package worlds

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
	worldloop "amandacore/services/internal/worlds/loop"
	"amandacore/services/internal/worlds/replication"
)

const stonewakeCommandTimeout = 3 * time.Second

type stonewakeHTTPResponse struct {
	Status      int
	Body        any
	SinceCursor replication.Cursor
}

type stonewakeCommandHTTPError struct {
	status  int
	code    string
	message string
}

func (e stonewakeCommandHTTPError) Error() string {
	return e.message
}

func newStonewakeCommandHTTPError(status int, code string, message string) error {
	return stonewakeCommandHTTPError{status: status, code: code, message: message}
}

func (s *worldServer) startStonewakeLoop() {
	shardLoop := worldloop.NewShardLoop(worldloop.ShardLoopConfig{
		ShardID:        worldloop.StonewakeShardID,
		ZoneID:         defaultZoneID,
		QueueLimit:     1024,
		CommandTimeout: stonewakeCommandTimeout,
		Observer:       s.observeStonewakeLoopEvent,
	})
	if err := shardLoop.Start(); err != nil {
		observability.LogEvent("world-service", "world.loop_start_failed", map[string]any{
			"shardId": worldloop.StonewakeShardID,
			"zoneId":  defaultZoneID,
			"reason":  err.Error(),
		})
		return
	}
	s.stonewakeLoop = shardLoop
}

func (s *worldServer) submitStonewakeHTTPCommand(
	ctx context.Context,
	kind worldloop.CommandKind,
	sessionToken string,
	actorID string,
	payload map[string]any,
	apply func(state *worldloop.ShardState) (stonewakeHTTPResponse, error),
) (stonewakeHTTPResponse, error) {
	if s.stonewakeLoop == nil {
		return stonewakeHTTPResponse{}, fmt.Errorf("stonewake world loop is not available")
	}

	result, err := s.stonewakeLoop.Submit(ctx, worldloop.CommandFunc{
		CommandKind: kind,
		Token:       sessionToken,
		Actor:       actorID,
		Payload:     payload,
		ApplyCommand: func(state *worldloop.ShardState, commandContext worldloop.CommandContext) (worldloop.CommandResult, error) {
			response, err := apply(state)
			if err != nil {
				return worldloop.CommandResult{}, err
			}
			return worldloop.CommandResult{
				CommandID: commandContext.CommandID,
				Sequence:  commandContext.Sequence,
				Kind:      kind,
				Tick:      commandContext.Tick,
				Snapshot:  state.Snapshot(),
				Payload:   response,
			}, nil
		},
	})
	if err != nil {
		return stonewakeHTTPResponse{}, err
	}
	response, ok := result.Payload.(stonewakeHTTPResponse)
	if !ok {
		return stonewakeHTTPResponse{}, fmt.Errorf("stonewake command returned unexpected payload %T", result.Payload)
	}
	response = s.attachStonewakeReplicationMetadata(response, result)
	return response, nil
}

func (s *worldServer) submitStonewakeSessionMutation(
	ctx context.Context,
	kind worldloop.CommandKind,
	sessionToken string,
	payload map[string]any,
	errorCode string,
	mutate func(session *worldSessionState, state *worldloop.ShardState) error,
) (stonewakeHTTPResponse, error) {
	return s.submitStonewakeHTTPCommand(ctx, kind, sessionToken, "", payload, func(state *worldloop.ShardState) (stonewakeHTTPResponse, error) {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		if err := s.advanceWorldLocked(time.Now()); err != nil {
			return stonewakeHTTPResponse{}, newStonewakeCommandHTTPError(http.StatusInternalServerError, "world_advance_failed", err.Error())
		}
		session, ok := s.sessionsByToken[sessionToken]
		if !ok {
			return stonewakeHTTPResponse{}, newStonewakeCommandHTTPError(http.StatusNotFound, "world_session_missing", "World session token was not found.")
		}
		if err := mutate(session, state); err != nil {
			return stonewakeHTTPResponse{}, newStonewakeCommandHTTPError(http.StatusBadRequest, errorCode, err.Error())
		}
		s.syncStonewakeStateLocked(state, session)
		return stonewakeHTTPResponse{Status: http.StatusOK, Body: s.buildResponse(session)}, nil
	})
}

func (s *worldServer) attachStonewakeReplicationMetadata(response stonewakeHTTPResponse, result worldloop.CommandResult) stonewakeHTTPResponse {
	body, ok := response.Body.(map[string]any)
	if !ok {
		return response
	}

	frame := result.Replication
	if !response.SinceCursor.Empty() && s.stonewakeLoop != nil {
		frame = s.stonewakeLoop.ReplicationFrameSince(response.SinceCursor, replication.DeltaReasonPoll)
	}
	if frame.ProtocolVersion == "" {
		return response
	}

	cursorToken := frame.Cursor.Token()
	body["snapshotVersion"] = frame.SnapshotVersion
	body["deltaVersion"] = frame.DeltaVersion
	body["cursor"] = cursorToken
	body["fullSnapshot"] = frame.FullSnapshot
	body["resyncRequired"] = frame.ResyncRequired
	body["changed"] = frame.Changed
	replicationBody := map[string]any{
		"protocolVersion": frame.ProtocolVersion,
		"kind":            frame.Kind,
		"snapshotVersion": frame.SnapshotVersion,
		"deltaVersion":    frame.DeltaVersion,
		"cursor":          cursorToken,
		"cursorState":     frame.Cursor,
		"fullSnapshot":    frame.FullSnapshot,
		"resyncRequired":  frame.ResyncRequired,
		"changed":         frame.Changed,
		"reason":          frame.Reason,
	}
	body["replication"] = replicationBody
	if frame.Snapshot != nil {
		replicationBody["snapshot"] = frame.Snapshot
	}
	if frame.Delta != nil {
		replicationBody["delta"] = frame.Delta
	}

	eventName := observability.EventReplicationCursorAccepted
	if !response.SinceCursor.Empty() && frame.ResyncRequired {
		eventName = observability.EventReplicationCursorStale
	}
	observability.LogEvent("world-service", eventName, map[string]any{
		"cursor":         cursorToken,
		"stateVersion":   frame.SnapshotVersion,
		"resyncRequired": frame.ResyncRequired,
	})
	if frame.ResyncRequired {
		observability.LogEvent("world-service", observability.EventReplicationResyncRequired, map[string]any{
			"cursor":       cursorToken,
			"stateVersion": frame.SnapshotVersion,
		})
	}
	return response
}

func (s *worldServer) writeStonewakeCommandError(w http.ResponseWriter, err error) {
	var httpErr stonewakeCommandHTTPError
	if errors.As(err, &httpErr) {
		httpapi.Error(w, httpErr.status, httpErr.code, httpErr.message)
		return
	}
	if errors.Is(err, worldloop.ErrCommandTimeout) {
		httpapi.Error(w, http.StatusGatewayTimeout, "world_command_timeout", err.Error())
		return
	}
	if worldloop.IsStopped(err) {
		httpapi.Error(w, http.StatusServiceUnavailable, "world_loop_unavailable", err.Error())
		return
	}
	httpapi.Error(w, http.StatusInternalServerError, "world_command_failed", err.Error())
}

func (s *worldServer) observeStonewakeLoopEvent(event worldloop.LoopEvent) {
	fields := map[string]any{
		"shardId":    event.ShardID,
		"zoneId":     event.ZoneID,
		"queueDepth": event.QueueDepth,
	}
	if event.CommandID != "" {
		fields["commandId"] = event.CommandID
	}
	if event.CommandKind != "" {
		fields["commandKind"] = string(event.CommandKind)
	}
	if event.SessionToken != "" {
		fields["worldSessionToken"] = event.SessionToken
	}
	if event.ActorID != "" {
		fields["actorId"] = event.ActorID
	}
	if event.Sequence != 0 {
		fields["sequence"] = event.Sequence
	}
	if event.Tick != 0 {
		fields["tick"] = event.Tick
	}
	if event.Latency > 0 {
		fields["latencyMs"] = float64(event.Latency.Microseconds()) / 1000.0
	}
	if event.Err != nil {
		fields["reason"] = event.Err.Error()
	}

	observability.LogEvent("world-service", event.Name, fields)
	if event.Name == "world.reconnect_restored" {
		observability.LogEvent("world-service", observability.EventWorldReconnectRestored, fields)
	}
}

func (s *worldServer) syncStonewakeStateLocked(state *worldloop.ShardState, session *worldSessionState) {
	if state == nil {
		return
	}
	s.syncStonewakeNPCsLocked(state)
	s.syncStonewakeSessionLocked(state, session)
}

func (s *worldServer) syncStonewakeSessionLocked(state *worldloop.ShardState, session *worldSessionState) {
	if state == nil || session == nil || session.Token == "" {
		return
	}
	if session.ZoneID != defaultZoneID || session.InstanceID != "" || session.HousingSpaceID != "" {
		state.RemoveSession(session.Token)
		return
	}

	state.UpsertPlayer(worldloop.PlayerState{
		SessionToken:     session.Token,
		AccountID:        session.AccountID,
		CharacterID:      session.CharacterID,
		DisplayName:      session.DisplayName,
		ZoneID:           session.ZoneID,
		Position:         worldloop.Position{X: session.X, Y: session.Y, Z: session.Z},
		Connected:        session.Connected,
		Health:           session.Health,
		MaxHealth:        session.MaxHealth,
		Resource:         session.Resource,
		MaxResource:      session.MaxResource,
		Alive:            session.Alive,
		TargetID:         session.CurrentTargetID,
		AutoAttackActive: session.AutoAttackActive,
		QuestProgress:    stonewakeQuestProgress(session.QuestProgress),
		InventorySlots:   stonewakeInventorySlots(session.Inventory),
		ActionBarSlots:   stonewakeActionBarSlots(session.ActionBarSlots, session.LearnedAbilityIDs),
	})
}

func (s *worldServer) syncStonewakeNPCsLocked(state *worldloop.ShardState) {
	if state == nil {
		return
	}
	for _, mobID := range s.mobOrder {
		mob := s.mobs[mobID]
		if mob == nil || mob.ZoneID != defaultZoneID || mob.InstanceID != "" {
			continue
		}
		state.UpsertNPC(worldloop.NpcState{
			ID:          mob.ID,
			ZoneID:      mob.ZoneID,
			Position:    worldloop.Position{X: mob.X, Y: mob.Y, Z: mob.Z},
			Health:      mob.Health,
			MaxHealth:   mob.MaxHealth,
			Alive:       mob.Alive,
			Targetable:  mob.Targetable,
			TargetID:    mob.CurrentTargetCharacter,
			DisplayName: mob.DisplayName,
			Kind:        mob.Kind,
			Threat:      stonewakeThreatTable(mob.ThreatByCharacter),
			RespawnTick: uint64(mob.RespawnTick),
		})
	}
}

func stonewakeQuestProgress(source map[string]platform.CharacterQuestProgress) map[string]int {
	if len(source) == 0 {
		return nil
	}
	progress := make(map[string]int, len(source))
	for questID, questProgress := range source {
		progress[questID] = questProgress.CurrentCount
	}
	return progress
}

func stonewakeInventorySlots(source []platform.CharacterInventorySlot) map[int]string {
	slots := map[int]string{}
	for _, slot := range platform.NormalizeInventorySlots(source) {
		if slot.ItemID != "" && slot.StackCount > 0 {
			slots[slot.SlotIndex] = slot.ItemID
		}
	}
	if len(slots) == 0 {
		return nil
	}
	return slots
}

func stonewakeActionBarSlots(source []platform.CharacterActionBarSlot, learnedAbilityIDs []string) map[int]string {
	slots := map[int]string{}
	for _, slot := range platform.NormalizeActionBarSlots(source, learnedAbilityIDs) {
		if slot.AbilityID != "" {
			slots[slot.SlotIndex] = slot.AbilityID
		}
	}
	if len(slots) == 0 {
		return nil
	}
	return slots
}

func stonewakeThreatTable(source map[string]float64) map[string]float64 {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]float64, len(source))
	for characterID, threat := range source {
		if threat > 0 {
			cloned[characterID] = threat
		}
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}
