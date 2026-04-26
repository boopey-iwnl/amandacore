package worlds

import (
	"testing"
	"time"
)

func TestShardCoordinatorAssignsZonesDeterministically(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	coordinator := NewInProcessShardCoordinator(runtime, ShardCoordinatorConfig{ShardCount: 2, QueueDepthLimit: 8})

	if len(coordinator.Assignments) != len(runtime.Zones) {
		t.Fatalf("expected one shard assignment per zone, got %d", len(coordinator.Assignments))
	}
	if coordinator.Assignments["dawnwake_landing"].ShardID != "shard-01" {
		t.Fatalf("expected dawnwake_landing on shard-01, got %s", coordinator.Assignments["dawnwake_landing"].ShardID)
	}
	if coordinator.Assignments["amberglass_fields"].ShardID != "shard-02" {
		t.Fatalf("expected amberglass_fields on shard-02, got %s", coordinator.Assignments["amberglass_fields"].ShardID)
	}
	if coordinator.Assignments["mistwood_reach"].ShardID != "shard-01" {
		t.Fatalf("expected mistwood_reach on shard-01, got %s", coordinator.Assignments["mistwood_reach"].ShardID)
	}
}

func TestShardCoordinatorRoutesCommandToOwningShardAfterTransfer(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	coordinator := NewInProcessShardCoordinator(runtime, ShardCoordinatorConfig{ShardCount: 2, QueueDepthLimit: 8})
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_sharded"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	before, err := coordinator.RouteCommand(WorldCommand{CharacterID: "char_sharded", Name: "move"})
	if err != nil {
		t.Fatalf("route before transfer failed: %v", err)
	}
	if before.ZoneID != "dawnwake_landing" || before.ShardID != "shard-01" {
		t.Fatalf("unexpected route before transfer: %#v", before)
	}

	_, _ = runtime.MoveCharacter("char_sharded", 176, 0, 0)
	transfer, err := runtime.MoveCharacter("char_sharded", 10, 0, 0)
	if err != nil {
		t.Fatalf("transfer failed: %v", err)
	}
	if !transfer.Completed {
		t.Fatalf("expected completed transfer, got %#v", transfer)
	}

	after, err := coordinator.RouteCommand(WorldCommand{CharacterID: "char_sharded", Name: "ability"})
	if err != nil {
		t.Fatalf("route after transfer failed: %v", err)
	}
	if after.ZoneID != "amberglass_fields" || after.ShardID != "shard-02" {
		t.Fatalf("unexpected route after transfer: %#v", after)
	}
	if err := coordinator.ValidateSingleZoneOwnership(); err != nil {
		t.Fatalf("single-zone ownership failed: %v", err)
	}
}

func TestShardCoordinatorBackpressureTracksQueueDepth(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	coordinator := NewInProcessShardCoordinator(runtime, ShardCoordinatorConfig{ShardCount: 1, QueueDepthLimit: 2})
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_pressure"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	first, err := coordinator.TryEnqueueCommand(WorldCommand{CommandID: "cmd-1", CharacterID: "char_pressure", Name: "move"})
	if err != nil || !first.Accepted {
		t.Fatalf("expected first command accepted, result %#v err %v", first, err)
	}
	second, err := coordinator.TryEnqueueCommand(WorldCommand{CommandID: "cmd-2", CharacterID: "char_pressure", Name: "move"})
	if err != nil || !second.Accepted {
		t.Fatalf("expected second command accepted, result %#v err %v", second, err)
	}
	third, err := coordinator.TryEnqueueCommand(WorldCommand{CommandID: "cmd-3", CharacterID: "char_pressure", Name: "move"})
	if err != nil {
		t.Fatalf("third command should report backpressure without route error: %v", err)
	}
	if !third.Backpressured || third.Accepted {
		t.Fatalf("expected third command backpressured, got %#v", third)
	}

	if err := coordinator.CompleteCommand(first, SimulationTick{Duration: time.Millisecond, QueueDepth: first.QueueDepth}); err != nil {
		t.Fatalf("complete first failed: %v", err)
	}
	if err := coordinator.CompleteCommand(second, SimulationTick{Duration: 2 * time.Millisecond, QueueDepth: second.QueueDepth}); err != nil {
		t.Fatalf("complete second failed: %v", err)
	}
	snapshot := coordinator.Snapshot()
	if snapshot.BackpressureCount != 1 {
		t.Fatalf("expected one backpressure event, got %d", snapshot.BackpressureCount)
	}
	if snapshot.MaxQueueDepth != 2 {
		t.Fatalf("expected max queue depth 2, got %d", snapshot.MaxQueueDepth)
	}
	if snapshot.CommandsAccepted != 2 || snapshot.CommandsProcessed != 2 {
		t.Fatalf("unexpected command counters: %#v", snapshot)
	}
}

func TestTickDurationSummaryUsesNearestRankPercentiles(t *testing.T) {
	summary := SummarizeTickDurations([]time.Duration{
		time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		5 * time.Millisecond,
	})

	if summary.Count != 5 {
		t.Fatalf("expected 5 samples, got %d", summary.Count)
	}
	if summary.P50 != 3*time.Millisecond {
		t.Fatalf("expected p50 3ms, got %s", summary.P50)
	}
	if summary.P95 != 5*time.Millisecond || summary.P99 != 5*time.Millisecond || summary.Max != 5*time.Millisecond {
		t.Fatalf("unexpected upper percentiles: %#v", summary)
	}
}

func TestRepeatedZoneTransfersDoNotDuplicatePlayerOwnership(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	coordinator := NewInProcessShardCoordinator(runtime, ShardCoordinatorConfig{ShardCount: 2, QueueDepthLimit: 8})
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_loop"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	for index := 0; index < 3; index++ {
		toAmberglass, err := moveThroughFirstGate(runtime, "char_loop")
		if err != nil || !toAmberglass.Completed {
			t.Fatalf("expected transfer to amberglass on loop %d, result %#v err %v", index, toAmberglass, err)
		}
		if err := coordinator.ValidateSingleZoneOwnership(); err != nil {
			t.Fatalf("ownership failed after amberglass transfer: %v", err)
		}

		toLanding, err := moveThroughFirstGate(runtime, "char_loop")
		if err != nil || !toLanding.Completed {
			t.Fatalf("expected transfer to landing on loop %d, result %#v err %v", index, toLanding, err)
		}
		if err := coordinator.ValidateSingleZoneOwnership(); err != nil {
			t.Fatalf("ownership failed after landing transfer: %v", err)
		}
	}
}

func TestShardPopulationDistributionFollowsPlacedCharacters(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	coordinator := NewInProcessShardCoordinator(runtime, ShardCoordinatorConfig{ShardCount: 2, QueueDepthLimit: 8})

	if _, _, err := runtime.PlaceCharacterAtEntry("char_landing", "dawnwake_landing", "landing_shore"); err != nil {
		t.Fatalf("place landing failed: %v", err)
	}
	if _, _, err := runtime.PlaceCharacterAtEntry("char_amberglass", "amberglass_fields", "amberglass_west_road"); err != nil {
		t.Fatalf("place amberglass failed: %v", err)
	}

	zonePopulation := coordinator.ZonePopulation()
	if zonePopulation["dawnwake_landing"] != 1 || zonePopulation["amberglass_fields"] != 1 {
		t.Fatalf("unexpected zone population: %#v", zonePopulation)
	}
	shardPopulation := coordinator.ShardPopulation()
	if shardPopulation["shard-01"] != 1 || shardPopulation["shard-02"] != 1 {
		t.Fatalf("unexpected shard population: %#v", shardPopulation)
	}
}

func moveThroughFirstGate(runtime *ContinentRuntime, characterID string) (ZoneTransferResult, error) {
	state := runtime.Characters[characterID]
	zoneRuntime := runtime.Zones[state.ZoneID]
	if zoneRuntime == nil || len(zoneRuntime.Definition.TransitionGates) == 0 {
		return ZoneTransferResult{}, nil
	}
	gate := zoneRuntime.Definition.TransitionGates[0]
	center := WorldPosition{
		ZoneID: state.ZoneID,
		X:      (gate.GateBounds.MinX + gate.GateBounds.MaxX) / 2,
		Y:      (gate.GateBounds.MinY + gate.GateBounds.MaxY) / 2,
		Z:      (gate.GateBounds.MinZ + gate.GateBounds.MaxZ) / 2,
	}
	if _, err := runtime.MoveCharacter(characterID, center.X-state.Position.X, center.Y-state.Position.Y, center.Z-state.Position.Z); err != nil {
		return ZoneTransferResult{}, err
	}
	deltaX, deltaY := exitDeltaForTest(zoneRuntime.Definition.Bounds, gate.GateBounds)
	return runtime.MoveCharacter(characterID, deltaX, deltaY, 0)
}

func exitDeltaForTest(zone ZoneBounds, gate ZoneBounds) (float64, float64) {
	switch {
	case gate.MaxX >= zone.MaxX:
		return 10, 0
	case gate.MinX <= zone.MinX:
		return -10, 0
	case gate.MaxY >= zone.MaxY:
		return 0, 10
	case gate.MinY <= zone.MinY:
		return 0, -10
	default:
		return 10, 0
	}
}
