package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"amandacore/services/internal/config"
	"amandacore/services/internal/store"
)

func main() {
	var dryRun bool
	var status bool
	var jsonOutput bool
	var storePath string
	flags := flag.NewFlagSet("dbmigrate", flag.ExitOnError)
	flags.BoolVar(&dryRun, "dry-run", false, "validate migrations without writing")
	flags.BoolVar(&status, "status", false, "print applied migration history")
	flags.BoolVar(&jsonOutput, "json", false, "print JSON output")
	flags.StringVar(&storePath, "store", "", "platform state store path")
	_ = flags.Parse(os.Args[1:])

	cfg := config.Load("dbmigrate", "0")
	if storePath == "" {
		storePath = cfg.StorePath
	}

	fileStore, err := store.NewFileStoreWithOptions(storePath, cfg.BuildID, cfg.WorldEndpoint, store.FileStoreOpenOptions{
		ApplyMigrations: false,
		SaveOnOpen:      false,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbmigrate: open store: %v\n", err)
		os.Exit(1)
	}

	if status {
		history, err := fileStore.MigrationHistory()
		if err != nil {
			fmt.Fprintf(os.Stderr, "dbmigrate: status: %v\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			writeJSON(history)
			return
		}
		fmt.Printf("AmandaCore migration status for %s\n", storePath)
		if len(history) == 0 {
			fmt.Println("no migrations applied")
			return
		}
		for _, record := range history {
			fmt.Printf("%s %s checksum=%s appliedAt=%d durationMs=%d\n", record.ID, record.Description, record.Checksum, record.AppliedAt, record.DurationMs)
		}
		return
	}

	result, err := fileStore.ApplyMigrations(store.MigrationOptions{DryRun: dryRun})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbmigrate: apply migrations: %v\n", err)
		os.Exit(1)
	}
	if jsonOutput {
		writeJSON(result)
		return
	}
	mode := "applied"
	if dryRun {
		mode = "validated"
	}
	fmt.Printf("AmandaCore migrations %s for %s\n", mode, storePath)
	fmt.Printf("current=%s applied=%d alreadyApplied=%d dryRun=%v\n", result.CurrentID, len(result.Applied), len(result.AlreadyApplied), result.DryRun)
	for _, record := range result.Applied {
		fmt.Printf("+ %s %s\n", record.ID, record.Description)
	}
}

func writeJSON(value any) {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbmigrate: encode JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(payload))
}
