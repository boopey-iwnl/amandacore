package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type httpContractManifest struct {
	ContractID     string                      `json:"contractId"`
	Status         string                      `json:"status"`
	Routes         []httpContractRoute         `json:"routes"`
	ResponseShapes []httpContractResponseShape `json:"responseShapes"`
}

type httpContractRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type httpContractResponseShape struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
}

func TestHTTPContractManifestMatchesRegisteredRoutes(t *testing.T) {
	repoRoot := findRepoRoot(t)
	manifestRoutes := loadManifestRoutes(t, filepath.Join(repoRoot, "Docs", "Contracts", "http-api-v1.json"))
	sourceRoutes := collectRegisteredRoutes(t, filepath.Join(repoRoot, "Services"))

	assertSameRouteSet(t, manifestRoutes, sourceRoutes)
}

func TestHTTPContractManifestIsSortedAndUnique(t *testing.T) {
	repoRoot := findRepoRoot(t)
	manifest := loadManifest(t, filepath.Join(repoRoot, "Docs", "Contracts", "http-api-v1.json"))

	var previous string
	seen := make(map[string]bool, len(manifest.Routes))
	for index, route := range manifest.Routes {
		key := routeKey(route.Method, route.Path)
		if seen[key] {
			t.Fatalf("duplicate route in manifest: %s", key)
		}
		seen[key] = true

		if index > 0 && key < previous {
			t.Fatalf("manifest routes must be sorted by method then path: %s appears after %s", key, previous)
		}
		previous = key
	}
}

func TestHTTPContractManifestDocumentsInventoryEquipmentPayloads(t *testing.T) {
	repoRoot := findRepoRoot(t)
	manifest := loadManifest(t, filepath.Join(repoRoot, "Docs", "Contracts", "http-api-v1.json"))

	requiredShapes := map[string][]string{
		"worldSession.inventory.slots[]": {
			"slotIndex",
			"itemId",
			"displayName",
			"stackCount",
			"itemType",
			"itemSubtype",
			"quality",
			"iconKind",
			"description",
			"equipSlot",
			"requiredArchetype",
			"requiredLevel",
			"sellPriceCopper",
			"strength",
			"stamina",
			"armor",
		},
		"worldSession.equipment.slots[]": {
			"slot",
			"itemId",
			"displayName",
			"itemType",
			"itemSubtype",
			"quality",
			"iconKind",
			"description",
			"equipSlot",
			"requiredArchetype",
			"requiredLevel",
			"sellPriceCopper",
			"strength",
			"stamina",
			"armor",
		},
	}

	shapesByName := map[string]map[string]bool{}
	for _, shape := range manifest.ResponseShapes {
		fields := map[string]bool{}
		for _, field := range shape.Fields {
			fields[field] = true
		}
		shapesByName[shape.Name] = fields
	}

	for name, fields := range requiredShapes {
		shapeFields, found := shapesByName[name]
		if !found {
			t.Fatalf("response shape %s is not documented", name)
		}
		for _, field := range fields {
			if !shapeFields[field] {
				t.Fatalf("response shape %s missing field %s", name, field)
			}
		}
	}
}

func loadManifestRoutes(t *testing.T, path string) map[string]httpContractRoute {
	t.Helper()

	manifest := loadManifest(t, path)
	routes := make(map[string]httpContractRoute, len(manifest.Routes))
	for _, route := range manifest.Routes {
		route.Method = strings.TrimSpace(route.Method)
		route.Path = strings.TrimSpace(route.Path)
		if route.Method == "" || route.Path == "" {
			t.Fatalf("manifest route must include method and path: %#v", route)
		}
		routes[routeKey(route.Method, route.Path)] = route
	}
	return routes
}

func loadManifest(t *testing.T, path string) httpContractManifest {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read HTTP contract manifest: %v", err)
	}

	var manifest httpContractManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to parse HTTP contract manifest: %v", err)
	}
	if manifest.ContractID != "amandacore.http-api.v1" {
		t.Fatalf("unexpected contract id %q", manifest.ContractID)
	}
	if strings.TrimSpace(manifest.Status) == "" {
		t.Fatal("manifest status is required")
	}

	return manifest
}

func collectRegisteredRoutes(t *testing.T, servicesRoot string) map[string]httpContractRoute {
	t.Helper()

	routePattern := regexp.MustCompile(`"(GET|POST) ([^"]+)"`)
	routes := map[string]httpContractRoute{}

	err := filepath.WalkDir(servicesRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, match := range routePattern.FindAllStringSubmatch(string(data), -1) {
			method := match[1]
			path := match[2]
			routes[routeKey(method, path)] = httpContractRoute{Method: method, Path: path}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed to collect registered routes: %v", err)
	}

	return routes
}

func assertSameRouteSet(t *testing.T, manifestRoutes map[string]httpContractRoute, sourceRoutes map[string]httpContractRoute) {
	t.Helper()

	missingFromManifest := differenceKeys(sourceRoutes, manifestRoutes)
	extraInManifest := differenceKeys(manifestRoutes, sourceRoutes)

	if len(missingFromManifest) > 0 || len(extraInManifest) > 0 {
		t.Fatalf(
			"HTTP contract manifest drift detected\nmissing from manifest:\n%s\nextra in manifest:\n%s",
			formatRouteKeys(missingFromManifest),
			formatRouteKeys(extraInManifest))
	}
}

func differenceKeys(left map[string]httpContractRoute, right map[string]httpContractRoute) []string {
	var diff []string
	for key := range left {
		if _, ok := right[key]; !ok {
			diff = append(diff, key)
		}
	}
	sort.Strings(diff)
	return diff
}

func formatRouteKeys(keys []string) string {
	if len(keys) == 0 {
		return "  <none>"
	}

	return "  " + strings.Join(keys, "\n  ")
}

func routeKey(method string, path string) string {
	return strings.TrimSpace(method) + " " + strings.TrimSpace(path)
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if fileExists(filepath.Join(dir, "project.json")) && dirExists(filepath.Join(dir, "Services")) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("failed to locate repository root from test working directory")
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
