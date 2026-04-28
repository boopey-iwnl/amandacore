package content

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"amandacore/services/internal/observability"
)

const CompiledContentProtocol = "amandacore.content.v1"

type ContentCompileResult struct {
	Package    CompiledContentPackage  `json:"package"`
	Validation ContentValidationReport `json:"validation"`
}

type CompiledContentPackage struct {
	Protocol      string                   `json:"protocol"`
	PackageID     string                   `json:"package_id"`
	Version       string                   `json:"version"`
	SchemaVersion string                   `json:"schema_version"`
	DisplayName   string                   `json:"display_name"`
	CleanRoomNote string                   `json:"clean_room_note"`
	Catalogs      []CompiledCatalogSummary `json:"catalogs"`
	Files         []string                 `json:"files"`
	IDs           CompiledContentIDs       `json:"ids"`
	Hooks         []CompiledHookSummary    `json:"hooks"`
	ContentSHA256 string                   `json:"content_sha256"`
}

type CompiledCatalogSummary struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

type CompiledContentIDs struct {
	Continents   []string `json:"continents,omitempty"`
	Zones        []string `json:"zones"`
	MapExports   []string `json:"map_exports,omitempty"`
	NPCs         []string `json:"npcs"`
	Items        []string `json:"items"`
	LootTables   []string `json:"loot_tables"`
	Quests       []string `json:"quests"`
	Abilities    []string `json:"abilities"`
	Auras        []string `json:"auras,omitempty"`
	Vendors      []string `json:"vendors,omitempty"`
	Trainers     []string `json:"trainers,omitempty"`
	Dialogues    []string `json:"dialogues,omitempty"`
	HookBindings []string `json:"hook_bindings,omitempty"`
}

type CompiledHookSummary struct {
	BindingID string   `json:"binding_id"`
	Hook      string   `json:"hook"`
	SourceID  string   `json:"source_id"`
	Priority  int      `json:"priority"`
	Actions   []string `json:"actions"`
}

func CompileContentPackage(manifestPath string) ContentCompileResult {
	loadResult := NewContentPackageLoader().Load(manifestPath)
	if !loadResult.Validation.Valid() {
		observability.LogEvent("content-compiler", observability.EventContentValidationFailed, map[string]any{
			"errorCount": len(loadResult.Validation.Errors),
		})
		return ContentCompileResult{Validation: loadResult.Validation}
	}
	return CompileLoadedContentPackage(loadResult.Package)
}

func CompileLoadedContentPackage(loaded LoadedContentPackage) ContentCompileResult {
	validation := ValidateLoadedContentPackage(loaded)
	if !validation.Valid() {
		observability.LogEvent("content-compiler", observability.EventContentValidationFailed, map[string]any{
			"packageId":  loaded.Manifest.PackageID,
			"errorCount": len(validation.Errors),
		})
		return ContentCompileResult{Validation: validation}
	}
	compiled := buildCompiledContentPackage(loaded)
	compiled.ContentSHA256 = compiledContentHash(compiled)
	observability.LogEvent("content-compiler", observability.EventContentPackageCompiled, map[string]any{
		"packageId": loaded.Manifest.PackageID,
		"version":   loaded.Manifest.Version,
		"hash":      compiled.ContentSHA256,
	})
	return ContentCompileResult{Package: compiled, Validation: validation}
}

func MarshalCompiledContentPackage(compiled CompiledContentPackage) ([]byte, error) {
	return json.MarshalIndent(compiled, "", "  ")
}

func buildCompiledContentPackage(loaded LoadedContentPackage) CompiledContentPackage {
	return CompiledContentPackage{
		Protocol:      CompiledContentProtocol,
		PackageID:     loaded.Manifest.PackageID,
		Version:       loaded.Manifest.Version,
		SchemaVersion: loaded.Manifest.SchemaVersion,
		DisplayName:   loaded.Manifest.DisplayName,
		CleanRoomNote: "AmandaCore-original content package; no external MMO schemas, IDs, scripts, text, or data.",
		Catalogs:      compiledCatalogSummaries(loaded),
		Files:         compiledPackageFiles(loaded.Manifest),
		IDs: CompiledContentIDs{
			Continents:   sortedIDs(loaded.Continents, func(item ContinentDefinition) string { return item.ContinentID }),
			Zones:        sortedIDs(loaded.Zones, func(item ZoneDefinition) string { return item.ZoneID }),
			MapExports:   sortedIDs(loaded.MapExports, func(item MapExportDefinition) string { return item.MapID }),
			NPCs:         sortedIDs(loaded.NPCs, func(item NpcArchetype) string { return item.ArchetypeID }),
			Items:        sortedIDs(loaded.Items, func(item ItemDefinition) string { return item.ItemID }),
			LootTables:   sortedIDs(loaded.LootTables, func(item LootTableDefinition) string { return item.LootTableID }),
			Quests:       sortedIDs(loaded.Quests, func(item QuestDefinition) string { return item.QuestID }),
			Abilities:    sortedIDs(loaded.Abilities, func(item AbilityDefinition) string { return item.AbilityID }),
			Auras:        sortedIDs(loaded.Auras, func(item AuraDefinition) string { return item.AuraID }),
			Vendors:      sortedIDs(loaded.Vendors, func(item VendorDefinition) string { return item.VendorID }),
			Trainers:     sortedIDs(loaded.Trainers, func(item TrainerDefinition) string { return item.TrainerID }),
			Dialogues:    sortedIDs(loaded.Dialogues, func(item DialogueDefinition) string { return item.DialogueID }),
			HookBindings: sortedIDs(loaded.HookBindings, func(item HookBindingDefinition) string { return item.BindingID }),
		},
		Hooks: compiledHookSummaries(loaded.HookBindings),
	}
}

func compiledContentHash(compiled CompiledContentPackage) string {
	withoutHash := compiled
	withoutHash.ContentSHA256 = ""
	payload, _ := json.Marshal(withoutHash)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func compiledCatalogSummaries(loaded LoadedContentPackage) []CompiledCatalogSummary {
	catalogs := []CompiledCatalogSummary{
		{Kind: "continents", Count: len(loaded.Continents)},
		{Kind: "zones", Count: len(loaded.Zones)},
		{Kind: "map_exports", Count: len(loaded.MapExports)},
		{Kind: "npcs", Count: len(loaded.NPCs)},
		{Kind: "items", Count: len(loaded.Items)},
		{Kind: "loot_tables", Count: len(loaded.LootTables)},
		{Kind: "quests", Count: len(loaded.Quests)},
		{Kind: "abilities", Count: len(loaded.Abilities)},
		{Kind: "auras", Count: len(loaded.Auras)},
		{Kind: "vendors", Count: len(loaded.Vendors)},
		{Kind: "trainers", Count: len(loaded.Trainers)},
		{Kind: "dialogues", Count: len(loaded.Dialogues)},
		{Kind: "hook_bindings", Count: len(loaded.HookBindings)},
	}
	sort.Slice(catalogs, func(i int, j int) bool {
		return catalogs[i].Kind < catalogs[j].Kind
	})
	return catalogs
}

func compiledPackageFiles(manifest ContentPackageManifest) []string {
	files := make([]string, 0)
	files = append(files, manifest.ContinentFiles...)
	files = append(files, manifest.Zones...)
	files = append(files, manifest.MapExports...)
	files = append(files, manifest.NPCCatalogs...)
	files = append(files, manifest.ItemCatalogs...)
	files = append(files, manifest.LootCatalogs...)
	files = append(files, manifest.QuestCatalogs...)
	files = append(files, manifest.AbilityCatalogs...)
	files = append(files, manifest.AuraCatalogs...)
	files = append(files, manifest.VendorCatalogs...)
	files = append(files, manifest.TrainerCatalogs...)
	files = append(files, manifest.DialogueCatalogs...)
	files = append(files, manifest.HookCatalogs...)
	sort.Strings(files)
	return files
}

func compiledHookSummaries(bindings []HookBindingDefinition) []CompiledHookSummary {
	copied := append([]HookBindingDefinition(nil), bindings...)
	sortHookBindings(copied)
	result := make([]CompiledHookSummary, 0, len(copied))
	for _, binding := range copied {
		actions := make([]string, 0, len(binding.Actions))
		for _, action := range binding.Actions {
			actions = append(actions, action.Action)
		}
		sort.Strings(actions)
		result = append(result, CompiledHookSummary{
			BindingID: binding.BindingID,
			Hook:      binding.Hook,
			SourceID:  binding.SourceID,
			Priority:  binding.Priority,
			Actions:   actions,
		})
	}
	return result
}

func sortedIDs[T any](values []T, id func(T) string) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		if next := strings.TrimSpace(id(value)); next != "" {
			ids = append(ids, next)
		}
	}
	sort.Strings(ids)
	return ids
}
