package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/worlds"
)

type options struct {
	Clients  int
	Duration time.Duration
	CmdRate  int
	Scenario string
	Content  string
}

type report struct {
	ContentPackageLoaded      bool
	ContinentActivated        bool
	ZonesActivated            int
	TransitionGatesLoaded     int
	PlayersAttached           int
	ZoneTransitionsRequested  int
	ZoneTransitionsCompleted  int
	ZoneTransitionsRejected   int
	VisibilityEvaluations     int
	VisibilityEnterExitDeltas int
	NPCsSpawned               int
	CombatInteractions        int
	AverageTickDuration       time.Duration
	MaxTickDuration           time.Duration
	MaxQueueDepth             int
	Errors                    []string
}

func main() {
	opts := parseOptions()
	if opts.Scenario != "dawnwake-traversal-basic" {
		exitf("unsupported scenario %q", opts.Scenario)
	}
	if opts.Content == "" {
		exitf("--content is required")
	}
	observability.LogEvent("loadsim", observability.EventLoadsimDawnwakeStarted, map[string]any{
		"scenario": opts.Scenario,
		"clients":  opts.Clients,
		"duration": opts.Duration.String(),
		"cmdRate":  opts.CmdRate,
		"content":  opts.Content,
	})

	started := time.Now()
	result := runDawnwakeTraversal(opts)
	result.AverageTickDuration, result.MaxTickDuration = tickDurations(result, time.Since(started))
	observability.LogEvent("loadsim", observability.EventLoadsimDawnwakeCompleted, map[string]any{
		"scenario":              opts.Scenario,
		"errors":                len(result.Errors),
		"transitionsCompleted":  result.ZoneTransitionsCompleted,
		"visibilityEvaluations": result.VisibilityEvaluations,
	})

	printReport(result)
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
	flag.StringVar(&opts.Scenario, "scenario", "dawnwake-traversal-basic", "scenario name")
	flag.StringVar(&opts.Content, "content", "", "content package manifest path")
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
	return opts
}

func runDawnwakeTraversal(opts options) report {
	result := report{}
	registry, err := worlds.NewContentPackageLoader().Load(opts.Content)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("content load failed: %v", err))
		return result
	}
	result.ContentPackageLoaded = true

	runtime, err := registry.ActivateContinent("dawnwake_isles")
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("continent activation failed: %v", err))
		return result
	}
	result.ContinentActivated = true
	result.ZonesActivated = len(runtime.Zones)
	result.TransitionGatesLoaded = countTransitionGates(runtime)
	result.NPCsSpawned = countNPCs(runtime)

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
		gate := zoneRuntime.Definition.TransitionGates[0]
		gateCenter := center(gate.GateBounds, state.ZoneID)
		_, _ = runtime.MoveCharacter(characterID, gateCenter.X-state.Position.X, gateCenter.Y-state.Position.Y, gateCenter.Z-state.Position.Z)
		exitDeltaX, exitDeltaY := exitDelta(zoneRuntime.Definition.Bounds, gate.GateBounds)
		transfer, err := runtime.MoveCharacter(characterID, exitDeltaX, exitDeltaY, 0)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s transfer move failed: %v", characterID, err))
			continue
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
		if delta, err := runtime.EvaluateVisibility(characterID, worlds.InterestProfile{Radius: 90, IncludeAdjacentStreamingHints: true}); err == nil {
			result.VisibilityEvaluations++
			result.VisibilityEnterExitDeltas += len(delta.Entered) + len(delta.Exited)
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("%s visibility after transfer failed: %v", characterID, err))
		}
	}
	result.MaxQueueDepth = opts.Clients
	return result
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

func tickDurations(result report, elapsed time.Duration) (time.Duration, time.Duration) {
	ticks := result.VisibilityEvaluations + result.ZoneTransitionsRequested + result.PlayersAttached
	if ticks <= 0 {
		return 0, 0
	}
	average := elapsed / time.Duration(ticks)
	return average, elapsed
}

func printReport(result report) {
	fmt.Println("Dawnwake traversal loadsim report")
	fmt.Printf("content package loaded: %t\n", result.ContentPackageLoaded)
	fmt.Printf("continent activated: %t\n", result.ContinentActivated)
	fmt.Printf("zones activated: %d\n", result.ZonesActivated)
	fmt.Printf("transition gates loaded: %d\n", result.TransitionGatesLoaded)
	fmt.Printf("players attached: %d\n", result.PlayersAttached)
	fmt.Printf("zone transitions requested: %d\n", result.ZoneTransitionsRequested)
	fmt.Printf("zone transitions completed: %d\n", result.ZoneTransitionsCompleted)
	fmt.Printf("zone transitions rejected: %d\n", result.ZoneTransitionsRejected)
	fmt.Printf("visibility evaluations: %d\n", result.VisibilityEvaluations)
	fmt.Printf("visibility enter/exit deltas: %d\n", result.VisibilityEnterExitDeltas)
	fmt.Printf("NPCs spawned: %d\n", result.NPCsSpawned)
	fmt.Printf("combat interactions: %d\n", result.CombatInteractions)
	fmt.Printf("average tick duration: %s\n", result.AverageTickDuration)
	fmt.Printf("max tick duration: %s\n", result.MaxTickDuration)
	fmt.Printf("max queue depth: %d\n", result.MaxQueueDepth)
	fmt.Printf("errors: %d\n", len(result.Errors))
	for _, err := range result.Errors {
		fmt.Printf("- %s\n", err)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
