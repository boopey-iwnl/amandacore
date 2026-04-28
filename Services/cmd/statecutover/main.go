package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"amandacore/services/internal/statecutover"
)

func main() {
	var source string
	var sqlitePath string
	var apply bool
	var includeExpiredTickets bool
	var jsonOutput bool
	flags := flag.NewFlagSet("statecutover", flag.ExitOnError)
	flags.StringVar(&source, "source", "", "legacy platform-state.json path")
	flags.StringVar(&sqlitePath, "sqlite", "", "explicit target sqlite database path for future writable imports")
	flags.BoolVar(&apply, "apply", false, "perform writable import; currently disabled")
	flags.BoolVar(&includeExpiredTickets, "include-expired-tickets", false, "include expired runtime join tickets in importable row counts")
	flags.BoolVar(&jsonOutput, "json", false, "print JSON output")
	_ = flags.Parse(os.Args[1:])

	if strings.TrimSpace(source) == "" {
		fmt.Fprintln(os.Stderr, "statecutover: --source is required")
		os.Exit(2)
	}
	if apply {
		if strings.TrimSpace(sqlitePath) == "" {
			fmt.Fprintln(os.Stderr, "statecutover: --sqlite is required for writable import")
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "statecutover: writable import is intentionally disabled in this milestone; use dry-run report for the manual cutover gate")
		os.Exit(2)
	}

	report, err := statecutover.AnalyzeFile(statecutover.Options{
		SourcePath:            source,
		IncludeExpiredTickets: includeExpiredTickets,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "statecutover: analyze: %v\n", err)
		os.Exit(1)
	}
	if jsonOutput {
		writeJSON(report)
		return
	}

	fmt.Printf("AmandaCore legacy state cutover dry run for %s\n", source)
	fmt.Printf("accounts=%d characters=%d sessions=%d tickets=%d importableTickets=%d\n",
		report.Counts.Accounts,
		report.Counts.Characters,
		report.Counts.Sessions,
		report.Counts.WorldJoinTickets,
		report.Counts.ImportableWorldTickets)
	fmt.Printf("inventorySlots=%d questStates=%d actionBarSlots=%d socialRows=%d economyRows=%d\n",
		report.Counts.InventorySlots,
		report.Counts.QuestStates,
		report.Counts.ActionBarSlots,
		report.Counts.Friends+report.Counts.Parties+report.Counts.Guilds+report.Counts.GuildInvites,
		report.Counts.Auctions+report.Counts.Mail)
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func writeJSON(value any) {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "statecutover: encode JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(payload))
}
