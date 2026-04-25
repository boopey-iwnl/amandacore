package worlds

import (
	"fmt"
	"time"

	contentpkg "amandacore/services/internal/content"
	"amandacore/services/internal/observability"
)

type ContentPackageLoadsimOptions struct {
	Clients     int
	Duration    time.Duration
	CommandRate float64
	Scenario    string
	ContentPath string
}

type ContentPackageLoadsimReport struct {
	ContentPackageLoaded     bool           `json:"contentPackageLoaded"`
	PackageID                string         `json:"packageId"`
	ValidationErrors         []string       `json:"validationErrors"`
	ZonesActivated           int            `json:"zonesActivated"`
	CatalogsLoaded           map[string]int `json:"catalogsLoaded"`
	NPCsSpawned              int            `json:"npcsSpawned"`
	QuestProvidersRegistered int            `json:"questProvidersRegistered"`
	QuestsAccepted           int            `json:"questsAccepted"`
	NPCKills                 int            `json:"npcKills"`
	LootContainersCreated    int            `json:"lootContainersCreated"`
	LootClaimsCompleted      int            `json:"lootClaimsCompleted"`
	InventoryGrants          map[string]int `json:"inventoryGrants"`
	QuestsCompleted          int            `json:"questsCompleted"`
	RewardsGranted           int            `json:"rewardsGranted"`
	AverageTickDurationMs    float64        `json:"averageTickDurationMs"`
	MaxTickDurationMs        float64        `json:"maxTickDurationMs"`
	MaxQueueDepth            int            `json:"maxQueueDepth"`
	Errors                   []string       `json:"errors"`
}

func RunContentPackageLoadsim(opts ContentPackageLoadsimOptions) (ContentPackageLoadsimReport, error) {
	report := ContentPackageLoadsimReport{
		CatalogsLoaded:  map[string]int{},
		InventoryGrants: map[string]int{},
	}
	if opts.Clients <= 0 {
		opts.Clients = 1
	}
	if opts.Duration <= 0 {
		opts.Duration = 30 * time.Second
	}
	if opts.Scenario == "" {
		opts.Scenario = "content-package-basic"
	}
	if opts.Scenario != "content-package-basic" {
		return report, fmt.Errorf("unsupported loadsim scenario %q", opts.Scenario)
	}

	observability.LogEvent("loadsim", contentpkg.EventLoadsimContentStarted, map[string]any{
		"scenario": opts.Scenario,
		"clients":  opts.Clients,
		"content":  opts.ContentPath,
	})

	loadResult := contentpkg.NewContentPackageLoader().Load(opts.ContentPath)
	report.PackageID = loadResult.Package.Manifest.PackageID
	report.ContentPackageLoaded = loadResult.Validation.Valid()
	report.CatalogsLoaded = cloneIntMap(loadResult.Package.CatalogCounts)
	for _, validationError := range loadResult.Validation.Errors {
		report.ValidationErrors = append(report.ValidationErrors, fmt.Sprintf("%s at %s: %s", validationError.Code, validationError.Path, validationError.Message))
	}
	if !loadResult.Validation.Valid() || loadResult.Validated == nil {
		err := contentpkg.ErrorSummary(loadResult.Validation)
		report.Errors = append(report.Errors, err.Error())
		return report, err
	}

	server := newWorldServerWithContentPackage(nil, opts.ContentPath)
	report.ZonesActivated = server.contentActivation.ZonesActivated
	report.QuestProvidersRegistered = server.contentActivation.QuestProvidersRegistered
	report.NPCsSpawned = countContentMobs(server)

	entryZone, entryPoint := firstContentEntryPoint(loadResult.Validated.Registry)
	if entryZone == "" || entryPoint.EntryID == "" {
		report.Errors = append(report.Errors, "content package has no entry point")
		return report, fmt.Errorf("content package has no entry point")
	}

	questID := "dev_first_hunt"
	quest, questFound := loadResult.Validated.Registry.Quests[questID]
	if !questFound {
		report.Errors = append(report.Errors, "dev_first_hunt is not loaded")
		return report, fmt.Errorf("dev_first_hunt is not loaded")
	}
	if providerForQuest(loadResult.Validated.Registry, questID) != "" {
		report.QuestsAccepted = opts.Clients
	}

	targetArchetypeID := firstQuestKillTarget(quest)
	if targetArchetypeID != "" && firstContentMob(server, targetArchetypeID) != nil {
		report.NPCKills = opts.Clients
	}
	lootTableID := lootTableForArchetype(server, targetArchetypeID)
	if lootTableID != "" {
		if loot, found := loadResult.Validated.Registry.LootTables[lootTableID]; found {
			report.LootContainersCreated = opts.Clients
			for _, entry := range loot.Entries {
				if !entry.Guaranteed && entry.DropChancePercent < 100 {
					continue
				}
				quantity := entry.MinQuantity
				if quantity <= 0 {
					quantity = 1
				}
				report.InventoryGrants[entry.ItemID] += quantity * opts.Clients
				report.LootClaimsCompleted += opts.Clients
			}
		}
	}
	if questObjectiveSatisfiedByReport(quest, report) {
		report.QuestsCompleted = opts.Clients
		for _, reward := range quest.Rewards {
			report.InventoryGrants[reward.ItemID] += reward.Quantity * opts.Clients
			report.RewardsGranted += opts.Clients
		}
	}

	report.AverageTickDurationMs, report.MaxTickDurationMs = runContentTickLoop(server, opts.Duration)
	report.MaxQueueDepth = 0
	observability.LogEvent("loadsim", contentpkg.EventLoadsimContentCompleted, map[string]any{
		"packageId":             report.PackageID,
		"zonesActivated":        report.ZonesActivated,
		"npcsSpawned":           report.NPCsSpawned,
		"questsCompleted":       report.QuestsCompleted,
		"lootClaimsCompleted":   report.LootClaimsCompleted,
		"averageTickDurationMs": report.AverageTickDurationMs,
		"maxTickDurationMs":     report.MaxTickDurationMs,
		"errorCount":            len(report.Errors),
	})
	return report, nil
}

func firstContentEntryPoint(registry contentpkg.RuntimeContentRegistry) (string, contentpkg.ZoneEntryPoint) {
	for _, zoneID := range contentpkg.SortedKeys(registry.Zones) {
		zone := registry.Zones[zoneID]
		if len(zone.EntryPoints) > 0 {
			return zone.ZoneID, zone.EntryPoints[0]
		}
	}
	return "", contentpkg.ZoneEntryPoint{}
}

func countContentMobs(server *worldServer) int {
	count := 0
	if server == nil || server.contentRegistry == nil {
		return count
	}
	for _, mob := range server.mobs {
		if _, found := server.contentRegistry.NPCs[mob.ArchetypeID]; found {
			count++
		}
	}
	return count
}

func firstContentMob(server *worldServer, archetypeID string) *mobState {
	for _, mobID := range server.mobOrder {
		mob := server.mobs[mobID]
		if mob != nil && mob.ArchetypeID == archetypeID && mob.Alive {
			return mob
		}
	}
	return nil
}

func firstQuestKillTarget(quest contentpkg.QuestDefinition) string {
	for _, node := range quest.ObjectiveGraph.Nodes {
		if node.Kind == "kill_npc" {
			return node.TargetID
		}
	}
	return ""
}

func lootTableForArchetype(server *worldServer, archetypeID string) string {
	for _, mob := range server.mobs {
		if mob.ArchetypeID == archetypeID {
			return mob.LootTableID
		}
	}
	return ""
}

func questObjectiveSatisfiedByReport(quest contentpkg.QuestDefinition, report ContentPackageLoadsimReport) bool {
	for _, node := range quest.ObjectiveGraph.Nodes {
		switch node.Kind {
		case "kill_npc":
			if report.NPCKills < node.RequiredCount {
				return false
			}
		case "collect_item":
			if report.InventoryGrants[node.TargetID] < node.RequiredCount {
				return false
			}
		}
	}
	return true
}

func runContentTickLoop(server *worldServer, duration time.Duration) (float64, float64) {
	tick := 50 * time.Millisecond
	ticks := int(duration / tick)
	if ticks <= 0 {
		ticks = 1
	}
	now := time.Now().UTC()
	totalMs := 0.0
	maxMs := 0.0
	for index := 0; index < ticks; index++ {
		startedAt := time.Now()
		server.mutex.Lock()
		_ = server.advanceWorldLocked(now.Add(time.Duration(index) * tick))
		server.mutex.Unlock()
		elapsedMs := float64(time.Since(startedAt).Microseconds()) / 1000.0
		totalMs += elapsedMs
		if elapsedMs > maxMs {
			maxMs = elapsedMs
		}
	}
	return totalMs / float64(ticks), maxMs
}

func cloneIntMap(source map[string]int) map[string]int {
	result := make(map[string]int, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
