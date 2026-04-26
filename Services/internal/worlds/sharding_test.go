package worlds

import (
	"testing"

	contentpkg "amandacore/services/internal/content"
)

func TestBuildContentZoneShardAssignmentsStableAcrossSortedZones(t *testing.T) {
	registry := loadDawnwakeRegistryForShardingTest(t)

	assignments, err := BuildContentZoneShardAssignments(registry, ShardAssignmentPolicy{ShardCount: 2})
	if err != nil {
		t.Fatalf("expected assignments: %v", err)
	}
	if len(assignments) != len(registry.Zones) {
		t.Fatalf("expected one assignment per zone, got %d for %d zones", len(assignments), len(registry.Zones))
	}
	if assignments["amberglass_fields"].ShardID != "zone_shard_01" {
		t.Fatalf("expected first sorted zone on zone_shard_01, got %#v", assignments["amberglass_fields"])
	}
	if assignments["dawnwake_landing"].ShardID != "zone_shard_02" {
		t.Fatalf("expected second sorted zone on zone_shard_02, got %#v", assignments["dawnwake_landing"])
	}
}

func TestResolveZoneShardRejectsUnknownZone(t *testing.T) {
	registry := loadDawnwakeRegistryForShardingTest(t)
	assignments, err := BuildContentZoneShardAssignments(registry, ShardAssignmentPolicy{ShardCount: 2})
	if err != nil {
		t.Fatalf("expected assignments: %v", err)
	}

	if _, err := ResolveZoneShard(assignments, "missing_zone"); err == nil {
		t.Fatalf("expected missing zone to be rejected")
	}
}

func loadDawnwakeRegistryForShardingTest(t *testing.T) contentpkg.RuntimeContentRegistry {
	t.Helper()
	result := contentpkg.NewContentPackageLoader().Load(defaultDawnwakePackagePath)
	if !result.Validation.Valid() || result.Validated == nil {
		t.Fatalf("expected Dawnwake package to validate, got %#v", result.Validation.Errors)
	}
	return result.Validated.Registry
}
