package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ContentExporterID = "amandacore-content-exporter"

type MapAuthoringDefinition struct {
	MapID           string                         `json:"map_id"`
	ZoneID          string                         `json:"zone_id"`
	DisplayName     string                         `json:"display_name"`
	CoordinateSpace string                         `json:"coordinate_space"`
	Bounds          ZoneBounds                     `json:"bounds"`
	Entries         []MapAuthoringEntryMarker      `json:"entries"`
	AdjacentZones   []MapAdjacentZoneDefinition    `json:"adjacent_zones"`
	Transitions     []MapAuthoringTransitionMarker `json:"transitions"`
	StreamingCells  []MapAuthoringCellMarker       `json:"streaming_cells"`
	Landmarks       []MapAuthoringLandmarkMarker   `json:"landmarks"`
	AuthoringSource string                         `json:"authoring_source"`
	SourceScene     string                         `json:"source_scene"`
	Tags            []string                       `json:"tags"`
}

type MapAuthoringEntryMarker struct {
	MarkerID   string   `json:"marker_id"`
	EntityName string   `json:"entity_name"`
	EntryID    string   `json:"entry_id"`
	Position   Position `json:"position"`
	FacingYaw  float64  `json:"facing_yaw"`
	Tags       []string `json:"tags"`
}

type MapAuthoringTransitionMarker struct {
	MarkerID           string   `json:"marker_id"`
	EntityName         string   `json:"entity_name"`
	TransitionID       string   `json:"transition_id"`
	DisplayName        string   `json:"display_name"`
	TargetZoneID       string   `json:"target_zone_id"`
	DestinationEntryID string   `json:"destination_entry_id"`
	StreamingCellID    string   `json:"streaming_cell_id"`
	Hint               string   `json:"hint"`
	Position           Position `json:"position"`
	Radius             float64  `json:"radius"`
	Tags               []string `json:"tags"`
}

type MapAuthoringCellMarker struct {
	MarkerID    string     `json:"marker_id"`
	EntityName  string     `json:"entity_name"`
	CellID      string     `json:"cell_id"`
	DisplayName string     `json:"display_name"`
	Bounds      ZoneBounds `json:"bounds"`
	Priority    int        `json:"priority"`
	Tags        []string   `json:"tags"`
}

type MapAuthoringLandmarkMarker struct {
	MarkerID    string   `json:"marker_id"`
	EntityName  string   `json:"entity_name"`
	LandmarkID  string   `json:"landmark_id"`
	DisplayName string   `json:"display_name"`
	Kind        string   `json:"kind"`
	Position    Position `json:"position"`
	Tags        []string `json:"tags"`
}

type MapAuthoringExportResult struct {
	AuthoringFiles []string
	Exports        []MapExportDefinition
	Validation     ContentValidationReport
}

type MapExportWriteResult struct {
	Written []string
	Changed []string
}

type MapExportCheckResult struct {
	Compared []string
	Drift    []string
	Missing  []string
}

func GenerateMapExportsFromAuthoringDirectory(inputDir string) MapAuthoringExportResult {
	report := ContentValidationReport{}
	resolvedInput := resolveContentPath(inputDir)
	info, err := os.Stat(resolvedInput)
	if err != nil || !info.IsDir() {
		report.Addf(ErrorMissingFile, "authoring.input", "authoring directory %q could not be read", inputDir)
		return MapAuthoringExportResult{Validation: report}
	}

	files, err := filepath.Glob(filepath.Join(resolvedInput, "*.authoring.json"))
	if err != nil {
		report.Addf(ErrorMalformedJSON, "authoring.input", "authoring glob failed for %q: %v", inputDir, err)
		return MapAuthoringExportResult{Validation: report}
	}
	sort.Strings(files)
	if len(files) == 0 {
		report.Addf(ErrorMissingFile, "authoring.input", "authoring directory %q has no *.authoring.json files", inputDir)
		return MapAuthoringExportResult{Validation: report}
	}

	result := MapAuthoringExportResult{AuthoringFiles: append([]string(nil), files...)}
	seenMapIDs := map[string]struct{}{}
	seenZoneIDs := map[string]struct{}{}
	for index, path := range files {
		payload, err := os.ReadFile(path)
		if err != nil {
			report.Addf(ErrorMissingFile, fmt.Sprintf("authoring.files[%d]", index), "authoring file %q could not be read: %v", path, err)
			continue
		}
		var authoring MapAuthoringDefinition
		if err := json.Unmarshal(payload, &authoring); err != nil {
			report.Addf(ErrorMalformedJSON, fmt.Sprintf("authoring.files[%d]", index), "authoring file %q is malformed: %v", path, err)
			continue
		}
		validateMapAuthoring(authoring, index, seenMapIDs, seenZoneIDs, &report)
		result.Exports = append(result.Exports, mapAuthoringToExport(authoring))
	}
	sort.Slice(result.Exports, func(left, right int) bool {
		return result.Exports[left].ZoneID < result.Exports[right].ZoneID
	})
	result.Validation = report
	return result
}

func WriteMapExports(outputDir string, exports []MapExportDefinition) (MapExportWriteResult, error) {
	resolvedOutput := resolveContentPath(outputDir)
	if err := os.MkdirAll(resolvedOutput, 0o755); err != nil {
		return MapExportWriteResult{}, err
	}
	result := MapExportWriteResult{}
	for _, export := range exports {
		path := filepath.Join(resolvedOutput, export.ZoneID+".map.json")
		payload, err := EncodeMapExport(export)
		if err != nil {
			return result, err
		}
		existing, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(existing, payload) {
			result.Changed = append(result.Changed, path)
		}
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			return result, err
		}
		result.Written = append(result.Written, path)
	}
	return result, nil
}

func CheckMapExports(outputDir string, exports []MapExportDefinition) (MapExportCheckResult, error) {
	resolvedOutput := resolveContentPath(outputDir)
	result := MapExportCheckResult{}
	for _, export := range exports {
		path := filepath.Join(resolvedOutput, export.ZoneID+".map.json")
		expected, err := EncodeMapExport(export)
		if err != nil {
			return result, err
		}
		actual, err := os.ReadFile(path)
		if err != nil {
			result.Missing = append(result.Missing, path)
			continue
		}
		result.Compared = append(result.Compared, path)
		if !bytes.Equal(actual, expected) {
			result.Drift = append(result.Drift, path)
		}
	}
	return result, nil
}

func EncodeMapExport(export MapExportDefinition) ([]byte, error) {
	normalized := normalizeMapExport(export)
	payload, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func mapAuthoringToExport(authoring MapAuthoringDefinition) MapExportDefinition {
	export := MapExportDefinition{
		MapID:           authoring.MapID,
		ZoneID:          authoring.ZoneID,
		DisplayName:     authoring.DisplayName,
		CoordinateSpace: authoring.CoordinateSpace,
		Bounds:          authoring.Bounds,
		AdjacentZones:   append([]MapAdjacentZoneDefinition(nil), authoring.AdjacentZones...),
		AuthoringSource: authoring.AuthoringSource,
		GeneratedBy:     ContentExporterID,
		Tags:            append([]string(nil), authoring.Tags...),
	}
	for _, entry := range authoring.Entries {
		export.EntryPoints = append(export.EntryPoints, ZoneEntryPoint{
			EntryID:   entry.EntryID,
			Position:  entry.Position,
			FacingYaw: entry.FacingYaw,
		})
	}
	for _, transition := range authoring.Transitions {
		export.TransitionPoints = append(export.TransitionPoints, MapTransitionPointDefinition{
			TransitionID:       transition.TransitionID,
			DisplayName:        transition.DisplayName,
			TargetZoneID:       transition.TargetZoneID,
			DestinationEntryID: transition.DestinationEntryID,
			StreamingCellID:    transition.StreamingCellID,
			Hint:               transition.Hint,
			Position:           transition.Position,
			Radius:             transition.Radius,
			Tags:               append([]string(nil), transition.Tags...),
		})
	}
	for _, cell := range authoring.StreamingCells {
		export.StreamingCells = append(export.StreamingCells, StreamingCellDefinition{
			CellID:      cell.CellID,
			DisplayName: cell.DisplayName,
			Bounds:      cell.Bounds,
			Priority:    cell.Priority,
			Tags:        append([]string(nil), cell.Tags...),
		})
	}
	for _, landmark := range authoring.Landmarks {
		export.Landmarks = append(export.Landmarks, MapLandmarkDefinition{
			LandmarkID:  landmark.LandmarkID,
			DisplayName: landmark.DisplayName,
			Kind:        landmark.Kind,
			Position:    landmark.Position,
			Tags:        append([]string(nil), landmark.Tags...),
		})
	}
	return normalizeMapExport(export)
}

func normalizeMapExport(export MapExportDefinition) MapExportDefinition {
	sort.Slice(export.EntryPoints, func(left, right int) bool {
		return export.EntryPoints[left].EntryID < export.EntryPoints[right].EntryID
	})
	sort.Slice(export.AdjacentZones, func(left, right int) bool {
		leftKey := export.AdjacentZones[left].ZoneID + "." + export.AdjacentZones[left].TransitionID
		rightKey := export.AdjacentZones[right].ZoneID + "." + export.AdjacentZones[right].TransitionID
		return leftKey < rightKey
	})
	sort.Slice(export.TransitionPoints, func(left, right int) bool {
		return export.TransitionPoints[left].TransitionID < export.TransitionPoints[right].TransitionID
	})
	sort.Slice(export.StreamingCells, func(left, right int) bool {
		return export.StreamingCells[left].CellID < export.StreamingCells[right].CellID
	})
	sort.Slice(export.Landmarks, func(left, right int) bool {
		return export.Landmarks[left].LandmarkID < export.Landmarks[right].LandmarkID
	})
	for index := range export.AdjacentZones {
		sort.Strings(export.AdjacentZones[index].Tags)
	}
	for index := range export.TransitionPoints {
		sort.Strings(export.TransitionPoints[index].Tags)
	}
	for index := range export.StreamingCells {
		sort.Strings(export.StreamingCells[index].Tags)
	}
	for index := range export.Landmarks {
		sort.Strings(export.Landmarks[index].Tags)
	}
	sort.Strings(export.Tags)
	return export
}

func validateMapAuthoring(authoring MapAuthoringDefinition, index int, seenMapIDs map[string]struct{}, seenZoneIDs map[string]struct{}, report *ContentValidationReport) {
	path := fmt.Sprintf("authoring[%d]", index)
	requiredID(report, path+".map_id", authoring.MapID)
	requiredID(report, path+".zone_id", authoring.ZoneID)
	requiredString(report, path+".display_name", authoring.DisplayName)
	requiredString(report, path+".coordinate_space", authoring.CoordinateSpace)
	requiredString(report, path+".authoring_source", authoring.AuthoringSource)
	if authoring.MapID != "" {
		if _, exists := seenMapIDs[authoring.MapID]; exists {
			report.Addf(ErrorDuplicateID, path+".map_id", "map id %q is duplicated", authoring.MapID)
		}
		seenMapIDs[authoring.MapID] = struct{}{}
	}
	if authoring.ZoneID != "" {
		if _, exists := seenZoneIDs[authoring.ZoneID]; exists {
			report.Addf(ErrorDuplicateID, path+".zone_id", "zone id %q has multiple authoring files", authoring.ZoneID)
		}
		seenZoneIDs[authoring.ZoneID] = struct{}{}
	}
	if authoring.CoordinateSpace != "" && !validEnum(authoring.CoordinateSpace, "amandacore_server", "o3de_placeholder") {
		report.Addf(ErrorInvalidEnum, path+".coordinate_space", "coordinate_space %q is not valid", authoring.CoordinateSpace)
	}
	boundsValid := validateBounds(path+".bounds", authoring.Bounds, report)
	validateAuthoringMarkers(path+".entries", authoring.Entries, func(entry MapAuthoringEntryMarker) (string, string, Position) {
		return entry.MarkerID, entry.EntityName, entry.Position
	}, boundsValid, authoring.Bounds, report)
	validateAuthoringMarkers(path+".transitions", authoring.Transitions, func(transition MapAuthoringTransitionMarker) (string, string, Position) {
		return transition.MarkerID, transition.EntityName, transition.Position
	}, boundsValid, authoring.Bounds, report)
	validateAuthoringMarkers(path+".landmarks", authoring.Landmarks, func(landmark MapAuthoringLandmarkMarker) (string, string, Position) {
		return landmark.MarkerID, landmark.EntityName, landmark.Position
	}, boundsValid, authoring.Bounds, report)

	cellIDs := map[string]struct{}{}
	for cellIndex, cell := range authoring.StreamingCells {
		cellPath := fmt.Sprintf("%s.streaming_cells[%d]", path, cellIndex)
		requiredID(report, cellPath+".marker_id", cell.MarkerID)
		requiredString(report, cellPath+".entity_name", cell.EntityName)
		requiredID(report, cellPath+".cell_id", cell.CellID)
		requiredString(report, cellPath+".display_name", cell.DisplayName)
		if cell.CellID != "" {
			if _, exists := cellIDs[cell.CellID]; exists {
				report.Addf(ErrorDuplicateID, cellPath+".cell_id", "streaming cell id %q is duplicated", cell.CellID)
			}
			cellIDs[cell.CellID] = struct{}{}
		}
		cellBoundsValid := validateBounds(cellPath+".bounds", cell.Bounds, report)
		if boundsValid && cellBoundsValid && !boundsContain(authoring.Bounds, cell.Bounds) {
			report.Addf(ErrorPositionOutOfBounds, cellPath+".bounds", "streaming cell %q is outside authoring map bounds", cell.CellID)
		}
	}
	for transitionIndex, transition := range authoring.Transitions {
		transitionPath := fmt.Sprintf("%s.transitions[%d]", path, transitionIndex)
		requiredID(report, transitionPath+".transition_id", transition.TransitionID)
		requiredString(report, transitionPath+".display_name", transition.DisplayName)
		requiredID(report, transitionPath+".target_zone_id", transition.TargetZoneID)
		requiredID(report, transitionPath+".destination_entry_id", transition.DestinationEntryID)
		requiredID(report, transitionPath+".streaming_cell_id", transition.StreamingCellID)
		if transition.StreamingCellID != "" && !containsID(cellIDs, transition.StreamingCellID) {
			report.Addf(ErrorBrokenReference, transitionPath+".streaming_cell_id", "transition %q references missing streaming cell %q", transition.TransitionID, transition.StreamingCellID)
		}
		if transition.Radius <= 0 {
			report.Add(ErrorInvalidNumberRange, transitionPath+".radius", "transition radius must be positive")
		}
	}
	for adjacentIndex, adjacent := range authoring.AdjacentZones {
		adjacentPath := fmt.Sprintf("%s.adjacent_zones[%d]", path, adjacentIndex)
		requiredID(report, adjacentPath+".zone_id", adjacent.ZoneID)
		requiredID(report, adjacentPath+".transition_id", adjacent.TransitionID)
		requiredString(report, adjacentPath+".direction", adjacent.Direction)
	}
}

func validateAuthoringMarkers[T any](path string, markers []T, markerParts func(T) (string, string, Position), boundsValid bool, bounds ZoneBounds, report *ContentValidationReport) {
	seenMarkerIDs := map[string]struct{}{}
	for index, marker := range markers {
		markerID, entityName, position := markerParts(marker)
		markerPath := fmt.Sprintf("%s[%d]", path, index)
		requiredID(report, markerPath+".marker_id", markerID)
		requiredString(report, markerPath+".entity_name", entityName)
		if markerID != "" {
			if _, exists := seenMarkerIDs[markerID]; exists {
				report.Addf(ErrorDuplicateID, markerPath+".marker_id", "marker id %q is duplicated", markerID)
			}
			seenMarkerIDs[markerID] = struct{}{}
		}
		if boundsValid && !positionInBounds(position, bounds) {
			report.Addf(ErrorPositionOutOfBounds, markerPath+".position", "marker %q is outside authoring map bounds", markerID)
		}
	}
}

func resolveContentPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return trimmed
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed)
	}
	if found, ok := resolveRelativeFromParents(trimmed); ok {
		return found
	}
	return filepath.Clean(trimmed)
}
