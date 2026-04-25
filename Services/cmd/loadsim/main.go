package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"amandacore/services/internal/worlds"
)

func main() {
	var clients int
	var durationText string
	var commandRate float64
	var scenario string
	var contentPath string
	flag.IntVar(&clients, "clients", 1, "number of simulated clients")
	flag.StringVar(&durationText, "duration", "30s", "simulated duration, for example 30s or 1m")
	flag.Float64Var(&commandRate, "cmd-rate", 2, "simulated command rate per second")
	flag.StringVar(&scenario, "scenario", "content-package-basic", "scenario to run")
	flag.StringVar(&contentPath, "content", "", "content package manifest path")
	flag.Parse()

	duration, err := time.ParseDuration(durationText)
	if err != nil {
		exitf("invalid duration: %v", err)
	}

	report, err := worlds.RunContentPackageLoadsim(worlds.ContentPackageLoadsimOptions{
		Clients:     clients,
		Duration:    duration,
		CommandRate: commandRate,
		Scenario:    scenario,
		ContentPath: contentPath,
	})
	printReport(report)
	if err != nil {
		os.Exit(1)
	}
}

func printReport(report worlds.ContentPackageLoadsimReport) {
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
	os.Exit(1)
}
