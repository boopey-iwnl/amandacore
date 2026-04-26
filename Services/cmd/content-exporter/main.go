package main

import (
	"flag"
	"fmt"
	"os"

	contentpkg "amandacore/services/internal/content"
)

func main() {
	input := flag.String("input", "", "authoring metadata directory")
	output := flag.String("output", "", "map export output directory")
	check := flag.Bool("check", false, "compare generated exports with output files without writing")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "--input and --output are required")
		os.Exit(2)
	}

	result := contentpkg.GenerateMapExportsFromAuthoringDirectory(*input)
	if !result.Validation.Valid() {
		for _, validationError := range result.Validation.Errors {
			fmt.Fprintf(os.Stderr, "%s at %s: %s\n", validationError.Code, validationError.Path, validationError.Message)
		}
		os.Exit(1)
	}

	if *check {
		checkResult, err := contentpkg.CheckMapExports(*output, result.Exports)
		if err != nil {
			fmt.Fprintf(os.Stderr, "check failed: %v\n", err)
			os.Exit(1)
		}
		for _, missing := range checkResult.Missing {
			fmt.Fprintf(os.Stderr, "missing generated export: %s\n", missing)
		}
		for _, drift := range checkResult.Drift {
			fmt.Fprintf(os.Stderr, "generated export is stale: %s\n", drift)
		}
		if len(checkResult.Missing) > 0 || len(checkResult.Drift) > 0 {
			os.Exit(1)
		}
		fmt.Printf("content-exporter check passed: %d exports compared\n", len(checkResult.Compared))
		return
	}

	writeResult, err := contentpkg.WriteMapExports(*output, result.Exports)
	if err != nil {
		fmt.Fprintf(os.Stderr, "export failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("content-exporter wrote %d map exports (%d changed)\n", len(writeResult.Written), len(writeResult.Changed))
	for _, path := range writeResult.Written {
		fmt.Printf("- %s\n", path)
	}
}
