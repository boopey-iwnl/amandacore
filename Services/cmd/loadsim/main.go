package main

import (
	"context"
	"fmt"
	"os"

	"amandacore/services/internal/loadsim"
)

func main() {
	cfg, err := loadsim.ParseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadsim: %v\n", err)
		os.Exit(2)
	}
	report, err := loadsim.Run(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadsim run failed: %v\n", err)
		fmt.Print(loadsim.RenderTextReport(report))
		os.Exit(1)
	}
	if err := loadsim.WriteJSONReport(cfg.ReportPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "write report failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(loadsim.RenderTextReport(report))
}
