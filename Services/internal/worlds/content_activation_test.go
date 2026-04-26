package worlds

import (
	"testing"

	contentpkg "amandacore/services/internal/content"
)

func TestValidatedDevPackageActivatesIntoRuntimeContentRegistry(t *testing.T) {
	server := newWorldServerWithContentPackage(nil, contentpkg.DefaultPackagePath)

	if server.contentRegistry == nil {
		t.Fatalf("expected runtime content registry")
	}
	if server.contentRegistry.PackageID != "dev_foundation" {
		t.Fatalf("unexpected package id %q", server.contentRegistry.PackageID)
	}
	if _, found := server.contentRegistry.Zones["dev_isle_edge"]; !found {
		t.Fatalf("expected dev_isle_edge in runtime registry")
	}
	if _, found := server.contentRegistry.Items["dev_stalker_fang"]; !found {
		t.Fatalf("expected dev_stalker_fang in runtime registry")
	}
}

func TestZoneRuntimeCreatedFromLoadedZoneDefinition(t *testing.T) {
	server := newWorldServerWithContentPackage(nil, contentpkg.DefaultPackagePath)

	runtime := server.zoneRuntimes["dev_isle_edge"]
	if runtime == nil {
		t.Fatalf("expected dev_isle_edge zone runtime")
	}
	if runtime.RuntimeConfig.TickMS != 50 {
		t.Fatalf("expected runtime tick 50ms, got %d", runtime.RuntimeConfig.TickMS)
	}
	if runtime.SpawnGroupCount != 1 || runtime.QuestProviderCount != 1 {
		t.Fatalf("unexpected runtime counts: %#v", runtime)
	}
}

func TestNPCsSpawnFromLoadedSpawnGroups(t *testing.T) {
	server := newWorldServerWithContentPackage(nil, contentpkg.DefaultPackagePath)

	found := 0
	for _, mob := range server.mobs {
		if mob.ArchetypeID == "dev_isle_stalker" && mob.ZoneID == "dev_isle_edge" {
			found++
			if mob.SpawnPointID == "" {
				t.Fatalf("expected spawned content mob to keep spawn point id: %#v", mob)
			}
			if mob.LootTableID != "dev_isle_stalker_cache" {
				t.Fatalf("expected content mob loot table, got %q", mob.LootTableID)
			}
		}
	}
	if found != 2 {
		t.Fatalf("expected 2 dev isle stalkers, got %d", found)
	}
}

func TestQuestProviderRegistersLoadedDevFirstHunt(t *testing.T) {
	server := newWorldServerWithContentPackage(nil, contentpkg.DefaultPackagePath)

	provider := server.friendlyNPCs["provider_dev_pathfinder"]
	if provider.ID == "" {
		t.Fatalf("expected provider_dev_pathfinder to be registered")
	}
	if provider.ZoneID != "dev_isle_edge" {
		t.Fatalf("expected provider in dev_isle_edge, got %q", provider.ZoneID)
	}
	if len(provider.Services) != 1 || provider.Services[0].ServiceID != "dev_first_hunt" {
		t.Fatalf("expected provider to offer dev_first_hunt, got %#v", provider.Services)
	}
	quest := server.quests["dev_first_hunt"]
	if quest.ID == "" {
		t.Fatalf("expected dev_first_hunt to be registered")
	}
	if quest.TargetMobType != "dev_isle_stalker" {
		t.Fatalf("expected quest target dev_isle_stalker, got %q", quest.TargetMobType)
	}
}

func TestDawnwakePackageActivatesMultipleZoneRuntimes(t *testing.T) {
	server := newWorldServerWithContentPackage(nil, "Content/Packs/dawnwake_isles/package.json")

	if server.contentRegistry == nil {
		t.Fatalf("expected Dawnwake runtime content registry")
	}
	if server.contentRegistry.PackageID != "dawnwake_isles" {
		t.Fatalf("unexpected package id %q", server.contentRegistry.PackageID)
	}
	if server.contentActivation.ZonesActivated != 3 {
		t.Fatalf("expected 3 zones activated, got %#v", server.contentActivation)
	}
	if server.contentActivation.TransitionsLoaded != 4 {
		t.Fatalf("expected 4 transitions loaded, got %#v", server.contentActivation)
	}
	if runtime := server.zoneRuntimes["dawnwake_tideglass_shoal"]; runtime == nil || runtime.TransitionCount != 2 {
		t.Fatalf("expected Tideglass runtime with two transitions, got %#v", runtime)
	}
	if mob := firstContentMob(server, "dw_tideglass_mote"); mob == nil || mob.ZoneID != "dawnwake_tideglass_shoal" {
		t.Fatalf("expected Tideglass mote to spawn from loaded content, got %#v", mob)
	}
	if provider := server.friendlyNPCs["dw_provider_lantern_pier"]; provider.ID == "" || provider.ZoneID != "dawnwake_landing" {
		t.Fatalf("expected Dawnwake quest provider in landing, got %#v", provider)
	}
}

func TestContentZoneTransitionMovesSessionToDestinationEntry(t *testing.T) {
	server := newWorldServerWithContentPackage(nil, "Content/Packs/dawnwake_isles/package.json")
	landing := server.contentRegistry.Zones["dawnwake_landing"]
	transition := landing.Transitions[0]
	session := &worldSessionState{
		CharacterID: "char_transition_test",
		ZoneID:      "dawnwake_landing",
		X:           transition.Position.X,
		Y:           transition.Position.Y,
		Z:           transition.Position.Z,
		Alive:       true,
	}

	result, err := server.applyContentZoneTransitionsLocked(session)
	if err != nil {
		t.Fatalf("transition failed: %v", err)
	}
	if !result.Completed {
		t.Fatalf("expected transition to complete, got %#v", result)
	}
	if session.ZoneID != "dawnwake_tideglass_shoal" {
		t.Fatalf("expected session in Tideglass Shoal, got %q", session.ZoneID)
	}
	entry, _ := server.contentEntryPointLocked("dawnwake_tideglass_shoal", "from_landing_causeway")
	if session.X != entry.Position.X || session.Y != entry.Position.Y {
		t.Fatalf("expected session at destination entry, got %.1f %.1f", session.X, session.Y)
	}
}

func TestDawnwakeTraversalLoadsimCompletes(t *testing.T) {
	report, err := RunContentPackageLoadsim(ContentPackageLoadsimOptions{
		Clients:     1,
		Duration:    100000000,
		CommandRate: 2,
		Scenario:    "dawnwake-traversal-basic",
		ContentPath: "Content/Packs/dawnwake_isles/package.json",
	})
	if err != nil {
		t.Fatalf("loadsim failed: %v report=%#v", err, report)
	}
	if report.ZonesActivated != 3 || report.TransitionsCompleted != 1 {
		t.Fatalf("unexpected traversal report: %#v", report)
	}
	if report.QuestsCompleted != 1 || report.LootClaimsCompleted != 1 || report.InventoryGrants["dw_tideglass_splinter"] != 1 {
		t.Fatalf("expected Dawnwake quest, loot, and inventory flow, got %#v", report)
	}
}
