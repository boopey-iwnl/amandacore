package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"amandacore/services/internal/worlds"
)

type options struct {
	Clients         int
	Duration        time.Duration
	CommandRate     float64
	Scenario        string
	ContentPath     string
	TransitionLoops int
	Seed            int64
	ShardCount      int
}

func main() {
	opts := parseOptions()

	switch opts.Scenario {
	case "quest-basic":
		report, err := worlds.RunQuestBasicLoadsim(worlds.QuestBasicLoadsimOptions{
			Clients:  opts.Clients,
			Duration: opts.Duration,
			CmdRate:  questCommandRate(opts.CommandRate),
		})
		printQuestReport(report)
		exitForQuestReport(report, err)
	case "content-package-basic":
		report, err := worlds.RunContentPackageLoadsim(worlds.ContentPackageLoadsimOptions{
			Clients:     opts.Clients,
			Duration:    opts.Duration,
			CommandRate: opts.CommandRate,
			Scenario:    opts.Scenario,
			ContentPath: opts.ContentPath,
		})
		printContentReport(report)
		exitForContentReport(report, err)
	case "dawnwake-streaming-basic", "dawnwake-traversal-basic", "dawnwake-multizone-sharding-basic":
		report, err := worlds.RunDawnwakeStreamingLoadsim(worlds.DawnwakeStreamingLoadsimOptions{
			Clients:         opts.Clients,
			Duration:        opts.Duration,
			CommandRate:     opts.CommandRate,
			Scenario:        opts.Scenario,
			ContentPath:     opts.ContentPath,
			TransitionLoops: opts.TransitionLoops,
			Seed:            opts.Seed,
			ShardCount:      opts.ShardCount,
		})
		printDawnwakeReport(report)
		exitForDawnwakeReport(report, err)
	default:
		exitf("unsupported scenario %q", opts.Scenario)
	}
}

func parseOptions() options {
	var durationText string
	opts := options{}
	flag.IntVar(&opts.Clients, "clients", 1, "number of simulated players")
	flag.StringVar(&durationText, "duration", "30s", "scenario duration budget, for example 30s")
	flag.Float64Var(&opts.CommandRate, "cmd-rate", 2, "nominal commands per second per player")
	flag.StringVar(&opts.Scenario, "scenario", "quest-basic", "scenario name")
	flag.StringVar(&opts.ContentPath, "content", "", "content package manifest path")
	flag.IntVar(&opts.TransitionLoops, "transition-loops", 0, "transition probes per simulated player for Dawnwake scenarios")
	flag.Int64Var(&opts.Seed, "seed", 42, "deterministic loadsim seed")
	flag.IntVar(&opts.ShardCount, "shards", 2, "zone shard count for Dawnwake multizone loadsim")
	flag.Parse()

	duration, err := time.ParseDuration(durationText)
	if err != nil {
		exitf("invalid duration: %v", err)
	}
	opts.Duration = duration
	opts.Scenario = strings.ToLower(strings.TrimSpace(opts.Scenario))
	if opts.Clients <= 0 {
		exitf("--clients must be greater than zero")
	}
	if opts.CommandRate <= 0 {
		exitf("--cmd-rate must be greater than zero")
	}
	if opts.TransitionLoops < 0 {
		exitf("--transition-loops must be zero or greater")
	}
	if opts.ShardCount <= 0 {
		exitf("--shards must be greater than zero")
	}
	return opts
}

func printQuestReport(report worlds.QuestBasicLoadsimReport) {
	fmt.Println("Quest basic loadsim report")
	fmt.Printf("- simulated clients: %d\n", report.SimulatedClients)
	fmt.Printf("- quests accepted: %d\n", report.QuestsAccepted)
	fmt.Printf("- NPC kills: %d\n", report.NPCKills)
	fmt.Printf("- kill credits awarded: %d\n", report.KillCreditsAwarded)
	fmt.Printf("- loot containers created: %d\n", report.LootContainersCreated)
	fmt.Printf("- loot claims attempted: %d\n", report.LootClaimsAttempted)
	fmt.Printf("- loot claims completed: %d\n", report.LootClaimsCompleted)
	fmt.Printf("- inventory grants: %d\n", report.InventoryGrants)
	fmt.Printf("- objective updates: %d\n", report.ObjectiveUpdates)
	fmt.Printf("- quests ready: %d\n", report.QuestsReady)
	fmt.Printf("- quests completed: %d\n", report.QuestsCompleted)
	fmt.Printf("- rewards granted: %d\n", report.RewardsGranted)
	fmt.Printf("- rejected commands: %d\n", report.RejectedCommands)
	fmt.Printf("- average tick duration: %s\n", report.AverageTickDuration)
	fmt.Printf("- max tick duration: %s\n", report.MaxTickDuration)
	fmt.Printf("- max queue depth: %d\n", report.MaxQueueDepth)
	if len(report.Errors) == 0 {
		fmt.Println("- errors: 0")
		return
	}
	fmt.Printf("- errors: %d\n", len(report.Errors))
	for _, errText := range report.Errors {
		fmt.Printf("  - %s\n", errText)
	}
}

func questCommandRate(commandRate float64) int {
	rate := int(commandRate)
	if rate < 1 {
		return 1
	}
	return rate
}

func printContentReport(report worlds.ContentPackageLoadsimReport) {
	fmt.Println("Content package loadsim report")
	fmt.Printf("- content package loaded: %v\n", report.ContentPackageLoaded)
	fmt.Printf("- package id: %s\n", report.PackageID)
	fmt.Printf("- validation errors: %d\n", len(report.ValidationErrors))
	for _, validationError := range report.ValidationErrors {
		fmt.Printf("  - %s\n", validationError)
	}
	fmt.Printf("- zones activated: %d\n", report.ZonesActivated)
	fmt.Printf("- catalogs loaded: %s\n", formatCounts(report.CatalogsLoaded))
	fmt.Printf("- NPCs spawned: %d\n", report.NPCsSpawned)
	fmt.Printf("- quest providers registered: %d\n", report.QuestProvidersRegistered)
	fmt.Printf("- quests accepted: %d\n", report.QuestsAccepted)
	fmt.Printf("- NPC kills: %d\n", report.NPCKills)
	fmt.Printf("- loot containers created: %d\n", report.LootContainersCreated)
	fmt.Printf("- loot claims completed: %d\n", report.LootClaimsCompleted)
	fmt.Printf("- inventory grants: %s\n", formatCounts(report.InventoryGrants))
	fmt.Printf("- quests completed: %d\n", report.QuestsCompleted)
	fmt.Printf("- rewards granted: %d\n", report.RewardsGranted)
	fmt.Printf("- average tick duration: %.3fms\n", report.AverageTickDurationMs)
	fmt.Printf("- max tick duration: %.3fms\n", report.MaxTickDurationMs)
	fmt.Printf("- max queue depth: %d\n", report.MaxQueueDepth)
	fmt.Printf("- errors: %d\n", len(report.Errors))
	for _, runError := range report.Errors {
		fmt.Printf("  - %s\n", runError)
	}
	encoded, err := json.Marshal(report)
	if err == nil {
		fmt.Printf("- json: %s\n", string(encoded))
	}
}

func printDawnwakeReport(report worlds.DawnwakeStreamingLoadsimReport) {
	fmt.Println("Dawnwake streaming loadsim report")
	fmt.Printf("- scenario: %s\n", report.Scenario)
	fmt.Printf("- content package loaded: %v\n", report.ContentPackageLoaded)
	fmt.Printf("- package id: %s\n", report.PackageID)
	fmt.Printf("- continent id: %s\n", report.ContinentID)
	fmt.Printf("- validation errors: %d\n", len(report.ValidationErrors))
	for _, validationError := range report.ValidationErrors {
		fmt.Printf("  - %s\n", validationError)
	}
	fmt.Printf("- zones activated: %d\n", report.ZonesActivated)
	fmt.Printf("- catalogs loaded: %s\n", formatCounts(report.CatalogsLoaded))
	fmt.Printf("- transition gates loaded: %d\n", report.TransitionGatesLoaded)
	fmt.Printf("- players attached: %d\n", report.PlayersAttached)
	fmt.Printf("- transition loops: %d\n", report.TransitionLoops)
	fmt.Printf("- seed: %d\n", report.Seed)
	fmt.Printf("- shard count: %d\n", report.ShardCount)
	fmt.Printf("- shard assignments: %s\n", formatStringMap(report.ShardAssignments))
	fmt.Printf("- shard population: %s\n", formatCounts(report.ShardPopulation))
	fmt.Printf("- zone population: %s\n", formatCounts(report.ZonePopulation))
	fmt.Printf("- transition counts: %s\n", formatCounts(report.TransitionCounts))
	fmt.Printf("- zone transition counts: %s\n", formatCounts(report.ZoneTransitionCounts))
	fmt.Printf("- zone transitions requested: %d\n", report.ZoneTransitionsRequested)
	fmt.Printf("- zone transitions completed: %d\n", report.ZoneTransitionsCompleted)
	fmt.Printf("- zone transitions rejected: %d\n", report.ZoneTransitionsRejected)
	fmt.Printf("- visibility evaluations: %d\n", report.VisibilityEvaluations)
	fmt.Printf("- streaming hints emitted: %d\n", report.StreamingHintsEmitted)
	fmt.Printf("- NPCs spawned: %d\n", report.NPCsSpawned)
	fmt.Printf("- quest providers registered: %d\n", report.QuestProvidersRegistered)
	fmt.Printf("- average tick duration: %.3fms\n", report.AverageTickDurationMs)
	fmt.Printf("- p50 tick duration: %.3fms\n", report.P50TickDurationMs)
	fmt.Printf("- p95 tick duration: %.3fms\n", report.P95TickDurationMs)
	fmt.Printf("- p99 tick duration: %.3fms\n", report.P99TickDurationMs)
	fmt.Printf("- max tick duration: %.3fms\n", report.MaxTickDurationMs)
	fmt.Printf("- max queue depth: %d\n", report.MaxQueueDepth)
	fmt.Printf("- rejected commands: %d\n", report.RejectedCommands)
	fmt.Printf("- errors: %d\n", len(report.Errors))
	for _, runError := range report.Errors {
		fmt.Printf("  - %s\n", runError)
	}
	encoded, err := json.Marshal(report)
	if err == nil {
		fmt.Printf("- json: %s\n", string(encoded))
	}
}

func formatStringMap(values map[string]string) string {
	if len(values) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := "{"
	for index, key := range keys {
		if index > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s:%s", key, values[key])
	}
	return result + "}"
}

func formatCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := "{"
	for index, key := range keys {
		if index > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s:%d", key, counts[key])
	}
	return result + "}"
}

func exitForQuestReport(report worlds.QuestBasicLoadsimReport, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadsim failed: %v\n", err)
		os.Exit(1)
	}
	if len(report.Errors) > 0 {
		os.Exit(1)
	}
}

func exitForContentReport(report worlds.ContentPackageLoadsimReport, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadsim failed: %v\n", err)
		os.Exit(1)
	}
	if len(report.Errors) > 0 || len(report.ValidationErrors) > 0 {
		os.Exit(1)
	}
}

func exitForDawnwakeReport(report worlds.DawnwakeStreamingLoadsimReport, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadsim failed: %v\n", err)
		os.Exit(1)
	}
	if len(report.Errors) > 0 || len(report.ValidationErrors) > 0 {
		os.Exit(1)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
