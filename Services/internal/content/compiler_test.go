package content

import (
	"bytes"
	"testing"
)

func TestContentCompilerCompilesDevPackage(t *testing.T) {
	result := CompileContentPackage(DefaultPackagePath)
	if !result.Validation.Valid() {
		t.Fatalf("expected content compiler to validate dev package, got %#v", result.Validation.Errors)
	}
	if result.Package.Protocol != CompiledContentProtocol {
		t.Fatalf("unexpected protocol %q", result.Package.Protocol)
	}
	if result.Package.PackageID != "dev_foundation" {
		t.Fatalf("unexpected package id %q", result.Package.PackageID)
	}
	if result.Package.ContentSHA256 == "" {
		t.Fatalf("expected deterministic content hash")
	}
	if !containsString(result.Package.IDs.Vendors, "vendor_dev_pathfinder_cache") {
		t.Fatalf("expected compiled vendor IDs, got %#v", result.Package.IDs.Vendors)
	}
	if !containsString(result.Package.IDs.Trainers, "trainer_dev_pathfinder") {
		t.Fatalf("expected compiled trainer IDs, got %#v", result.Package.IDs.Trainers)
	}
	if !containsString(result.Package.IDs.HookBindings, "hook_dev_first_hunt_accept") {
		t.Fatalf("expected compiled hook IDs, got %#v", result.Package.IDs.HookBindings)
	}
}

func TestContentCompilerRejectsDuplicateIDs(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Items = append(loaded.Items, loaded.Items[0])

	result := CompileLoadedContentPackage(loaded)
	assertValidationCode(t, result.Validation, ErrorDuplicateID)
}

func TestContentCompilerRejectsMissingQuestObjectiveReference(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Quests[0].ObjectiveGraph.Nodes[0].TargetID = "missing_npc"

	result := CompileLoadedContentPackage(loaded)
	assertValidationCode(t, result.Validation, ErrorBrokenReference)
	assertValidationPathContains(t, result.Validation, "objective_graph.nodes[0].target_id")
}

func TestContentCompilerRejectsMissingLootItemReference(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.LootTables[0].Entries[0].ItemID = "missing_item"

	result := CompileLoadedContentPackage(loaded)
	assertValidationCode(t, result.Validation, ErrorBrokenReference)
	assertValidationPathContains(t, result.Validation, "loot_tables[0].entries[0].item_id")
}

func TestContentCompilerRejectsMissingTrainerAbilityReference(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Trainers[0].Abilities[0].AbilityID = "missing_ability"

	result := CompileLoadedContentPackage(loaded)
	assertValidationCode(t, result.Validation, ErrorBrokenReference)
	assertValidationPathContains(t, result.Validation, "trainers[0].abilities[0].ability_id")
}

func TestContentCompilerRejectsMissingVendorItemReference(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.Vendors[0].Items[0].ItemID = "missing_item"

	result := CompileLoadedContentPackage(loaded)
	assertValidationCode(t, result.Validation, ErrorBrokenReference)
	assertValidationPathContains(t, result.Validation, "vendors[0].items[0].item_id")
}

func TestContentCompilerRejectsInvalidHookName(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	loaded.HookBindings[0].Hook = "on_unsafe_script"

	result := CompileLoadedContentPackage(loaded)
	assertValidationCode(t, result.Validation, ErrorInvalidEnum)
	assertValidationPathContains(t, result.Validation, "hook_bindings[0].hook")
}

func TestContentCompilerOutputIsDeterministic(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	first := CompileLoadedContentPackage(loaded)
	if !first.Validation.Valid() {
		t.Fatalf("expected first compile to pass: %#v", first.Validation.Errors)
	}

	reordered := loaded
	if len(reordered.Items) >= 2 {
		reordered.Items[0], reordered.Items[1] = reordered.Items[1], reordered.Items[0]
	}
	if len(reordered.HookBindings) >= 2 {
		reordered.HookBindings[0], reordered.HookBindings[1] = reordered.HookBindings[1], reordered.HookBindings[0]
	}
	second := CompileLoadedContentPackage(reordered)
	if !second.Validation.Valid() {
		t.Fatalf("expected second compile to pass: %#v", second.Validation.Errors)
	}

	firstPayload, err := MarshalCompiledContentPackage(first.Package)
	if err != nil {
		t.Fatalf("marshal first package: %v", err)
	}
	secondPayload, err := MarshalCompiledContentPackage(second.Package)
	if err != nil {
		t.Fatalf("marshal second package: %v", err)
	}
	if !bytes.Equal(firstPayload, secondPayload) {
		t.Fatalf("expected deterministic compiler output\nfirst:\n%s\nsecond:\n%s", firstPayload, secondPayload)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
