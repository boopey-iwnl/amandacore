package worlds

import (
	"testing"
	"time"
)

func TestBuildZoneShardAssignmentsStableAcrossSortedZones(t *testing.T) {
	zones := map[string]zoneDefinition{
		"zone_c": {ID: "zone_c"},
		"zone_a": {ID: "zone_a"},
		"zone_b": {ID: "zone_b"},
	}

	assignments, err := BuildZoneShardAssignments(zones, ShardAssignmentPolicy{ShardCount: 2})
	if err != nil {
		t.Fatalf("expected assignments: %v", err)
	}
	if assignments["zone_a"].ShardID != "zone_shard_01" {
		t.Fatalf("expected zone_a on first shard, got %#v", assignments["zone_a"])
	}
	if assignments["zone_b"].ShardID != "zone_shard_02" {
		t.Fatalf("expected zone_b on second shard, got %#v", assignments["zone_b"])
	}
	if assignments["zone_c"].ShardID != "zone_shard_01" {
		t.Fatalf("expected zone_c to wrap to first shard, got %#v", assignments["zone_c"])
	}
}

func TestZoneHandoffCompletesAndUpdatesOwnership(t *testing.T) {
	server := newWorldServer(nil)
	session := newProgressionTestSession(server, "handoff_complete")
	session.X = 470
	session.Y = 260

	decision, err := server.requestZoneHandoffLocked(session, "to_brindlebrook")
	if err != nil {
		t.Fatalf("handoff failed: %v", err)
	}
	if decision.Status != ZoneHandoffCompleted || !decision.Accepted {
		t.Fatalf("expected completed accepted handoff, got %#v", decision)
	}
	if session.ZoneID != secondZoneID || session.X != secondZoneEntryX || session.Y != secondZoneEntryY {
		t.Fatalf("expected session at Brindlebrook entry, got zone=%s x=%.1f y=%.1f", session.ZoneID, session.X, session.Y)
	}
	ownership, found := server.shardCoordinator.CharacterOwnership(session.CharacterID)
	if !found || ownership.ZoneID != secondZoneID {
		t.Fatalf("expected destination ownership, got %#v found=%v", ownership, found)
	}
	if got := server.shardCoordinator.MaxQueueDepth(); got != 1 {
		t.Fatalf("expected queue max depth 1, got %d", got)
	}
	if len(server.shardCoordinator.Journal()) != 3 {
		t.Fatalf("expected requested, accepted, completed journal entries, got %#v", server.shardCoordinator.Journal())
	}
}

func TestZoneHandoffRejectsOutOfRangeWithoutMovingCharacter(t *testing.T) {
	server := newWorldServer(nil)
	session := newProgressionTestSession(server, "handoff_far")
	originalZone := session.ZoneID
	originalX := session.X
	originalY := session.Y

	decision, err := server.requestZoneHandoffLocked(session, "to_brindlebrook")
	if err == nil {
		t.Fatalf("expected handoff rejection")
	}
	if decision.Reason != ZoneHandoffRejectOutOfRange {
		t.Fatalf("expected OutOfRange, got %#v", decision)
	}
	if session.ZoneID != originalZone || session.X != originalX || session.Y != originalY {
		t.Fatalf("rejected handoff moved character: zone=%s x=%.1f y=%.1f", session.ZoneID, session.X, session.Y)
	}
}

func TestZoneHandoffDestinationUnavailableCanRetry(t *testing.T) {
	server := newWorldServer(nil)
	session := newProgressionTestSession(server, "handoff_retry")
	session.X = 470
	session.Y = 260
	destination, err := server.shardCoordinator.ResolveZone(secondZoneID)
	if err != nil {
		t.Fatalf("expected destination shard: %v", err)
	}

	if err := server.setShardWorkerStateLocked(destination.ShardID, ShardWorkerUnavailable, "test"); err != nil {
		t.Fatalf("set unavailable failed: %v", err)
	}
	decision, err := server.requestZoneHandoffLocked(session, "to_brindlebrook")
	if err == nil {
		t.Fatalf("expected unavailable shard rejection")
	}
	if decision.Reason != ZoneHandoffRejectDestinationShardUnavailable || !decision.Retryable {
		t.Fatalf("expected retryable unavailable shard rejection, got %#v", decision)
	}
	if session.ZoneID != defaultZoneID {
		t.Fatalf("expected source zone to remain after rejection, got %s", session.ZoneID)
	}

	if err := server.setShardWorkerStateLocked(destination.ShardID, ShardWorkerActive, "test_retry"); err != nil {
		t.Fatalf("set active failed: %v", err)
	}
	decision, err = server.requestZoneHandoffLocked(session, "to_brindlebrook")
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if decision.Status != ZoneHandoffCompleted || session.ZoneID != secondZoneID {
		t.Fatalf("expected retry completion, decision=%#v zone=%s", decision, session.ZoneID)
	}
}

func TestZoneHandoffQueueBackpressureRejected(t *testing.T) {
	server := newWorldServer(nil)
	session := newProgressionTestSession(server, "handoff_queue")
	session.X = 470
	session.Y = 260
	queue := server.shardCoordinator.queues[secondZoneID]
	if queue == nil {
		t.Fatalf("expected destination queue")
	}
	queue.Depth = queue.Capacity

	decision, err := server.requestZoneHandoffLocked(session, "to_brindlebrook")
	if err == nil {
		t.Fatalf("expected queue rejection")
	}
	if decision.Reason != ZoneHandoffRejectQueueFull {
		t.Fatalf("expected QueueFull, got %#v", decision)
	}
	if got := server.countEvents(EventZoneQueueBackpressure); got != 1 {
		t.Fatalf("expected queue backpressure event, got %d", got)
	}
}

func TestZoneHandoffReconnectCorrectionUsesCoordinatorOwnership(t *testing.T) {
	server := newWorldServer(nil)
	session := newProgressionTestSession(server, "handoff_reconnect")
	server.shardCoordinator.ownership[session.CharacterID] = CharacterZoneOwnership{
		CharacterID: session.CharacterID,
		ZoneID:      secondZoneID,
		ShardID:     server.shardCoordinator.assignments[secondZoneID].ShardID,
		X:           secondZoneEntryX,
		Y:           secondZoneEntryY,
		Z:           playableGroundZ,
		UpdatedAtMs: nowMillis(),
	}

	if corrected := server.correctSessionFromShardOwnershipLocked(session); !corrected {
		t.Fatalf("expected session correction")
	}
	if session.ZoneID != secondZoneID || session.X != secondZoneEntryX || session.Y != secondZoneEntryY {
		t.Fatalf("expected corrected Brindlebrook position, got zone=%s x=%.1f y=%.1f", session.ZoneID, session.X, session.Y)
	}
}

func TestZoneHandoffLoadsimCompletesWithExpectedRetry(t *testing.T) {
	report, err := RunZoneHandoffLoadsim(ZoneHandoffLoadsimOptions{
		Clients:         3,
		Duration:        100 * time.Millisecond,
		CmdRate:         4,
		TransitionLoops: 2,
		Shards:          2,
		QueueCapacity:   8,
	})
	if err != nil {
		t.Fatalf("expected loadsim to run: %v", err)
	}
	if len(report.Errors) > 0 {
		t.Fatalf("expected no loadsim errors, got %#v", report.Errors)
	}
	if report.HandoffsCompleted != 6 {
		t.Fatalf("expected 6 completed handoffs, got %#v", report)
	}
	if report.HandoffsRejected != 1 || report.ExpectedRejections != 1 || report.HandoffsRetried != 1 {
		t.Fatalf("expected one rejected and retried handoff, got %#v", report)
	}
	if len(report.ZonePopulation) < 1 || len(report.ShardPopulation) < 1 {
		t.Fatalf("expected population summaries, got zones=%#v shards=%#v", report.ZonePopulation, report.ShardPopulation)
	}
}
