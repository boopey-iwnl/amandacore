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

func TestDawnwakePackageRegistersHandoffGates(t *testing.T) {
	server := newWorldServerWithContentPackage(nil, contentpkg.DawnwakePackagePath)

	if server.contentRegistry == nil {
		t.Fatalf("expected runtime content registry")
	}
	if server.contentRegistry.PackageID != "dawnwake_isles_foundation" {
		t.Fatalf("unexpected package id %q", server.contentRegistry.PackageID)
	}
	if server.contentActivation.HandoffGatesRegistered != 2 {
		t.Fatalf("expected two handoff gates, got %d", server.contentActivation.HandoffGatesRegistered)
	}
	gate := server.handoffGateLocked("gate_dawnwake_landing_to_tideglass")
	if gate.TransitionID == "" {
		t.Fatalf("expected package handoff gate")
	}
	if gate.FromZoneID != "dawnwake_landing" || gate.ToZoneID != "dawnwake_tideglass_shoal" {
		t.Fatalf("unexpected handoff route: %#v", gate)
	}
	if gate.ArrivalX != 14 || gate.ArrivalY != 50 {
		t.Fatalf("expected arrival from package spawn point, got %#v", gate)
	}
}
