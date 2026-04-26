package content

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDawnwakeAuthoringExportsGenerateExpectedMaps(t *testing.T) {
	result := GenerateMapExportsFromAuthoringDirectory("Content/Authoring/DawnwakeIsles")
	if !result.Validation.Valid() {
		t.Fatalf("expected authoring metadata to validate, got %#v", result.Validation.Errors)
	}
	if len(result.Exports) != 3 {
		t.Fatalf("expected three map exports, got %d", len(result.Exports))
	}
	ids := map[string]bool{}
	for _, export := range result.Exports {
		ids[export.MapID] = true
		if export.GeneratedBy != ContentExporterID {
			t.Fatalf("expected generated_by marker %q, got %#v", ContentExporterID, export)
		}
	}
	for _, expected := range []string{"dw_map_landing", "dw_map_tideglass_shoal", "dw_map_windspur_rise"} {
		if !ids[expected] {
			t.Fatalf("expected generated map id %q, got %#v", expected, ids)
		}
	}
}

func TestDawnwakeAuthoringExportsMatchCommittedMaps(t *testing.T) {
	result := GenerateMapExportsFromAuthoringDirectory("Content/Authoring/DawnwakeIsles")
	if !result.Validation.Valid() {
		t.Fatalf("expected authoring metadata to validate, got %#v", result.Validation.Errors)
	}
	check, err := CheckMapExports("Content/Packs/dawnwake_isles/maps", result.Exports)
	if err != nil {
		t.Fatalf("check map exports: %v", err)
	}
	if len(check.Missing) > 0 || len(check.Drift) > 0 {
		t.Fatalf("expected committed map exports to match generated output, missing=%#v drift=%#v", check.Missing, check.Drift)
	}
}

func TestMapAuthoringMissingStreamingCellRejected(t *testing.T) {
	tempDir := t.TempDir()
	authoring := `{
  "map_id": "test_map",
  "zone_id": "test_zone",
  "display_name": "Test Map",
  "coordinate_space": "o3de_placeholder",
  "authoring_source": "AmandaCore test metadata.",
  "bounds": {"min_x":0,"min_y":0,"min_z":0,"max_x":10,"max_y":10,"max_z":10},
  "entries": [],
  "adjacent_zones": [],
  "transitions": [{
    "marker_id": "transition_marker",
    "entity_name": "TransitionMarker",
    "transition_id": "to_missing",
    "display_name": "To Missing",
    "target_zone_id": "missing_zone",
    "destination_entry_id": "entry",
    "streaming_cell_id": "missing_cell",
    "position": {"x":1,"y":1,"z":1},
    "radius": 2
  }],
  "streaming_cells": [],
  "landmarks": []
}`
	if err := os.WriteFile(filepath.Join(tempDir, "test.authoring.json"), []byte(authoring), 0o644); err != nil {
		t.Fatalf("write authoring: %v", err)
	}
	result := GenerateMapExportsFromAuthoringDirectory(tempDir)
	assertValidationCode(t, result.Validation, ErrorBrokenReference)
	assertValidationPathContains(t, result.Validation, "streaming_cell_id")
}

func TestWriteMapExportsIsDeterministic(t *testing.T) {
	result := GenerateMapExportsFromAuthoringDirectory("Content/Authoring/DawnwakeIsles")
	if !result.Validation.Valid() {
		t.Fatalf("expected authoring metadata to validate, got %#v", result.Validation.Errors)
	}
	output := t.TempDir()
	if _, err := WriteMapExports(output, result.Exports); err != nil {
		t.Fatalf("write exports: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(output, "dawnwake_landing.map.json"))
	if err != nil {
		t.Fatalf("read first export: %v", err)
	}
	if _, err := WriteMapExports(output, result.Exports); err != nil {
		t.Fatalf("rewrite exports: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(output, "dawnwake_landing.map.json"))
	if err != nil {
		t.Fatalf("read second export: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("expected deterministic export bytes")
	}
}
