package worlds

import (
	"context"
	"testing"
	"time"

	"amandacore/services/internal/platform"
	"amandacore/services/internal/simcore"
)

type memoryCharacterWriter struct {
	saved map[simcore.CharacterID]DirtyCharacterState
}

func (w *memoryCharacterWriter) UpdateCharacterState(characterID string, zoneID string, x float64, y float64, z float64) (*platform.Character, error) {
	if w.saved == nil {
		w.saved = map[simcore.CharacterID]DirtyCharacterState{}
	}
	w.saved[simcore.CharacterID(characterID)] = DirtyCharacterState{
		CharacterID: simcore.CharacterID(characterID),
		ZoneID:      simcore.ZoneID(zoneID),
		Position:    simcore.Vector3{X: x, Y: y, Z: z},
	}
	return &platform.Character{ID: characterID, ZoneID: zoneID, PositionX: x, PositionY: y, PositionZ: z}, nil
}

func TestSessionGatewayRejectsUnauthenticatedAndUnboundState(t *testing.T) {
	gateway := NewSessionGateway()

	_, validation := gateway.ValidateCommand(simcore.CommandEnvelope{
		SessionID:   "missing",
		CharacterID: "runner",
		Payload:     simcore.HeartbeatIntentCommand{CharacterID: "runner"},
	})
	if validation.Accepted {
		t.Fatalf("expected missing session command to be rejected")
	}
	if validation.Reason != simcore.RejectionUnauthenticated {
		t.Fatalf("expected unauthenticated rejection, got %s", validation.Reason)
	}

	_, err := gateway.Attach(AttachSessionRequest{
		SessionID:   "session_without_account",
		CharacterID: "runner",
		RealmID:     "realm",
		ZoneID:      "zone",
	})
	if err == nil {
		t.Fatalf("expected attach without account to fail")
	}
}

func TestSessionGatewayDuplicateCharacterSessionIsDeterministic(t *testing.T) {
	gateway := NewSessionGateway()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)

	first, err := gateway.Attach(AttachSessionRequest{
		SessionID:   "session_001",
		AccountID:   "account",
		CharacterID: "runner",
		RealmID:     "realm",
		ZoneID:      "zone",
		Now:         now,
	})
	if err != nil {
		t.Fatalf("first attach failed: %v", err)
	}
	second, err := gateway.Attach(AttachSessionRequest{
		SessionID:   "session_002",
		AccountID:   "account",
		CharacterID: "runner",
		RealmID:     "realm",
		ZoneID:      "zone",
		Now:         now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("second attach failed: %v", err)
	}
	if second.Replaced == nil {
		t.Fatalf("expected second attach to replace first")
	}
	if second.Replaced.SessionID != first.Session.SessionID {
		t.Fatalf("expected replaced session %s, got %s", first.Session.SessionID, second.Replaced.SessionID)
	}

	_, validation := gateway.ValidateCommand(simcore.CommandEnvelope{
		SessionID:   first.Session.SessionID,
		CharacterID: "runner",
		Payload:     simcore.HeartbeatIntentCommand{CharacterID: "runner"},
	})
	if validation.Accepted || validation.Reason != simcore.RejectionSessionInactive {
		t.Fatalf("expected old session to be inactive, got accepted=%v reason=%s", validation.Accepted, validation.Reason)
	}
}

func TestCommandQueuePreservesDeterministicOrdering(t *testing.T) {
	runtime := newTestRuntime(t, 10)
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)

	attachTestSession(t, runtime, "session_b", "char_b", simcore.Vector3{X: 2, Y: 0, Z: playableGroundZ}, now)
	attachTestSession(t, runtime, "session_a", "char_a", simcore.Vector3{X: 1, Y: 0, Z: playableGroundZ}, now)

	commands := []simcore.CommandEnvelope{
		moveEnvelope("cmd_3", "session_b", "char_b", now, 1, 1),
		moveEnvelope("cmd_1", "session_a", "char_a", now, 1, 1),
		moveEnvelope("cmd_2", "session_a", "char_a", now.Add(time.Millisecond), 1, 1),
	}
	for _, command := range commands {
		if _, err := runtime.Enqueue(command); err != nil {
			t.Fatalf("enqueue failed: %v", err)
		}
	}

	result := runtime.RunTick(now)
	if result.CommandsProcessed != 3 {
		t.Fatalf("expected 3 commands, got %d", result.CommandsProcessed)
	}
	if len(result.Diffs) != 1 || len(result.Diffs[0].Deltas) < 3 {
		t.Fatalf("expected movement deltas, got %#v", result.Diffs)
	}
	first := result.Diffs[0].Deltas[0].(simcore.PositionDelta)
	second := result.Diffs[0].Deltas[1].(simcore.PositionDelta)
	third := result.Diffs[0].Deltas[2].(simcore.PositionDelta)
	if first.EntityID != "char_a" || second.EntityID != "char_b" || third.EntityID != "char_a" {
		t.Fatalf("unexpected deterministic order: %s, %s, %s", first.EntityID, second.EntityID, third.EntityID)
	}
}

func TestFullCommandQueueRejectsWithClearReason(t *testing.T) {
	runtime := newTestRuntime(t, 1)
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	attachTestSession(t, runtime, "session", "char", simcore.Vector3{X: 1, Y: 0, Z: playableGroundZ}, now)

	if _, err := runtime.Enqueue(moveEnvelope("cmd_1", "session", "char", now, 1, 1)); err != nil {
		t.Fatalf("first enqueue failed: %v", err)
	}
	if _, err := runtime.Enqueue(moveEnvelope("cmd_2", "session", "char", now, 1, 1)); err == nil {
		t.Fatalf("expected second enqueue to fail")
	}
}

func TestWorldTickProcessesMovementAndProducesStateDiff(t *testing.T) {
	runtime := newTestRuntime(t, 10)
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	attachTestSession(t, runtime, "session", "char", simcore.Vector3{X: 10, Y: 10, Z: playableGroundZ}, now)

	if _, err := runtime.Enqueue(moveEnvelope("cmd_move", "session", "char", now, 4, 2)); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	result := runtime.RunTick(now)
	if result.CommandsRejected != 0 {
		t.Fatalf("expected movement to be accepted, got rejections %#v", result.Rejections)
	}
	if len(result.Diffs) != 1 {
		t.Fatalf("expected one state diff, got %d", len(result.Diffs))
	}
	position := result.Diffs[0].Deltas[0].(simcore.PositionDelta)
	if position.To.X != 14 || position.To.Y != 12 {
		t.Fatalf("expected authoritative move to 14,12 got %#v", position.To)
	}
}

func TestInvalidMovementIsCorrected(t *testing.T) {
	runtime := newTestRuntime(t, 10)
	runtime.movementRules.MaxStepDistance = 2
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	attachTestSession(t, runtime, "session", "char", simcore.Vector3{X: 10, Y: 10, Z: playableGroundZ}, now)

	if _, err := runtime.Enqueue(moveEnvelope("cmd_fast", "session", "char", now, 20, 0)); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	result := runtime.RunTick(now)
	if result.CommandsRejected != 0 {
		t.Fatalf("expected correction, not rejection: %#v", result.Rejections)
	}
	var correction simcore.CorrectionDelta
	found := false
	for _, delta := range result.Diffs[0].Deltas {
		if typed, ok := delta.(simcore.CorrectionDelta); ok {
			correction = typed
			found = true
		}
	}
	if !found {
		t.Fatalf("expected correction delta, got %#v", result.Diffs[0].Deltas)
	}
	if correction.AuthoritativePosition.X != 12 || correction.ReasonCode != "speed_limited" {
		t.Fatalf("unexpected correction %#v", correction)
	}
}

func TestDirtyCharacterPositionFlushesThroughPersistenceHandoff(t *testing.T) {
	writer := &memoryCharacterWriter{}
	persistence := NewPersistenceHandoff(writer)
	runtime := NewWorldRuntime(WorldRuntimeConfig{}, NewSessionGateway(), persistence)
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	attachTestSession(t, runtime, "session", "char", simcore.Vector3{X: 10, Y: 10, Z: playableGroundZ}, now)

	if _, err := runtime.Enqueue(moveEnvelope("cmd_move", "session", "char", now, 2, 0)); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	runtime.RunTick(now)
	results := persistence.FlushDirty(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected one flush result, got %d", len(results))
	}
	if results[0].Error != nil {
		t.Fatalf("flush failed: %v", results[0].Error)
	}
	saved := writer.saved["char"]
	if saved.Position.X != 12 || saved.Position.Y != 10 {
		t.Fatalf("unexpected saved position %#v", saved.Position)
	}
}

func TestDisconnectReconnectRestoresAuthoritativePosition(t *testing.T) {
	runtime := newTestRuntime(t, 10)
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	attachTestSession(t, runtime, "session", "char", simcore.Vector3{X: 10, Y: 10, Z: playableGroundZ}, now)

	if _, err := runtime.Enqueue(moveEnvelope("cmd_move", "session", "char", now, 4, 2)); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	runtime.RunTick(now)
	if _, err := runtime.gateway.Disconnect("session", "test_disconnect", now); err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}
	if _, err := runtime.gateway.CompleteReconnect("session", simcore.Vector3{X: 14, Y: 12, Z: playableGroundZ}, now); err != nil {
		t.Fatalf("reconnect failed: %v", err)
	}
	session, ok := runtime.gateway.Session("session")
	if !ok {
		t.Fatalf("missing session after reconnect")
	}
	if session.AuthoritativePosition.X != 14 || session.AuthoritativePosition.Y != 12 {
		t.Fatalf("expected reconnect at 14,12 got %#v", session.AuthoritativePosition)
	}
}

func newTestRuntime(t *testing.T, capacity int) *WorldRuntime {
	t.Helper()
	gateway := NewSessionGateway()
	persistence := NewPersistenceHandoff(&memoryCharacterWriter{})
	return NewWorldRuntime(WorldRuntimeConfig{
		CommandQueueLimit: capacity,
		MovementRules: MovementRules{
			MaxStepDistance: 12,
			Bounds:          RuntimeBounds{MinX: 0, MinY: 0, MaxX: 100, MaxY: 100},
			ServerZ:         playableGroundZ,
			ControlZ:        true,
		},
	}, gateway, persistence)
}

func attachTestSession(t *testing.T, runtime *WorldRuntime, sessionID simcore.SessionID, characterID simcore.CharacterID, position simcore.Vector3, now time.Time) {
	t.Helper()
	if _, err := runtime.gateway.Attach(AttachSessionRequest{
		SessionID:             sessionID,
		AccountID:             "account_" + simcore.AccountID(characterID),
		CharacterID:           characterID,
		RealmID:               "realm",
		ZoneID:                "zone",
		AuthoritativePosition: position,
		Now:                   now,
	}); err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if err := runtime.RegisterOrUpdateEntity(RuntimeEntity{
		ID:       simcore.EntityID(characterID),
		Kind:     "player",
		ZoneID:   "zone",
		Position: position,
	}); err != nil {
		t.Fatalf("register entity failed: %v", err)
	}
}

func moveEnvelope(commandID simcore.CommandID, sessionID simcore.SessionID, characterID simcore.CharacterID, receivedAt time.Time, deltaX float64, deltaY float64) simcore.CommandEnvelope {
	return simcore.CommandEnvelope{
		CommandID:         commandID,
		SessionID:         sessionID,
		AccountID:         "account_" + simcore.AccountID(characterID),
		CharacterID:       characterID,
		RealmID:           "realm",
		ZoneID:            "zone",
		ServerReceiveTime: receivedAt,
		IntendedTick:      1,
		Payload: simcore.MoveIntentCommand{
			CharacterID: characterID,
			Delta:       simcore.Vector3{X: deltaX, Y: deltaY},
		},
	}
}
