package worlds

import (
	"fmt"
	"time"

	contentpkg "amandacore/services/internal/content"
	"amandacore/services/internal/observability"
)

const defaultDawnwakePackagePath = "Content/Packs/dawnwake_isles/package.json"

type DawnwakeStreamingLoadsimOptions struct {
	Clients         int
	Duration        time.Duration
	CommandRate     float64
	Scenario        string
	ContentPath     string
	TransitionLoops int
	Seed            int64
	ShardCount      int
}

type DawnwakeStreamingLoadsimReport struct {
	Scenario                 string            `json:"scenario"`
	ContentPackageLoaded     bool              `json:"contentPackageLoaded"`
	PackageID                string            `json:"packageId"`
	ContinentID              string            `json:"continentId"`
	ValidationErrors         []string          `json:"validationErrors"`
	ZonesActivated           int               `json:"zonesActivated"`
	CatalogsLoaded           map[string]int    `json:"catalogsLoaded"`
	TransitionGatesLoaded    int               `json:"transitionGatesLoaded"`
	PlayersAttached          int               `json:"playersAttached"`
	ZoneTransitionsRequested int               `json:"zoneTransitionsRequested"`
	ZoneTransitionsCompleted int               `json:"zoneTransitionsCompleted"`
	ZoneTransitionsRejected  int               `json:"zoneTransitionsRejected"`
	TransitionLoops          int               `json:"transitionLoops"`
	Seed                     int64             `json:"seed"`
	ShardCount               int               `json:"shardCount"`
	ShardAssignments         map[string]string `json:"shardAssignments"`
	ShardPopulation          map[string]int    `json:"shardPopulation"`
	ZonePopulation           map[string]int    `json:"zonePopulation"`
	TransitionCounts         map[string]int    `json:"transitionCounts"`
	ZoneTransitionCounts     map[string]int    `json:"zoneTransitionCounts"`
	VisibilityEvaluations    int               `json:"visibilityEvaluations"`
	StreamingHintsEmitted    int               `json:"streamingHintsEmitted"`
	NPCsSpawned              int               `json:"npcsSpawned"`
	QuestProvidersRegistered int               `json:"questProvidersRegistered"`
	AverageTickDurationMs    float64           `json:"averageTickDurationMs"`
	P50TickDurationMs        float64           `json:"p50TickDurationMs"`
	P95TickDurationMs        float64           `json:"p95TickDurationMs"`
	P99TickDurationMs        float64           `json:"p99TickDurationMs"`
	MaxTickDurationMs        float64           `json:"maxTickDurationMs"`
	MaxQueueDepth            int               `json:"maxQueueDepth"`
	RejectedCommands         int               `json:"rejectedCommands"`
	Errors                   []string          `json:"errors"`
}

func RunDawnwakeStreamingLoadsim(opts DawnwakeStreamingLoadsimOptions) (DawnwakeStreamingLoadsimReport, error) {
	report := DawnwakeStreamingLoadsimReport{
		CatalogsLoaded:       map[string]int{},
		ShardAssignments:     map[string]string{},
		ShardPopulation:      map[string]int{},
		ZonePopulation:       map[string]int{},
		TransitionCounts:     map[string]int{},
		ZoneTransitionCounts: map[string]int{},
	}
	if opts.Clients <= 0 {
		opts.Clients = 1
	}
	if opts.Duration <= 0 {
		opts.Duration = 30 * time.Second
	}
	if opts.ContentPath == "" {
		opts.ContentPath = defaultDawnwakePackagePath
	}
	if opts.Scenario == "" {
		opts.Scenario = "dawnwake-streaming-basic"
	}
	multizone := opts.Scenario == "dawnwake-multizone-sharding-basic"
	if opts.Scenario != "dawnwake-streaming-basic" && opts.Scenario != "dawnwake-traversal-basic" && !multizone {
		return report, fmt.Errorf("unsupported loadsim scenario %q", opts.Scenario)
	}
	if opts.Seed == 0 {
		opts.Seed = 42
	}
	if opts.TransitionLoops <= 0 {
		opts.TransitionLoops = 1
		if multizone {
			opts.TransitionLoops = 3
		}
	}
	if opts.ShardCount <= 0 {
		opts.ShardCount = 2
	}
	report.Scenario = opts.Scenario
	report.Seed = opts.Seed
	report.TransitionLoops = opts.TransitionLoops
	report.ShardCount = opts.ShardCount

	observability.LogEvent("loadsim", contentpkg.EventLoadsimDawnwakeStarted, map[string]any{
		"scenario":        opts.Scenario,
		"clients":         opts.Clients,
		"content":         opts.ContentPath,
		"transitionLoops": opts.TransitionLoops,
		"seed":            opts.Seed,
		"shardCount":      opts.ShardCount,
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

	registry := loadResult.Validated.Registry
	continent := firstDawnwakeContinent(registry)
	if continent.ContinentID == "" {
		err := fmt.Errorf("dawnwake package has no continent definition")
		report.Errors = append(report.Errors, err.Error())
		return report, err
	}
	report.ContinentID = continent.ContinentID
	report.TransitionGatesLoaded = countContentTransitionGates(registry)

	server := newWorldServerWithContentPackage(nil, opts.ContentPath)
	report.ZonesActivated = server.contentActivation.ZonesActivated
	report.QuestProvidersRegistered = server.contentActivation.QuestProvidersRegistered
	report.NPCsSpawned = countContentMobs(server)

	assignments, err := BuildContentZoneShardAssignments(registry, ShardAssignmentPolicy{ShardCount: opts.ShardCount})
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		return report, err
	}
	report.ShardAssignments = shardAssignmentSummary(assignments)

	entryZoneID := continent.DefaultEntry.ZoneID
	if entryZoneID == "" {
		entryZoneID, _ = firstContentEntryPoint(registry)
	}
	if entryZoneID == "" {
		err := fmt.Errorf("dawnwake package has no default entry zone")
		report.Errors = append(report.Errors, err.Error())
		return report, err
	}

	for index := 0; index < opts.Clients; index++ {
		report.PlayersAttached++
		currentZoneID := entryZoneID
		report.VisibilityEvaluations++
		report.StreamingHintsEmitted += adjacentHintCount(continent, currentZoneID)

		for step := 0; step < opts.TransitionLoops; step++ {
			gate, found := selectEnabledTransition(registry.Zones[currentZoneID], index, step, opts.Seed)
			if !found {
				report.ZoneTransitionsRejected++
				report.RejectedCommands++
				report.Errors = append(report.Errors, fmt.Sprintf("client %d zone %q has no enabled transition gate", index+1, currentZoneID))
				break
			}
			report.ZoneTransitionsRequested++
			if _, zoneFound := registry.Zones[gate.ToZoneID]; !zoneFound {
				report.ZoneTransitionsRejected++
				report.RejectedCommands++
				report.Errors = append(report.Errors, fmt.Sprintf("client %d transition %q has missing destination zone %q", index+1, gate.TransitionID, gate.ToZoneID))
				break
			}
			if !zoneHasContentEntryPoint(registry.Zones[gate.ToZoneID], gate.EntryPointIDOnArrival) {
				report.ZoneTransitionsRejected++
				report.RejectedCommands++
				report.Errors = append(report.Errors, fmt.Sprintf("client %d transition %q has missing arrival entry %q", index+1, gate.TransitionID, gate.EntryPointIDOnArrival))
				break
			}
			if _, err := ResolveZoneShard(assignments, gate.ToZoneID); err != nil {
				report.ZoneTransitionsRejected++
				report.RejectedCommands++
				report.Errors = append(report.Errors, fmt.Sprintf("client %d transition %q has no destination shard: %v", index+1, gate.TransitionID, err))
				break
			}
			report.ZoneTransitionsCompleted++
			report.TransitionCounts[gate.TransitionID]++
			report.ZoneTransitionCounts[currentZoneID]++
			currentZoneID = gate.ToZoneID
			report.VisibilityEvaluations++
			report.StreamingHintsEmitted += adjacentHintCount(continent, currentZoneID)
		}
		report.ZonePopulation[currentZoneID]++
		if assignment, err := ResolveZoneShard(assignments, currentZoneID); err == nil {
			report.ShardPopulation[string(assignment.ShardID)]++
		}
	}

	tickStats := runContentTickStats(server, opts.Duration)
	report.AverageTickDurationMs = tickStats.AverageMs
	report.P50TickDurationMs = tickStats.P50Ms
	report.P95TickDurationMs = tickStats.P95Ms
	report.P99TickDurationMs = tickStats.P99Ms
	report.MaxTickDurationMs = tickStats.MaxMs
	report.MaxQueueDepth = opts.Clients * opts.TransitionLoops
	observability.LogEvent("loadsim", contentpkg.EventLoadsimDawnwakeCompleted, map[string]any{
		"packageId":             report.PackageID,
		"continentId":           report.ContinentID,
		"scenario":              report.Scenario,
		"zonesActivated":        report.ZonesActivated,
		"transitionGatesLoaded": report.TransitionGatesLoaded,
		"transitionsCompleted":  report.ZoneTransitionsCompleted,
		"transitionLoops":       report.TransitionLoops,
		"shardCount":            report.ShardCount,
		"visibilityEvaluations": report.VisibilityEvaluations,
		"averageTickDurationMs": report.AverageTickDurationMs,
		"p95TickDurationMs":     report.P95TickDurationMs,
		"maxTickDurationMs":     report.MaxTickDurationMs,
		"errorCount":            len(report.Errors),
	})
	return report, nil
}

func firstDawnwakeContinent(registry contentpkg.RuntimeContentRegistry) contentpkg.ContinentDefinition {
	if continent, found := registry.Continents["dawnwake_isles"]; found {
		return continent
	}
	for _, continentID := range contentpkg.SortedKeys(registry.Continents) {
		return registry.Continents[continentID]
	}
	return contentpkg.ContinentDefinition{}
}

func countContentTransitionGates(registry contentpkg.RuntimeContentRegistry) int {
	total := 0
	for _, zoneID := range contentpkg.SortedKeys(registry.Zones) {
		total += len(registry.Zones[zoneID].TransitionGates)
	}
	return total
}

func firstEnabledTransition(zone contentpkg.ZoneDefinition) (contentpkg.ZoneTransitionGate, bool) {
	for _, gate := range zone.TransitionGates {
		if !gate.Disabled {
			return gate, true
		}
	}
	return contentpkg.ZoneTransitionGate{}, false
}

func selectEnabledTransition(zone contentpkg.ZoneDefinition, clientIndex int, step int, seed int64) (contentpkg.ZoneTransitionGate, bool) {
	enabled := make([]contentpkg.ZoneTransitionGate, 0, len(zone.TransitionGates))
	for _, gate := range zone.TransitionGates {
		if !gate.Disabled {
			enabled = append(enabled, gate)
		}
	}
	if len(enabled) == 0 {
		return contentpkg.ZoneTransitionGate{}, false
	}
	offset := int(seed % int64(len(enabled)))
	if offset < 0 {
		offset = -offset
	}
	return enabled[(clientIndex+step+offset)%len(enabled)], true
}

func zoneHasContentEntryPoint(zone contentpkg.ZoneDefinition, entryPointID string) bool {
	for _, entry := range zone.EntryPoints {
		if entry.EntryID == entryPointID {
			return true
		}
	}
	return false
}

func adjacentHintCount(continent contentpkg.ContinentDefinition, zoneID string) int {
	count := 0
	for _, adjacency := range continent.Adjacency {
		if adjacency.FromZoneID == zoneID || adjacency.ToZoneID == zoneID {
			count++
		}
	}
	return count
}
