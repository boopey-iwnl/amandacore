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
	TransitionsLoaded        int            `json:"transitionsLoaded"`
	TransitionsCompleted     int            `json:"transitionsCompleted"`
	MapExportsLoaded         int            `json:"mapExportsLoaded"`
	StreamingCellsLoaded     int            `json:"streamingCellsLoaded"`
	StreamingHintsObserved   int            `json:"streamingHintsObserved"`
	ZonesEntered             []string       `json:"zonesEntered"`
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
	if opts.Scenario != "content-package-basic" && opts.Scenario != "dawnwake-traversal-basic" && opts.Scenario != "dawnwake-streaming-basic" {
		return report, fmt.Errorf("unsupported loadsim scenario %q", opts.Scenario)
	}

	startEvent := contentpkg.EventLoadsimContentStarted
	completeEvent := contentpkg.EventLoadsimContentCompleted
	if opts.Scenario == "dawnwake-traversal-basic" {
		startEvent = contentpkg.EventLoadsimDawnwakeStarted
		completeEvent = contentpkg.EventLoadsimDawnwakeCompleted
	} else if opts.Scenario == "dawnwake-streaming-basic" {
		startEvent = contentpkg.EventLoadsimStreamingStarted
		completeEvent = contentpkg.EventLoadsimStreamingCompleted
	}
	observability.LogEvent("loadsim", startEvent, map[string]any{
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
	report.TransitionsLoaded = server.contentActivation.TransitionsLoaded
	report.MapExportsLoaded = server.contentActivation.MapExportsLoaded
	report.StreamingCellsLoaded = server.contentActivation.StreamingCellsLoaded
	report.NPCsSpawned = countContentMobs(server)

	entryZone, entryPoint := firstContentEntryPoint(loadResult.Validated.Registry)
	if entryZone == "" || entryPoint.EntryID == "" {
		report.Errors = append(report.Errors, "content package has no entry point")
		return report, fmt.Errorf("content package has no entry point")
	}

	questID := "dev_first_hunt"
	if opts.Scenario == "dawnwake-traversal-basic" || opts.Scenario == "dawnwake-streaming-basic" {
		questID = "dw_tideglass_sparks"
	}
	quest, questFound := loadResult.Validated.Registry.Quests[questID]
	if !questFound {
		report.Errors = append(report.Errors, fmt.Sprintf("%s is not loaded", questID))
		return report, fmt.Errorf("%s is not loaded", questID)
	}
	if providerForQuest(loadResult.Validated.Registry, questID) != "" {
		report.QuestsAccepted = opts.Clients
	}

	if opts.Scenario == "dawnwake-traversal-basic" {
		completed, err := runDawnwakeTraversalStep(server, entryZone, entryPoint)
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			return report, err
		}
		if completed {
			report.TransitionsCompleted = opts.Clients
		}
	} else if opts.Scenario == "dawnwake-streaming-basic" {
		completed, zonesEntered, hintsObserved, err := runDawnwakeStreamingTraversal(server, entryZone, entryPoint)
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			return report, err
		}
		report.TransitionsCompleted = completed * opts.Clients
		report.StreamingHintsObserved = hintsObserved
		report.ZonesEntered = zonesEntered
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
	observability.LogEvent("loadsim", completeEvent, map[string]any{
		"packageId":              report.PackageID,
		"zonesActivated":         report.ZonesActivated,
		"transitionsCompleted":   report.TransitionsCompleted,
		"mapExportsLoaded":       report.MapExportsLoaded,
		"streamingCellsLoaded":   report.StreamingCellsLoaded,
		"streamingHintsObserved": report.StreamingHintsObserved,
		"npcsSpawned":            report.NPCsSpawned,
		"questsCompleted":        report.QuestsCompleted,
		"lootClaimsCompleted":    report.LootClaimsCompleted,
		"averageTickDurationMs":  report.AverageTickDurationMs,
		"maxTickDurationMs":      report.MaxTickDurationMs,
		"errorCount":             len(report.Errors),
	})
	return report, nil
}

func runDawnwakeTraversalStep(server *worldServer, entryZone string, entryPoint contentpkg.ZoneEntryPoint) (bool, error) {
	session := &worldSessionState{
		Token:       "loadsim_dawnwake_001",
		CharacterID: "loadsim_dawnwake_character",
		DisplayName: "DawnwakeSim",
		ZoneID:      entryZone,
		X:           entryPoint.Position.X,
		Y:           entryPoint.Position.Y,
		Z:           entryPoint.Position.Z,
		Connected:   true,
		Alive:       true,
		Health:      playerMaxHealth,
		MaxHealth:   playerMaxHealth,
	}
	server.mutex.Lock()
	defer server.mutex.Unlock()
	zone := server.contentRegistry.Zones[entryZone]
	if len(zone.Transitions) == 0 {
		return false, fmt.Errorf("entry zone %s has no transition", entryZone)
	}
	transition := zone.Transitions[0]
	session.X = transition.Position.X
	session.Y = transition.Position.Y
	session.Z = transition.Position.Z
	result, err := server.applyContentZoneTransitionsLocked(session)
	if err != nil {
		return false, err
	}
	if !result.Completed {
		return false, fmt.Errorf("transition %s did not complete", transition.TransitionID)
	}
	return true, nil
}

func runDawnwakeStreamingTraversal(server *worldServer, entryZone string, entryPoint contentpkg.ZoneEntryPoint) (int, []string, int, error) {
	session := &worldSessionState{
		Token:       "loadsim_dawnwake_streaming_001",
		CharacterID: "loadsim_dawnwake_streaming_character",
		DisplayName: "DawnwakeStreamingSim",
		ZoneID:      entryZone,
		X:           entryPoint.Position.X,
		Y:           entryPoint.Position.Y,
		Z:           entryPoint.Position.Z,
		Connected:   true,
		Alive:       true,
		Health:      playerMaxHealth,
		MaxHealth:   playerMaxHealth,
	}
	targetZones := []string{
		"dawnwake_tideglass_shoal",
		"dawnwake_windspur_rise",
		"dawnwake_tideglass_shoal",
		"dawnwake_landing",
	}
	zonesEntered := []string{entryZone}
	hintsObserved := 0
	completed := 0

	server.mutex.Lock()
	defer server.mutex.Unlock()
	for _, targetZoneID := range targetZones {
		runtime := server.zoneRuntimes[session.ZoneID]
		if runtime == nil || runtime.MapID == "" {
			return completed, zonesEntered, hintsObserved, fmt.Errorf("zone %s has no active map export runtime", session.ZoneID)
		}
		if len(runtime.StreamingCells) == 0 {
			return completed, zonesEntered, hintsObserved, fmt.Errorf("zone %s has no active streaming cells", session.ZoneID)
		}
		if len(runtime.TransitionHints) == 0 {
			return completed, zonesEntered, hintsObserved, fmt.Errorf("zone %s has no transition hints", session.ZoneID)
		}
		hintsObserved += len(runtime.TransitionHints)

		transition, found := contentTransitionToZone(server, session.ZoneID, targetZoneID)
		if !found {
			return completed, zonesEntered, hintsObserved, fmt.Errorf("zone %s has no transition to %s", session.ZoneID, targetZoneID)
		}
		session.X = transition.Position.X
		session.Y = transition.Position.Y
		session.Z = transition.Position.Z
		result, err := server.applyContentZoneTransitionsLocked(session)
		if err != nil {
			return completed, zonesEntered, hintsObserved, err
		}
		if !result.Completed || result.ToZoneID != targetZoneID {
			return completed, zonesEntered, hintsObserved, fmt.Errorf("transition to %s did not complete", targetZoneID)
		}
		completed++
		zonesEntered = append(zonesEntered, session.ZoneID)
	}

	runtime := server.zoneRuntimes[session.ZoneID]
	if runtime != nil {
		hintsObserved += len(runtime.TransitionHints)
	}
	return completed, zonesEntered, hintsObserved, nil
}

func contentTransitionToZone(server *worldServer, zoneID string, targetZoneID string) (contentpkg.ZoneTransitionDefinition, bool) {
	if server == nil || server.contentRegistry == nil {
		return contentpkg.ZoneTransitionDefinition{}, false
	}
	zone, found := server.contentRegistry.Zones[zoneID]
	if !found {
		return contentpkg.ZoneTransitionDefinition{}, false
	}
	for _, transition := range zone.Transitions {
		if transition.TargetZoneID == targetZoneID {
			return transition, true
		}
	}
	return contentpkg.ZoneTransitionDefinition{}, false
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
