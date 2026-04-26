package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type bounds2D struct {
	MinX float64 `json:"min_x"`
	MinY float64 `json:"min_y"`
	MaxX float64 `json:"max_x"`
	MaxY float64 `json:"max_y"`
}

type landmark struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
}

type route struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

type authoredMap struct {
	ZoneID           string     `json:"zone_id"`
	DisplayName      string     `json:"display_name"`
	NormalizedBounds bounds2D   `json:"normalized_bounds"`
	WorldBounds      bounds2D   `json:"world_bounds"`
	Landmarks        []landmark `json:"landmarks"`
	Routes           []route    `json:"routes"`
}

type exportedMap struct {
	DisplayName      string     `json:"display_name"`
	Landmarks        []landmark `json:"landmarks"`
	NormalizedBounds bounds2D   `json:"normalized_bounds"`
	Routes           []route    `json:"routes"`
	SchemaVersion    string     `json:"schema_version"`
	Source           string     `json:"source"`
	WorldBounds      bounds2D   `json:"world_bounds"`
	ZoneID           string     `json:"zone_id"`
}

func main() {
	var inputDir string
	var outputDir string
	var checkOnly bool
	flag.StringVar(&inputDir, "input", "", "authoring directory")
	flag.StringVar(&outputDir, "output", "", "exported map output directory")
	flag.BoolVar(&checkOnly, "check", false, "verify exported files are up to date without writing")
	flag.Parse()

	if strings.TrimSpace(inputDir) == "" {
		exitf("--input is required")
	}
	if strings.TrimSpace(outputDir) == "" {
		exitf("--output is required")
	}
	files, err := filepath.Glob(filepath.Join(inputDir, "*.authoring.json"))
	if err != nil {
		exitf("find authoring files: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		exitf("no *.authoring.json files found in %s", inputDir)
	}
	if !checkOnly {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			exitf("create output directory: %v", err)
		}
	}

	failures := 0
	for _, path := range files {
		payload, err := os.ReadFile(path)
		if err != nil {
			exitf("read %s: %v", path, err)
		}
		var authored authoredMap
		if err := json.Unmarshal(payload, &authored); err != nil {
			exitf("decode %s: %v", path, err)
		}
		if err := validateAuthoredMap(authored, path); err != nil {
			exitf("%v", err)
		}
		exported := exportedMap{
			DisplayName:      authored.DisplayName,
			Landmarks:        authored.Landmarks,
			NormalizedBounds: authored.NormalizedBounds,
			Routes:           authored.Routes,
			SchemaVersion:    "1",
			Source:           "amandacore.content_exporter",
			WorldBounds:      authored.WorldBounds,
			ZoneID:           authored.ZoneID,
		}
		out, err := json.MarshalIndent(exported, "", "  ")
		if err != nil {
			exitf("encode %s: %v", authored.ZoneID, err)
		}
		out = append(out, '\n')
		outputPath := filepath.Join(outputDir, authored.ZoneID+".map.json")
		if checkOnly {
			existing, err := os.ReadFile(outputPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "missing exported map %s: %v\n", outputPath, err)
				failures++
				continue
			}
			if !bytes.Equal(existing, out) {
				fmt.Fprintf(os.Stderr, "exported map %s is out of date\n", outputPath)
				failures++
			}
			continue
		}
		if err := os.WriteFile(outputPath, out, 0o644); err != nil {
			exitf("write %s: %v", outputPath, err)
		}
		fmt.Printf("exported %s\n", outputPath)
	}
	if failures > 0 {
		os.Exit(1)
	}
	if checkOnly {
		fmt.Printf("content-exporter check passed for %d map file(s)\n", len(files))
	}
}

func validateAuthoredMap(authored authoredMap, path string) error {
	if strings.TrimSpace(authored.ZoneID) == "" {
		return fmt.Errorf("%s: zone_id is required", path)
	}
	if strings.TrimSpace(authored.DisplayName) == "" {
		return fmt.Errorf("%s: display_name is required", path)
	}
	if authored.NormalizedBounds.MaxX <= authored.NormalizedBounds.MinX || authored.NormalizedBounds.MaxY <= authored.NormalizedBounds.MinY {
		return fmt.Errorf("%s: normalized_bounds are malformed", path)
	}
	if authored.WorldBounds.MaxX <= authored.WorldBounds.MinX || authored.WorldBounds.MaxY <= authored.WorldBounds.MinY {
		return fmt.Errorf("%s: world_bounds are malformed", path)
	}
	landmarks := map[string]struct{}{}
	for index, landmark := range authored.Landmarks {
		if strings.TrimSpace(landmark.ID) == "" {
			return fmt.Errorf("%s: landmarks[%d].id is required", path, index)
		}
		if _, exists := landmarks[landmark.ID]; exists {
			return fmt.Errorf("%s: landmark id %q is duplicated", path, landmark.ID)
		}
		landmarks[landmark.ID] = struct{}{}
		if landmark.X < authored.NormalizedBounds.MinX || landmark.X > authored.NormalizedBounds.MaxX ||
			landmark.Y < authored.NormalizedBounds.MinY || landmark.Y > authored.NormalizedBounds.MaxY {
			return fmt.Errorf("%s: landmark %q is outside normalized bounds", path, landmark.ID)
		}
	}
	for index, route := range authored.Routes {
		if strings.TrimSpace(route.ID) == "" {
			return fmt.Errorf("%s: routes[%d].id is required", path, index)
		}
		if _, found := landmarks[route.From]; !found {
			return fmt.Errorf("%s: route %q references missing from landmark %q", path, route.ID, route.From)
		}
		if _, found := landmarks[route.To]; !found {
			return fmt.Errorf("%s: route %q references missing to landmark %q", path, route.ID, route.To)
		}
	}
	return nil
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
