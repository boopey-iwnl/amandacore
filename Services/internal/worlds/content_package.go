package worlds

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"amandacore/services/internal/observability"
)

const CurrentContentSchemaVersion = "amandacore.content.v1"

type ContentPackageLoader struct{}

type RuntimeContentRegistry struct {
	Package       ContentPackageManifest
	PackageRoot   string
	Continents    map[string]ContinentDefinition
	Zones         map[string]ZoneDefinition
	NpcArchetypes map[string]NpcArchetype
	NpcSpawns     map[string]NpcSpawnPoint
	Items         map[string]ContentStubDefinition
	LootTables    map[string]ContentStubDefinition
	Quests        map[string]ContentStubDefinition
	Abilities     map[string]ContentStubDefinition
	Auras         map[string]ContentStubDefinition
}

type ContentPackageManifest struct {
	PackageID     string         `json:"package_id"`
	DisplayName   string         `json:"display_name"`
	SchemaVersion string         `json:"schema_version"`
	Version       string         `json:"version"`
	Continent     string         `json:"continent,omitempty"`
	Zones         []string       `json:"zones,omitempty"`
	NPCs          string         `json:"npcs,omitempty"`
	Items         string         `json:"items,omitempty"`
	Loot          string         `json:"loot,omitempty"`
	Quests        string         `json:"quests,omitempty"`
	Abilities     string         `json:"abilities,omitempty"`
	Auras         string         `json:"auras,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type ContentStubDefinition struct {
	ID          string         `json:"id"`
	ItemID      string         `json:"item_id,omitempty"`
	LootTableID string         `json:"loot_table_id,omitempty"`
	QuestID     string         `json:"quest_id,omitempty"`
	AbilityID   string         `json:"ability_id,omitempty"`
	AuraID      string         `json:"aura_id,omitempty"`
	DisplayName string         `json:"display_name,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type continentContentFile struct {
	Continent ContinentDefinition `json:"continent"`
}

type zoneContentFile struct {
	Zone ZoneDefinition `json:"zone"`
}

type npcContentFile struct {
	NpcArchetypes []NpcArchetype  `json:"npc_archetypes"`
	Archetypes    []NpcArchetype  `json:"archetypes,omitempty"`
	SpawnPoints   []NpcSpawnPoint `json:"spawn_points,omitempty"`
}

type stubContentFile struct {
	Items     []ContentStubDefinition `json:"items,omitempty"`
	Loot      []ContentStubDefinition `json:"loot_tables,omitempty"`
	Quests    []ContentStubDefinition `json:"quests,omitempty"`
	Abilities []ContentStubDefinition `json:"abilities,omitempty"`
	Auras     []ContentStubDefinition `json:"auras,omitempty"`
}

type ContinentID string

type ContinentDefinition struct {
	ContinentID  string                   `json:"continent_id"`
	DisplayName  string                   `json:"display_name"`
	Description  string                   `json:"description,omitempty"`
	Origin       CoordinateOrigin         `json:"origin"`
	Units        string                   `json:"units"`
	Zones        []string                 `json:"zones"`
	Adjacency    []ZoneAdjacency          `json:"adjacency,omitempty"`
	DefaultEntry ContinentEntryRef        `json:"default_entry"`
	Streaming    ContinentStreamingConfig `json:"streaming,omitempty"`
	Tags         []string                 `json:"tags,omitempty"`
	Metadata     map[string]any           `json:"metadata,omitempty"`
}

type CoordinateOrigin struct {
	Description string  `json:"description,omitempty"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Z           float64 `json:"z"`
}

type ContinentEntryRef struct {
	ZoneID       string `json:"zone_id"`
	EntryPointID string `json:"entry_point_id"`
}

type ContinentStreamingConfig struct {
	ActivationPolicy        string  `json:"activation_policy,omitempty"`
	DefaultInterestRadius   float64 `json:"default_interest_radius,omitempty"`
	TransitionHintRadius    float64 `json:"transition_hint_radius,omitempty"`
	GateHintRadius          float64 `json:"gate_hint_radius,omitempty"`
	FutureStreamingHookName string  `json:"future_streaming_hook_name,omitempty"`
}

type ZoneAdjacency struct {
	FromZoneID     string   `json:"from_zone_id"`
	ToZoneID       string   `json:"to_zone_id"`
	TransitionIDs  []string `json:"transition_ids,omitempty"`
	Kind           string   `json:"kind,omitempty"`
	Bidirectional  bool     `json:"bidirectional,omitempty"`
	StreamingNotes string   `json:"streaming_notes,omitempty"`
}

type ZoneDefinition struct {
	ZoneID          string                    `json:"zone_id"`
	ContinentID     string                    `json:"continent_id,omitempty"`
	DisplayName     string                    `json:"display_name"`
	Description     string                    `json:"description,omitempty"`
	Bounds          ZoneBounds                `json:"bounds"`
	EntryPoints     []ZoneEntryPoint          `json:"entry_points,omitempty"`
	TransitionGates []ZoneTransitionGate      `json:"transition_gates,omitempty"`
	SpawnGroups     []ZoneSpawnGroup          `json:"spawn_groups,omitempty"`
	QuestProviders  []QuestProviderDefinition `json:"quest_providers,omitempty"`
	Runtime         ZoneRuntimeConfig         `json:"runtime,omitempty"`
	RuntimeConfig   ZoneRuntimeConfig         `json:"runtime_config,omitempty"`
	Streaming       ZoneStreamingConfig       `json:"streaming,omitempty"`
	StreamingHints  ZoneStreamingConfig       `json:"streaming_hints,omitempty"`
	Tags            []string                  `json:"tags,omitempty"`
	Metadata        map[string]any            `json:"metadata,omitempty"`
}

type ZoneBounds struct {
	MinX     float64 `json:"min_x"`
	MinY     float64 `json:"min_y"`
	MinZ     float64 `json:"min_z"`
	MaxX     float64 `json:"max_x"`
	MaxY     float64 `json:"max_y"`
	MaxZ     float64 `json:"max_z"`
	Accuracy string  `json:"accuracy,omitempty"`
	Source   string  `json:"source,omitempty"`
}

type WorldPosition struct {
	ZoneID string  `json:"zone_id,omitempty"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
}

type ZoneEntryPoint struct {
	EntryPointID string        `json:"entry_point_id"`
	DisplayName  string        `json:"display_name,omitempty"`
	Position     WorldPosition `json:"position"`
	Facing       float64       `json:"facing,omitempty"`
	FacingYaw    float64       `json:"facing_yaw,omitempty"`
	Tags         []string      `json:"tags,omitempty"`
}

type ZoneTransitionGate struct {
	TransitionID          string     `json:"transition_id"`
	FromZoneID            string     `json:"from_zone_id"`
	ToZoneID              string     `json:"to_zone_id"`
	Kind                  string     `json:"kind"`
	GateBounds            ZoneBounds `json:"gate_bounds"`
	EntryPointIDOnArrival string     `json:"entry_point_id_on_arrival"`
	RequiredFlags         []string   `json:"required_flags,omitempty"`
	Disabled              bool       `json:"disabled,omitempty"`
	Tags                  []string   `json:"tags,omitempty"`
}

type ZoneSpawnGroup struct {
	SpawnGroupID string          `json:"spawn_group_id"`
	DisplayName  string          `json:"display_name,omitempty"`
	ArchetypeID  string          `json:"archetype_id"`
	Disposition  string          `json:"disposition,omitempty"`
	SpawnPoints  []NpcSpawnPoint `json:"spawn_points,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
}

type NpcArchetype struct {
	ArchetypeID string   `json:"archetype_id"`
	DisplayName string   `json:"display_name"`
	Disposition string   `json:"disposition,omitempty"`
	Level       int      `json:"level,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type NpcSpawnPoint struct {
	SpawnPointID string        `json:"spawn_point_id"`
	ArchetypeID  string        `json:"archetype_id"`
	Position     WorldPosition `json:"position"`
	Tags         []string      `json:"tags,omitempty"`
}

type QuestProviderDefinition struct {
	ProviderID  string        `json:"provider_id"`
	DisplayName string        `json:"display_name"`
	Kind        string        `json:"kind,omitempty"`
	Position    WorldPosition `json:"position"`
	QuestIDs    []string      `json:"quest_ids,omitempty"`
	Services    []string      `json:"services,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
}

type ZoneRuntimeConfig struct {
	ActivationPolicy          string   `json:"activation_policy,omitempty"`
	ExpectedPlayerDensity     string   `json:"expected_player_density,omitempty"`
	HighPlayerDensityExpected bool     `json:"high_player_density_expected,omitempty"`
	HostileDensity            string   `json:"hostile_density,omitempty"`
	FutureHooks               []string `json:"future_hooks,omitempty"`
}

type ZoneStreamingConfig struct {
	InterestRadius          float64  `json:"interest_radius,omitempty"`
	PreloadRadius           float64  `json:"preload_radius,omitempty"`
	AdjacentPreloadDistance float64  `json:"adjacent_preload_distance,omitempty"`
	AdjacentZoneHinting     bool     `json:"adjacent_zone_hinting,omitempty"`
	Priority                string   `json:"priority,omitempty"`
	Notes                   []string `json:"notes,omitempty"`
	Tags                    []string `json:"tags,omitempty"`
}

type ValidationCode string

const (
	ValidationMissingContinentZone         ValidationCode = "MissingContinentZone"
	ValidationMissingDefaultEntry          ValidationCode = "MissingDefaultEntry"
	ValidationMissingTransitionDestination ValidationCode = "MissingTransitionDestination"
	ValidationMissingTransitionEntryPoint  ValidationCode = "MissingTransitionEntryPoint"
	ValidationTransitionGateOutOfBounds    ValidationCode = "TransitionGateOutOfBounds"
	ValidationZoneBoundsOverlap            ValidationCode = "ZoneBoundsOverlap"
	ValidationDuplicateTransitionID        ValidationCode = "DuplicateTransitionID"
	ValidationInvalidTopology              ValidationCode = "InvalidTopology"
	ValidationDuplicateZoneID              ValidationCode = "DuplicateZoneID"
	ValidationEntryPointOutOfBounds        ValidationCode = "EntryPointOutOfBounds"
	ValidationSpawnPointOutOfBounds        ValidationCode = "SpawnPointOutOfBounds"
)

type ValidationError struct {
	Code    ValidationCode
	Path    string
	Message string
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	parts := make([]string, 0, len(e))
	for _, err := range e {
		if err.Path == "" {
			parts = append(parts, fmt.Sprintf("%s: %s", err.Code, err.Message))
		} else {
			parts = append(parts, fmt.Sprintf("%s at %s: %s", err.Code, err.Path, err.Message))
		}
	}
	return strings.Join(parts, "; ")
}

func (e ValidationErrors) Has(code ValidationCode) bool {
	for _, err := range e {
		if err.Code == code {
			return true
		}
	}
	return false
}

func NewContentPackageLoader() ContentPackageLoader {
	return ContentPackageLoader{}
}

func (l ContentPackageLoader) Load(packagePath string) (*RuntimeContentRegistry, error) {
	manifest, err := readJSONFile[ContentPackageManifest](packagePath)
	if err != nil {
		return nil, err
	}
	root := filepath.Dir(packagePath)
	registry := &RuntimeContentRegistry{
		Package:       manifest,
		PackageRoot:   root,
		Continents:    map[string]ContinentDefinition{},
		Zones:         map[string]ZoneDefinition{},
		NpcArchetypes: map[string]NpcArchetype{},
		NpcSpawns:     map[string]NpcSpawnPoint{},
		Items:         map[string]ContentStubDefinition{},
		LootTables:    map[string]ContentStubDefinition{},
		Quests:        map[string]ContentStubDefinition{},
		Abilities:     map[string]ContentStubDefinition{},
		Auras:         map[string]ContentStubDefinition{},
	}

	if manifest.SchemaVersion == "" {
		return nil, ValidationErrors{{Code: ValidationInvalidTopology, Path: "package.schema_version", Message: "schema_version is required"}}
	}
	if manifest.SchemaVersion != CurrentContentSchemaVersion {
		return nil, ValidationErrors{{Code: ValidationInvalidTopology, Path: "package.schema_version", Message: fmt.Sprintf("unsupported schema_version %q", manifest.SchemaVersion)}}
	}
	if manifest.PackageID == "" {
		return nil, ValidationErrors{{Code: ValidationInvalidTopology, Path: "package.package_id", Message: "package_id is required"}}
	}

	for index, zonePath := range manifest.Zones {
		zone, err := loadZoneDefinition(filepath.Join(root, filepath.FromSlash(zonePath)))
		if err != nil {
			return nil, fmt.Errorf("load zone %s: %w", zonePath, err)
		}
		if zone.ZoneID == "" {
			return nil, ValidationErrors{{Code: ValidationInvalidTopology, Path: fmt.Sprintf("zones[%d].zone_id", index), Message: "zone_id is required"}}
		}
		if _, exists := registry.Zones[zone.ZoneID]; exists {
			return nil, ValidationErrors{{Code: ValidationDuplicateZoneID, Path: fmt.Sprintf("zones[%d].zone_id", index), Message: fmt.Sprintf("duplicate zone_id %s", zone.ZoneID)}}
		}
		normalizeZoneDefinition(&zone)
		registry.Zones[zone.ZoneID] = zone
	}

	if manifest.Continent != "" {
		continent, err := loadContinentDefinition(filepath.Join(root, filepath.FromSlash(manifest.Continent)))
		if err != nil {
			return nil, fmt.Errorf("load continent %s: %w", manifest.Continent, err)
		}
		if continent.ContinentID == "" {
			return nil, ValidationErrors{{Code: ValidationInvalidTopology, Path: "continent.continent_id", Message: "continent_id is required"}}
		}
		registry.Continents[continent.ContinentID] = continent
		observability.LogEvent("world-service", observability.EventContentContinentLoaded, map[string]any{
			"packageId":   manifest.PackageID,
			"continentId": continent.ContinentID,
			"zoneCount":   len(continent.Zones),
		})
		for _, adjacency := range continent.Adjacency {
			observability.LogEvent("world-service", observability.EventContentZoneAdjacencyLoaded, map[string]any{
				"continentId": continent.ContinentID,
				"fromZoneId":  adjacency.FromZoneID,
				"toZoneId":    adjacency.ToZoneID,
			})
		}
	}

	if manifest.NPCs != "" {
		if err := loadNPCContent(registry, filepath.Join(root, filepath.FromSlash(manifest.NPCs))); err != nil {
			return nil, err
		}
	}
	if manifest.Items != "" {
		if err := loadStubContent(filepath.Join(root, filepath.FromSlash(manifest.Items)), "items", registry.Items); err != nil {
			return nil, err
		}
	}
	if manifest.Loot != "" {
		if err := loadStubContent(filepath.Join(root, filepath.FromSlash(manifest.Loot)), "loot", registry.LootTables); err != nil {
			return nil, err
		}
	}
	if manifest.Quests != "" {
		if err := loadStubContent(filepath.Join(root, filepath.FromSlash(manifest.Quests)), "quests", registry.Quests); err != nil {
			return nil, err
		}
	}
	if manifest.Abilities != "" {
		if err := loadStubContent(filepath.Join(root, filepath.FromSlash(manifest.Abilities)), "abilities", registry.Abilities); err != nil {
			return nil, err
		}
	}
	if manifest.Auras != "" {
		if err := loadStubContent(filepath.Join(root, filepath.FromSlash(manifest.Auras)), "auras", registry.Auras); err != nil {
			return nil, err
		}
	}

	if err := registry.Validate(); err != nil {
		observability.LogEvent("world-service", observability.EventContentContinentValidationFailed, map[string]any{
			"packageId": manifest.PackageID,
			"error":     err.Error(),
		})
		return nil, err
	}
	return registry, nil
}

func loadContinentDefinition(path string) (ContinentDefinition, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ContinentDefinition{}, err
	}
	var wrapped continentContentFile
	if err := json.Unmarshal(content, &wrapped); err != nil {
		return ContinentDefinition{}, err
	}
	if wrapped.Continent.ContinentID != "" {
		return wrapped.Continent, nil
	}

	var raw struct {
		ContinentID  string                   `json:"continent_id"`
		DisplayName  string                   `json:"display_name"`
		Description  string                   `json:"description,omitempty"`
		Origin       CoordinateOrigin         `json:"origin"`
		Units        string                   `json:"units"`
		Zones        []json.RawMessage        `json:"zones"`
		Adjacency    []ZoneAdjacency          `json:"adjacency,omitempty"`
		DefaultEntry ContinentEntryRef        `json:"default_entry"`
		Streaming    ContinentStreamingConfig `json:"streaming,omitempty"`
		Tags         []string                 `json:"tags,omitempty"`
		Metadata     map[string]any           `json:"metadata,omitempty"`
	}
	if err := json.Unmarshal(content, &raw); err != nil {
		return ContinentDefinition{}, err
	}
	zones := make([]string, 0, len(raw.Zones))
	for _, zoneRef := range raw.Zones {
		var zoneID string
		if err := json.Unmarshal(zoneRef, &zoneID); err == nil && zoneID != "" {
			zones = append(zones, zoneID)
			continue
		}
		var objectRef struct {
			ZoneID string `json:"zone_id"`
		}
		if err := json.Unmarshal(zoneRef, &objectRef); err != nil {
			return ContinentDefinition{}, err
		}
		zones = append(zones, objectRef.ZoneID)
	}
	return ContinentDefinition{
		ContinentID:  raw.ContinentID,
		DisplayName:  raw.DisplayName,
		Description:  raw.Description,
		Origin:       raw.Origin,
		Units:        raw.Units,
		Zones:        zones,
		Adjacency:    raw.Adjacency,
		DefaultEntry: raw.DefaultEntry,
		Streaming:    raw.Streaming,
		Tags:         raw.Tags,
		Metadata:     raw.Metadata,
	}, nil
}

func loadZoneDefinition(path string) (ZoneDefinition, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ZoneDefinition{}, err
	}
	var wrapped zoneContentFile
	if err := json.Unmarshal(content, &wrapped); err != nil {
		return ZoneDefinition{}, err
	}
	if wrapped.Zone.ZoneID != "" {
		normalizeZoneDefinition(&wrapped.Zone)
		return wrapped.Zone, nil
	}
	var zone ZoneDefinition
	if err := json.Unmarshal(content, &zone); err != nil {
		return ZoneDefinition{}, err
	}
	normalizeZoneDefinition(&zone)
	return zone, nil
}

func (r *RuntimeContentRegistry) Validate() error {
	var validation ValidationErrors
	for _, continent := range r.Continents {
		validation = append(validation, r.validateContinent(continent)...)
	}
	for _, zone := range r.Zones {
		validation = append(validation, validateZone(zone, r.NpcArchetypes)...)
	}
	if len(validation) > 0 {
		sort.Slice(validation, func(i, j int) bool {
			if validation[i].Code == validation[j].Code {
				return validation[i].Path < validation[j].Path
			}
			return validation[i].Code < validation[j].Code
		})
		return validation
	}
	return nil
}

func (r *RuntimeContentRegistry) validateContinent(continent ContinentDefinition) ValidationErrors {
	var validation ValidationErrors
	seenZones := map[string]bool{}
	for index, zoneID := range continent.Zones {
		if _, found := r.Zones[zoneID]; !found {
			validation = append(validation, ValidationError{Code: ValidationMissingContinentZone, Path: fmt.Sprintf("continent.zones[%d]", index), Message: fmt.Sprintf("zone %s is referenced by continent %s but is not loaded", zoneID, continent.ContinentID)})
		}
		if seenZones[zoneID] {
			validation = append(validation, ValidationError{Code: ValidationDuplicateZoneID, Path: fmt.Sprintf("continent.zones[%d]", index), Message: fmt.Sprintf("zone %s is referenced more than once", zoneID)})
		}
		seenZones[zoneID] = true
	}

	defaultZone, found := r.Zones[continent.DefaultEntry.ZoneID]
	if !found {
		validation = append(validation, ValidationError{Code: ValidationMissingDefaultEntry, Path: "continent.default_entry.zone_id", Message: fmt.Sprintf("default entry zone %s is not loaded", continent.DefaultEntry.ZoneID)})
	} else if _, ok := defaultZone.entryPoint(continent.DefaultEntry.EntryPointID); !ok {
		validation = append(validation, ValidationError{Code: ValidationMissingDefaultEntry, Path: "continent.default_entry.entry_point_id", Message: fmt.Sprintf("default entry point %s does not exist in zone %s", continent.DefaultEntry.EntryPointID, continent.DefaultEntry.ZoneID)})
	}

	transitionIDs := map[string]string{}
	for _, zoneID := range continent.Zones {
		zone, found := r.Zones[zoneID]
		if !found {
			continue
		}
		for index, transition := range zone.TransitionGates {
			path := fmt.Sprintf("zones.%s.transition_gates[%d]", zone.ZoneID, index)
			if transition.FromZoneID != zone.ZoneID {
				validation = append(validation, ValidationError{Code: ValidationInvalidTopology, Path: path + ".from_zone_id", Message: "transition from_zone_id must match containing zone"})
			}
			if _, exists := transitionIDs[transition.TransitionID]; exists {
				validation = append(validation, ValidationError{Code: ValidationDuplicateTransitionID, Path: path + ".transition_id", Message: fmt.Sprintf("transition_id %s is already used by %s", transition.TransitionID, transitionIDs[transition.TransitionID])})
			}
			transitionIDs[transition.TransitionID] = zone.ZoneID
			destination, destinationFound := r.Zones[transition.ToZoneID]
			if !destinationFound {
				validation = append(validation, ValidationError{Code: ValidationMissingTransitionDestination, Path: path + ".to_zone_id", Message: fmt.Sprintf("destination zone %s is not loaded", transition.ToZoneID)})
			} else if _, ok := destination.entryPoint(transition.EntryPointIDOnArrival); !ok {
				validation = append(validation, ValidationError{Code: ValidationMissingTransitionEntryPoint, Path: path + ".entry_point_id_on_arrival", Message: fmt.Sprintf("entry point %s is missing from destination zone %s", transition.EntryPointIDOnArrival, transition.ToZoneID)})
			}
			if !zone.Bounds.ContainsBounds(transition.GateBounds) {
				validation = append(validation, ValidationError{Code: ValidationTransitionGateOutOfBounds, Path: path + ".gate_bounds", Message: fmt.Sprintf("transition gate %s is outside source zone bounds", transition.TransitionID)})
			}
			observability.LogEvent("world-service", observability.EventContentZoneTransitionLoaded, map[string]any{
				"continentId":  continent.ContinentID,
				"transitionId": transition.TransitionID,
				"fromZoneId":   transition.FromZoneID,
				"toZoneId":     transition.ToZoneID,
				"disabled":     transition.Disabled,
			})
		}
	}

	for index, adjacency := range continent.Adjacency {
		path := fmt.Sprintf("continent.adjacency[%d]", index)
		if _, found := r.Zones[adjacency.FromZoneID]; !found {
			validation = append(validation, ValidationError{Code: ValidationMissingContinentZone, Path: path + ".from_zone_id", Message: fmt.Sprintf("adjacency source zone %s is not loaded", adjacency.FromZoneID)})
		}
		if _, found := r.Zones[adjacency.ToZoneID]; !found {
			validation = append(validation, ValidationError{Code: ValidationMissingContinentZone, Path: path + ".to_zone_id", Message: fmt.Sprintf("adjacency destination zone %s is not loaded", adjacency.ToZoneID)})
		}
		for _, transitionID := range adjacency.TransitionIDs {
			if _, found := transitionIDs[transitionID]; !found {
				validation = append(validation, ValidationError{Code: ValidationInvalidTopology, Path: path + ".transition_ids", Message: fmt.Sprintf("adjacency references unknown transition %s", transitionID)})
			}
		}
	}

	for i := 0; i < len(continent.Zones); i++ {
		left, leftFound := r.Zones[continent.Zones[i]]
		if !leftFound {
			continue
		}
		for j := i + 1; j < len(continent.Zones); j++ {
			right, rightFound := r.Zones[continent.Zones[j]]
			if rightFound && left.Bounds.Overlaps(right.Bounds) {
				validation = append(validation, ValidationError{Code: ValidationZoneBoundsOverlap, Path: fmt.Sprintf("continent.zones[%d:%d]", i, j), Message: fmt.Sprintf("zone bounds overlap between %s and %s", left.ZoneID, right.ZoneID)})
			}
		}
	}

	if kingsfall, found := r.Zones["kingsfall_harbor"]; found {
		if !hasString(kingsfall.Tags, "city") || (!kingsfall.Runtime.HighPlayerDensityExpected && kingsfall.Runtime.ExpectedPlayerDensity == "") || kingsfall.Runtime.HostileDensity == "" {
			validation = append(validation, ValidationError{Code: ValidationInvalidTopology, Path: "zones.kingsfall_harbor.runtime", Message: "Kingsfall Harbor must carry city tags and city-specific runtime hints"})
		}
	}
	return validation
}

func validateZone(zone ZoneDefinition, archetypes map[string]NpcArchetype) ValidationErrors {
	var validation ValidationErrors
	if !zone.Bounds.Valid() {
		validation = append(validation, ValidationError{Code: ValidationInvalidTopology, Path: "zones." + zone.ZoneID + ".bounds", Message: "zone bounds must have max values greater than min values"})
	}
	for index, entry := range zone.EntryPoints {
		if !zone.Bounds.Contains(entry.Position) {
			validation = append(validation, ValidationError{Code: ValidationEntryPointOutOfBounds, Path: fmt.Sprintf("zones.%s.entry_points[%d]", zone.ZoneID, index), Message: fmt.Sprintf("entry point %s is outside zone bounds", entry.EntryPointID)})
		}
	}
	for groupIndex, group := range zone.SpawnGroups {
		if group.ArchetypeID != "" {
			if _, found := archetypes[group.ArchetypeID]; !found && len(archetypes) > 0 {
				validation = append(validation, ValidationError{Code: ValidationInvalidTopology, Path: fmt.Sprintf("zones.%s.spawn_groups[%d].archetype_id", zone.ZoneID, groupIndex), Message: fmt.Sprintf("spawn group references unknown archetype %s", group.ArchetypeID)})
			}
		}
		for spawnIndex, spawn := range group.SpawnPoints {
			if !zone.Bounds.Contains(spawn.Position) {
				validation = append(validation, ValidationError{Code: ValidationSpawnPointOutOfBounds, Path: fmt.Sprintf("zones.%s.spawn_groups[%d].spawn_points[%d]", zone.ZoneID, groupIndex, spawnIndex), Message: fmt.Sprintf("spawn point %s is outside zone bounds", spawn.SpawnPointID)})
			}
		}
	}
	for index, provider := range zone.QuestProviders {
		if !zone.Bounds.Contains(provider.Position) {
			validation = append(validation, ValidationError{Code: ValidationSpawnPointOutOfBounds, Path: fmt.Sprintf("zones.%s.quest_providers[%d]", zone.ZoneID, index), Message: fmt.Sprintf("quest provider %s is outside zone bounds", provider.ProviderID)})
		}
	}
	return validation
}

func normalizeZoneDefinition(zone *ZoneDefinition) {
	if zone.Runtime.ActivationPolicy == "" && zone.RuntimeConfig.ActivationPolicy != "" {
		zone.Runtime = zone.RuntimeConfig
	}
	if zone.Streaming.InterestRadius == 0 && zone.StreamingHints.InterestRadius != 0 {
		zone.Streaming = zone.StreamingHints
	}
	for index := range zone.EntryPoints {
		if zone.EntryPoints[index].Position.ZoneID == "" {
			zone.EntryPoints[index].Position.ZoneID = zone.ZoneID
		}
		if zone.EntryPoints[index].Facing == 0 && zone.EntryPoints[index].FacingYaw != 0 {
			zone.EntryPoints[index].Facing = zone.EntryPoints[index].FacingYaw
		}
	}
	for index := range zone.TransitionGates {
		if zone.TransitionGates[index].FromZoneID == "" {
			zone.TransitionGates[index].FromZoneID = zone.ZoneID
		}
	}
	for groupIndex := range zone.SpawnGroups {
		for spawnIndex := range zone.SpawnGroups[groupIndex].SpawnPoints {
			spawn := &zone.SpawnGroups[groupIndex].SpawnPoints[spawnIndex]
			if spawn.Position.ZoneID == "" {
				spawn.Position.ZoneID = zone.ZoneID
			}
			if spawn.ArchetypeID == "" {
				spawn.ArchetypeID = zone.SpawnGroups[groupIndex].ArchetypeID
			}
		}
	}
	for index := range zone.QuestProviders {
		if zone.QuestProviders[index].Position.ZoneID == "" {
			zone.QuestProviders[index].Position.ZoneID = zone.ZoneID
		}
	}
}

func loadNPCContent(registry *RuntimeContentRegistry, path string) error {
	loaded, err := readJSONFile[npcContentFile](path)
	if err != nil {
		return fmt.Errorf("load npcs %s: %w", path, err)
	}
	archetypes := loaded.NpcArchetypes
	if len(archetypes) == 0 {
		archetypes = loaded.Archetypes
	}
	for _, archetype := range archetypes {
		if archetype.ArchetypeID == "" {
			return ValidationErrors{{Code: ValidationInvalidTopology, Path: "npcs.npc_archetypes", Message: "archetype_id is required"}}
		}
		registry.NpcArchetypes[archetype.ArchetypeID] = archetype
	}
	for _, spawn := range loaded.SpawnPoints {
		if spawn.SpawnPointID == "" {
			return ValidationErrors{{Code: ValidationInvalidTopology, Path: "npcs.spawn_points", Message: "spawn_point_id is required"}}
		}
		registry.NpcSpawns[spawn.SpawnPointID] = spawn
	}
	return nil
}

func loadStubContent(path string, field string, target map[string]ContentStubDefinition) error {
	loaded, err := readJSONFile[stubContentFile](path)
	if err != nil {
		return fmt.Errorf("load %s %s: %w", field, path, err)
	}
	var source []ContentStubDefinition
	switch field {
	case "items":
		source = loaded.Items
	case "loot":
		source = loaded.Loot
	case "quests":
		source = loaded.Quests
	case "abilities":
		source = loaded.Abilities
	case "auras":
		source = loaded.Auras
	}
	for _, stub := range source {
		if stub.ID == "" {
			switch field {
			case "items":
				stub.ID = stub.ItemID
			case "loot":
				stub.ID = stub.LootTableID
			case "quests":
				stub.ID = stub.QuestID
			case "abilities":
				stub.ID = stub.AbilityID
			case "auras":
				stub.ID = stub.AuraID
			}
		}
		if stub.ID == "" {
			return ValidationErrors{{Code: ValidationInvalidTopology, Path: field, Message: "id is required"}}
		}
		target[stub.ID] = stub
	}
	return nil
}

func readJSONFile[T any](path string) (T, error) {
	var target T
	content, err := os.ReadFile(path)
	if err != nil {
		return target, err
	}
	if err := json.Unmarshal(content, &target); err != nil {
		return target, err
	}
	return target, nil
}

func (z ZoneDefinition) entryPoint(entryPointID string) (ZoneEntryPoint, bool) {
	for _, entry := range z.EntryPoints {
		if entry.EntryPointID == entryPointID {
			return entry, true
		}
	}
	return ZoneEntryPoint{}, false
}

func (z ZoneDefinition) transition(transitionID string) (ZoneTransitionGate, bool) {
	for _, transition := range z.TransitionGates {
		if transition.TransitionID == transitionID {
			return transition, true
		}
	}
	return ZoneTransitionGate{}, false
}

func (b ZoneBounds) Valid() bool {
	return b.MaxX > b.MinX && b.MaxY > b.MinY && b.MaxZ >= b.MinZ
}

func (b ZoneBounds) Contains(position WorldPosition) bool {
	return position.X >= b.MinX && position.X <= b.MaxX &&
		position.Y >= b.MinY && position.Y <= b.MaxY &&
		position.Z >= b.MinZ && position.Z <= b.MaxZ
}

func (b ZoneBounds) ContainsBounds(inner ZoneBounds) bool {
	return inner.MinX >= b.MinX && inner.MaxX <= b.MaxX &&
		inner.MinY >= b.MinY && inner.MaxY <= b.MaxY &&
		inner.MinZ >= b.MinZ && inner.MaxZ <= b.MaxZ
}

func (b ZoneBounds) Overlaps(other ZoneBounds) bool {
	return b.MinX < other.MaxX && b.MaxX > other.MinX &&
		b.MinY < other.MaxY && b.MaxY > other.MinY &&
		b.MinZ < other.MaxZ && b.MaxZ > other.MinZ
}

type ZoneActivationPolicy string

const (
	ZoneActivationPolicyEager    ZoneActivationPolicy = "Eager"
	ZoneActivationPolicyOnDemand ZoneActivationPolicy = "OnDemand"
)

type ZoneRuntimeFactory struct{}

type ZoneRuntime struct {
	ZoneID     string
	Definition ZoneDefinition
	Entities   EntityRegistry
	Characters map[string]*CharacterZoneState
	CommandQ   ZoneCommandQueue
	ShardID    ShardID
	Active     bool
}

type EntityRegistry struct {
	Entities  map[string]RuntimeEntity
	ZoneIndex map[string]string
}

type RuntimeEntity struct {
	EntityID    string        `json:"entity_id"`
	DisplayName string        `json:"display_name"`
	Kind        string        `json:"kind"`
	ZoneID      string        `json:"zone_id"`
	Position    WorldPosition `json:"position"`
	Tags        []string      `json:"tags,omitempty"`
}

type CharacterZoneState struct {
	CharacterID string
	ZoneID      string
	Position    WorldPosition
	Facing      float64
	Connected   bool
}

type SimulationTick struct {
	TickID     int64
	StartedAt  time.Time
	Duration   time.Duration
	QueueDepth int
}

type ZoneCommandQueue struct {
	Capacity           int
	Pending            []WorldCommand
	TotalEnqueued      int
	TotalDequeued      int
	TotalBackpressured int
	MaxDepth           int
}

type CommandQueueResult struct {
	Accepted      bool
	Backpressured bool
	Reason        string
	ZoneID        string
	CommandID     string
	CommandName   string
	QueueDepth    int
	MaxQueueDepth int
}

type CommandQueueStats struct {
	ZoneID             string
	Capacity           int
	Depth              int
	MaxDepth           int
	TotalEnqueued      int
	TotalDequeued      int
	TotalBackpressured int
}

type ShardID string

type ShardAssignmentPolicy struct {
	Prefix           string
	MaxZonesPerShard int
	ZoneOrder        []string
}

type ShardAssignment struct {
	ShardID string
	ZoneID  string
	Reason  string
}

type ZoneShardHandle struct {
	ShardID string
	ZoneID  string
	Runtime *ZoneRuntime
}

type ShardRuntimeIndex struct {
	ZoneToShard  map[string]string
	ShardToZones map[string][]string
}

func (f ZoneRuntimeFactory) CreateZoneRuntime(zone ZoneDefinition) *ZoneRuntime {
	runtime := &ZoneRuntime{
		ZoneID:     zone.ZoneID,
		Definition: zone,
		Entities: EntityRegistry{
			Entities:  map[string]RuntimeEntity{},
			ZoneIndex: map[string]string{},
		},
		Characters: map[string]*CharacterZoneState{},
		CommandQ:   NewZoneCommandQueue(0),
		Active:     true,
	}
	for _, group := range zone.SpawnGroups {
		for _, spawn := range group.SpawnPoints {
			runtime.Entities.Add(RuntimeEntity{
				EntityID:    spawn.SpawnPointID,
				DisplayName: spawn.ArchetypeID,
				Kind:        hostileMobKind,
				ZoneID:      zone.ZoneID,
				Position:    spawn.Position,
				Tags:        append([]string(nil), spawn.Tags...),
			})
		}
	}
	for _, provider := range zone.QuestProviders {
		runtime.Entities.Add(RuntimeEntity{
			EntityID:    provider.ProviderID,
			DisplayName: provider.DisplayName,
			Kind:        questGiverNPCKind,
			ZoneID:      zone.ZoneID,
			Position:    provider.Position,
			Tags:        append([]string(nil), provider.Tags...),
		})
	}
	observability.LogEvent("world-service", observability.EventWorldZoneRuntimeCreated, map[string]any{
		"zoneId":      zone.ZoneID,
		"displayName": zone.DisplayName,
		"entityCount": len(runtime.Entities.Entities),
	})
	return runtime
}

func (r *EntityRegistry) Add(entity RuntimeEntity) {
	if r.Entities == nil {
		r.Entities = map[string]RuntimeEntity{}
	}
	if r.ZoneIndex == nil {
		r.ZoneIndex = map[string]string{}
	}
	r.Entities[entity.EntityID] = entity
	r.ZoneIndex[entity.EntityID] = entity.ZoneID
}

func (r *EntityRegistry) Remove(entityID string) {
	delete(r.Entities, entityID)
	delete(r.ZoneIndex, entityID)
}

func NewZoneCommandQueue(capacity int) ZoneCommandQueue {
	if capacity < 0 {
		capacity = 0
	}
	return ZoneCommandQueue{Capacity: capacity, Pending: []WorldCommand{}}
}

func (z *ZoneRuntime) ConfigureCommandQueue(capacity int) {
	z.CommandQ = NewZoneCommandQueue(capacity)
	observability.LogEvent("world-service", observability.EventWorldZoneQueueDepthSampled, map[string]any{
		"zoneId":       z.ZoneID,
		"queueDepth":   0,
		"queueMax":     0,
		"queueCap":     z.CommandQ.Capacity,
		"sampleReason": "configured",
	})
}

func (z *ZoneRuntime) EnqueueCommand(command WorldCommand) CommandQueueResult {
	if z.CommandQ.Pending == nil {
		z.CommandQ.Pending = []WorldCommand{}
	}
	depth := len(z.CommandQ.Pending)
	if z.CommandQ.Capacity > 0 && depth >= z.CommandQ.Capacity {
		z.CommandQ.TotalBackpressured++
		result := CommandQueueResult{
			Accepted:      false,
			Backpressured: true,
			Reason:        "zone command queue capacity reached",
			ZoneID:        z.ZoneID,
			CommandID:     command.CommandID,
			CommandName:   command.Name,
			QueueDepth:    depth,
			MaxQueueDepth: z.CommandQ.MaxDepth,
		}
		observability.LogEvent("world-service", observability.EventWorldZoneCommandBackpressured, result.eventFields())
		return result
	}
	z.CommandQ.Pending = append(z.CommandQ.Pending, command)
	z.CommandQ.TotalEnqueued++
	depth = len(z.CommandQ.Pending)
	if depth > z.CommandQ.MaxDepth {
		z.CommandQ.MaxDepth = depth
	}
	result := CommandQueueResult{
		Accepted:      true,
		ZoneID:        z.ZoneID,
		CommandID:     command.CommandID,
		CommandName:   command.Name,
		QueueDepth:    depth,
		MaxQueueDepth: z.CommandQ.MaxDepth,
	}
	observability.LogEvent("world-service", observability.EventWorldZoneCommandEnqueued, result.eventFields())
	observability.LogEvent("world-service", observability.EventWorldZoneQueueDepthSampled, map[string]any{
		"zoneId":       z.ZoneID,
		"queueDepth":   depth,
		"queueMax":     z.CommandQ.MaxDepth,
		"queueCap":     z.CommandQ.Capacity,
		"sampleReason": "enqueue",
	})
	return result
}

func (z *ZoneRuntime) DequeueCommand() (WorldCommand, bool) {
	if len(z.CommandQ.Pending) == 0 {
		return WorldCommand{}, false
	}
	command := z.CommandQ.Pending[0]
	copy(z.CommandQ.Pending, z.CommandQ.Pending[1:])
	z.CommandQ.Pending = z.CommandQ.Pending[:len(z.CommandQ.Pending)-1]
	z.CommandQ.TotalDequeued++
	observability.LogEvent("world-service", observability.EventWorldZoneCommandDequeued, map[string]any{
		"zoneId":      z.ZoneID,
		"commandId":   command.CommandID,
		"commandName": command.Name,
		"queueDepth":  len(z.CommandQ.Pending),
	})
	observability.LogEvent("world-service", observability.EventWorldZoneQueueDepthSampled, map[string]any{
		"zoneId":       z.ZoneID,
		"queueDepth":   len(z.CommandQ.Pending),
		"queueMax":     z.CommandQ.MaxDepth,
		"queueCap":     z.CommandQ.Capacity,
		"sampleReason": "dequeue",
	})
	return command, true
}

func (z *ZoneRuntime) DrainCommandQueue(limit int) []WorldCommand {
	if limit <= 0 || limit > len(z.CommandQ.Pending) {
		limit = len(z.CommandQ.Pending)
	}
	commands := make([]WorldCommand, 0, limit)
	for len(commands) < limit {
		command, ok := z.DequeueCommand()
		if !ok {
			break
		}
		commands = append(commands, command)
	}
	return commands
}

func (z *ZoneRuntime) QueueStats() CommandQueueStats {
	return CommandQueueStats{
		ZoneID:             z.ZoneID,
		Capacity:           z.CommandQ.Capacity,
		Depth:              len(z.CommandQ.Pending),
		MaxDepth:           z.CommandQ.MaxDepth,
		TotalEnqueued:      z.CommandQ.TotalEnqueued,
		TotalDequeued:      z.CommandQ.TotalDequeued,
		TotalBackpressured: z.CommandQ.TotalBackpressured,
	}
}

func (r CommandQueueResult) eventFields() map[string]any {
	return map[string]any{
		"zoneId":        r.ZoneID,
		"commandId":     r.CommandID,
		"commandName":   r.CommandName,
		"accepted":      r.Accepted,
		"backpressured": r.Backpressured,
		"reason":        r.Reason,
		"queueDepth":    r.QueueDepth,
		"maxQueueDepth": r.MaxQueueDepth,
	}
}

type ContinentRuntime struct {
	Registry        *RuntimeContentRegistry
	Definition      ContinentDefinition
	Zones           map[string]*ZoneRuntime
	EntityZoneIndex map[string]string
	Characters      map[string]*CharacterZoneState
	Visibility      map[string]VisibilitySet
	Shards          *ShardRuntimeIndex
	Events          []DomainEvent
}

type ZoneRuntimeHandle struct {
	ZoneID  string
	Runtime *ZoneRuntime
}

type WorldRuntime struct {
	Registry            *RuntimeContentRegistry
	Continents          map[string]*ContinentRuntime
	CharacterContinents map[string]string
}

func (r *RuntimeContentRegistry) ActivateContinent(continentID string) (*ContinentRuntime, error) {
	continent, found := r.Continents[continentID]
	if !found {
		observability.LogEvent("world-service", observability.EventWorldContinentActivationFailed, map[string]any{
			"continentId": continentID,
			"reason":      "continent not loaded",
		})
		return nil, fmt.Errorf("continent %s is not loaded", continentID)
	}
	if err := r.validateContinent(continent); err != nil {
		observability.LogEvent("world-service", observability.EventWorldContinentActivationFailed, map[string]any{
			"continentId": continentID,
			"reason":      err.Error(),
		})
		return nil, err
	}
	runtime := &ContinentRuntime{
		Registry:        r,
		Definition:      continent,
		Zones:           map[string]*ZoneRuntime{},
		EntityZoneIndex: map[string]string{},
		Characters:      map[string]*CharacterZoneState{},
		Visibility:      map[string]VisibilitySet{},
	}
	factory := ZoneRuntimeFactory{}
	for _, zoneID := range continent.Zones {
		zoneRuntime := factory.CreateZoneRuntime(r.Zones[zoneID])
		runtime.Zones[zoneID] = zoneRuntime
		for entityID, entity := range zoneRuntime.Entities.Entities {
			runtime.EntityZoneIndex[entityID] = entity.ZoneID
		}
	}
	observability.LogEvent("world-service", observability.EventWorldContinentActivated, map[string]any{
		"continentId": continentID,
		"zoneCount":   len(runtime.Zones),
	})
	if _, err := runtime.AssignZonesToShards(ShardAssignmentPolicy{Prefix: continentID + "-shard"}); err != nil {
		observability.LogEvent("world-service", observability.EventWorldShardAssignmentFailed, map[string]any{
			"continentId": continentID,
			"reason":      err.Error(),
		})
		return nil, err
	}
	return runtime, nil
}

func NewWorldRuntime(registry *RuntimeContentRegistry) *WorldRuntime {
	return &WorldRuntime{
		Registry:            registry,
		Continents:          map[string]*ContinentRuntime{},
		CharacterContinents: map[string]string{},
	}
}

func (w *WorldRuntime) ActivateAllContinents() error {
	for continentID := range w.Registry.Continents {
		runtime, err := w.Registry.ActivateContinent(continentID)
		if err != nil {
			return err
		}
		w.Continents[continentID] = runtime
	}
	return nil
}

func (w *WorldRuntime) RouteCommand(command WorldCommand) (ZoneRuntimeHandle, error) {
	continentID := w.CharacterContinents[command.CharacterID]
	if continentID == "" {
		for id, continent := range w.Continents {
			if _, found := continent.Characters[command.CharacterID]; found {
				continentID = id
				w.CharacterContinents[command.CharacterID] = id
				break
			}
		}
	}
	if continentID == "" {
		return ZoneRuntimeHandle{}, fmt.Errorf("character %s is not assigned to a continent runtime", command.CharacterID)
	}
	return w.Continents[continentID].RouteCommand(command)
}

func (w *WorldRuntime) RouteCommandToQueue(command WorldCommand) (CommandQueueResult, error) {
	handle, err := w.RouteCommand(command)
	if err != nil {
		return CommandQueueResult{}, err
	}
	return handle.Runtime.EnqueueCommand(command), nil
}

type WorldCommand struct {
	CommandID   string
	CharacterID string
	Name        string
	Payload     map[string]any
}

func (r *ContinentRuntime) SpawnCharacterAtDefaultEntry(characterID string) (CharacterZoneState, []StateDiff, error) {
	zone, entry, err := r.defaultEntry()
	if err != nil {
		return CharacterZoneState{}, nil, err
	}
	return r.SpawnCharacterAtEntry(characterID, zone.ZoneID, entry.EntryPointID)
}

func (r *ContinentRuntime) SpawnCharacterAtEntry(characterID string, zoneID string, entryPointID string) (CharacterZoneState, []StateDiff, error) {
	zone, found := r.Registry.Zones[zoneID]
	if !found {
		return CharacterZoneState{}, nil, fmt.Errorf("zone %s is not loaded", zoneID)
	}
	entry, found := zone.entryPoint(entryPointID)
	if !found {
		return CharacterZoneState{}, nil, fmt.Errorf("entry point %s is not loaded in zone %s", entryPointID, zoneID)
	}
	state := CharacterZoneState{
		CharacterID: characterID,
		ZoneID:      zone.ZoneID,
		Position:    entry.Position,
		Facing:      entry.Facing,
		Connected:   true,
	}
	return state, r.setCharacterZoneState(state, "", true), nil
}

func (r *ContinentRuntime) PlaceCharacterAtEntry(characterID string, zoneID string, entryPointID string) (CharacterZoneState, []StateDiff, error) {
	zone, found := r.Registry.Zones[zoneID]
	if !found {
		return CharacterZoneState{}, nil, fmt.Errorf("zone %s is not loaded", zoneID)
	}
	entry, found := zone.entryPoint(entryPointID)
	if !found {
		return CharacterZoneState{}, nil, fmt.Errorf("entry point %s is not loaded in zone %s", entryPointID, zoneID)
	}
	previousZoneID := ""
	if existing, found := r.Characters[characterID]; found {
		previousZoneID = existing.ZoneID
	}
	state := CharacterZoneState{
		CharacterID: characterID,
		ZoneID:      zone.ZoneID,
		Position:    entry.Position,
		Facing:      entry.Facing,
		Connected:   true,
	}
	return state, r.setCharacterZoneState(state, previousZoneID, true), nil
}

func (r *ContinentRuntime) RouteCommand(command WorldCommand) (ZoneRuntimeHandle, error) {
	state, found := r.Characters[command.CharacterID]
	if !found {
		return ZoneRuntimeHandle{}, fmt.Errorf("character %s is not assigned to a zone runtime", command.CharacterID)
	}
	runtime, found := r.Zones[state.ZoneID]
	if !found {
		return ZoneRuntimeHandle{}, fmt.Errorf("character %s is assigned to missing zone %s", command.CharacterID, state.ZoneID)
	}
	return ZoneRuntimeHandle{ZoneID: state.ZoneID, Runtime: runtime}, nil
}

func (r *ContinentRuntime) RouteCommandToQueue(command WorldCommand) (CommandQueueResult, error) {
	handle, err := r.RouteCommand(command)
	if err != nil {
		return CommandQueueResult{}, err
	}
	return handle.Runtime.EnqueueCommand(command), nil
}

func (r *ContinentRuntime) MoveCharacter(characterID string, deltaX float64, deltaY float64, deltaZ float64) (ZoneTransferResult, error) {
	state, found := r.Characters[characterID]
	if !found {
		return ZoneTransferResult{}, fmt.Errorf("character %s is not active", characterID)
	}
	zone := r.Registry.Zones[state.ZoneID]
	next := state.Position
	next.X += deltaX
	next.Y += deltaY
	next.Z += deltaZ
	if zone.Bounds.Contains(next) {
		state.Position = next
		if zoneRuntime := r.Zones[state.ZoneID]; zoneRuntime != nil {
			entity := zoneRuntime.Entities.Entities[playerEntityID(characterID)]
			entity.Position = next
			zoneRuntime.Entities.Entities[playerEntityID(characterID)] = entity
		}
		return ZoneTransferResult{CharacterID: characterID, FromZoneID: zone.ZoneID, ToZoneID: zone.ZoneID}, nil
	}
	for _, gate := range zone.TransitionGates {
		if gate.GateBounds.Contains(state.Position) || gate.GateBounds.Contains(next) {
			return r.RequestZoneTransfer(ZoneTransferRequest{
				CharacterID:  characterID,
				FromZoneID:   zone.ZoneID,
				ToZoneID:     gate.ToZoneID,
				TransitionID: gate.TransitionID,
				Position:     state.Position,
			})
		}
	}
	result := ZoneTransferResult{
		CharacterID:     characterID,
		FromZoneID:      zone.ZoneID,
		Rejected:        true,
		RejectionReason: "movement left zone bounds without an enabled transition gate",
		Events: []DomainEvent{newDomainEvent(observability.EventWorldZoneTransitionRejected, characterID, "", zone.ZoneID, map[string]any{
			"reason": "movement left zone bounds without an enabled transition gate",
		})},
	}
	observability.LogEvent("world-service", observability.EventWorldZoneTransitionRejected, result.eventFields())
	return result, nil
}

type ZoneTransferRequest struct {
	CharacterID  string
	FromZoneID   string
	ToZoneID     string
	TransitionID string
	Position     WorldPosition
}

type ZoneTransferResult struct {
	CharacterID     string
	FromZoneID      string
	ToZoneID        string
	TransitionID    string
	Requested       bool
	Completed       bool
	Rejected        bool
	RejectionReason string
	Diffs           []StateDiff
	Events          []DomainEvent
}

func (r *ContinentRuntime) RequestZoneTransfer(request ZoneTransferRequest) (ZoneTransferResult, error) {
	result := ZoneTransferResult{
		CharacterID:  request.CharacterID,
		FromZoneID:   request.FromZoneID,
		ToZoneID:     request.ToZoneID,
		TransitionID: request.TransitionID,
		Requested:    true,
	}
	observability.LogEvent("world-service", observability.EventWorldZoneTransitionRequested, result.eventFields())
	result.Events = append(result.Events, newDomainEvent(observability.EventWorldZoneTransitionRequested, request.CharacterID, "", request.FromZoneID, result.eventFields()))

	reject := func(reason string) (ZoneTransferResult, error) {
		result.Rejected = true
		result.RejectionReason = reason
		result.Events = append(result.Events, newDomainEvent(observability.EventWorldZoneTransitionRejected, request.CharacterID, "", request.FromZoneID, map[string]any{
			"transitionId": request.TransitionID,
			"toZoneId":     request.ToZoneID,
			"reason":       reason,
		}))
		observability.LogEvent("world-service", observability.EventWorldZoneTransitionRejected, result.eventFields())
		return result, nil
	}

	sourceRuntime, sourceFound := r.Zones[request.FromZoneID]
	if !sourceFound {
		return reject("source zone does not exist")
	}
	destinationRuntime, destinationFound := r.Zones[request.ToZoneID]
	if !destinationFound {
		return reject("destination zone does not exist")
	}
	state, characterFound := r.Characters[request.CharacterID]
	if !characterFound {
		return reject("character is not active")
	}
	if state.ZoneID != request.FromZoneID {
		return reject("character is not owned by source zone")
	}
	if r.Shards != nil && len(r.Shards.ZoneToShard) > 0 {
		if _, assigned := r.Shards.ZoneToShard[request.FromZoneID]; !assigned {
			return reject("source zone is not bound to a shard")
		}
		if _, assigned := r.Shards.ZoneToShard[request.ToZoneID]; !assigned {
			return reject("destination zone is not bound to a shard")
		}
	}
	gate, gateFound := sourceRuntime.Definition.transition(request.TransitionID)
	if !gateFound {
		return reject("transition does not exist")
	}
	if gate.Disabled {
		return reject("transition is disabled")
	}
	if gate.ToZoneID != request.ToZoneID {
		return reject("transition destination mismatch")
	}
	if !gate.GateBounds.Contains(state.Position) && !gate.GateBounds.Contains(request.Position) {
		return reject("character is not inside transition gate")
	}
	entry, entryFound := destinationRuntime.Definition.entryPoint(gate.EntryPointIDOnArrival)
	if !entryFound {
		return reject("destination entry point does not exist")
	}

	previousZoneID := state.ZoneID
	state.ZoneID = request.ToZoneID
	state.Position = entry.Position
	state.Facing = entry.Facing
	sourceRuntime.Entities.Remove(playerEntityID(request.CharacterID))
	delete(sourceRuntime.Characters, request.CharacterID)
	destinationRuntime.Characters[request.CharacterID] = state
	destinationRuntime.Entities.Add(RuntimeEntity{
		EntityID:    playerEntityID(request.CharacterID),
		DisplayName: request.CharacterID,
		Kind:        "player",
		ZoneID:      request.ToZoneID,
		Position:    state.Position,
	})
	r.EntityZoneIndex[playerEntityID(request.CharacterID)] = request.ToZoneID
	delete(r.Visibility, request.CharacterID)

	result.Completed = true
	result.Diffs = append(result.Diffs,
		newStateDiff(observability.EventWorldZoneExited, request.CharacterID, "", previousZoneID, nil),
		newStateDiff(observability.EventWorldZoneEntered, request.CharacterID, "", request.ToZoneID, map[string]any{"x": state.Position.X, "y": state.Position.Y, "z": state.Position.Z}),
		newStateDiff(observability.EventWorldZoneRoutingUpdated, request.CharacterID, "", request.ToZoneID, nil),
	)
	result.Events = append(result.Events,
		newDomainEvent(observability.EventWorldZoneExited, request.CharacterID, "", previousZoneID, map[string]any{"toZoneId": request.ToZoneID}),
		newDomainEvent(observability.EventWorldZoneEntered, request.CharacterID, "", request.ToZoneID, map[string]any{"fromZoneId": previousZoneID}),
		newDomainEvent(observability.EventWorldZoneTransitionCompleted, request.CharacterID, "", request.ToZoneID, result.eventFields()),
		newDomainEvent(observability.EventWorldZoneRoutingUpdated, request.CharacterID, "", request.ToZoneID, map[string]any{"fromZoneId": previousZoneID}),
	)
	r.Events = append(r.Events, result.Events...)
	observability.LogEvent("world-service", observability.EventWorldZoneTransitionCompleted, result.eventFields())
	observability.LogEvent("world-service", observability.EventWorldZoneRoutingUpdated, map[string]any{
		"characterId": request.CharacterID,
		"fromZoneId":  previousZoneID,
		"toZoneId":    request.ToZoneID,
	})
	return result, nil
}

func (r ZoneTransferResult) eventFields() map[string]any {
	return map[string]any{
		"characterId":     r.CharacterID,
		"fromZoneId":      r.FromZoneID,
		"toZoneId":        r.ToZoneID,
		"transitionId":    r.TransitionID,
		"completed":       r.Completed,
		"rejected":        r.Rejected,
		"rejectionReason": r.RejectionReason,
	}
}

type InterestProfile struct {
	Radius                        float64
	IncludeAdjacentStreamingHints bool
}

type VisibilityQuery struct {
	CharacterID string
	ZoneID      string
	Position    WorldPosition
	Radius      float64
}

type VisibilitySet struct {
	EntityIDs map[string]bool
}

type VisibilityDelta struct {
	CharacterID    string
	ZoneID         string
	Entered        []RuntimeEntity
	Exited         []RuntimeEntity
	StreamingHints []StreamingRegionHint
}

type StreamingRegionHint struct {
	TransitionID string `json:"transition_id"`
	FromZoneID   string `json:"from_zone_id"`
	ToZoneID     string `json:"to_zone_id"`
	Kind         string `json:"kind"`
	Reason       string `json:"reason"`
}

type ZoneInterestBoundary struct {
	ZoneID string
	Bounds ZoneBounds
}

type NearbyEntityRadius float64

func (r *ContinentRuntime) EvaluateVisibility(characterID string, profile InterestProfile) (VisibilityDelta, error) {
	state, found := r.Characters[characterID]
	if !found {
		return VisibilityDelta{}, fmt.Errorf("character %s is not active", characterID)
	}
	radius := profile.Radius
	if radius <= 0 {
		radius = r.Definition.Streaming.DefaultInterestRadius
	}
	if radius <= 0 {
		radius = 48
	}
	query := VisibilityQuery{
		CharacterID: characterID,
		ZoneID:      state.ZoneID,
		Position:    state.Position,
		Radius:      radius,
	}
	current := r.visibilitySet(query)
	previous := r.Visibility[characterID]
	delta := VisibilityDelta{CharacterID: characterID, ZoneID: state.ZoneID}
	zoneRuntime := r.Zones[state.ZoneID]

	for entityID := range current.EntityIDs {
		if previous.EntityIDs == nil || !previous.EntityIDs[entityID] {
			entity := zoneRuntime.Entities.Entities[entityID]
			delta.Entered = append(delta.Entered, entity)
			observability.LogEvent("world-service", observability.EventWorldVisibilityEntityEntered, map[string]any{
				"characterId": characterID,
				"entityId":    entityID,
				"zoneId":      state.ZoneID,
			})
		}
	}
	for entityID := range previous.EntityIDs {
		if !current.EntityIDs[entityID] {
			entity := zoneRuntime.Entities.Entities[entityID]
			delta.Exited = append(delta.Exited, entity)
			observability.LogEvent("world-service", observability.EventWorldVisibilityEntityExited, map[string]any{
				"characterId": characterID,
				"entityId":    entityID,
				"zoneId":      state.ZoneID,
			})
		}
	}
	if profile.IncludeAdjacentStreamingHints {
		delta.StreamingHints = r.streamingHintsForPosition(state.ZoneID, state.Position)
		for _, hint := range delta.StreamingHints {
			observability.LogEvent("world-service", observability.EventWorldStreamingHintEmitted, map[string]any{
				"characterId":  characterID,
				"transitionId": hint.TransitionID,
				"fromZoneId":   hint.FromZoneID,
				"toZoneId":     hint.ToZoneID,
			})
		}
	}
	r.Visibility[characterID] = current
	observability.LogEvent("world-service", observability.EventWorldVisibilityEvaluated, map[string]any{
		"characterId": characterID,
		"zoneId":      state.ZoneID,
		"visible":     len(current.EntityIDs),
		"entered":     len(delta.Entered),
		"exited":      len(delta.Exited),
	})
	return delta, nil
}

func (r *ContinentRuntime) visibilitySet(query VisibilityQuery) VisibilitySet {
	visible := VisibilitySet{EntityIDs: map[string]bool{}}
	zoneRuntime := r.Zones[query.ZoneID]
	if zoneRuntime == nil {
		return visible
	}
	for entityID, entity := range zoneRuntime.Entities.Entities {
		if entityID == playerEntityID(query.CharacterID) {
			continue
		}
		if distance2D(query.Position.X, query.Position.Y, entity.Position.X, entity.Position.Y) <= query.Radius {
			visible.EntityIDs[entityID] = true
		}
	}
	return visible
}

func (r *ContinentRuntime) streamingHintsForPosition(zoneID string, position WorldPosition) []StreamingRegionHint {
	zoneRuntime := r.Zones[zoneID]
	if zoneRuntime == nil {
		return nil
	}
	hintRadius := r.Definition.Streaming.TransitionHintRadius
	if hintRadius <= 0 {
		hintRadius = r.Definition.Streaming.GateHintRadius
	}
	if zoneRuntime.Definition.Streaming.AdjacentPreloadDistance > 0 {
		hintRadius = zoneRuntime.Definition.Streaming.AdjacentPreloadDistance
	}
	if hintRadius <= 0 {
		hintRadius = 20
	}
	hints := []StreamingRegionHint{}
	for _, gate := range zoneRuntime.Definition.TransitionGates {
		if gate.Disabled {
			continue
		}
		if gate.GateBounds.Contains(position) || distanceToBounds2D(position, gate.GateBounds) <= hintRadius {
			hints = append(hints, StreamingRegionHint{
				TransitionID: gate.TransitionID,
				FromZoneID:   gate.FromZoneID,
				ToZoneID:     gate.ToZoneID,
				Kind:         gate.Kind,
				Reason:       "near_transition_gate",
			})
		}
	}
	return hints
}

type CharacterZoneStore interface {
	SaveCharacterZoneState(state CharacterZoneState) error
	LoadCharacterZoneState(characterID string) (CharacterZoneState, bool, error)
}

type MemoryCharacterZoneStore struct {
	states map[string]CharacterZoneState
}

func NewMemoryCharacterZoneStore() *MemoryCharacterZoneStore {
	return &MemoryCharacterZoneStore{states: map[string]CharacterZoneState{}}
}

func (s *MemoryCharacterZoneStore) SaveCharacterZoneState(state CharacterZoneState) error {
	if s.states == nil {
		s.states = map[string]CharacterZoneState{}
	}
	s.states[state.CharacterID] = state
	return nil
}

func (s *MemoryCharacterZoneStore) LoadCharacterZoneState(characterID string) (CharacterZoneState, bool, error) {
	state, found := s.states[characterID]
	return state, found, nil
}

func (r *ContinentRuntime) SaveCharacterZoneState(store CharacterZoneStore, characterID string) error {
	state, found := r.Characters[characterID]
	if !found {
		return fmt.Errorf("character %s is not active", characterID)
	}
	if err := store.SaveCharacterZoneState(*state); err != nil {
		return err
	}
	observability.LogEvent("world-service", observability.EventWorldCharacterZoneSaved, map[string]any{
		"characterId": characterID,
		"zoneId":      state.ZoneID,
		"x":           state.Position.X,
		"y":           state.Position.Y,
		"z":           state.Position.Z,
	})
	return nil
}

func (r *ContinentRuntime) RestoreCharacterZoneState(store CharacterZoneStore, characterID string) (CharacterZoneState, []StateDiff, error) {
	state, found, err := store.LoadCharacterZoneState(characterID)
	if err != nil {
		return CharacterZoneState{}, nil, err
	}
	if !found {
		return r.SpawnCharacterAtDefaultEntry(characterID)
	}
	zoneRuntime, zoneFound := r.Zones[state.ZoneID]
	if !zoneFound || !zoneRuntime.Definition.Bounds.Contains(state.Position) {
		_, entry, defaultErr := r.defaultEntry()
		if defaultErr != nil {
			return CharacterZoneState{}, nil, defaultErr
		}
		originalZoneID := state.ZoneID
		state.ZoneID = entry.Position.ZoneID
		state.Position = entry.Position
		state.Connected = true
		diffs := r.setCharacterZoneState(state, originalZoneID, true)
		diffs = append(diffs, newStateDiff(observability.EventWorldCharacterZoneRestoreCorrected, characterID, "", state.ZoneID, map[string]any{"fromZoneId": originalZoneID}))
		observability.LogEvent("world-service", observability.EventWorldCharacterZoneRestoreCorrected, map[string]any{
			"characterId": characterID,
			"fromZoneId":  originalZoneID,
			"toZoneId":    state.ZoneID,
		})
		return state, diffs, nil
	}
	state.Connected = true
	diffs := r.setCharacterZoneState(state, "", true)
	diffs = append(diffs, newStateDiff(observability.EventWorldCharacterZoneRestored, characterID, "", state.ZoneID, nil))
	observability.LogEvent("world-service", observability.EventWorldCharacterZoneRestored, map[string]any{
		"characterId": characterID,
		"zoneId":      state.ZoneID,
	})
	return state, diffs, nil
}

func (r *ContinentRuntime) setCharacterZoneState(state CharacterZoneState, previousZoneID string, emitEnter bool) []StateDiff {
	state.Position.ZoneID = state.ZoneID
	current := state
	r.Characters[state.CharacterID] = &current
	for zoneID, zoneRuntime := range r.Zones {
		if zoneID != state.ZoneID {
			delete(zoneRuntime.Characters, state.CharacterID)
			zoneRuntime.Entities.Remove(playerEntityID(state.CharacterID))
		}
	}
	zoneRuntime := r.Zones[state.ZoneID]
	zoneRuntime.Characters[state.CharacterID] = &current
	zoneRuntime.Entities.Add(RuntimeEntity{
		EntityID:    playerEntityID(state.CharacterID),
		DisplayName: state.CharacterID,
		Kind:        "player",
		ZoneID:      state.ZoneID,
		Position:    state.Position,
	})
	r.EntityZoneIndex[playerEntityID(state.CharacterID)] = state.ZoneID

	diffs := []StateDiff{newStateDiff(observability.EventWorldZoneRoutingUpdated, state.CharacterID, "", state.ZoneID, nil)}
	if previousZoneID != "" && previousZoneID != state.ZoneID {
		diffs = append([]StateDiff{newStateDiff(observability.EventWorldZoneExited, state.CharacterID, "", previousZoneID, nil)}, diffs...)
	}
	if emitEnter {
		diffs = append(diffs, newStateDiff(observability.EventWorldZoneEntered, state.CharacterID, "", state.ZoneID, nil))
		observability.LogEvent("world-service", observability.EventWorldZoneEntered, map[string]any{
			"characterId": state.CharacterID,
			"zoneId":      state.ZoneID,
		})
	}
	observability.LogEvent("world-service", observability.EventWorldZoneRoutingUpdated, map[string]any{
		"characterId": state.CharacterID,
		"zoneId":      state.ZoneID,
	})
	return diffs
}

func (r *ContinentRuntime) defaultEntry() (ZoneDefinition, ZoneEntryPoint, error) {
	zone, found := r.Registry.Zones[r.Definition.DefaultEntry.ZoneID]
	if !found {
		return ZoneDefinition{}, ZoneEntryPoint{}, fmt.Errorf("default entry zone %s is not loaded", r.Definition.DefaultEntry.ZoneID)
	}
	entry, found := zone.entryPoint(r.Definition.DefaultEntry.EntryPointID)
	if !found {
		return ZoneDefinition{}, ZoneEntryPoint{}, fmt.Errorf("default entry point %s is not loaded", r.Definition.DefaultEntry.EntryPointID)
	}
	return zone, entry, nil
}

func playerEntityID(characterID string) string {
	return "player:" + characterID
}

func distanceToBounds2D(position WorldPosition, bounds ZoneBounds) float64 {
	x := 0.0
	if position.X < bounds.MinX {
		x = bounds.MinX - position.X
	} else if position.X > bounds.MaxX {
		x = position.X - bounds.MaxX
	}
	y := 0.0
	if position.Y < bounds.MinY {
		y = bounds.MinY - position.Y
	} else if position.Y > bounds.MaxY {
		y = position.Y - bounds.MaxY
	}
	return distance2D(0, 0, x, y)
}

func hasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (r *ContinentRuntime) AssignZonesToShards(policy ShardAssignmentPolicy) (*ShardRuntimeIndex, error) {
	if len(r.Zones) == 0 {
		return nil, fmt.Errorf("continent runtime has no zones to assign")
	}
	if policy.Prefix == "" {
		policy.Prefix = string(r.Definition.ContinentID) + "-shard"
	}
	if policy.MaxZonesPerShard <= 0 {
		policy.MaxZonesPerShard = len(r.Zones)
	}
	zoneOrder := append([]string(nil), policy.ZoneOrder...)
	if len(zoneOrder) == 0 {
		zoneOrder = append(zoneOrder, r.Definition.Zones...)
	}
	index := &ShardRuntimeIndex{
		ZoneToShard:  map[string]string{},
		ShardToZones: map[string][]string{},
	}
	for assignmentIndex, zoneID := range zoneOrder {
		zoneRuntime, found := r.Zones[zoneID]
		if !found {
			observability.LogEvent("world-service", observability.EventWorldShardAssignmentFailed, map[string]any{
				"continentId": r.Definition.ContinentID,
				"zoneId":      zoneID,
				"reason":      "zone is not active",
			})
			return nil, fmt.Errorf("zone %s is not active", zoneID)
		}
		if _, exists := index.ZoneToShard[zoneID]; exists {
			observability.LogEvent("world-service", observability.EventWorldShardAssignmentFailed, map[string]any{
				"continentId": r.Definition.ContinentID,
				"zoneId":      zoneID,
				"reason":      "zone assigned twice",
			})
			return nil, fmt.Errorf("zone %s assigned twice", zoneID)
		}
		shardID := fmt.Sprintf("%s-%03d", policy.Prefix, assignmentIndex/policy.MaxZonesPerShard+1)
		index.ZoneToShard[zoneID] = shardID
		index.ShardToZones[shardID] = append(index.ShardToZones[shardID], zoneID)
		zoneRuntime.ShardID = ShardID(shardID)
		observability.LogEvent("world-service", observability.EventWorldShardAssigned, map[string]any{
			"continentId": r.Definition.ContinentID,
			"zoneId":      zoneID,
			"shardId":     shardID,
		})
		observability.LogEvent("world-service", observability.EventWorldShardZoneBound, map[string]any{
			"continentId": r.Definition.ContinentID,
			"zoneId":      zoneID,
			"shardId":     shardID,
		})
	}
	for zoneID := range r.Zones {
		if _, assigned := index.ZoneToShard[zoneID]; !assigned {
			observability.LogEvent("world-service", observability.EventWorldShardAssignmentFailed, map[string]any{
				"continentId": r.Definition.ContinentID,
				"zoneId":      zoneID,
				"reason":      "active zone missing from assignment policy",
			})
			return nil, fmt.Errorf("active zone %s missing from shard assignment policy", zoneID)
		}
	}
	r.Shards = index
	return index, nil
}

func (r *ContinentRuntime) ZoneShard(zoneID string) (ZoneShardHandle, bool) {
	if r.Shards == nil {
		return ZoneShardHandle{}, false
	}
	shardID, found := r.Shards.ZoneToShard[zoneID]
	if !found {
		return ZoneShardHandle{}, false
	}
	runtime, found := r.Zones[zoneID]
	if !found {
		return ZoneShardHandle{}, false
	}
	return ZoneShardHandle{ShardID: shardID, ZoneID: zoneID, Runtime: runtime}, true
}

func AsValidationErrors(err error) (ValidationErrors, bool) {
	var validation ValidationErrors
	if errors.As(err, &validation) {
		return validation, true
	}
	return nil, false
}
