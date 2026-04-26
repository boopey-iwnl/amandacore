package worlds

import (
	"path/filepath"
	"strings"
	"testing"

	"amandacore/services/internal/observability"
)

func TestDawnwakeIslesPackageManifestLoads(t *testing.T) {
	registry := loadDawnwakeRegistry(t)

	if registry.Package.PackageID != "dawnwake_isles" {
		t.Fatalf("expected dawnwake_isles package, got %s", registry.Package.PackageID)
	}
	if registry.Package.SchemaVersion != CurrentContentSchemaVersion {
		t.Fatalf("expected schema %s, got %s", CurrentContentSchemaVersion, registry.Package.SchemaVersion)
	}
	if registry.Package.Version != "0.1.0" {
		t.Fatalf("expected package version 0.1.0, got %s", registry.Package.Version)
	}
}

func TestDawnwakeIslesContinentDefinitionLoads(t *testing.T) {
	registry := loadDawnwakeRegistry(t)
	continent, found := registry.Continents["dawnwake_isles"]
	if !found {
		t.Fatalf("expected dawnwake_isles continent to load")
	}
	if continent.DefaultEntry.ZoneID != "dawnwake_landing" || continent.DefaultEntry.EntryPointID == "" {
		t.Fatalf("unexpected default entry: %#v", continent.DefaultEntry)
	}
	if len(continent.Zones) != 5 {
		t.Fatalf("expected 5 continent zones, got %d", len(continent.Zones))
	}
}

func TestDawnwakeTopologyReferencesOnlyExistingZones(t *testing.T) {
	registry := loadDawnwakeRegistry(t)
	continent := registry.Continents["dawnwake_isles"]

	seenZones := map[string]bool{}
	for _, zoneID := range continent.Zones {
		if seenZones[zoneID] {
			t.Fatalf("duplicate zone ID %s", zoneID)
		}
		seenZones[zoneID] = true
		if _, found := registry.Zones[zoneID]; !found {
			t.Fatalf("continent references missing zone %s", zoneID)
		}
	}
	for _, adjacency := range continent.Adjacency {
		if _, found := registry.Zones[adjacency.FromZoneID]; !found {
			t.Fatalf("adjacency references missing source zone %s", adjacency.FromZoneID)
		}
		if _, found := registry.Zones[adjacency.ToZoneID]; !found {
			t.Fatalf("adjacency references missing destination zone %s", adjacency.ToZoneID)
		}
	}
}

func TestDawnwakeDefaultEntryReferencesValidZoneAndEntryPoint(t *testing.T) {
	registry := loadDawnwakeRegistry(t)
	continent := registry.Continents["dawnwake_isles"]
	zone, found := registry.Zones[continent.DefaultEntry.ZoneID]
	if !found {
		t.Fatalf("default zone %s missing", continent.DefaultEntry.ZoneID)
	}
	entry, found := zone.entryPoint(continent.DefaultEntry.EntryPointID)
	if !found {
		t.Fatalf("default entry %s missing", continent.DefaultEntry.EntryPointID)
	}
	if !zone.Bounds.Contains(entry.Position) {
		t.Fatalf("default entry is outside zone bounds")
	}
}

func TestDawnwakeTransitionTopologyIsValid(t *testing.T) {
	registry := loadDawnwakeRegistry(t)
	seenTransitions := map[string]bool{}
	for zoneID, zone := range registry.Zones {
		for _, gate := range zone.TransitionGates {
			if seenTransitions[gate.TransitionID] {
				t.Fatalf("duplicate transition ID %s", gate.TransitionID)
			}
			seenTransitions[gate.TransitionID] = true
			if gate.FromZoneID != zoneID {
				t.Fatalf("transition %s has from zone %s in zone %s", gate.TransitionID, gate.FromZoneID, zoneID)
			}
			destination, found := registry.Zones[gate.ToZoneID]
			if !found {
				t.Fatalf("transition %s destination zone %s missing", gate.TransitionID, gate.ToZoneID)
			}
			if _, found := destination.entryPoint(gate.EntryPointIDOnArrival); !found {
				t.Fatalf("transition %s arrival entry %s missing", gate.TransitionID, gate.EntryPointIDOnArrival)
			}
			if !zone.Bounds.ContainsBounds(gate.GateBounds) {
				t.Fatalf("transition %s gate bounds outside source zone", gate.TransitionID)
			}
		}
	}
	if len(seenTransitions) < 8 {
		t.Fatalf("expected at least 8 Dawnwake transitions, got %d", len(seenTransitions))
	}
}

func TestDawnwakeEntryAndSpawnPointsAreInsideZoneBounds(t *testing.T) {
	registry := loadDawnwakeRegistry(t)
	for zoneID, zone := range registry.Zones {
		for _, entry := range zone.EntryPoints {
			if !zone.Bounds.Contains(entry.Position) {
				t.Fatalf("entry point %s is outside zone %s", entry.EntryPointID, zoneID)
			}
		}
		for _, group := range zone.SpawnGroups {
			for _, spawn := range group.SpawnPoints {
				if !zone.Bounds.Contains(spawn.Position) {
					t.Fatalf("spawn point %s is outside zone %s", spawn.SpawnPointID, zoneID)
				}
			}
		}
		for _, provider := range zone.QuestProviders {
			if !zone.Bounds.Contains(provider.Position) {
				t.Fatalf("quest provider %s is outside zone %s", provider.ProviderID, zoneID)
			}
		}
	}
}

func TestInvalidTopologyPackageIsRejectedWithValidationErrors(t *testing.T) {
	registry := loadDawnwakeRegistry(t)
	landing := registry.Zones["dawnwake_landing"]
	landing.TransitionGates[0].ToZoneID = "missing_zone"
	registry.Zones["dawnwake_landing"] = landing

	err := registry.Validate()
	if err == nil {
		t.Fatalf("expected invalid topology to fail validation")
	}
	validation, ok := AsValidationErrors(err)
	if !ok {
		t.Fatalf("expected validation errors, got %T: %v", err, err)
	}
	if !validation.Has(ValidationMissingTransitionDestination) {
		t.Fatalf("expected %s, got %#v", ValidationMissingTransitionDestination, validation)
	}
}

func TestContinentRuntimeActivatesMultipleZones(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	if len(runtime.Zones) != 5 {
		t.Fatalf("expected 5 active zones, got %d", len(runtime.Zones))
	}
	for _, zoneID := range []string{"dawnwake_landing", "amberglass_fields", "mistwood_reach", "highroad_pass", "kingsfall_harbor"} {
		if runtime.Zones[zoneID] == nil {
			t.Fatalf("expected zone runtime %s to be active", zoneID)
		}
	}
}

func TestCharacterSpawnedAtDefaultEntryIsAssignedToOwningZoneRuntime(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	state, diffs, err := runtime.SpawnCharacterAtDefaultEntry("char_dawnwake")
	if err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	if state.ZoneID != "dawnwake_landing" {
		t.Fatalf("expected dawnwake_landing, got %s", state.ZoneID)
	}
	handle, err := runtime.RouteCommand(WorldCommand{CharacterID: "char_dawnwake", Name: "movement"})
	if err != nil {
		t.Fatalf("route command failed: %v", err)
	}
	if handle.ZoneID != "dawnwake_landing" {
		t.Fatalf("expected owning zone dawnwake_landing, got %s", handle.ZoneID)
	}
	if !diffsContain(diffs, observability.EventWorldZoneEntered) {
		t.Fatalf("expected zone entered diff, got %#v", diffs)
	}
}

func TestMovementThroughTransitionGateRequestsAndCompletesTransfer(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_dawnwake"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	if result, err := runtime.MoveCharacter("char_dawnwake", 176, 0, 0); err != nil || result.Requested {
		t.Fatalf("expected first move to reach gate without transfer, result %#v err %v", result, err)
	}

	result, err := runtime.MoveCharacter("char_dawnwake", 10, 0, 0)
	if err != nil {
		t.Fatalf("transition move failed: %v", err)
	}
	if !result.Requested || !result.Completed || result.Rejected {
		t.Fatalf("unexpected transfer result: %#v", result)
	}
	if result.ToZoneID != "amberglass_fields" {
		t.Fatalf("expected transfer to amberglass_fields, got %s", result.ToZoneID)
	}
	if runtime.Characters["char_dawnwake"].ZoneID != "amberglass_fields" {
		t.Fatalf("character was not handed off to destination runtime")
	}
	if !diffsContain(result.Diffs, observability.EventWorldZoneRoutingUpdated) {
		t.Fatalf("expected routing update diff, got %#v", result.Diffs)
	}
}

func TestInvalidTransitionIsRejected(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_dawnwake"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	gate := runtime.Zones["dawnwake_landing"].Definition.TransitionGates[0]
	result, err := runtime.RequestZoneTransfer(ZoneTransferRequest{
		CharacterID:  "char_dawnwake",
		FromZoneID:   "dawnwake_landing",
		ToZoneID:     gate.ToZoneID,
		TransitionID: gate.TransitionID,
		Position:     WorldPosition{ZoneID: "dawnwake_landing", X: 32, Y: 92, Z: 0},
	})
	if err != nil {
		t.Fatalf("rejected transfer should not return transport error: %v", err)
	}
	if !result.Requested || !result.Rejected || result.Completed {
		t.Fatalf("expected rejection, got %#v", result)
	}
	if result.RejectionReason == "" {
		t.Fatalf("expected rejection reason")
	}
}

func TestCommandsRouteToOwningZoneAfterTransfer(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_dawnwake"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	_, _ = runtime.MoveCharacter("char_dawnwake", 176, 0, 0)
	if _, err := runtime.MoveCharacter("char_dawnwake", 10, 0, 0); err != nil {
		t.Fatalf("transition move failed: %v", err)
	}
	handle, err := runtime.RouteCommand(WorldCommand{CharacterID: "char_dawnwake", Name: "ability"})
	if err != nil {
		t.Fatalf("route command failed: %v", err)
	}
	if handle.ZoneID != "amberglass_fields" {
		t.Fatalf("expected command route to amberglass_fields, got %s", handle.ZoneID)
	}
}

func TestVisibilityQueryIncludesNearbyAndExcludesFarSameZoneEntity(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_dawnwake"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	runtime.Zones["dawnwake_landing"].Entities.Add(RuntimeEntity{
		EntityID:    "test_far_entity",
		DisplayName: "Far Entity",
		Kind:        "npc",
		ZoneID:      "dawnwake_landing",
		Position:    WorldPosition{ZoneID: "dawnwake_landing", X: 300, Y: 200, Z: 0},
	})

	delta, err := runtime.EvaluateVisibility("char_dawnwake", InterestProfile{Radius: 45})
	if err != nil {
		t.Fatalf("visibility failed: %v", err)
	}
	if !entitiesContain(delta.Entered, "dawnwake_landing_pathfinder") {
		t.Fatalf("expected nearby provider to enter visibility, got %#v", delta.Entered)
	}
	if entitiesContain(delta.Entered, "test_far_entity") {
		t.Fatalf("far same-zone entity should be excluded")
	}
}

func TestVisibilityDeltaChangesAfterZoneTransfer(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_dawnwake"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	before, err := runtime.EvaluateVisibility("char_dawnwake", InterestProfile{Radius: 80})
	if err != nil {
		t.Fatalf("visibility before transfer failed: %v", err)
	}
	_, _ = runtime.MoveCharacter("char_dawnwake", 176, 0, 0)
	if _, err := runtime.MoveCharacter("char_dawnwake", 10, 0, 0); err != nil {
		t.Fatalf("transition move failed: %v", err)
	}
	after, err := runtime.EvaluateVisibility("char_dawnwake", InterestProfile{Radius: 90, IncludeAdjacentStreamingHints: true})
	if err != nil {
		t.Fatalf("visibility after transfer failed: %v", err)
	}
	if before.ZoneID == after.ZoneID {
		t.Fatalf("expected visibility zone to change")
	}
	if len(after.Entered) == 0 {
		t.Fatalf("expected visibility entered delta after zone transfer")
	}
	if len(after.StreamingHints) == 0 {
		t.Fatalf("expected adjacent-zone streaming hints near arrival gate")
	}
}

func TestReconnectRestoresZoneIDOrCorrectsInvalidPosition(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	store := NewMemoryCharacterZoneStore()
	state := CharacterZoneState{
		CharacterID: "char_dawnwake",
		ZoneID:      "amberglass_fields",
		Position:    WorldPosition{ZoneID: "amberglass_fields", X: 360, Y: 120, Z: 0},
		Connected:   true,
	}
	if err := store.SaveCharacterZoneState(state); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	restored, _, err := runtime.RestoreCharacterZoneState(store, "char_dawnwake")
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if restored.ZoneID != "amberglass_fields" {
		t.Fatalf("expected restore to amberglass_fields, got %s", restored.ZoneID)
	}

	if err := store.SaveCharacterZoneState(CharacterZoneState{
		CharacterID: "char_corrected",
		ZoneID:      "missing_zone",
		Position:    WorldPosition{ZoneID: "missing_zone", X: 999, Y: 999, Z: 0},
	}); err != nil {
		t.Fatalf("save invalid failed: %v", err)
	}
	corrected, diffs, err := runtime.RestoreCharacterZoneState(store, "char_corrected")
	if err != nil {
		t.Fatalf("correction failed: %v", err)
	}
	if corrected.ZoneID != "dawnwake_landing" {
		t.Fatalf("expected correction to dawnwake_landing, got %s", corrected.ZoneID)
	}
	if !diffsContain(diffs, observability.EventWorldCharacterZoneRestoreCorrected) {
		t.Fatalf("expected restore correction diff, got %#v", diffs)
	}
}

func TestZoneCommandQueueBackpressureAndDrain(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_queue"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	zoneRuntime := runtime.Zones["dawnwake_landing"]
	zoneRuntime.ConfigureCommandQueue(1)

	first, err := runtime.RouteCommandToQueue(WorldCommand{CommandID: "cmd_001", CharacterID: "char_queue", Name: "move"})
	if err != nil {
		t.Fatalf("first command route failed: %v", err)
	}
	if !first.Accepted || first.Backpressured {
		t.Fatalf("expected first command accepted, got %#v", first)
	}
	second, err := runtime.RouteCommandToQueue(WorldCommand{CommandID: "cmd_002", CharacterID: "char_queue", Name: "ability"})
	if err != nil {
		t.Fatalf("second command route failed: %v", err)
	}
	if !second.Backpressured || second.Accepted {
		t.Fatalf("expected second command backpressured, got %#v", second)
	}
	stats := zoneRuntime.QueueStats()
	if stats.Depth != 1 || stats.TotalBackpressured != 1 || stats.MaxDepth != 1 {
		t.Fatalf("unexpected queue stats before drain: %#v", stats)
	}
	command, ok := zoneRuntime.DequeueCommand()
	if !ok || command.CommandID != "cmd_001" {
		t.Fatalf("unexpected dequeued command: %#v ok=%t", command, ok)
	}
	stats = zoneRuntime.QueueStats()
	if stats.Depth != 0 || stats.TotalDequeued != 1 {
		t.Fatalf("unexpected queue stats after drain: %#v", stats)
	}
}

func TestShardAssignmentMapsEveryZoneToSingleOwner(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	index, err := runtime.AssignZonesToShards(ShardAssignmentPolicy{Prefix: "test-shard", MaxZonesPerShard: 1})
	if err != nil {
		t.Fatalf("assign shards failed: %v", err)
	}
	if len(index.ZoneToShard) != len(runtime.Zones) {
		t.Fatalf("expected every zone assigned, got %#v", index.ZoneToShard)
	}
	seenZones := map[string]bool{}
	for shardID, zoneIDs := range index.ShardToZones {
		if len(zoneIDs) != 1 {
			t.Fatalf("expected one zone per shard for %s, got %#v", shardID, zoneIDs)
		}
		for _, zoneID := range zoneIDs {
			if seenZones[zoneID] {
				t.Fatalf("zone %s assigned more than once", zoneID)
			}
			seenZones[zoneID] = true
			handle, found := runtime.ZoneShard(zoneID)
			if !found || handle.ShardID != shardID || handle.Runtime == nil {
				t.Fatalf("bad zone shard handle for %s: %#v found=%t", zoneID, handle, found)
			}
		}
	}
}

func TestTransferRejectsDestinationWithoutShardOwner(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_shard_transfer"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	_, _ = runtime.MoveCharacter("char_shard_transfer", 176, 0, 0)
	delete(runtime.Shards.ZoneToShard, "amberglass_fields")

	result, err := runtime.MoveCharacter("char_shard_transfer", 10, 0, 0)
	if err != nil {
		t.Fatalf("transfer should reject without transport error: %v", err)
	}
	if !result.Rejected || !strings.Contains(result.RejectionReason, "destination zone is not bound") {
		t.Fatalf("expected destination shard rejection, got %#v", result)
	}
	if runtime.Characters["char_shard_transfer"].ZoneID != "dawnwake_landing" {
		t.Fatalf("character should remain in source zone after rejection")
	}
}

func TestRepeatedTransitionStressKeepsSingleZoneOwnership(t *testing.T) {
	runtime := newDawnwakeRuntime(t)
	if _, _, err := runtime.SpawnCharacterAtDefaultEntry("char_transition_stress"); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	for index := 0; index < 8; index++ {
		if err := moveThroughFirstEnabledGate(runtime, "char_transition_stress"); err != nil {
			t.Fatalf("transition %d failed: %v", index, err)
		}
		activeZones := 0
		for _, zone := range runtime.Zones {
			if _, found := zone.Characters["char_transition_stress"]; found {
				activeZones++
			}
		}
		if activeZones != 1 {
			t.Fatalf("expected one owning zone after transition %d, got %d", index, activeZones)
		}
	}
}

func loadDawnwakeRegistry(t *testing.T) *RuntimeContentRegistry {
	t.Helper()
	registry, err := NewContentPackageLoader().Load(filepath.Join("..", "..", "..", "Content", "Packs", "dawnwake_isles", "package.json"))
	if err != nil {
		t.Fatalf("failed to load Dawnwake package: %v", err)
	}
	return registry
}

func newDawnwakeRuntime(t *testing.T) *ContinentRuntime {
	t.Helper()
	runtime, err := loadDawnwakeRegistry(t).ActivateContinent("dawnwake_isles")
	if err != nil {
		t.Fatalf("failed to activate Dawnwake continent: %v", err)
	}
	return runtime
}

func diffsContain(diffs []StateDiff, diffType string) bool {
	for _, diff := range diffs {
		if diff.DiffType == diffType {
			return true
		}
	}
	return false
}

func entitiesContain(entities []RuntimeEntity, entityID string) bool {
	for _, entity := range entities {
		if entity.EntityID == entityID {
			return true
		}
	}
	return false
}

func moveThroughFirstEnabledGate(runtime *ContinentRuntime, characterID string) error {
	state := runtime.Characters[characterID]
	if state == nil {
		return nil
	}
	zoneRuntime := runtime.Zones[state.ZoneID]
	if zoneRuntime == nil {
		return nil
	}
	var gate ZoneTransitionGate
	for _, candidate := range zoneRuntime.Definition.TransitionGates {
		if !candidate.Disabled {
			gate = candidate
			break
		}
	}
	if gate.TransitionID == "" {
		return nil
	}
	center := WorldPosition{
		ZoneID: state.ZoneID,
		X:      (gate.GateBounds.MinX + gate.GateBounds.MaxX) / 2,
		Y:      (gate.GateBounds.MinY + gate.GateBounds.MaxY) / 2,
		Z:      (gate.GateBounds.MinZ + gate.GateBounds.MaxZ) / 2,
	}
	if _, err := runtime.MoveCharacter(characterID, center.X-state.Position.X, center.Y-state.Position.Y, center.Z-state.Position.Z); err != nil {
		return err
	}
	deltaX, deltaY := testExitDelta(zoneRuntime.Definition.Bounds, gate.GateBounds)
	result, err := runtime.MoveCharacter(characterID, deltaX, deltaY, 0)
	if err != nil {
		return err
	}
	if !result.Completed || result.Rejected {
		return &testError{message: "transition did not complete"}
	}
	return nil
}

func testExitDelta(zone ZoneBounds, gate ZoneBounds) (float64, float64) {
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

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
