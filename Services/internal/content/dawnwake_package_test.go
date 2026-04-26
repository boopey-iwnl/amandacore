package content

import "testing"

const dawnwakePackagePath = "Content/Packs/dawnwake_isles/package.json"

func TestDawnwakePackageLoadsContinentAndZones(t *testing.T) {
	result := NewContentPackageLoader().Load(dawnwakePackagePath)
	if !result.Validation.Valid() {
		t.Fatalf("expected Dawnwake package to validate, got %#v", result.Validation.Errors)
	}
	if result.Validated == nil {
		t.Fatalf("expected validated Dawnwake package")
	}
	if result.Package.Manifest.PackageID != "dawnwake_isles" {
		t.Fatalf("unexpected package id %q", result.Package.Manifest.PackageID)
	}
	if len(result.Package.Continents) != 1 {
		t.Fatalf("expected one continent, got %d", len(result.Package.Continents))
	}
	if len(result.Package.Zones) != 5 {
		t.Fatalf("expected five zones, got %d", len(result.Package.Zones))
	}
	if result.Package.Continents[0].DefaultEntry.ZoneID != "dawnwake_landing" {
		t.Fatalf("unexpected default entry: %#v", result.Package.Continents[0].DefaultEntry)
	}
}

func TestDawnwakeTransitionReferencesValidate(t *testing.T) {
	loaded := mustLoadDawnwakePackage(t)
	report := ValidateLoadedContentPackage(loaded)
	if !report.Valid() {
		t.Fatalf("expected Dawnwake package to validate, got %#v", report.Errors)
	}
	transitionCount := 0
	for _, zone := range loaded.Zones {
		transitionCount += len(zone.TransitionGates)
	}
	if transitionCount < 8 {
		t.Fatalf("expected at least 8 transition gates, got %d", transitionCount)
	}
}

func TestDawnwakeMissingTransitionDestinationRejected(t *testing.T) {
	loaded := mustLoadDawnwakePackage(t)
	loaded.Zones[0].TransitionGates[0].ToZoneID = "missing_zone"

	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorBrokenReference)
	assertValidationPathContains(t, report, "transition_gates[0].to_zone_id")
}

func TestDawnwakeMissingArrivalEntryRejected(t *testing.T) {
	loaded := mustLoadDawnwakePackage(t)
	loaded.Zones[0].TransitionGates[0].EntryPointIDOnArrival = "missing_entry"

	report := ValidateLoadedContentPackage(loaded)
	assertValidationCode(t, report, ErrorBrokenReference)
	assertValidationPathContains(t, report, "entry_point_id_on_arrival")
}

func TestDawnwakeRuntimeRegistryIncludesContinents(t *testing.T) {
	loaded := mustLoadDawnwakePackage(t)
	registry := NewRuntimeContentRegistry(loaded)

	if _, found := registry.Continents["dawnwake_isles"]; !found {
		t.Fatalf("expected Dawnwake continent in runtime registry")
	}
	if _, found := registry.Zones["kingsfall_harbor"]; !found {
		t.Fatalf("expected Kingsfall Harbor zone in runtime registry")
	}
}

func mustLoadDawnwakePackage(t *testing.T) LoadedContentPackage {
	t.Helper()
	result := NewContentPackageLoader().Load(dawnwakePackagePath)
	if !result.Validation.Valid() {
		t.Fatalf("expected Dawnwake package to load, got errors: %#v", result.Validation.Errors)
	}
	return result.Package
}
