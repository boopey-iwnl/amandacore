package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	contentpkg "amandacore/services/internal/content"
)

func main() {
	packagePath := flag.String("package", "", "content package manifest path")
	outPath := flag.String("out", "", "compiled content output path")
	check := flag.Bool("check", false, "validate package and, when --out is set, compare compiled output without writing")
	flag.Parse()

	result := contentpkg.CompileContentPackage(*packagePath)
	if !result.Validation.Valid() {
		for _, validationError := range result.Validation.Errors {
			fmt.Fprintf(os.Stderr, "%s at %s: %s\n", validationError.Code, validationError.Path, validationError.Message)
		}
		os.Exit(1)
	}

	payload, err := contentpkg.MarshalCompiledContentPackage(result.Package)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile failed: %v\n", err)
		os.Exit(1)
	}
	payload = append(payload, '\n')

	if *check {
		if *outPath == "" {
			fmt.Printf("content-compiler check passed: %s %s (%s)\n", result.Package.PackageID, result.Package.Version, result.Package.ContentSHA256)
			return
		}
		existing, err := os.ReadFile(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "compiled output %q could not be read: %v\n", *outPath, err)
			os.Exit(1)
		}
		if !bytes.Equal(existing, payload) {
			fmt.Fprintf(os.Stderr, "compiled output is stale: %s\n", *outPath)
			os.Exit(1)
		}
		fmt.Printf("content-compiler check passed: %s\n", *outPath)
		return
	}

	if *outPath == "" {
		fmt.Print(string(payload))
		return
	}
	if err := os.WriteFile(*outPath, payload, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "compiled output %q could not be written: %v\n", *outPath, err)
		os.Exit(1)
	}
	fmt.Printf("content-compiler wrote %s (%s)\n", *outPath, result.Package.ContentSHA256)
}
