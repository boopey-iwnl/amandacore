package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"amandacore/services/internal/observability"
)

const (
	SupportedSchemaVersion = "1"
	DefaultPackagePath     = "Content/Packs/dev_foundation/package.json"
)

const (
	EventPackageLoadStarted         = "content.package.load_started"
	EventPackageLoadCompleted       = "content.package.load_completed"
	EventPackageLoadFailed          = "content.package.load_failed"
	EventPackageValidationStarted   = "content.package.validation_started"
	EventPackageValidationCompleted = "content.package.validation_completed"
	EventPackageValidationFailed    = "content.package.validation_failed"
	EventPackageActivated           = "content.package.activated"
	EventPackageActivationFailed    = "content.package.activation_failed"
	EventZoneLoaded                 = "content.zone.loaded"
	EventZoneValidationFailed       = "content.zone.validation_failed"
	EventCatalogLoaded              = "content.catalog.loaded"
	EventCatalogValidationFailed    = "content.catalog.validation_failed"
	EventReferenceResolved          = "content.reference.resolved"
	EventReferenceBroken            = "content.reference.broken"
	EventQuestProviderRegistered    = "content.quest_provider.registered"
	EventContinentLoaded            = "content.continent.loaded"
	EventContinentValidationFailed  = "content.continent.validation_failed"
	EventZoneTransitionLoaded       = "content.zone_transition.loaded"
	EventZoneAdjacencyLoaded        = "content.zone_adjacency.loaded"
	EventWorldZoneRuntimeCreated    = "world.zone.runtime_created"
	EventLoadsimContentStarted      = "loadsim.content.started"
	EventLoadsimContentCompleted    = "loadsim.content.completed"
	EventLoadsimDawnwakeStarted     = "loadsim.dawnwake.started"
	EventLoadsimDawnwakeCompleted   = "loadsim.dawnwake.completed"
)

type ContentPackageID string
type ContentPackageVersion string
type ContentValidationErrorCode string
type ContentValidationPath string

const (
	ErrorMissingFile              ContentValidationErrorCode = "MissingFile"
	ErrorMalformedJSON            ContentValidationErrorCode = "MalformedJson"
	ErrorUnsupportedSchemaVersion ContentValidationErrorCode = "UnsupportedSchemaVersion"
	ErrorMissingRequiredField     ContentValidationErrorCode = "MissingRequiredField"
	ErrorDuplicateID              ContentValidationErrorCode = "DuplicateID"
	ErrorInvalidID                ContentValidationErrorCode = "InvalidID"
	ErrorInvalidEnum              ContentValidationErrorCode = "InvalidEnum"
	ErrorInvalidNumberRange       ContentValidationErrorCode = "InvalidNumberRange"
	ErrorBrokenReference          ContentValidationErrorCode = "BrokenReference"
	ErrorPositionOutOfBounds      ContentValidationErrorCode = "PositionOutOfBounds"
	ErrorObjectiveGraphCycle      ContentValidationErrorCode = "ObjectiveGraphCycle"
	ErrorRuntimeConfigInvalid     ContentValidationErrorCode = "RuntimeConfigInvalid"
	ErrorTransitionInvalid        ContentValidationErrorCode = "TransitionInvalid"
)

type ContentPackageManifest struct {
	PackageID       string   `json:"package_id"`
	DisplayName     string   `json:"display_name"`
	Version         string   `json:"version"`
	SchemaVersion   string   `json:"schema_version"`
	Description     string   `json:"description"`
	Authorship      string   `json:"authorship"`
	ContinentFiles  []string `json:"continent_files"`
	Zones           []string `json:"zones"`
	NPCCatalogs     []string `json:"npc_catalogs"`
	ItemCatalogs    []string `json:"item_catalogs"`
	LootCatalogs    []string `json:"loot_catalogs"`
	QuestCatalogs   []string `json:"quest_catalogs"`
	AbilityCatalogs []string `json:"ability_catalogs"`
	AuraCatalogs    []string `json:"aura_catalogs"`
	Tags            []string `json:"tags"`
}

type ContentPackageSource struct {
	ManifestPath string `json:"manifest_path"`
	RootDir      string `json:"root_dir"`
}

type ContentPackageLoadResult struct {
	Package    LoadedContentPackage    `json:"package"`
	Validation ContentValidationReport `json:"validation"`
	Validated  *ValidatedContentPackage
}

type LoadedContentPackage struct {
	Manifest      ContentPackageManifest `json:"manifest"`
	Source        ContentPackageSource   `json:"source"`
	Continents    []ContinentDefinition  `json:"continents"`
	Zones         []ZoneDefinition       `json:"zones"`
	NPCs          []NpcArchetype         `json:"npcs"`
	Items         []ItemDefinition       `json:"items"`
	LootTables    []LootTableDefinition  `json:"loot_tables"`
	Quests        []QuestDefinition      `json:"quests"`
	Abilities     []AbilityDefinition    `json:"abilities"`
	Auras         []AuraDefinition       `json:"auras"`
	LoadedFiles   []string               `json:"loaded_files"`
	CatalogCounts map[string]int         `json:"catalog_counts"`
}

type ValidatedContentPackage struct {
	Loaded   LoadedContentPackage
	Registry RuntimeContentRegistry
}

type RuntimeContentRegistry struct {
	PackageID     string
	Version       string
	Continents    map[string]ContinentDefinition
	Zones         map[string]ZoneDefinition
	NPCs          map[string]NpcArchetype
	Items         map[string]ItemDefinition
	LootTables    map[string]LootTableDefinition
	Quests        map[string]QuestDefinition
	Abilities     map[string]AbilityDefinition
	Auras         map[string]AuraDefinition
	QuestProvider map[string]QuestProviderDefinition
}

type ContentValidationReport struct {
	Errors []ContentValidationError `json:"errors"`
}

type ContentValidationError struct {
	Code    ContentValidationErrorCode `json:"code"`
	Path    ContentValidationPath      `json:"path"`
	Message string                     `json:"message"`
}

func (r ContentValidationReport) Valid() bool {
	return len(r.Errors) == 0
}

func (r *ContentValidationReport) Add(code ContentValidationErrorCode, path string, message string) {
	r.Errors = append(r.Errors, ContentValidationError{
		Code:    code,
		Path:    ContentValidationPath(path),
		Message: message,
	})
}

func (r *ContentValidationReport) Addf(code ContentValidationErrorCode, path string, format string, args ...any) {
	r.Add(code, path, fmt.Sprintf(format, args...))
}

func (r ContentValidationReport) HasCode(code ContentValidationErrorCode) bool {
	for _, err := range r.Errors {
		if err.Code == code {
			return true
		}
	}
	return false
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type ZoneBounds struct {
	MinX float64 `json:"min_x"`
	MinY float64 `json:"min_y"`
	MinZ float64 `json:"min_z"`
	MaxX float64 `json:"max_x"`
	MaxY float64 `json:"max_y"`
	MaxZ float64 `json:"max_z"`
}

type ZoneEntryPoint struct {
	EntryID   string   `json:"entry_id"`
	Position  Position `json:"position"`
	FacingYaw float64  `json:"facing_yaw"`
}

type ZoneRuntimeConfig struct {
	TickMS      int `json:"tick_ms"`
	MaxPlayers  int `json:"max_players"`
	MaxEntities int `json:"max_entities"`
}

type ContinentDefinition struct {
	ContinentID  string                   `json:"continent_id"`
	DisplayName  string                   `json:"display_name"`
	Description  string                   `json:"description"`
	Origin       Position                 `json:"origin"`
	Units        string                   `json:"units"`
	Zones        []ContinentZoneRef       `json:"zones"`
	Adjacency    []ZoneAdjacency          `json:"adjacency"`
	DefaultEntry ContinentEntryRef        `json:"default_entry"`
	Streaming    ContinentStreamingConfig `json:"streaming"`
	Tags         []string                 `json:"tags"`
	Metadata     map[string]any           `json:"metadata"`
}

type ContinentZoneRef struct {
	ZoneID      string   `json:"zone_id"`
	DisplayName string   `json:"display_name"`
	Tags        []string `json:"tags"`
}

type ZoneAdjacency struct {
	AdjacencyID   string   `json:"adjacency_id"`
	FromZoneID    string   `json:"from_zone_id"`
	ToZoneID      string   `json:"to_zone_id"`
	TransitionIDs []string `json:"transition_ids"`
	Kind          string   `json:"kind"`
	Bidirectional bool     `json:"bidirectional"`
	Tags          []string `json:"tags"`
}

type ContinentEntryRef struct {
	ZoneID       string `json:"zone_id"`
	EntryPointID string `json:"entry_point_id"`
}

type ContinentStreamingConfig struct {
	ActivationPolicy      string  `json:"activation_policy"`
	DefaultInterestRadius float64 `json:"default_interest_radius"`
	GateHintRadius        float64 `json:"gate_hint_radius"`
}

type ZoneTransitionGate struct {
	TransitionID          string     `json:"transition_id"`
	FromZoneID            string     `json:"from_zone_id"`
	ToZoneID              string     `json:"to_zone_id"`
	Kind                  string     `json:"kind"`
	GateBounds            ZoneBounds `json:"gate_bounds"`
	EntryPointIDOnArrival string     `json:"entry_point_id_on_arrival"`
	Disabled              bool       `json:"disabled"`
	Tags                  []string   `json:"tags"`
}

type ZoneStreamingConfig struct {
	InterestRadius          float64  `json:"interest_radius"`
	AdjacentPreloadDistance float64  `json:"adjacent_preload_distance"`
	Priority                string   `json:"priority"`
	Notes                   []string `json:"notes"`
}

type ZoneDefinition struct {
	ZoneID          string                    `json:"zone_id"`
	DisplayName     string                    `json:"display_name"`
	Description     string                    `json:"description"`
	ContinentID     string                    `json:"continent_id"`
	Bounds          ZoneBounds                `json:"bounds"`
	EntryPoints     []ZoneEntryPoint          `json:"entry_points"`
	TransitionGates []ZoneTransitionGate      `json:"transition_gates"`
	SpawnGroups     []SpawnGroupDefinition    `json:"spawn_groups"`
	QuestProviders  []QuestProviderDefinition `json:"quest_providers"`
	Runtime         ZoneRuntimeConfig         `json:"runtime"`
	Streaming       ZoneStreamingConfig       `json:"streaming"`
	Tags            []string                  `json:"tags"`
	Metadata        map[string]any            `json:"metadata"`
}

type SpawnGroupDefinition struct {
	SpawnGroupID   string                 `json:"spawn_group_id"`
	NPCArchetypeID string                 `json:"npc_archetype_id"`
	LootTableID    string                 `json:"loot_table_id"`
	SpawnPoints    []SpawnPointDefinition `json:"spawn_points"`
	RespawnSeconds int                    `json:"respawn_seconds"`
	MaxAlive       int                    `json:"max_alive"`
	Tags           []string               `json:"tags"`
}

type SpawnPointDefinition struct {
	SpawnPointID string   `json:"spawn_point_id"`
	Position     Position `json:"position"`
	FacingYaw    float64  `json:"facing_yaw"`
}

type QuestProviderDefinition struct {
	ProviderID      string   `json:"provider_id"`
	DisplayName     string   `json:"display_name"`
	Position        Position `json:"position"`
	OfferedQuestIDs []string `json:"offered_quest_ids"`
	Tags            []string `json:"tags"`
}

type NpcArchetype struct {
	ArchetypeID       string   `json:"archetype_id"`
	DisplayName       string   `json:"display_name"`
	Level             int      `json:"level"`
	MaxHealth         float64  `json:"max_health"`
	Disposition       string   `json:"disposition"`
	AttackRange       float64  `json:"attack_range"`
	AggroRange        float64  `json:"aggro_range"`
	LeashRange        float64  `json:"leash_range"`
	BaseDamage        float64  `json:"base_damage"`
	AttackIntervalMS  int      `json:"attack_interval_ms"`
	DefaultAbilityIDs []string `json:"default_ability_ids"`
	Tags              []string `json:"tags"`
}

type ItemDefinition struct {
	ItemID      string   `json:"item_id"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Kind        string   `json:"kind"`
	Quality     string   `json:"quality"`
	MaxStack    int      `json:"max_stack"`
	Tags        []string `json:"tags"`
}

type LootTableDefinition struct {
	LootTableID string           `json:"loot_table_id"`
	Entries     []LootTableEntry `json:"entries"`
	AllowEmpty  bool             `json:"allow_empty"`
	Tags        []string         `json:"tags"`
}

type LootTableEntry struct {
	ItemID            string   `json:"item_id"`
	MinQuantity       int      `json:"min_quantity"`
	MaxQuantity       int      `json:"max_quantity"`
	DropChancePercent float64  `json:"drop_chance_percent"`
	Guaranteed        bool     `json:"guaranteed"`
	Tags              []string `json:"tags"`
}

type QuestDefinition struct {
	QuestID              string              `json:"quest_id"`
	DisplayName          string              `json:"display_name"`
	Summary              string              `json:"summary"`
	RequiredLevel        int                 `json:"required_level"`
	PrerequisiteQuestIDs []string            `json:"prerequisite_quest_ids"`
	ObjectiveGraph       QuestObjectiveGraph `json:"objective_graph"`
	Rewards              []QuestReward       `json:"rewards"`
	Tags                 []string            `json:"tags"`
}

type QuestObjectiveGraph struct {
	Nodes []QuestObjectiveNode `json:"nodes"`
}

type QuestObjectiveNode struct {
	NodeID        string   `json:"node_id"`
	Kind          string   `json:"kind"`
	TargetID      string   `json:"target_id"`
	RequiredCount int      `json:"required_count"`
	DependsOn     []string `json:"depends_on"`
}

type QuestReward struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

type AbilityDefinition struct {
	AbilityID   string          `json:"ability_id"`
	DisplayName string          `json:"display_name"`
	School      string          `json:"school"`
	TargetRule  string          `json:"target_rule"`
	Range       float64         `json:"range"`
	Timing      AbilityTiming   `json:"timing"`
	CooldownMS  int             `json:"cooldown_ms"`
	Effects     []AbilityEffect `json:"effects"`
	Tags        []string        `json:"tags"`
}

type AbilityTiming struct {
	CastMS int `json:"cast_ms"`
}

type AbilityEffect struct {
	Kind      string  `json:"kind"`
	AuraID    string  `json:"aura_id"`
	Magnitude float64 `json:"magnitude"`
}

type AuraDefinition struct {
	AuraID      string         `json:"aura_id"`
	DisplayName string         `json:"display_name"`
	Kind        string         `json:"kind"`
	DurationMS  int            `json:"duration_ms"`
	MaxStacks   int            `json:"max_stacks"`
	StackRule   string         `json:"stack_rule"`
	TickRule    string         `json:"tick_rule"`
	Modifiers   []AuraModifier `json:"modifiers"`
	Tags        []string       `json:"tags"`
}

type AuraModifier struct {
	Stat      string  `json:"stat"`
	Operation string  `json:"operation"`
	Value     float64 `json:"value"`
}

type ContentPackageLoader struct{}

func NewContentPackageLoader() ContentPackageLoader {
	return ContentPackageLoader{}
}

func (ContentPackageLoader) Load(manifestPath string) ContentPackageLoadResult {
	report := ContentValidationReport{}
	resolvedPath, requestedPath := resolvePackagePath(manifestPath)
	observability.LogEvent("content-loader", EventPackageLoadStarted, map[string]any{
		"manifestPath": requestedPath,
	})

	manifestBytes, err := os.ReadFile(resolvedPath)
	if err != nil {
		report.Addf(ErrorMissingFile, "package", "content package manifest %q could not be read: %v", requestedPath, err)
		observability.LogEvent("content-loader", EventPackageLoadFailed, map[string]any{
			"manifestPath": requestedPath,
			"errorCount":   len(report.Errors),
		})
		return ContentPackageLoadResult{Validation: report}
	}

	var manifest ContentPackageManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		report.Addf(ErrorMalformedJSON, "package", "content package manifest %q is malformed: %v", requestedPath, err)
		observability.LogEvent("content-loader", EventPackageLoadFailed, map[string]any{
			"manifestPath": requestedPath,
			"errorCount":   len(report.Errors),
		})
		return ContentPackageLoadResult{Validation: report}
	}

	loaded := LoadedContentPackage{
		Manifest: manifest,
		Source: ContentPackageSource{
			ManifestPath: resolvedPath,
			RootDir:      filepath.Dir(resolvedPath),
		},
		CatalogCounts: map[string]int{},
	}

	loadFiles(&loaded, &report)
	observability.LogEvent("content-loader", EventPackageValidationStarted, map[string]any{
		"packageId": manifest.PackageID,
	})
	validation := ValidateLoadedContentPackage(loaded)
	report.Errors = append(report.Errors, validation.Errors...)

	result := ContentPackageLoadResult{Package: loaded, Validation: report}
	if report.Valid() {
		registry := NewRuntimeContentRegistry(loaded)
		result.Validated = &ValidatedContentPackage{Loaded: loaded, Registry: registry}
		observability.LogEvent("content-loader", EventPackageValidationCompleted, map[string]any{
			"packageId":  manifest.PackageID,
			"errorCount": 0,
		})
		observability.LogEvent("content-loader", EventPackageLoadCompleted, map[string]any{
			"packageId":  manifest.PackageID,
			"continents": len(loaded.Continents),
			"zones":      len(loaded.Zones),
			"npcs":       len(loaded.NPCs),
			"items":      len(loaded.Items),
			"loot":       len(loaded.LootTables),
			"quests":     len(loaded.Quests),
		})
		return result
	}

	observability.LogEvent("content-loader", EventPackageValidationFailed, map[string]any{
		"packageId":  manifest.PackageID,
		"errorCount": len(report.Errors),
	})
	observability.LogEvent("content-loader", EventPackageLoadFailed, map[string]any{
		"packageId":  manifest.PackageID,
		"errorCount": len(report.Errors),
	})
	return result
}

func resolvePackagePath(manifestPath string) (string, string) {
	requested := strings.TrimSpace(manifestPath)
	if requested == "" {
		requested = strings.TrimSpace(os.Getenv("AMANDACORE_CONTENT_PACKAGE"))
	}
	if requested == "" {
		requested = DefaultPackagePath
	}
	if filepath.IsAbs(requested) {
		return filepath.Clean(requested), requested
	}

	if found, ok := resolveRelativeFromParents(requested); ok {
		return found, requested
	}
	return filepath.Clean(requested), requested
}

func ResolvePackagePath(manifestPath string) string {
	resolved, _ := resolvePackagePath(manifestPath)
	return resolved
}

func resolveRelativeFromParents(relative string) (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	current := cwd
	for {
		candidate := filepath.Clean(filepath.Join(current, relative))
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func loadFiles(loaded *LoadedContentPackage, report *ContentValidationReport) {
	loadContinentFiles(loaded, report)
	loadZoneFiles(loaded, report)
	loadCatalogFiles(loaded.Manifest.NPCCatalogs, loaded, report, "npc_catalogs", "npc", func(target *LoadedContentPackage, payload []byte) error {
		var file struct {
			NPCs []NpcArchetype `json:"npcs"`
		}
		if err := json.Unmarshal(payload, &file); err != nil {
			return err
		}
		target.NPCs = append(target.NPCs, file.NPCs...)
		target.CatalogCounts["npcs"] += len(file.NPCs)
		return nil
	})
	loadCatalogFiles(loaded.Manifest.ItemCatalogs, loaded, report, "item_catalogs", "item", func(target *LoadedContentPackage, payload []byte) error {
		var file struct {
			Items []ItemDefinition `json:"items"`
		}
		if err := json.Unmarshal(payload, &file); err != nil {
			return err
		}
		target.Items = append(target.Items, file.Items...)
		target.CatalogCounts["items"] += len(file.Items)
		return nil
	})
	loadCatalogFiles(loaded.Manifest.LootCatalogs, loaded, report, "loot_catalogs", "loot", func(target *LoadedContentPackage, payload []byte) error {
		var file struct {
			LootTables []LootTableDefinition `json:"loot_tables"`
		}
		if err := json.Unmarshal(payload, &file); err != nil {
			return err
		}
		target.LootTables = append(target.LootTables, file.LootTables...)
		target.CatalogCounts["loot_tables"] += len(file.LootTables)
		return nil
	})
	loadCatalogFiles(loaded.Manifest.QuestCatalogs, loaded, report, "quest_catalogs", "quest", func(target *LoadedContentPackage, payload []byte) error {
		var file struct {
			Quests []QuestDefinition `json:"quests"`
		}
		if err := json.Unmarshal(payload, &file); err != nil {
			return err
		}
		target.Quests = append(target.Quests, file.Quests...)
		target.CatalogCounts["quests"] += len(file.Quests)
		return nil
	})
	loadCatalogFiles(loaded.Manifest.AbilityCatalogs, loaded, report, "ability_catalogs", "ability", func(target *LoadedContentPackage, payload []byte) error {
		var file struct {
			Abilities []AbilityDefinition `json:"abilities"`
		}
		if err := json.Unmarshal(payload, &file); err != nil {
			return err
		}
		target.Abilities = append(target.Abilities, file.Abilities...)
		target.CatalogCounts["abilities"] += len(file.Abilities)
		return nil
	})
	loadCatalogFiles(loaded.Manifest.AuraCatalogs, loaded, report, "aura_catalogs", "aura", func(target *LoadedContentPackage, payload []byte) error {
		var file struct {
			Auras []AuraDefinition `json:"auras"`
		}
		if err := json.Unmarshal(payload, &file); err != nil {
			return err
		}
		target.Auras = append(target.Auras, file.Auras...)
		target.CatalogCounts["auras"] += len(file.Auras)
		return nil
	})
}

func loadContinentFiles(loaded *LoadedContentPackage, report *ContentValidationReport) {
	for index, relative := range loaded.Manifest.ContinentFiles {
		path := filepath.Clean(filepath.Join(loaded.Source.RootDir, relative))
		payload, err := os.ReadFile(path)
		if err != nil {
			report.Addf(ErrorMissingFile, fmt.Sprintf("continent_files[%d]", index), "continent file %q could not be read: %v", relative, err)
			continue
		}
		var continent ContinentDefinition
		if err := json.Unmarshal(payload, &continent); err != nil {
			report.Addf(ErrorMalformedJSON, fmt.Sprintf("continent_files[%d]", index), "continent file %q is malformed: %v", relative, err)
			continue
		}
		loaded.Continents = append(loaded.Continents, continent)
		loaded.LoadedFiles = append(loaded.LoadedFiles, path)
		loaded.CatalogCounts["continents"]++
		observability.LogEvent("content-loader", EventContinentLoaded, map[string]any{
			"continentId": continent.ContinentID,
			"path":        relative,
		})
		for _, adjacency := range continent.Adjacency {
			observability.LogEvent("content-loader", EventZoneAdjacencyLoaded, map[string]any{
				"continentId": continent.ContinentID,
				"adjacencyId": adjacency.AdjacencyID,
				"fromZoneId":  adjacency.FromZoneID,
				"toZoneId":    adjacency.ToZoneID,
			})
		}
	}
}

func loadZoneFiles(loaded *LoadedContentPackage, report *ContentValidationReport) {
	for index, relative := range loaded.Manifest.Zones {
		path := filepath.Clean(filepath.Join(loaded.Source.RootDir, relative))
		payload, err := os.ReadFile(path)
		if err != nil {
			report.Addf(ErrorMissingFile, fmt.Sprintf("zones[%d]", index), "zone file %q could not be read: %v", relative, err)
			continue
		}
		var zone ZoneDefinition
		if err := json.Unmarshal(payload, &zone); err != nil {
			report.Addf(ErrorMalformedJSON, fmt.Sprintf("zones[%d]", index), "zone file %q is malformed: %v", relative, err)
			continue
		}
		loaded.Zones = append(loaded.Zones, zone)
		loaded.LoadedFiles = append(loaded.LoadedFiles, path)
		loaded.CatalogCounts["zones"]++
		observability.LogEvent("content-loader", EventZoneLoaded, map[string]any{
			"zoneId": zone.ZoneID,
			"path":   relative,
		})
		for _, gate := range zone.TransitionGates {
			observability.LogEvent("content-loader", EventZoneTransitionLoaded, map[string]any{
				"zoneId":       zone.ZoneID,
				"transitionId": gate.TransitionID,
				"fromZoneId":   gate.FromZoneID,
				"toZoneId":     gate.ToZoneID,
				"disabled":     gate.Disabled,
			})
		}
	}
}

func loadCatalogFiles(paths []string, loaded *LoadedContentPackage, report *ContentValidationReport, manifestField string, catalogKind string, apply func(*LoadedContentPackage, []byte) error) {
	for index, relative := range paths {
		path := filepath.Clean(filepath.Join(loaded.Source.RootDir, relative))
		payload, err := os.ReadFile(path)
		if err != nil {
			report.Addf(ErrorMissingFile, fmt.Sprintf("%s[%d]", manifestField, index), "%s catalog file %q could not be read: %v", catalogKind, relative, err)
			continue
		}
		if err := apply(loaded, payload); err != nil {
			report.Addf(ErrorMalformedJSON, fmt.Sprintf("%s[%d]", manifestField, index), "%s catalog file %q is malformed: %v", catalogKind, relative, err)
			continue
		}
		loaded.LoadedFiles = append(loaded.LoadedFiles, path)
		observability.LogEvent("content-loader", EventCatalogLoaded, map[string]any{
			"catalog": catalogKind,
			"path":    relative,
		})
	}
}

func ValidateLoadedContentPackage(loaded LoadedContentPackage) ContentValidationReport {
	report := ContentValidationReport{}
	validateManifest(loaded.Manifest, &report)

	continentIDs := validateIDs("continents", loaded.Continents, func(continent ContinentDefinition) string { return continent.ContinentID }, &report)
	zoneIDs := validateIDs("zones", loaded.Zones, func(zone ZoneDefinition) string { return zone.ZoneID }, &report)
	npcIDs := validateIDs("npcs", loaded.NPCs, func(npc NpcArchetype) string { return npc.ArchetypeID }, &report)
	itemIDs := validateIDs("items", loaded.Items, func(item ItemDefinition) string { return item.ItemID }, &report)
	lootIDs := validateIDs("loot_tables", loaded.LootTables, func(loot LootTableDefinition) string { return loot.LootTableID }, &report)
	questIDs := validateIDs("quests", loaded.Quests, func(quest QuestDefinition) string { return quest.QuestID }, &report)
	abilityIDs := validateIDs("abilities", loaded.Abilities, func(ability AbilityDefinition) string { return ability.AbilityID }, &report)
	auraIDs := validateIDs("auras", loaded.Auras, func(aura AuraDefinition) string { return aura.AuraID }, &report)

	providerIDs := map[string]struct{}{}
	for zoneIndex, zone := range loaded.Zones {
		validateZone(zone, zoneIndex, continentIDs, npcIDs, lootIDs, questIDs, providerIDs, &report)
	}
	validateContinentTopology(loaded.Continents, loaded.Zones, zoneIDs, &report)
	for index, npc := range loaded.NPCs {
		validateNPC(npc, index, abilityIDs, len(loaded.Abilities) > 0, &report)
	}
	for index, item := range loaded.Items {
		validateItem(item, index, &report)
	}
	for index, loot := range loaded.LootTables {
		validateLootTable(loot, index, itemIDs, &report)
	}
	for index, quest := range loaded.Quests {
		validateQuest(quest, index, questIDs, npcIDs, itemIDs, providerIDs, &report)
	}
	for index, ability := range loaded.Abilities {
		validateAbility(ability, index, auraIDs, &report)
	}
	for index, aura := range loaded.Auras {
		validateAura(aura, index, &report)
	}

	return report
}

func validateManifest(manifest ContentPackageManifest, report *ContentValidationReport) {
	requiredString(report, "package.package_id", manifest.PackageID)
	requiredString(report, "package.display_name", manifest.DisplayName)
	requiredString(report, "package.version", manifest.Version)
	requiredString(report, "package.schema_version", manifest.SchemaVersion)
	if manifest.PackageID != "" && !validID(manifest.PackageID) {
		report.Addf(ErrorInvalidID, "package.package_id", "package_id %q is not a stable AmandaCore content id", manifest.PackageID)
	}
	if manifest.SchemaVersion != "" && manifest.SchemaVersion != SupportedSchemaVersion {
		report.Addf(ErrorUnsupportedSchemaVersion, "package.schema_version", "schema_version %q is not supported; expected %q", manifest.SchemaVersion, SupportedSchemaVersion)
	}
	if len(manifest.Zones) == 0 {
		report.Add(ErrorMissingRequiredField, "package.zones", "at least one zone file must be listed")
	}
}

func validateZone(zone ZoneDefinition, index int, continentIDs map[string]struct{}, npcIDs map[string]struct{}, lootIDs map[string]struct{}, questIDs map[string]struct{}, providerIDs map[string]struct{}, report *ContentValidationReport) {
	path := fmt.Sprintf("zones[%d]", index)
	requiredID(report, path+".zone_id", zone.ZoneID)
	requiredString(report, path+".display_name", zone.DisplayName)
	boundsValid := validateBounds(path+".bounds", zone.Bounds, report)
	if zone.ContinentID != "" && !containsID(continentIDs, zone.ContinentID) {
		report.Addf(ErrorBrokenReference, path+".continent_id", "zone %q references missing continent %q", zone.ZoneID, zone.ContinentID)
		logBrokenReference("zone", zone.ZoneID, "continent", zone.ContinentID)
	}
	for entryIndex, entry := range zone.EntryPoints {
		entryPath := fmt.Sprintf("%s.entry_points[%d]", path, entryIndex)
		requiredID(report, entryPath+".entry_id", entry.EntryID)
		if boundsValid && !positionInBounds(entry.Position, zone.Bounds) {
			report.Addf(ErrorPositionOutOfBounds, entryPath+".position", "entry point %q is outside zone bounds", entry.EntryID)
		}
	}
	for gateIndex, gate := range zone.TransitionGates {
		gatePath := fmt.Sprintf("%s.transition_gates[%d]", path, gateIndex)
		requiredID(report, gatePath+".transition_id", gate.TransitionID)
		requiredID(report, gatePath+".from_zone_id", gate.FromZoneID)
		requiredID(report, gatePath+".to_zone_id", gate.ToZoneID)
		requiredID(report, gatePath+".entry_point_id_on_arrival", gate.EntryPointIDOnArrival)
		if gate.FromZoneID != "" && zone.ZoneID != "" && gate.FromZoneID != zone.ZoneID {
			report.Addf(ErrorTransitionInvalid, gatePath+".from_zone_id", "transition %q must start from containing zone %q", gate.TransitionID, zone.ZoneID)
		}
		gateBoundsValid := validateBounds(gatePath+".gate_bounds", gate.GateBounds, report)
		if boundsValid && gateBoundsValid && !boundsContainBounds(zone.Bounds, gate.GateBounds) {
			report.Addf(ErrorPositionOutOfBounds, gatePath+".gate_bounds", "transition %q gate bounds are outside source zone bounds", gate.TransitionID)
		}
	}
	for groupIndex, group := range zone.SpawnGroups {
		groupPath := fmt.Sprintf("%s.spawn_groups[%d]", path, groupIndex)
		requiredID(report, groupPath+".spawn_group_id", group.SpawnGroupID)
		requiredID(report, groupPath+".npc_archetype_id", group.NPCArchetypeID)
		if group.NPCArchetypeID != "" && !containsID(npcIDs, group.NPCArchetypeID) {
			report.Addf(ErrorBrokenReference, groupPath+".npc_archetype_id", "spawn group %q references missing NPC archetype %q", group.SpawnGroupID, group.NPCArchetypeID)
			logBrokenReference("spawn_group", group.SpawnGroupID, "npc_archetype", group.NPCArchetypeID)
		}
		requiredID(report, groupPath+".loot_table_id", group.LootTableID)
		if group.LootTableID != "" && !containsID(lootIDs, group.LootTableID) {
			report.Addf(ErrorBrokenReference, groupPath+".loot_table_id", "spawn group %q references missing loot table %q", group.SpawnGroupID, group.LootTableID)
			logBrokenReference("spawn_group", group.SpawnGroupID, "loot_table", group.LootTableID)
		}
		if group.RespawnSeconds < 0 {
			report.Addf(ErrorInvalidNumberRange, groupPath+".respawn_seconds", "respawn_seconds must be non-negative")
		}
		if group.MaxAlive <= 0 {
			report.Addf(ErrorInvalidNumberRange, groupPath+".max_alive", "max_alive must be positive")
		}
		if len(group.SpawnPoints) == 0 {
			report.Add(ErrorMissingRequiredField, groupPath+".spawn_points", "spawn group must define at least one spawn point")
		}
		for spawnIndex, spawn := range group.SpawnPoints {
			spawnPath := fmt.Sprintf("%s.spawn_points[%d]", groupPath, spawnIndex)
			requiredID(report, spawnPath+".spawn_point_id", spawn.SpawnPointID)
			if boundsValid && !positionInBounds(spawn.Position, zone.Bounds) {
				report.Addf(ErrorPositionOutOfBounds, spawnPath+".position", "spawn point %q is outside zone bounds", spawn.SpawnPointID)
			}
		}
	}
	for providerIndex, provider := range zone.QuestProviders {
		providerPath := fmt.Sprintf("%s.quest_providers[%d]", path, providerIndex)
		requiredID(report, providerPath+".provider_id", provider.ProviderID)
		requiredString(report, providerPath+".display_name", provider.DisplayName)
		if provider.ProviderID != "" {
			if _, exists := providerIDs[provider.ProviderID]; exists {
				report.Addf(ErrorDuplicateID, providerPath+".provider_id", "quest provider id %q is duplicated", provider.ProviderID)
			}
			providerIDs[provider.ProviderID] = struct{}{}
		}
		if boundsValid && !positionInBounds(provider.Position, zone.Bounds) {
			report.Addf(ErrorPositionOutOfBounds, providerPath+".position", "quest provider %q is outside zone bounds", provider.ProviderID)
		}
		for questIndex, questID := range provider.OfferedQuestIDs {
			refPath := fmt.Sprintf("%s.offered_quest_ids[%d]", providerPath, questIndex)
			requiredID(report, refPath, questID)
			if questID != "" && !containsID(questIDs, questID) {
				report.Addf(ErrorBrokenReference, refPath, "quest provider %q references missing quest %q", provider.ProviderID, questID)
				logBrokenReference("quest_provider", provider.ProviderID, "quest", questID)
			}
		}
	}
	validateRuntime(path+".runtime", zone.Runtime, report)
}

func validateBounds(path string, bounds ZoneBounds, report *ContentValidationReport) bool {
	valid := true
	if bounds.MaxX <= bounds.MinX {
		report.Add(ErrorInvalidNumberRange, path+".max_x", "max_x must be greater than min_x")
		valid = false
	}
	if bounds.MaxY <= bounds.MinY {
		report.Add(ErrorInvalidNumberRange, path+".max_y", "max_y must be greater than min_y")
		valid = false
	}
	if bounds.MaxZ <= bounds.MinZ {
		report.Add(ErrorInvalidNumberRange, path+".max_z", "max_z must be greater than min_z")
		valid = false
	}
	return valid
}

func validateRuntime(path string, runtime ZoneRuntimeConfig, report *ContentValidationReport) {
	if runtime.TickMS <= 0 || runtime.TickMS < 16 || runtime.TickMS > 250 {
		report.Add(ErrorRuntimeConfigInvalid, path+".tick_ms", "tick_ms must be between 16 and 250")
	}
	if runtime.MaxPlayers < 0 {
		report.Add(ErrorRuntimeConfigInvalid, path+".max_players", "max_players must be positive when present")
	}
	if runtime.MaxEntities < 0 {
		report.Add(ErrorRuntimeConfigInvalid, path+".max_entities", "max_entities must be positive when present")
	}
	if runtime.MaxPlayers == 0 {
		report.Add(ErrorRuntimeConfigInvalid, path+".max_players", "max_players must be positive")
	}
	if runtime.MaxEntities == 0 {
		report.Add(ErrorRuntimeConfigInvalid, path+".max_entities", "max_entities must be positive")
	}
}

func validateContinentTopology(continents []ContinentDefinition, zones []ZoneDefinition, zoneIDs map[string]struct{}, report *ContentValidationReport) {
	zonesByID := map[string]ZoneDefinition{}
	transitionIDs := map[string]struct{}{}
	for _, zone := range zones {
		zonesByID[zone.ZoneID] = zone
		for _, gate := range zone.TransitionGates {
			if gate.TransitionID == "" {
				continue
			}
			if _, exists := transitionIDs[gate.TransitionID]; exists {
				report.Addf(ErrorDuplicateID, "zones.transition_gates", "transition id %q is duplicated", gate.TransitionID)
			}
			transitionIDs[gate.TransitionID] = struct{}{}
		}
	}

	for continentIndex, continent := range continents {
		path := fmt.Sprintf("continents[%d]", continentIndex)
		requiredID(report, path+".continent_id", continent.ContinentID)
		requiredString(report, path+".display_name", continent.DisplayName)
		for zoneIndex, zoneRef := range continent.Zones {
			refPath := fmt.Sprintf("%s.zones[%d].zone_id", path, zoneIndex)
			requiredID(report, refPath, zoneRef.ZoneID)
			if zoneRef.ZoneID != "" && !containsID(zoneIDs, zoneRef.ZoneID) {
				report.Addf(ErrorBrokenReference, refPath, "continent %q references missing zone %q", continent.ContinentID, zoneRef.ZoneID)
				logBrokenReference("continent", continent.ContinentID, "zone", zoneRef.ZoneID)
			}
		}
		requiredID(report, path+".default_entry.zone_id", continent.DefaultEntry.ZoneID)
		requiredID(report, path+".default_entry.entry_point_id", continent.DefaultEntry.EntryPointID)
		if continent.DefaultEntry.ZoneID != "" && !containsID(zoneIDs, continent.DefaultEntry.ZoneID) {
			report.Addf(ErrorBrokenReference, path+".default_entry.zone_id", "continent %q default entry references missing zone %q", continent.ContinentID, continent.DefaultEntry.ZoneID)
			logBrokenReference("continent", continent.ContinentID, "zone", continent.DefaultEntry.ZoneID)
		} else if continent.DefaultEntry.EntryPointID != "" && !zoneHasEntryPoint(zonesByID[continent.DefaultEntry.ZoneID], continent.DefaultEntry.EntryPointID) {
			report.Addf(ErrorBrokenReference, path+".default_entry.entry_point_id", "continent %q default entry references missing entry point %q", continent.ContinentID, continent.DefaultEntry.EntryPointID)
			logBrokenReference("continent", continent.ContinentID, "zone_entry", continent.DefaultEntry.EntryPointID)
		}
		for adjacencyIndex, adjacency := range continent.Adjacency {
			adjacencyPath := fmt.Sprintf("%s.adjacency[%d]", path, adjacencyIndex)
			requiredID(report, adjacencyPath+".adjacency_id", adjacency.AdjacencyID)
			requiredID(report, adjacencyPath+".from_zone_id", adjacency.FromZoneID)
			requiredID(report, adjacencyPath+".to_zone_id", adjacency.ToZoneID)
			if adjacency.FromZoneID != "" && !containsID(zoneIDs, adjacency.FromZoneID) {
				report.Addf(ErrorBrokenReference, adjacencyPath+".from_zone_id", "adjacency %q references missing source zone %q", adjacency.AdjacencyID, adjacency.FromZoneID)
				logBrokenReference("zone_adjacency", adjacency.AdjacencyID, "zone", adjacency.FromZoneID)
			}
			if adjacency.ToZoneID != "" && !containsID(zoneIDs, adjacency.ToZoneID) {
				report.Addf(ErrorBrokenReference, adjacencyPath+".to_zone_id", "adjacency %q references missing destination zone %q", adjacency.AdjacencyID, adjacency.ToZoneID)
				logBrokenReference("zone_adjacency", adjacency.AdjacencyID, "zone", adjacency.ToZoneID)
			}
			for transitionIndex, transitionID := range adjacency.TransitionIDs {
				transitionPath := fmt.Sprintf("%s.transition_ids[%d]", adjacencyPath, transitionIndex)
				requiredID(report, transitionPath, transitionID)
				if transitionID != "" && !containsID(transitionIDs, transitionID) {
					report.Addf(ErrorBrokenReference, transitionPath, "adjacency %q references missing transition gate %q", adjacency.AdjacencyID, transitionID)
					logBrokenReference("zone_adjacency", adjacency.AdjacencyID, "zone_transition", transitionID)
				}
			}
		}
	}

	for zoneIndex, zone := range zones {
		zonePath := fmt.Sprintf("zones[%d]", zoneIndex)
		for gateIndex, gate := range zone.TransitionGates {
			gatePath := fmt.Sprintf("%s.transition_gates[%d]", zonePath, gateIndex)
			if gate.ToZoneID != "" && !containsID(zoneIDs, gate.ToZoneID) {
				report.Addf(ErrorBrokenReference, gatePath+".to_zone_id", "transition %q references missing destination zone %q", gate.TransitionID, gate.ToZoneID)
				logBrokenReference("zone_transition", gate.TransitionID, "zone", gate.ToZoneID)
				continue
			}
			if gate.ToZoneID != "" && gate.EntryPointIDOnArrival != "" && !zoneHasEntryPoint(zonesByID[gate.ToZoneID], gate.EntryPointIDOnArrival) {
				report.Addf(ErrorBrokenReference, gatePath+".entry_point_id_on_arrival", "transition %q references missing destination entry point %q", gate.TransitionID, gate.EntryPointIDOnArrival)
				logBrokenReference("zone_transition", gate.TransitionID, "zone_entry", gate.EntryPointIDOnArrival)
			}
		}
	}
}

func zoneHasEntryPoint(zone ZoneDefinition, entryPointID string) bool {
	for _, entry := range zone.EntryPoints {
		if entry.EntryID == entryPointID {
			return true
		}
	}
	return false
}

func validateNPC(npc NpcArchetype, index int, abilityIDs map[string]struct{}, requireAbilityRefs bool, report *ContentValidationReport) {
	path := fmt.Sprintf("npcs[%d]", index)
	requiredID(report, path+".archetype_id", npc.ArchetypeID)
	requiredString(report, path+".display_name", npc.DisplayName)
	if npc.Level <= 0 {
		report.Add(ErrorInvalidNumberRange, path+".level", "level must be positive")
	}
	if npc.MaxHealth <= 0 {
		report.Add(ErrorInvalidNumberRange, path+".max_health", "max_health must be positive")
	}
	if !validEnum(npc.Disposition, "hostile", "neutral", "friendly") {
		report.Addf(ErrorInvalidEnum, path+".disposition", "disposition %q is not valid", npc.Disposition)
	}
	if npc.AttackRange < 0 || npc.AggroRange < 0 || npc.LeashRange < 0 || npc.BaseDamage < 0 {
		report.Add(ErrorInvalidNumberRange, path, "NPC ranges and base_damage must be non-negative")
	}
	if npc.LeashRange > 0 && npc.AggroRange > npc.LeashRange {
		report.Add(ErrorInvalidNumberRange, path+".leash_range", "leash_range must be greater than or equal to aggro_range")
	}
	if npc.AttackIntervalMS <= 0 {
		report.Add(ErrorInvalidNumberRange, path+".attack_interval_ms", "attack_interval_ms must be positive")
	}
	if requireAbilityRefs {
		for abilityIndex, abilityID := range npc.DefaultAbilityIDs {
			refPath := fmt.Sprintf("%s.default_ability_ids[%d]", path, abilityIndex)
			if !containsID(abilityIDs, abilityID) {
				report.Addf(ErrorBrokenReference, refPath, "NPC archetype %q references missing ability %q", npc.ArchetypeID, abilityID)
				logBrokenReference("npc_archetype", npc.ArchetypeID, "ability", abilityID)
			}
		}
	}
}

func validateItem(item ItemDefinition, index int, report *ContentValidationReport) {
	path := fmt.Sprintf("items[%d]", index)
	requiredID(report, path+".item_id", item.ItemID)
	requiredString(report, path+".display_name", item.DisplayName)
	if !validEnum(item.Kind, "weapon", "armor", "consumable", "material", "quest", "junk", "currency", "equipment") {
		report.Addf(ErrorInvalidEnum, path+".kind", "item kind %q is not valid", item.Kind)
	}
	if !validEnum(item.Quality, "poor", "common", "uncommon", "rare", "epic") {
		report.Addf(ErrorInvalidEnum, path+".quality", "item quality %q is not valid", item.Quality)
	}
	if item.MaxStack <= 0 {
		report.Add(ErrorInvalidNumberRange, path+".max_stack", "max_stack must be positive")
	}
}

func validateLootTable(loot LootTableDefinition, index int, itemIDs map[string]struct{}, report *ContentValidationReport) {
	path := fmt.Sprintf("loot_tables[%d]", index)
	requiredID(report, path+".loot_table_id", loot.LootTableID)
	if len(loot.Entries) == 0 && !loot.AllowEmpty {
		report.Add(ErrorMissingRequiredField, path+".entries", "loot table must have entries unless allow_empty is true")
	}
	for entryIndex, entry := range loot.Entries {
		entryPath := fmt.Sprintf("%s.entries[%d]", path, entryIndex)
		requiredID(report, entryPath+".item_id", entry.ItemID)
		if entry.ItemID != "" && !containsID(itemIDs, entry.ItemID) {
			report.Addf(ErrorBrokenReference, entryPath+".item_id", "loot table %q references missing item %q", loot.LootTableID, entry.ItemID)
			logBrokenReference("loot_table", loot.LootTableID, "item", entry.ItemID)
		}
		if entry.MinQuantity <= 0 || entry.MaxQuantity <= 0 || entry.MaxQuantity < entry.MinQuantity {
			report.Add(ErrorInvalidNumberRange, entryPath, "loot quantity range must be positive and max_quantity must be at least min_quantity")
		}
		if entry.DropChancePercent < 0 || entry.DropChancePercent > 100 {
			report.Add(ErrorInvalidNumberRange, entryPath+".drop_chance_percent", "drop_chance_percent must be between 0 and 100")
		}
	}
}

func validateQuest(quest QuestDefinition, index int, questIDs map[string]struct{}, npcIDs map[string]struct{}, itemIDs map[string]struct{}, providerIDs map[string]struct{}, report *ContentValidationReport) {
	path := fmt.Sprintf("quests[%d]", index)
	requiredID(report, path+".quest_id", quest.QuestID)
	requiredString(report, path+".display_name", quest.DisplayName)
	if quest.RequiredLevel < 0 {
		report.Add(ErrorInvalidNumberRange, path+".required_level", "required_level must be non-negative")
	}
	for prereqIndex, prereqID := range quest.PrerequisiteQuestIDs {
		refPath := fmt.Sprintf("%s.prerequisite_quest_ids[%d]", path, prereqIndex)
		if !containsID(questIDs, prereqID) {
			report.Addf(ErrorBrokenReference, refPath, "quest %q references missing prerequisite quest %q", quest.QuestID, prereqID)
		}
	}
	nodeIDs := map[string]struct{}{}
	for nodeIndex, node := range quest.ObjectiveGraph.Nodes {
		nodePath := fmt.Sprintf("%s.objective_graph.nodes[%d]", path, nodeIndex)
		requiredID(report, nodePath+".node_id", node.NodeID)
		if node.NodeID != "" {
			if _, exists := nodeIDs[node.NodeID]; exists {
				report.Addf(ErrorDuplicateID, nodePath+".node_id", "objective node id %q is duplicated in quest %q", node.NodeID, quest.QuestID)
			}
			nodeIDs[node.NodeID] = struct{}{}
		}
		if !validEnum(node.Kind, "kill_npc", "collect_item", "talk_provider") {
			report.Addf(ErrorInvalidEnum, nodePath+".kind", "objective kind %q is not valid", node.Kind)
		}
		if node.RequiredCount <= 0 {
			report.Add(ErrorInvalidNumberRange, nodePath+".required_count", "required_count must be positive")
		}
		switch node.Kind {
		case "kill_npc":
			if !containsID(npcIDs, node.TargetID) {
				report.Addf(ErrorBrokenReference, nodePath+".target_id", "quest objective references missing NPC archetype %q", node.TargetID)
				logBrokenReference("quest", quest.QuestID, "npc_archetype", node.TargetID)
			}
		case "collect_item":
			if !containsID(itemIDs, node.TargetID) {
				report.Addf(ErrorBrokenReference, nodePath+".target_id", "quest objective references missing item %q", node.TargetID)
				logBrokenReference("quest", quest.QuestID, "item", node.TargetID)
			}
		case "talk_provider":
			if !containsID(providerIDs, node.TargetID) {
				report.Addf(ErrorBrokenReference, nodePath+".target_id", "quest objective references missing quest provider %q", node.TargetID)
				logBrokenReference("quest", quest.QuestID, "quest_provider", node.TargetID)
			}
		}
		for depIndex, dependencyID := range node.DependsOn {
			depPath := fmt.Sprintf("%s.depends_on[%d]", nodePath, depIndex)
			if !containsID(nodeIDs, dependencyID) {
				// A second pass below catches forward references after all ids are known.
				requiredID(report, depPath, dependencyID)
			}
		}
	}
	for nodeIndex, node := range quest.ObjectiveGraph.Nodes {
		nodePath := fmt.Sprintf("%s.objective_graph.nodes[%d]", path, nodeIndex)
		for depIndex, dependencyID := range node.DependsOn {
			depPath := fmt.Sprintf("%s.depends_on[%d]", nodePath, depIndex)
			if !containsID(nodeIDs, dependencyID) {
				report.Addf(ErrorBrokenReference, depPath, "objective node %q depends on missing node %q", node.NodeID, dependencyID)
			}
		}
	}
	if len(quest.ObjectiveGraph.Nodes) == 0 {
		report.Add(ErrorMissingRequiredField, path+".objective_graph.nodes", "quest must define at least one objective node")
	} else if hasObjectiveCycle(quest.ObjectiveGraph.Nodes) {
		report.Addf(ErrorObjectiveGraphCycle, path+".objective_graph", "quest %q objective graph contains a cycle", quest.QuestID)
	}
	for rewardIndex, reward := range quest.Rewards {
		rewardPath := fmt.Sprintf("%s.rewards[%d]", path, rewardIndex)
		requiredID(report, rewardPath+".item_id", reward.ItemID)
		if reward.ItemID != "" && !containsID(itemIDs, reward.ItemID) {
			report.Addf(ErrorBrokenReference, rewardPath+".item_id", "quest reward references missing item %q", reward.ItemID)
			logBrokenReference("quest", quest.QuestID, "item", reward.ItemID)
		}
		if reward.Quantity <= 0 {
			report.Add(ErrorInvalidNumberRange, rewardPath+".quantity", "reward quantity must be positive")
		}
	}
}

func validateAbility(ability AbilityDefinition, index int, auraIDs map[string]struct{}, report *ContentValidationReport) {
	path := fmt.Sprintf("abilities[%d]", index)
	requiredID(report, path+".ability_id", ability.AbilityID)
	requiredString(report, path+".display_name", ability.DisplayName)
	if !validEnum(ability.School, "physical", "primal", "spirit", "arcane", "none") {
		report.Addf(ErrorInvalidEnum, path+".school", "ability school %q is not valid", ability.School)
	}
	if !validEnum(ability.TargetRule, "self", "enemy", "ally", "none") {
		report.Addf(ErrorInvalidEnum, path+".target_rule", "target_rule %q is not valid", ability.TargetRule)
	}
	if ability.Range < 0 || ability.CooldownMS < 0 || ability.Timing.CastMS < 0 {
		report.Add(ErrorInvalidNumberRange, path, "ability range, cooldown_ms, and timing.cast_ms must be non-negative")
	}
	for effectIndex, effect := range ability.Effects {
		effectPath := fmt.Sprintf("%s.effects[%d]", path, effectIndex)
		if !validEnum(effect.Kind, "direct_damage", "heal", "apply_aura") {
			report.Addf(ErrorInvalidEnum, effectPath+".kind", "ability effect kind %q is not valid", effect.Kind)
		}
		if effect.Kind == "apply_aura" {
			requiredID(report, effectPath+".aura_id", effect.AuraID)
			if effect.AuraID != "" && !containsID(auraIDs, effect.AuraID) {
				report.Addf(ErrorBrokenReference, effectPath+".aura_id", "ability %q references missing aura %q", ability.AbilityID, effect.AuraID)
				logBrokenReference("ability", ability.AbilityID, "aura", effect.AuraID)
			}
		}
	}
}

func validateAura(aura AuraDefinition, index int, report *ContentValidationReport) {
	path := fmt.Sprintf("auras[%d]", index)
	requiredID(report, path+".aura_id", aura.AuraID)
	requiredString(report, path+".display_name", aura.DisplayName)
	if !validEnum(aura.Kind, "buff", "debuff", "passive") {
		report.Addf(ErrorInvalidEnum, path+".kind", "aura kind %q is not valid", aura.Kind)
	}
	if aura.DurationMS < 0 || aura.MaxStacks <= 0 {
		report.Add(ErrorInvalidNumberRange, path, "duration_ms must be non-negative and max_stacks must be positive")
	}
	if !validEnum(aura.StackRule, "refresh", "stack", "ignore") {
		report.Addf(ErrorInvalidEnum, path+".stack_rule", "stack_rule %q is not valid", aura.StackRule)
	}
	if !validEnum(aura.TickRule, "none", "interval") {
		report.Addf(ErrorInvalidEnum, path+".tick_rule", "tick_rule %q is not valid", aura.TickRule)
	}
}

func NewRuntimeContentRegistry(loaded LoadedContentPackage) RuntimeContentRegistry {
	registry := RuntimeContentRegistry{
		PackageID:     loaded.Manifest.PackageID,
		Version:       loaded.Manifest.Version,
		Continents:    map[string]ContinentDefinition{},
		Zones:         map[string]ZoneDefinition{},
		NPCs:          map[string]NpcArchetype{},
		Items:         map[string]ItemDefinition{},
		LootTables:    map[string]LootTableDefinition{},
		Quests:        map[string]QuestDefinition{},
		Abilities:     map[string]AbilityDefinition{},
		Auras:         map[string]AuraDefinition{},
		QuestProvider: map[string]QuestProviderDefinition{},
	}
	for _, continent := range loaded.Continents {
		registry.Continents[continent.ContinentID] = continent
	}
	for _, zone := range loaded.Zones {
		registry.Zones[zone.ZoneID] = zone
		for _, provider := range zone.QuestProviders {
			registry.QuestProvider[provider.ProviderID] = provider
		}
	}
	for _, npc := range loaded.NPCs {
		registry.NPCs[npc.ArchetypeID] = npc
	}
	for _, item := range loaded.Items {
		registry.Items[item.ItemID] = item
	}
	for _, loot := range loaded.LootTables {
		registry.LootTables[loot.LootTableID] = loot
	}
	for _, quest := range loaded.Quests {
		registry.Quests[quest.QuestID] = quest
	}
	for _, ability := range loaded.Abilities {
		registry.Abilities[ability.AbilityID] = ability
	}
	for _, aura := range loaded.Auras {
		registry.Auras[aura.AuraID] = aura
	}
	return registry
}

func SortedKeys[T any](source map[string]T) []string {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validateIDs[T any](path string, values []T, id func(T) string, report *ContentValidationReport) map[string]struct{} {
	seen := map[string]struct{}{}
	for index, value := range values {
		valueID := strings.TrimSpace(id(value))
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if valueID == "" {
			report.Add(ErrorMissingRequiredField, itemPath, "id is required")
			continue
		}
		if !validID(valueID) {
			report.Addf(ErrorInvalidID, itemPath, "id %q is not a stable AmandaCore content id", valueID)
		}
		if _, exists := seen[valueID]; exists {
			report.Addf(ErrorDuplicateID, itemPath, "id %q is duplicated", valueID)
		}
		seen[valueID] = struct{}{}
	}
	return seen
}

var stableIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)

func validID(value string) bool {
	return stableIDPattern.MatchString(strings.TrimSpace(value))
}

func requiredID(report *ContentValidationReport, path string, value string) {
	requiredString(report, path, value)
	if value != "" && !validID(value) {
		report.Addf(ErrorInvalidID, path, "id %q is not a stable AmandaCore content id", value)
	}
}

func requiredString(report *ContentValidationReport, path string, value string) {
	if strings.TrimSpace(value) == "" {
		report.Add(ErrorMissingRequiredField, path, "value is required")
	}
}

func validEnum(value string, allowed ...string) bool {
	normalized := strings.TrimSpace(strings.ToLower(value))
	for _, candidate := range allowed {
		if normalized == candidate {
			return true
		}
	}
	return false
}

func containsID(source map[string]struct{}, value string) bool {
	_, ok := source[value]
	return ok
}

func positionInBounds(position Position, bounds ZoneBounds) bool {
	return position.X >= bounds.MinX && position.X <= bounds.MaxX &&
		position.Y >= bounds.MinY && position.Y <= bounds.MaxY &&
		position.Z >= bounds.MinZ && position.Z <= bounds.MaxZ
}

func boundsContainBounds(outer ZoneBounds, inner ZoneBounds) bool {
	return inner.MinX >= outer.MinX && inner.MaxX <= outer.MaxX &&
		inner.MinY >= outer.MinY && inner.MaxY <= outer.MaxY &&
		inner.MinZ >= outer.MinZ && inner.MaxZ <= outer.MaxZ
}

func hasObjectiveCycle(nodes []QuestObjectiveNode) bool {
	graph := map[string][]string{}
	for _, node := range nodes {
		graph[node.NodeID] = append([]string(nil), node.DependsOn...)
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) bool
	visit = func(nodeID string) bool {
		if visiting[nodeID] {
			return true
		}
		if visited[nodeID] {
			return false
		}
		visiting[nodeID] = true
		defer func() {
			visiting[nodeID] = false
			visited[nodeID] = true
		}()
		for _, dependencyID := range graph[nodeID] {
			if _, exists := graph[dependencyID]; !exists {
				continue
			}
			if visit(dependencyID) {
				return true
			}
		}
		return false
	}
	for _, node := range nodes {
		if visit(node.NodeID) {
			return true
		}
	}
	return false
}

func logBrokenReference(sourceKind string, sourceID string, targetKind string, targetID string) {
	observability.LogEvent("content-loader", EventReferenceBroken, map[string]any{
		"sourceKind": sourceKind,
		"sourceId":   sourceID,
		"targetKind": targetKind,
		"targetId":   targetID,
	})
}

func ErrorSummary(report ContentValidationReport) error {
	if report.Valid() {
		return nil
	}
	messages := make([]string, 0, len(report.Errors))
	for _, validationError := range report.Errors {
		messages = append(messages, fmt.Sprintf("%s at %s: %s", validationError.Code, validationError.Path, validationError.Message))
	}
	return errors.New(strings.Join(messages, "; "))
}
