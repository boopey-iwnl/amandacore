package loadsim

import "testing"

func testContentPackage() ContentPackage {
	return ContentPackage{
		PackageID: "test",
		Zones: map[string]ZoneSpec{
			"zone_a": {
				ZoneID:          "zone_a",
				Bounds:          Bounds{MaxX: 10, MaxY: 10, MaxZ: 1},
				EntryPoints:     []EntryPoint{{EntryID: "default", Position: Point{X: 1, Y: 1}}},
				TransitionGates: []TransitionGate{{TransitionID: "to_b", ToZoneID: "zone_b", EntryPointIDOnArrival: "default"}},
			},
			"zone_b": {
				ZoneID:          "zone_b",
				Bounds:          Bounds{MaxX: 10, MaxY: 10, MaxZ: 1},
				EntryPoints:     []EntryPoint{{EntryID: "default", Position: Point{X: 1, Y: 1}}},
				TransitionGates: []TransitionGate{{TransitionID: "to_a", ToZoneID: "zone_a", EntryPointIDOnArrival: "default"}},
			},
		},
		ZoneOrder: []string{"zone_a", "zone_b"},
	}
}

func TestShardRegistryRegistersLocalShardRuntimes(t *testing.T) {
	registry := NewShardRegistry()
	if err := registry.Register(NewShardRuntime("local-01", ShardCapacity{})); err != nil {
		t.Fatal(err)
	}
	if len(registry.Shards) != 1 {
		t.Fatalf("expected registered shard")
	}
}

func TestStaticAssignmentAssignsEachZoneToExactlyOneShard(t *testing.T) {
	router, err := BuildShardRouter(testContentPackage(), 2, AssignmentStatic, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(router.ZoneBindings) != 2 {
		t.Fatalf("expected each zone to be bound once, got %v", router.ZoneBindings)
	}
}

func TestHashZoneAssignmentIsDeterministic(t *testing.T) {
	left, err := BuildShardRouter(testContentPackage(), 2, AssignmentHashZone, 8)
	if err != nil {
		t.Fatal(err)
	}
	right, err := BuildShardRouter(testContentPackage(), 2, AssignmentHashZone, 8)
	if err != nil {
		t.Fatal(err)
	}
	for zoneID, shardID := range left.ZoneBindings {
		if right.ZoneBindings[zoneID] != shardID {
			t.Fatalf("hash assignment differs for %s", zoneID)
		}
	}
}

func TestLeastLoadedAssignmentRespectsCapacityHints(t *testing.T) {
	if _, err := BuildShardRouter(testContentPackage(), 1, AssignmentLeastLoaded, 8); err != nil {
		t.Fatalf("least-loaded should fit two zones with default capacity: %v", err)
	}
}

func TestShardRouterRoutesCommandToCorrectShardAndRejectsWrongZone(t *testing.T) {
	router, err := BuildShardRouter(testContentPackage(), 2, AssignmentStatic, 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := router.RegisterCharacter("c1", "zone_a", Point{X: 1, Y: 1}); err != nil {
		t.Fatal(err)
	}
	result := router.Submit(CommandEnvelope{CommandID: "c1:1", CharacterID: "c1", ZoneID: "zone_a", Type: "move"})
	if !result.Accepted {
		t.Fatalf("expected route to be accepted: %#v", result)
	}
	wrong := router.Submit(CommandEnvelope{CommandID: "c1:2", CharacterID: "c1", ZoneID: "zone_b", Type: "move"})
	if !wrong.Rejected || wrong.Reason != RejectCharacterZoneMismatch {
		t.Fatalf("expected wrong-zone rejection, got %#v", wrong)
	}
}

func TestCrossZoneCombatAndLootAreRejected(t *testing.T) {
	router, err := BuildShardRouter(testContentPackage(), 1, AssignmentStatic, 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := router.RegisterCharacter("c1", "zone_a", Point{X: 1, Y: 1}); err != nil {
		t.Fatal(err)
	}
	if result := router.Submit(CommandEnvelope{CommandID: "c1:1", CharacterID: "c1", ZoneID: "zone_a", Type: "combat", TargetZoneID: "zone_b"}); !result.Accepted {
		t.Fatalf("expected enqueue before tick, got %#v", result)
	}
	results := router.TickAll()
	if len(results) != 1 || !results[0].Rejected || results[0].Reason != RejectCrossZoneInteractionUnsupported {
		t.Fatalf("expected cross-zone combat rejection, got %#v", results)
	}
}

func TestZoneTransferUpdatesRouting(t *testing.T) {
	router, err := BuildShardRouter(testContentPackage(), 1, AssignmentStatic, 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := router.RegisterCharacter("c1", "zone_a", Point{X: 1, Y: 1}); err != nil {
		t.Fatal(err)
	}
	_ = router.Submit(CommandEnvelope{CommandID: "c1:1", CharacterID: "c1", ZoneID: "zone_a", Type: "transition", TargetZoneID: "zone_b"})
	results := router.TickAll()
	if len(results) != 1 || !results[0].TransitionCompleted {
		t.Fatalf("expected completed transition, got %#v", results)
	}
	if router.CharacterZone["c1"] != "zone_b" {
		t.Fatalf("expected routing to zone_b, got %s", router.CharacterZone["c1"])
	}
}

func TestQueueBackpressureRejectsAndIncrementsMetrics(t *testing.T) {
	router, err := BuildShardRouter(testContentPackage(), 1, AssignmentStatic, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := router.RegisterCharacter("c1", "zone_a", Point{X: 1, Y: 1}); err != nil {
		t.Fatal(err)
	}
	first := router.Submit(CommandEnvelope{CommandID: "c1:1", CharacterID: "c1", ZoneID: "zone_a", Type: "move"})
	second := router.Submit(CommandEnvelope{CommandID: "c1:2", CharacterID: "c1", ZoneID: "zone_a", Type: "move"})
	if !first.Accepted || !second.Rejected || second.Reason != RejectQueueFull {
		t.Fatalf("expected queue full rejection, got first=%#v second=%#v", first, second)
	}
	zone, _ := router.Resolve("zone_a")
	if zone.QueueMetrics.MaxDepth == 0 {
		t.Fatalf("expected queue metrics to be updated")
	}
}
