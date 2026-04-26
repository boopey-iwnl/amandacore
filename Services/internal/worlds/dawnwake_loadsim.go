package worlds

import (
	"fmt"
	"time"

	contentpkg "amandacore/services/internal/content"
	"amandacore/services/internal/observability"
)

const defaultDawnwakePackagePath = "Content/Packs/dawnwake_isles/package.json"

type DawnwakeStreamingLoadsimOptions struct {
	Clients     int
	Duration    time.Duration
	CommandRate float64
	Scenario    string
	ContentPath string
}

type DawnwakeStreamingLoadsimReport struct {
	ContentPackageLoaded     bool           `json:"contentPackageLoaded"`
	PackageID                string         `json:"packageId"`
	ContinentID              string         `json:"continentId"`
	ValidationErrors         []string       `json:"validationErrors"`
	ZonesActivated           int            `json:"zonesActivated"`
	CatalogsLoaded           map[string]int `json:"catalogsLoaded"`
	TransitionGatesLoaded    int            `json:"transitionGatesLoaded"`
	PlayersAttached          int            `json:"playersAttached"`
	ZoneTransitionsRequested int            `json:"zoneTransitionsRequested"`
	ZoneTransitionsCompleted int            `json:"zoneTransitionsCompleted"`
	ZoneTransitionsRejected  int            `json:"zoneTransitionsRejected"`
	VisibilityEvaluations    int            `json:"visibilityEvaluations"`
	StreamingHintsEmitted    int            `json:"streamingHintsEmitted"`
	NPCsSpawned              int            `json:"npcsSpawned"`
	QuestProvidersRegistered int            `json:"questProvidersRegistered"`
	AverageTickDurationMs    float64        `json:"averageTickDurationMs"`
	MaxTickDurationMs        float64        `json:"maxTickDurationMs"`
	MaxQueueDepth            int            `json:"maxQueueDepth"`
	Errors                   []string       `json:"errors"`
}

func RunDawnwakeStreamingLoadsim(opts DawnwakeStreamingLoadsimOptions) (DawnwakeStreamingLoadsimReport, error) {
	report := DawnwakeStreamingLoadsimReport{
		CatalogsLoaded: map[string]int{},
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
	if opts.Scenario != "dawnwake-streaming-basic" && opts.Scenario != "dawnwake-traversal-basic" {
		return report, fmt.Errorf("unsupported loadsim scenario %q", opts.Scenario)
	}

	observability.LogEvent("loadsim", contentpkg.EventLoadsimDawnwakeStarted, map[string]any{
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
		report.VisibilityEvaluations++
		report.StreamingHintsEmitted += adjacentHintCount(continent, entryZoneID)

		gate, found := firstEnabledTransition(registry.Zones[entryZoneID])
		if !found {
			report.ZoneTransitionsRejected++
			report.Errors = append(report.Errors, fmt.Sprintf("client %d default zone %q has no enabled transition gate", index+1, entryZoneID))
			continue
		}
		report.ZoneTransitionsRequested++
		if _, zoneFound := registry.Zones[gate.ToZoneID]; !zoneFound {
			report.ZoneTransitionsRejected++
			report.Errors = append(report.Errors, fmt.Sprintf("client %d transition %q has missing destination zone %q", index+1, gate.TransitionID, gate.ToZoneID))
			continue
		}
		if !zoneHasContentEntryPoint(registry.Zones[gate.ToZoneID], gate.EntryPointIDOnArrival) {
			report.ZoneTransitionsRejected++
			report.Errors = append(report.Errors, fmt.Sprintf("client %d transition %q has missing arrival entry %q", index+1, gate.TransitionID, gate.EntryPointIDOnArrival))
			continue
		}
		report.ZoneTransitionsCompleted++
		report.VisibilityEvaluations++
		report.StreamingHintsEmitted += adjacentHintCount(continent, gate.ToZoneID)
	}

	report.AverageTickDurationMs, report.MaxTickDurationMs = runContentTickLoop(server, opts.Duration)
	report.MaxQueueDepth = opts.Clients
	observability.LogEvent("loadsim", contentpkg.EventLoadsimDawnwakeCompleted, map[string]any{
		"packageId":             report.PackageID,
		"continentId":           report.ContinentID,
		"zonesActivated":        report.ZonesActivated,
		"transitionGatesLoaded": report.TransitionGatesLoaded,
		"transitionsCompleted":  report.ZoneTransitionsCompleted,
		"visibilityEvaluations": report.VisibilityEvaluations,
		"averageTickDurationMs": report.AverageTickDurationMs,
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
