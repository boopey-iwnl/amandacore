package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentPackageManifestLoadsSuccessfully(t *testing.T) {
	result := NewContentPackageLoader().Load(DefaultPackagePath)
	if !result.Validation.Valid() {
		t.Fatalf("expected dev package to load, got errors: %#v", result.Validation.Errors)
	}
	if result.Validated == nil {
		t.Fatalf("expected validated package")
	}
	if result.Package.Manifest.PackageID != "dev_foundation" {
		t.Fatalf("unexpected package id %q", result.Package.Manifest.PackageID)
	}
	if len(result.Package.Zones) != 1 {
		t.Fatalf("expected one zone, got %d", len(result.Package.Zones))
	}
}

func TestMissingManifestFileReturnsMissingFile(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing-package.json")
	result := NewContentPackageLoader().Load(missingPath)
	assertValidationCode(t, result.Validation, ErrorMissingFile)
}

func TestMalformedJSONReturnsMalformedJSON(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "package.json")
	if err := os.WriteFile(manifestPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("write malformed manifest: %v", err)
	}
	result := NewContentPackageLoader().Load(manifestPath)
	assertValidationCode(t, result.Validation, ErrorMalformedJSON)
}

func TestUnsupportedSchemaVersionRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Manifest.SchemaVersion = "2"
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorUnsupportedSchemaVersion)
}

func TestDevFoundationPackageValidatesSuccessfully(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	report := ValidateLoadedContentPackage(loaded)
	if !report.Valid() {
		t.Fatalf("expected dev_foundation to validate, got %#v", report.Errors)
	}
}

func TestDuplicateItemIDsRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Items = append(loaded.Items, loaded.Items[0])
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorDuplicateID)
}

func TestDuplicateNPCArchetypeIDsRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.NPCs = append(loaded.NPCs, loaded.NPCs[0])
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorDuplicateID)
}

func TestLootTableReferencingMissingItemRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.LootTables[0].Entries[0].ItemID = "missing_item"
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorBrokenReference)
	assertValidationPathContains(t, report, "loot_tables[0].entries[0].item_id")
}

func TestSpawnGroupReferencingMissingNPCArchetypeRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Zones[0].SpawnGroups[0].NPCArchetypeID = "missing_npc"
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorBrokenReference)
	assertValidationPathContains(t, report, "spawn_groups[0].npc_archetype_id")
}

func TestSpawnGroupReferencingMissingLootTableRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Zones[0].SpawnGroups[0].LootTableID = "missing_loot"
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorBrokenReference)
	assertValidationPathContains(t, report, "spawn_groups[0].loot_table_id")
}

func TestSpawnPointOutsideZoneBoundsRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Zones[0].SpawnGroups[0].SpawnPoints[0].Position.X = 999
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorPositionOutOfBounds)
}

func TestQuestRewardReferencingMissingItemRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Quests[0].Rewards[0].ItemID = "missing_item"
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorBrokenReference)
	assertValidationPathContains(t, report, "rewards[0].item_id")
}

func TestQuestObjectiveReferencingMissingNPCRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Quests[0].ObjectiveGraph.Nodes[0].TargetID = "missing_npc"
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorBrokenReference)
	assertValidationPathContains(t, report, "objective_graph.nodes[0].target_id")
}

func TestQuestObjectiveGraphCycleRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Quests[0].ObjectiveGraph.Nodes[0].DependsOn = []string{"recover_fang"}
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorObjectiveGraphCycle)
}

func TestAbilityReferencingMissingAuraRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Abilities[1].Effects[0].AuraID = "missing_aura"
	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorBrokenReference)
	assertValidationPathContains(t, report, "abilities[1].effects[0].aura_id")
}

func TestManifestListedMissingFileRejected(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "package.json")
	manifest := `{
  "package_id": "missing_file_test",
  "display_name": "Missing File Test",
  "version": "0.1.0",
  "schema_version": "1",
  "zones": ["zones/missing.zone.json"]
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	_ = loaded
	result := NewContentPackageLoader().Load(manifestPath)
	assertValidationCode(t, result.Validation, ErrorMissingFile)
}

func mustLoadDevPackage(t *testing.T) LoadedContentPackage {
	t.Helper()
	result := NewContentPackageLoader().Load(DefaultPackagePath)
	if !result.Validation.Valid() {
		t.Fatalf("expected dev package to load, got errors: %#v", result.Validation.Errors)
	}
	return result.Package
}

func assertValidationCode(t *testing.T, report ContentValidationReport, code ContentValidationErrorCode) {
	t.Helper()
	if !report.HasCode(code) {
		t.Fatalf("expected validation code %s, got %#v", code, report.Errors)
	}
}

func assertValidationPathContains(t *testing.T, report ContentValidationReport, fragment string) {
	t.Helper()
	for _, validationError := range report.Errors {
		if strings.Contains(string(validationError.Path), fragment) {
			return
		}
	}
	t.Fatalf("expected validation path containing %q, got %#v", fragment, report.Errors)
}
