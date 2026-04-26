package main

import (
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
	CmdRate         int
	Scenario        string
	TransitionLoops int
	Shards          int
	QueueCapacity   int
}

func main() {
	opts := parseOptions()
	switch opts.Scenario {
	case "quest-basic":
		report, err := worlds.RunQuestBasicLoadsim(worlds.QuestBasicLoadsimOptions{
			Clients:  opts.Clients,
			Duration: opts.Duration,
			CmdRate:  opts.CmdRate,
		})
		printQuestReport(report)
		if err != nil {
			fmt.Fprintf(os.Stderr, "loadsim failed: %v\n", err)
			os.Exit(1)
		}
		if len(report.Errors) > 0 {
			os.Exit(1)
		}
	case "zone-handoff-basic":
		report, err := worlds.RunZoneHandoffLoadsim(worlds.ZoneHandoffLoadsimOptions{
			Clients:         opts.Clients,
			Duration:        opts.Duration,
			CmdRate:         opts.CmdRate,
			TransitionLoops: opts.TransitionLoops,
			Shards:          opts.Shards,
			QueueCapacity:   opts.QueueCapacity,
		})
		printZoneHandoffReport(report)
		if err != nil {
			fmt.Fprintf(os.Stderr, "loadsim failed: %v\n", err)
			os.Exit(1)
		}
		if len(report.Errors) > 0 {
			os.Exit(1)
		}
	default:
		exitf("unsupported scenario %q", opts.Scenario)
	}
}

func parseOptions() options {
	var durationText string
	opts := options{}
	flag.IntVar(&opts.Clients, "clients", 1, "number of simulated players")
	flag.StringVar(&durationText, "duration", "30s", "scenario duration budget, for example 30s")
	flag.IntVar(&opts.CmdRate, "cmd-rate", 2, "nominal commands per second per player")
	flag.StringVar(&opts.Scenario, "scenario", "quest-basic", "scenario name")
	flag.IntVar(&opts.TransitionLoops, "transition-loops", 2, "zone handoff transition loops per simulated player")
	flag.IntVar(&opts.Shards, "shards", 2, "zone shard count for zone handoff loadsim")
	flag.IntVar(&opts.QueueCapacity, "queue-capacity", 64, "per-zone command queue capacity for zone handoff loadsim")
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
	if opts.CmdRate <= 0 {
		exitf("--cmd-rate must be greater than zero")
	}
	if opts.TransitionLoops < 0 {
		exitf("--transition-loops must be zero or greater")
	}
	if opts.Shards <= 0 {
		exitf("--shards must be greater than zero")
	}
	if opts.QueueCapacity <= 0 {
		exitf("--queue-capacity must be greater than zero")
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

func printZoneHandoffReport(report worlds.ZoneHandoffLoadsimReport) {
	fmt.Println("Zone handoff loadsim report")
	fmt.Printf("- simulated clients: %d\n", report.SimulatedClients)
	fmt.Printf("- transition loops: %d\n", report.TransitionLoops)
	fmt.Printf("- shard count: %d\n", report.ShardCount)
	fmt.Printf("- queue capacity: %d\n", report.QueueCapacity)
	fmt.Printf("- handoffs requested: %d\n", report.HandoffsRequested)
	fmt.Printf("- handoffs accepted: %d\n", report.HandoffsAccepted)
	fmt.Printf("- handoffs completed: %d\n", report.HandoffsCompleted)
	fmt.Printf("- handoffs rejected: %d\n", report.HandoffsRejected)
	fmt.Printf("- handoffs retried: %d\n", report.HandoffsRetried)
	fmt.Printf("- expected rejections: %d\n", report.ExpectedRejections)
	fmt.Printf("- journal entries: %d\n", report.JournalEntries)
	fmt.Printf("- shard assignments: %s\n", formatStringMap(report.ShardAssignments))
	fmt.Printf("- zone population: %s\n", formatCounts(report.ZonePopulation))
	fmt.Printf("- shard population: %s\n", formatCounts(report.ShardPopulation))
	fmt.Printf("- max queue depth: %d\n", report.MaxQueueDepth)
	fmt.Printf("- queue backpressure: %d\n", report.QueueBackpressure)
	fmt.Printf("- average tick duration: %s\n", report.AverageTickDuration)
	fmt.Printf("- max tick duration: %s\n", report.MaxTickDuration)
	if len(report.Errors) == 0 {
		fmt.Println("- errors: 0")
		return
	}
	fmt.Printf("- errors: %d\n", len(report.Errors))
	for _, errText := range report.Errors {
		fmt.Printf("  - %s\n", errText)
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

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
