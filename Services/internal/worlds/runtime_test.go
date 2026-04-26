package worlds

import (
	"testing"
	"time"

	"amandacore/services/internal/simcore"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestWorldRuntimeCommandQueuePreservesDeterministicOrder(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	runtime := NewWorldRuntime(WorldRuntimeConfig{Clock: fixedClock{now: now}})

	commands := []simcore.CommandEnvelope{
		{
			CommandID:  "cmd_c",
			Sequence:   3,
			ReceivedAt: now,
			ActorID:    "player_c",
			ZoneID:     "test_zone",
			Command:    simcore.MoveIntentCommand{EntityID: "player_c", From: simcore.Vector3{X: 2}, To: simcore.Vector3{X: 3}},
		},
		{
			CommandID:  "cmd_a",
			Sequence:   1,
			ReceivedAt: now,
			ActorID:    "player_a",
			ZoneID:     "test_zone",
			Command:    simcore.MoveIntentCommand{EntityID: "player_a", From: simcore.Vector3{X: 0}, To: simcore.Vector3{X: 1}},
		},
		{
			CommandID:  "cmd_b",
			Sequence:   2,
			ReceivedAt: now,
			ActorID:    "player_b",
			ZoneID:     "test_zone",
			Command:    simcore.MoveIntentCommand{EntityID: "player_b", From: simcore.Vector3{X: 1}, To: simcore.Vector3{X: 2}},
		},
	}

	for _, command := range commands {
		if _, err := runtime.Enqueue(command); err != nil {
			t.Fatalf("enqueue failed: %v", err)
		}
	}

	result := runtime.RunTick(now)
	if result.CommandsProcessed != 3 {
		t.Fatalf("expected 3 processed commands, got %d", result.CommandsProcessed)
	}
	if len(result.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(result.Events))
	}

	firstMove, ok := result.Events[0].(simcore.PlayerMovedEvent)
	if !ok {
		t.Fatalf("expected first event to be PlayerMovedEvent, got %T", result.Events[0])
	}
	if firstMove.EntityID != "player_a" {
		t.Fatalf("expected first command to be player_a, got %s", firstMove.EntityID)
	}

	secondMove := result.Events[1].(simcore.PlayerMovedEvent)
	thirdMove := result.Events[2].(simcore.PlayerMovedEvent)
	if secondMove.EntityID != "player_b" || thirdMove.EntityID != "player_c" {
		t.Fatalf("expected deterministic player_b/player_c order, got %s/%s", secondMove.EntityID, thirdMove.EntityID)
	}
}

func TestWorldRuntimeTickAcceptsCommandsAndEmitsDomainEvents(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	runtime := NewWorldRuntime(WorldRuntimeConfig{Clock: fixedClock{now: now}})

	_, err := runtime.Enqueue(simcore.CommandEnvelope{
		ActorID: "runner",
		ZoneID:  "test_zone",
		Command: simcore.MoveIntentCommand{
			EntityID: "runner",
			From:     simcore.Vector3{X: 10, Y: 10, Z: 0},
			Delta:    simcore.Vector3{X: 2, Y: -1, Z: 0},
			To:       simcore.Vector3{X: 12, Y: 9, Z: 0},
		},
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	result := runtime.RunTick(now)
	if result.Tick.ID != 1 {
		t.Fatalf("expected first tick id 1, got %d", result.Tick.ID)
	}
	if result.QueueDepthBeforeTick != 1 {
		t.Fatalf("expected queue depth 1, got %d", result.QueueDepthBeforeTick)
	}

	move, ok := result.Events[0].(simcore.PlayerMovedEvent)
	if !ok {
		t.Fatalf("expected PlayerMovedEvent, got %T", result.Events[0])
	}
	if move.DomainEventKind() != simcore.EventPlayerMoved {
		t.Fatalf("unexpected event kind %s", move.DomainEventKind())
	}
	if move.To.X != 12 || move.To.Y != 9 {
		t.Fatalf("expected resolved move to 12,9 got %#v", move.To)
	}
}

func TestZoneRuntimeRegistersAndLooksUpEntity(t *testing.T) {
	zone, err := NewZoneRuntime(ZoneDefinition{
		ID:          "test_zone",
		DisplayName: "Test Zone",
	})
	if err != nil {
		t.Fatalf("zone create failed: %v", err)
	}

	entity := RuntimeEntity{
		ID:          "player_one",
		Kind:        "player",
		DisplayName: "Player One",
		Position:    simcore.Vector3{X: 1, Y: 2, Z: 3},
	}
	if err := zone.RegisterEntity(entity); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	found, ok := zone.LookupEntity("player_one")
	if !ok {
		t.Fatalf("expected entity lookup to succeed")
	}
	if found.ZoneID != "test_zone" {
		t.Fatalf("expected zone id to be filled from runtime, got %q", found.ZoneID)
	}
	if found.Position.X != 1 || found.Position.Y != 2 || found.Position.Z != 3 {
		t.Fatalf("unexpected position %#v", found.Position)
	}
}
