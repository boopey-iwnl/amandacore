package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"amandacore/services/internal/worlds"
)

type options struct {
	Clients  int
	Duration time.Duration
	CmdRate  int
	Scenario string
}

func main() {
	opts := parseOptions()
	if opts.Scenario != "quest-basic" {
		exitf("unsupported scenario %q", opts.Scenario)
	}

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
}

func parseOptions() options {
	var durationText string
	opts := options{}
	flag.IntVar(&opts.Clients, "clients", 1, "number of simulated players")
	flag.StringVar(&durationText, "duration", "30s", "scenario duration budget, for example 30s")
	flag.IntVar(&opts.CmdRate, "cmd-rate", 2, "nominal commands per second per player")
	flag.StringVar(&opts.Scenario, "scenario", "quest-basic", "scenario name")
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

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
