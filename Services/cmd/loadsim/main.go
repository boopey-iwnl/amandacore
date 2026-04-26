package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/worlds"
)

const (
	scenarioDawnwakeTraversalBasic   = "dawnwake-traversal-basic"
	scenarioDawnwakeMultizoneSharded = "dawnwake-multizone-sharding"
	scenarioDawnwakePopulation       = "dawnwake-multizone-population"
	scenarioDawnwakePopulationAlias  = "dawnwake-population"
	scenarioDawnwakeTransitionStress = "dawnwake-transition-stress"
	scenarioDawnwakeCommandPressure  = "dawnwake-command-pressure"
	scenarioDawnwakeZoneIsolation    = "dawnwake-zone-isolation"
)

type options struct {
	Clients          int
	Duration         time.Duration
	CmdRate          int
	Scenario         string
	Content          string
	Shards           int
	QueueLimit       int
	TransitionRate   float64
	MovementPattern  string
	ZoneDistribution string
	SpawnZone        string
	ReportJSON       string
}

type report struct {
	ContentPackageLoaded      bool
	ContinentActivated        bool
	ZonesActivated            int
	TransitionGatesLoaded     int
	PlayersAttached           int
	SessionReconnects         int
	ZoneTransitionsRequested  int
	ZoneTransitionsCompleted  int
	ZoneTransitionsRejected   int
	VisibilityEvaluations     int
	VisibilityEnterExitDeltas int
	NPCsSpawned               int
	CombatInteractions        int
	ShardAssignments          []worlds.ShardAssignment
	ShardAssignmentCount      int
	ZonePopulation            map[string]int
	ShardPopulation           map[string]int
	CommandsAccepted          int
	CommandsProcessed         int
	CommandsBackpressured     int
	CommandsEnqueued          int
	CommandsDequeued          int
	BackpressureEvents        int
	CommandsByType            map[string]int
	RouteFailures             int
	AverageTickDuration       time.Duration
	TickP50                   time.Duration
	TickP95                   time.Duration
	TickP99                   time.Duration
	MaxTickDuration           time.Duration
	MaxQueueDepth             int
	ZoneQueueDepths           map[string]int
	Errors                    []string
}

func main() {
	opts := parseOptions()
	if opts.Content == "" {
		exitf("--content is required")
	}

	startedEvent := observability.EventLoadsimDawnwakeStarted
	completedEvent := observability.EventLoadsimDawnwakeCompleted
	if opts.Scenario != scenarioDawnwakeTraversalBasic {
		startedEvent = observability.EventLoadsimMultizoneStarted
		completedEvent = observability.EventLoadsimMultizoneCompleted
	}
	observability.LogEvent("loadsim", startedEvent, map[string]any{
		"scenario":         opts.Scenario,
		"clients":          opts.Clients,
		"duration":         opts.Duration.String(),
		"cmdRate":          opts.CmdRate,
		"content":          opts.Content,
		"shards":           opts.Shards,
		"queueLimit":       opts.QueueLimit,
		"transitionRate":   opts.TransitionRate,
		"movementPattern":  opts.MovementPattern,
		"zoneDistribution": opts.ZoneDistribution,
		"spawnZone":        opts.SpawnZone,
	})

	started := time.Now()
	result := runScenario(opts)
	if opts.Scenario == scenarioDawnwakeTraversalBasic && result.MaxTickDuration == 0 {
		result.AverageTickDuration, result.MaxTickDuration = fallbackTickDurations(result, time.Since(started))
	}
	observability.LogEvent("loadsim", completedEvent, map[string]any{
		"scenario":              opts.Scenario,
		"errors":                len(result.Errors),
		"transitionsCompleted":  result.ZoneTransitionsCompleted,
		"visibilityEvaluations": result.VisibilityEvaluations,
		"commandsAccepted":      result.CommandsAccepted,
		"commandsBackpressured": result.CommandsBackpressured,
		"maxQueueDepth":         result.MaxQueueDepth,
	})
	if opts.ReportJSON != "" {
		if err := writeJSONReport(opts.ReportJSON, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("report write failed: %v", err))
		}
	}

	printReport(opts.Scenario, result)
	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}

func parseOptions() options {
	var durationText string
	opts := options{}
	flag.IntVar(&opts.Clients, "clients", 1, "number of simulated players")
	flag.StringVar(&durationText, "duration", "30s", "scenario duration budget")
	flag.IntVar(&opts.CmdRate, "cmd-rate", 2, "simulated commands per second")
	flag.StringVar(&opts.Scenario, "scenario", scenarioDawnwakeTraversalBasic, "scenario name")
	flag.StringVar(&opts.Content, "content", "", "content package manifest path")
	flag.IntVar(&opts.Shards, "shards", 2, "in-process shard count for multizone scenarios")
	flag.IntVar(&opts.QueueLimit, "queue-limit", 64, "per-shard command queue pressure limit")
	flag.Float64Var(&opts.TransitionRate, "transition-rate", 0.35, "fraction of accepted commands that attempt a zone transition")
	flag.StringVar(&opts.MovementPattern, "movement-pattern", "mixed", "movement pattern: mixed, transitions, local")
	flag.StringVar(&opts.ZoneDistribution, "zone-distribution", "", "comma-separated zone population weights, such as dawnwake_landing=50,amberglass_fields=50")
	flag.StringVar(&opts.SpawnZone, "spawn-zone", "", "single zone override for spawning all simulated players")
	flag.StringVar(&opts.ReportJSON, "report-json", "", "optional path for a JSON report")
	flag.Parse()

	var err error
	opts.Duration, err = time.ParseDuration(durationText)
	if err != nil {
		exitf("invalid duration: %v", err)
	}
	if opts.Clients <= 0 {
		exitf("--clients must be greater than zero")
	}
	if opts.CmdRate <= 0 {
		exitf("--cmd-rate must be greater than zero")
	}
	if opts.Shards <= 0 {
		exitf("--shards must be greater than zero")
	}
	if opts.QueueLimit <= 0 {
		exitf("--queue-limit must be greater than zero")
	}
	if opts.TransitionRate < 0 {
		exitf("--transition-rate must be zero or greater")
	}
	return opts
}

type zoneWeight struct {
	ZoneID   string
	Fraction float64
}

func runScenario(opts options) report {
	opts = normalizeOptions(opts)
	switch opts.Scenario {
	case scenarioDawnwakeTraversalBasic:
		return runDawnwakeTraversal(opts)
	case scenarioDawnwakeMultizoneSharded:
		return runDawnwakeMultizoneSharding(opts)
	case scenarioDawnwakePopulation, scenarioDawnwakePopulationAlias:
		return runDawnwakePopulation(opts)
	case scenarioDawnwakeTransitionStress:
		opts.MovementPattern = "transitions"
		return runDawnwakeMultizoneSharding(opts)
	case scenarioDawnwakeCommandPressure:
		opts.QueueLimit = 1
		opts.MovementPattern = "local"
		return runDawnwakeMultizoneSharding(opts)
	case scenarioDawnwakeZoneIsolation:
		opts.Shards = 5
		opts.MovementPattern = "mixed"
		return runDawnwakeMultizoneSharding(opts)
	default:
		return report{Errors: []string{fmt.Sprintf("unsupported scenario %q", opts.Scenario)}}
	}
}

func normalizeOptions(opts options) options {
	if opts.Clients <= 0 {
		opts.Clients = 1
	}
	if opts.Duration <= 0 {
		opts.Duration = time.Second
	}
	if opts.CmdRate <= 0 {
		opts.CmdRate = 1
	}
	if opts.Shards <= 0 {
		opts.Shards = 2
	}
	if opts.QueueLimit <= 0 {
		opts.QueueLimit = 64
	}
	if opts.MovementPattern == "" {
		opts.MovementPattern = "mixed"
	}
	return opts
}

func runDawnwakeTraversal(opts options) report {
	result := report{}
	runtime, err := loadDawnwakeRuntime(opts.Content, &result)
	if err != nil {
		return result
	}

	for index := 0; index < opts.Clients; index++ {
		characterID := fmt.Sprintf("loadsim_dawnwake_%03d", index+1)
		state, _, err := runtime.SpawnCharacterAtDefaultEntry(characterID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s spawn failed: %v", characterID, err))
			continue
		}
		result.PlayersAttached++
		if delta, err := runtime.EvaluateVisibility(characterID, worlds.InterestProfile{Radius: 80, IncludeAdjacentStreamingHints: true}); err == nil {
			result.VisibilityEvaluations++
			result.VisibilityEnterExitDeltas += len(delta.Entered) + len(delta.Exited)
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("%s visibility before transfer failed: %v", characterID, err))
		}

		zoneRuntime := runtime.Zones[state.ZoneID]
		if zoneRuntime == nil || len(zoneRuntime.Definition.TransitionGates) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("%s default zone has no transition gates", characterID))
			continue
		}
		transfer, err := moveThroughFirstGate(runtime, characterID)
		recordTransfer(&result, transfer, err, characterID)
		if err != nil || transfer.Rejected {
			continue
		}
		if delta, err := runtime.EvaluateVisibility(characterID, worlds.InterestProfile{Radius: 90, IncludeAdjacentStreamingHints: true}); err == nil {
			result.VisibilityEvaluations++
			result.VisibilityEnterExitDeltas += len(delta.Entered) + len(delta.Exited)
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("%s visibility after transfer failed: %v", characterID, err))
		}
	}
	return result
}

func runDawnwakeMultizoneSharding(opts options) report {
	result := report{CommandsByType: map[string]int{}}
	runtime, err := loadDawnwakeRuntime(opts.Content, &result)
	if err != nil {
		return result
	}
	coordinator := worlds.NewInProcessShardCoordinator(runtime, worlds.ShardCoordinatorConfig{
		ShardCount:      opts.Shards,
		QueueDepthLimit: opts.QueueLimit,
	})
	store := worlds.NewMemoryCharacterZoneStore()
	characterIDs := attachDistributedPlayers(runtime, store, opts, &result)
	if len(characterIDs) == 0 {
		result.Errors = append(result.Errors, "no simulated players attached")
		return result
	}

	commandBudget := int(opts.Duration.Seconds()) * opts.CmdRate
	if commandBudget < opts.Clients {
		commandBudget = opts.Clients
	}
	transitionEvery := transitionInterval(opts.TransitionRate, opts.MovementPattern)
	for commandIndex := 0; commandIndex < commandBudget; commandIndex++ {
		characterID := characterIDs[commandIndex%len(characterIDs)]
		command := worlds.WorldCommand{
			CommandID:   fmt.Sprintf("multi-%06d", commandIndex+1),
			CharacterID: characterID,
			Name:        "move",
		}
		result.CommandsByType[command.Name]++
		started := time.Now()
		queued, err := coordinator.TryEnqueueCommand(command)
		if err != nil {
			result.RouteFailures++
			result.Errors = append(result.Errors, fmt.Sprintf("%s route failed: %v", characterID, err))
			continue
		}
		if queued.Backpressured {
			continue
		}
		if shouldAttemptTransition(commandIndex, transitionEvery, opts.MovementPattern) {
			transfer, err := moveThroughFirstGate(runtime, characterID)
			recordTransfer(&result, transfer, err, characterID)
		} else if err := moveInsideCurrentZone(runtime, characterID, commandIndex); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s local movement failed: %v", characterID, err))
		}
		if delta, err := runtime.EvaluateVisibility(characterID, worlds.InterestProfile{Radius: 90, IncludeAdjacentStreamingHints: true}); err == nil {
			result.VisibilityEvaluations++
			result.VisibilityEnterExitDeltas += len(delta.Entered) + len(delta.Exited)
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("%s visibility failed: %v", characterID, err))
		}
		if err := coordinator.CompleteCommand(queued, worlds.SimulationTick{Duration: elapsedAtLeast(started), QueueDepth: queued.QueueDepth}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s command complete failed: %v", characterID, err))
		}
		if commandIndex%25 == 0 {
			if err := coordinator.ValidateSingleZoneOwnership(); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("ownership validation failed: %v", err))
			}
		}
	}

	runReconnectProbe(runtime, store, characterIDs, &result)
	runBackpressureProbe(coordinator, characterIDs[0], &result)
	if err := coordinator.ValidateSingleZoneOwnership(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("final ownership validation failed: %v", err))
	}
	applyShardSnapshot(coordinator.Snapshot(), &result)
	return result
}

func runDawnwakePopulation(opts options) report {
	result := report{CommandsByType: map[string]int{}}
	runtime, err := loadDawnwakeRuntime(opts.Content, &result)
	if err != nil {
		return result
	}
	coordinator := worlds.NewInProcessShardCoordinator(runtime, worlds.ShardCoordinatorConfig{
		ShardCount:      opts.Shards,
		QueueDepthLimit: opts.QueueLimit,
	})
	store := worlds.NewMemoryCharacterZoneStore()
	characterIDs := attachDistributedPlayers(runtime, store, opts, &result)
	for _, characterID := range characterIDs {
		if delta, err := runtime.EvaluateVisibility(characterID, worlds.InterestProfile{Radius: 90, IncludeAdjacentStreamingHints: true}); err == nil {
			result.VisibilityEvaluations++
			result.VisibilityEnterExitDeltas += len(delta.Entered) + len(delta.Exited)
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("%s visibility failed: %v", characterID, err))
		}
	}
	if err := coordinator.ValidateSingleZoneOwnership(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("ownership validation failed: %v", err))
	}
	applyShardSnapshot(coordinator.Snapshot(), &result)
	return result
}

func loadDawnwakeRuntime(contentPath string, result *report) (*worlds.ContinentRuntime, error) {
	registry, err := worlds.NewContentPackageLoader().Load(contentPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("content load failed: %v", err))
		return nil, err
	}
	result.ContentPackageLoaded = true

	runtime, err := registry.ActivateContinent("dawnwake_isles")
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("continent activation failed: %v", err))
		return nil, err
	}
	result.ContinentActivated = true
	result.ZonesActivated = len(runtime.Zones)
	result.TransitionGatesLoaded = countTransitionGates(runtime)
	result.NPCsSpawned = countNPCs(runtime)
	return runtime, nil
}

func attachDistributedPlayers(runtime *worlds.ContinentRuntime, store *worlds.MemoryCharacterZoneStore, opts options, result *report) []string {
	zoneIDs := distributedZoneIDs(runtime, opts, result)
	characterIDs := make([]string, 0, opts.Clients)
	for index := 0; index < opts.Clients; index++ {
		zoneID := zoneIDs[index%len(zoneIDs)]
		zoneRuntime := runtime.Zones[zoneID]
		if zoneRuntime == nil || len(zoneRuntime.Definition.EntryPoints) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("zone %s has no entry point for simulated attach", zoneID))
			continue
		}
		entryID := zoneRuntime.Definition.EntryPoints[0].EntryPointID
		characterID := fmt.Sprintf("loadsim_multizone_%04d", index+1)
		state, _, err := runtime.PlaceCharacterAtEntry(characterID, zoneID, entryID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s attach failed: %v", characterID, err))
			continue
		}
		if err := store.SaveCharacterZoneState(state); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s initial store save failed: %v", characterID, err))
			continue
		}
		characterIDs = append(characterIDs, characterID)
		result.PlayersAttached++
	}
	return characterIDs
}

func distributedZoneIDs(runtime *worlds.ContinentRuntime, opts options, result *report) []string {
	defaultZones := append([]string(nil), runtime.Definition.Zones...)
	if len(defaultZones) == 0 {
		for zoneID := range runtime.Zones {
			defaultZones = append(defaultZones, zoneID)
		}
		sort.Strings(defaultZones)
	}
	if opts.SpawnZone != "" {
		for _, zoneID := range defaultZones {
			if zoneID == opts.SpawnZone {
				return []string{opts.SpawnZone}
			}
		}
		result.Errors = append(result.Errors, fmt.Sprintf("spawn zone %s is not active", opts.SpawnZone))
		return defaultZones
	}
	if opts.ZoneDistribution == "" {
		return defaultZones
	}
	validZones := map[string]bool{}
	for _, zoneID := range defaultZones {
		validZones[zoneID] = true
	}
	weights, err := parseZoneDistribution(opts.ZoneDistribution, validZones)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid zone distribution: %v", err))
		return defaultZones
	}
	zoneIDs := make([]string, 0, opts.Clients)
	for index := 0; index < opts.Clients; index++ {
		target := float64(index) / float64(opts.Clients)
		cumulative := 0.0
		selected := weights[len(weights)-1].ZoneID
		for _, weight := range weights {
			cumulative += weight.Fraction
			if target < cumulative {
				selected = weight.ZoneID
				break
			}
		}
		zoneIDs = append(zoneIDs, selected)
	}
	return zoneIDs
}

func runReconnectProbe(runtime *worlds.ContinentRuntime, store *worlds.MemoryCharacterZoneStore, characterIDs []string, result *report) {
	limit := len(characterIDs) / 10
	if limit < 1 {
		limit = 1
	}
	if limit > len(characterIDs) {
		limit = len(characterIDs)
	}
	for index := 0; index < limit; index++ {
		characterID := characterIDs[index]
		if err := runtime.SaveCharacterZoneState(store, characterID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s reconnect save failed: %v", characterID, err))
			continue
		}
		if _, _, err := runtime.RestoreCharacterZoneState(store, characterID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s reconnect restore failed: %v", characterID, err))
			continue
		}
		result.SessionReconnects++
	}
}

func runBackpressureProbe(coordinator *worlds.InProcessShardCoordinator, characterID string, result *report) {
	accepted := []worlds.ShardCommandResult{}
	for index := 0; index <= coordinator.QueueDepthLimit; index++ {
		command := worlds.WorldCommand{
			CommandID:   fmt.Sprintf("pressure-%03d", index+1),
			CharacterID: characterID,
			Name:        "pressure_probe",
		}
		queued, err := coordinator.TryEnqueueCommand(command)
		if err != nil {
			result.RouteFailures++
			result.Errors = append(result.Errors, fmt.Sprintf("%s pressure route failed: %v", characterID, err))
			continue
		}
		result.CommandsByType[command.Name]++
		if queued.Accepted {
			accepted = append(accepted, queued)
		}
	}
	for _, queued := range accepted {
		if err := coordinator.CompleteCommand(queued, worlds.SimulationTick{Duration: time.Microsecond, QueueDepth: queued.QueueDepth}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s pressure complete failed: %v", characterID, err))
		}
	}
}

func applyShardSnapshot(snapshot worlds.ShardCoordinatorSnapshot, result *report) {
	result.ShardAssignments = snapshot.Assignments
	result.ShardAssignmentCount = len(snapshot.Assignments)
	result.ZonePopulation = snapshot.ZonePopulation
	result.ShardPopulation = snapshot.ShardPopulation
	result.CommandsAccepted = snapshot.CommandsAccepted
	result.CommandsProcessed = snapshot.CommandsProcessed
	result.CommandsBackpressured = snapshot.BackpressureCount
	result.CommandsEnqueued = snapshot.CommandsAccepted
	result.CommandsDequeued = snapshot.CommandsProcessed
	result.BackpressureEvents = snapshot.BackpressureCount
	result.RouteFailures += snapshot.RouteFailures
	result.MaxQueueDepth = snapshot.MaxQueueDepth
	result.AverageTickDuration = snapshot.Tick.Average
	result.TickP50 = snapshot.Tick.P50
	result.TickP95 = snapshot.Tick.P95
	result.TickP99 = snapshot.Tick.P99
	result.MaxTickDuration = snapshot.Tick.Max
	result.ZoneQueueDepths = map[string]int{}
	for _, metric := range snapshot.ShardMetrics {
		for _, zoneID := range metric.ZoneIDs {
			result.ZoneQueueDepths[zoneID] = metric.QueueDepth
		}
	}
}

func moveThroughFirstGate(runtime *worlds.ContinentRuntime, characterID string) (worlds.ZoneTransferResult, error) {
	state := runtime.Characters[characterID]
	if state == nil {
		return worlds.ZoneTransferResult{}, fmt.Errorf("character %s is not active", characterID)
	}
	zoneRuntime := runtime.Zones[state.ZoneID]
	if zoneRuntime == nil || len(zoneRuntime.Definition.TransitionGates) == 0 {
		return worlds.ZoneTransferResult{CharacterID: characterID, FromZoneID: state.ZoneID, ToZoneID: state.ZoneID}, nil
	}
	gate := firstEnabledGate(zoneRuntime.Definition.TransitionGates)
	if gate.TransitionID == "" {
		return worlds.ZoneTransferResult{CharacterID: characterID, FromZoneID: state.ZoneID, ToZoneID: state.ZoneID}, nil
	}
	gateCenter := center(gate.GateBounds, state.ZoneID)
	if _, err := runtime.MoveCharacter(characterID, gateCenter.X-state.Position.X, gateCenter.Y-state.Position.Y, gateCenter.Z-state.Position.Z); err != nil {
		return worlds.ZoneTransferResult{}, err
	}
	exitDeltaX, exitDeltaY := exitDelta(zoneRuntime.Definition.Bounds, gate.GateBounds)
	return runtime.MoveCharacter(characterID, exitDeltaX, exitDeltaY, 0)
}

func firstEnabledGate(gates []worlds.ZoneTransitionGate) worlds.ZoneTransitionGate {
	for _, gate := range gates {
		if !gate.Disabled {
			return gate
		}
	}
	return worlds.ZoneTransitionGate{}
}

func moveInsideCurrentZone(runtime *worlds.ContinentRuntime, characterID string, commandIndex int) error {
	state := runtime.Characters[characterID]
	if state == nil {
		return fmt.Errorf("character is not active")
	}
	zone := runtime.Zones[state.ZoneID]
	if zone == nil {
		return fmt.Errorf("zone %s is not active", state.ZoneID)
	}
	deltaX := 1.0
	if commandIndex%2 == 1 {
		deltaX = -1.0
	}
	if state.Position.X+deltaX > zone.Definition.Bounds.MaxX-1 {
		deltaX = -1.0
	}
	if state.Position.X+deltaX < zone.Definition.Bounds.MinX+1 {
		deltaX = 1.0
	}
	_, err := runtime.MoveCharacter(characterID, deltaX, 0, 0)
	return err
}

func recordTransfer(result *report, transfer worlds.ZoneTransferResult, err error, characterID string) {
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s transfer move failed: %v", characterID, err))
		return
	}
	if transfer.Requested {
		result.ZoneTransitionsRequested++
	}
	if transfer.Completed {
		result.ZoneTransitionsCompleted++
	}
	if transfer.Rejected {
		result.ZoneTransitionsRejected++
		result.Errors = append(result.Errors, fmt.Sprintf("%s transition rejected: %s", characterID, transfer.RejectionReason))
	}
}

func transitionInterval(rate float64, pattern string) int {
	switch pattern {
	case "transitions":
		return 1
	case "local":
		return 0
	}
	if rate <= 0 {
		return 0
	}
	if rate >= 1 {
		return 1
	}
	interval := int(1 / rate)
	if interval < 1 {
		interval = 1
	}
	return interval
}

func parseZoneDistribution(raw string, validZones map[string]bool) ([]zoneWeight, error) {
	parts := strings.Split(raw, ",")
	weights := make([]zoneWeight, 0, len(parts))
	total := 0.0
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("distribution item %q must use zone=weight", part)
		}
		zoneID := strings.TrimSpace(keyValue[0])
		if !validZones[zoneID] {
			return nil, fmt.Errorf("unknown zone %s", zoneID)
		}
		valueText := strings.TrimSpace(strings.TrimSuffix(keyValue[1], "%"))
		value, err := strconv.ParseFloat(valueText, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid weight for %s: %w", zoneID, err)
		}
		if value <= 0 {
			return nil, fmt.Errorf("weight for %s must be greater than zero", zoneID)
		}
		weights = append(weights, zoneWeight{ZoneID: zoneID, Fraction: value})
		total += value
	}
	if len(weights) == 0 {
		return nil, fmt.Errorf("zone distribution is empty")
	}
	for index := range weights {
		weights[index].Fraction = weights[index].Fraction / total
	}
	return weights, nil
}

func calculateTickStats(durations []time.Duration) worlds.TickDurationSummary {
	return worlds.SummarizeTickDurations(durations)
}

func shouldAttemptTransition(commandIndex int, transitionEvery int, pattern string) bool {
	if pattern == "local" || transitionEvery <= 0 {
		return false
	}
	if pattern == "transitions" {
		return true
	}
	return commandIndex%transitionEvery == 0
}

func elapsedAtLeast(started time.Time) time.Duration {
	elapsed := time.Since(started)
	if elapsed <= 0 {
		return time.Microsecond
	}
	return elapsed
}

func center(bounds worlds.ZoneBounds, zoneID string) worlds.WorldPosition {
	return worlds.WorldPosition{
		ZoneID: zoneID,
		X:      (bounds.MinX + bounds.MaxX) / 2,
		Y:      (bounds.MinY + bounds.MaxY) / 2,
		Z:      (bounds.MinZ + bounds.MaxZ) / 2,
	}
}

func exitDelta(zone worlds.ZoneBounds, gate worlds.ZoneBounds) (float64, float64) {
	switch {
	case gate.MaxX >= zone.MaxX:
		return 10, 0
	case gate.MinX <= zone.MinX:
		return -10, 0
	case gate.MaxY >= zone.MaxY:
		return 0, 10
	case gate.MinY <= zone.MinY:
		return 0, -10
	default:
		return 10, 0
	}
}

func countTransitionGates(runtime *worlds.ContinentRuntime) int {
	total := 0
	for _, zone := range runtime.Zones {
		total += len(zone.Definition.TransitionGates)
	}
	return total
}

func countNPCs(runtime *worlds.ContinentRuntime) int {
	total := 0
	for _, zone := range runtime.Zones {
		for _, entity := range zone.Entities.Entities {
			if entity.Kind == "hostile_mob" || entity.Kind == "quest_giver_npc" {
				total++
			}
		}
	}
	return total
}

func fallbackTickDurations(result report, elapsed time.Duration) (time.Duration, time.Duration) {
	ticks := result.VisibilityEvaluations + result.ZoneTransitionsRequested + result.PlayersAttached
	if ticks <= 0 {
		return 0, 0
	}
	average := elapsed / time.Duration(ticks)
	return average, elapsed
}

func printReport(scenario string, result report) {
	fmt.Printf("%s loadsim report\n", scenario)
	fmt.Printf("content package loaded: %t\n", result.ContentPackageLoaded)
	fmt.Printf("continent activated: %t\n", result.ContinentActivated)
	fmt.Printf("zones activated: %d\n", result.ZonesActivated)
	fmt.Printf("transition gates loaded: %d\n", result.TransitionGatesLoaded)
	fmt.Printf("players attached: %d\n", result.PlayersAttached)
	fmt.Printf("session reconnects: %d\n", result.SessionReconnects)
	fmt.Printf("zone population distribution: %s\n", formatStringIntMap(result.ZonePopulation))
	fmt.Printf("shard assignments: %s\n", formatShardAssignments(result.ShardAssignments))
	fmt.Printf("shard population distribution: %s\n", formatStringIntMap(result.ShardPopulation))
	fmt.Printf("zone transitions requested: %d\n", result.ZoneTransitionsRequested)
	fmt.Printf("zone transitions completed: %d\n", result.ZoneTransitionsCompleted)
	fmt.Printf("zone transitions rejected: %d\n", result.ZoneTransitionsRejected)
	fmt.Printf("visibility evaluations: %d\n", result.VisibilityEvaluations)
	fmt.Printf("visibility enter/exit deltas: %d\n", result.VisibilityEnterExitDeltas)
	fmt.Printf("NPCs spawned: %d\n", result.NPCsSpawned)
	fmt.Printf("combat interactions: %d\n", result.CombatInteractions)
	fmt.Printf("shard assignment count: %d\n", result.ShardAssignmentCount)
	fmt.Printf("commands accepted: %d\n", result.CommandsAccepted)
	fmt.Printf("commands processed: %d\n", result.CommandsProcessed)
	fmt.Printf("commands enqueued: %d\n", result.CommandsEnqueued)
	fmt.Printf("commands dequeued: %d\n", result.CommandsDequeued)
	fmt.Printf("commands backpressured: %d\n", result.CommandsBackpressured)
	fmt.Printf("backpressure events: %d\n", result.BackpressureEvents)
	fmt.Printf("commands by type: %s\n", formatStringIntMap(result.CommandsByType))
	fmt.Printf("route failures: %d\n", result.RouteFailures)
	fmt.Printf("average tick duration: %s\n", result.AverageTickDuration)
	fmt.Printf("tick p50: %s\n", result.TickP50)
	fmt.Printf("tick p95: %s\n", result.TickP95)
	fmt.Printf("tick p99: %s\n", result.TickP99)
	fmt.Printf("max tick duration: %s\n", result.MaxTickDuration)
	fmt.Printf("max queue depth: %d\n", result.MaxQueueDepth)
	fmt.Printf("zone queue depths: %s\n", formatStringIntMap(result.ZoneQueueDepths))
	fmt.Printf("errors: %d\n", len(result.Errors))
	for _, err := range result.Errors {
		fmt.Printf("- %s\n", err)
	}
}

func writeJSONReport(path string, result report) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o644)
}

func formatShardAssignments(assignments []worlds.ShardAssignment) string {
	if len(assignments) == 0 {
		return "{}"
	}
	sorted := append([]worlds.ShardAssignment(nil), assignments...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ZoneID < sorted[j].ZoneID
	})
	parts := make([]string, 0, len(sorted))
	for _, assignment := range sorted {
		parts = append(parts, fmt.Sprintf("%s=%s", assignment.ZoneID, assignment.ShardID))
	}
	return "{" + join(parts, ", ") + "}"
}

func formatStringIntMap(values map[string]int) string {
	if len(values) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return "{" + join(parts, ", ") + "}"
}

func join(values []string, sep string) string {
	if len(values) == 0 {
		return ""
	}
	result := values[0]
	for _, value := range values[1:] {
		result += sep + value
	}
	return result
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
