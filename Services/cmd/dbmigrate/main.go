package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"amandacore/services/internal/config"
	"amandacore/services/internal/store"
	"amandacore/services/internal/store/sqlstore"
)

func main() {
	var dryRun bool
	var status bool
	var check bool
	var jsonOutput bool
	var backend string
	var storePath string
	var sqlitePath string
	flags := flag.NewFlagSet("dbmigrate", flag.ExitOnError)
	flags.BoolVar(&dryRun, "dry-run", false, "validate migrations without writing")
	flags.BoolVar(&status, "status", false, "print applied migration history")
	flags.BoolVar(&check, "check", false, "verify all migrations are applied and checksums match without writing")
	flags.BoolVar(&jsonOutput, "json", false, "print JSON output")
	flags.StringVar(&backend, "backend", "", "store backend: file or sqlite")
	flags.StringVar(&storePath, "store", "", "platform state store path")
	flags.StringVar(&sqlitePath, "sqlite", "", "sqlite database path")
	_ = flags.Parse(os.Args[1:])

	cfg := config.Load("dbmigrate", "0")
	if backend == "" {
		backend = cfg.StoreBackend
	}
	backend = strings.ToLower(strings.TrimSpace(backend))
	if backend == "" {
		backend = "file"
	}

	switch backend {
	case "file":
		runFileMigrations(cfg, storePath, dryRun, status, check, jsonOutput)
	case "sqlite":
		runSQLiteMigrations(cfg, sqlitePath, dryRun, status, check, jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "dbmigrate: unsupported backend %q\n", backend)
		os.Exit(2)
	}
}

func runFileMigrations(cfg config.ServiceConfig, storePath string, dryRun bool, status bool, check bool, jsonOutput bool) {
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

	result, err := fileStore.ApplyMigrations(store.MigrationOptions{DryRun: dryRun || check})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbmigrate: apply migrations: %v\n", err)
		os.Exit(1)
	}
	if check && len(result.Applied) > 0 {
		if jsonOutput {
			writeJSON(result)
		}
		fmt.Fprintf(os.Stderr, "dbmigrate: file store has %d pending migrations\n", len(result.Applied))
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

func runSQLiteMigrations(cfg config.ServiceConfig, sqlitePath string, dryRun bool, status bool, check bool, jsonOutput bool) {
	if sqlitePath == "" {
		sqlitePath = cfg.SQLitePath
	}
	if strings.TrimSpace(sqlitePath) == "" {
		fmt.Fprintln(os.Stderr, "dbmigrate: AMANDACORE_SQLITE_PATH or --sqlite is required for sqlite backend")
		os.Exit(2)
	}

	sqliteStore, err := sqlstore.OpenWithOptions(sqlitePath, sqlstore.OpenOptions{ApplyMigrations: false})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbmigrate: open sqlite: %v\n", err)
		os.Exit(1)
	}
	defer sqliteStore.Close()

	migrator, err := sqliteStore.Migrator()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbmigrate: sqlite migrator: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()

	if status {
		statuses, err := migrator.Status(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dbmigrate: sqlite status: %v\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			writeJSON(statuses)
			return
		}
		fmt.Printf("AmandaCore sqlite migration status for %s\n", sqlitePath)
		for _, status := range statuses {
			state := "pending"
			checksum := status.Migration.Checksum
			if status.Applied && status.Record != nil {
				state = "applied"
				checksum = status.Record.Checksum
			}
			fmt.Printf("%s %s %s checksum=%s\n", status.Migration.ID, status.Migration.Name, state, checksum)
		}
		return
	}

	if dryRun || check {
		result, err := migrator.Check(ctx)
		if jsonOutput {
			writeJSON(result)
		}
		if err != nil {
			if errors.Is(err, sqlstore.ErrChecksumMismatch) {
				fmt.Fprintf(os.Stderr, "dbmigrate: sqlite checksum mismatch detected\n")
			} else {
				fmt.Fprintf(os.Stderr, "dbmigrate: sqlite check: %v\n", err)
			}
			os.Exit(1)
		}
		if len(result.Pending) > 0 {
			if !jsonOutput {
				fmt.Printf("AmandaCore sqlite migrations pending for %s\n", sqlitePath)
				for _, migration := range result.Pending {
					fmt.Printf("- %s %s\n", migration.ID, migration.Name)
				}
			}
			os.Exit(1)
		}
		if !jsonOutput {
			fmt.Printf("AmandaCore sqlite migrations verified for %s applied=%d pending=0\n", sqlitePath, len(result.Applied))
		}
		return
	}

	result, err := migrator.Apply(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbmigrate: apply sqlite migrations: %v\n", err)
		os.Exit(1)
	}
	if jsonOutput {
		writeJSON(result)
		return
	}
	fmt.Printf("AmandaCore sqlite migrations applied for %s\n", sqlitePath)
	fmt.Printf("applied=%d alreadyApplied=%d\n", len(result.Applied), len(result.AlreadyApplied))
	for _, record := range result.Applied {
		fmt.Printf("+ %s %s\n", record.ID, record.Name)
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
