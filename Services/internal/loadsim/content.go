package loadsim

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const DefaultContentPath = "Content/Packs/dawnwake_isles/package.json"

type ContentPackage struct {
	PackageID   string              `json:"packageId"`
	DisplayName string              `json:"displayName"`
	Version     string              `json:"version"`
	Zones       map[string]ZoneSpec `json:"zones"`
	ZoneOrder   []string            `json:"zoneOrder"`
}

type packageManifest struct {
	PackageID   string   `json:"package_id"`
	DisplayName string   `json:"display_name"`
	Version     string   `json:"version"`
	Zones       []string `json:"zones"`
}

type ZoneSpec struct {
	ZoneID          string           `json:"zone_id"`
	DisplayName     string           `json:"display_name"`
	Bounds          Bounds           `json:"bounds"`
	EntryPoints     []EntryPoint     `json:"entry_points"`
	TransitionGates []TransitionGate `json:"transition_gates"`
	Runtime         ZoneRuntimeHints `json:"runtime"`
}

type Bounds struct {
	MinX float64 `json:"min_x"`
	MinY float64 `json:"min_y"`
	MinZ float64 `json:"min_z"`
	MaxX float64 `json:"max_x"`
	MaxY float64 `json:"max_y"`
	MaxZ float64 `json:"max_z"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type EntryPoint struct {
	EntryID      string `json:"entry_id"`
	EntryPointID string `json:"entry_point_id"`
	Position     Point  `json:"position"`
}

type TransitionGate struct {
	TransitionID                 string `json:"transition_id"`
	ToZoneID                     string `json:"to_zone_id"`
	EntryPointIDOnArrival        string `json:"entry_id_on_arrival"`
	EntryPointIDOnArrivalVerbose string `json:"entry_point_id_on_arrival"`
	Disabled                     bool   `json:"disabled"`
}

type ZoneRuntimeHints struct {
	MaxSessions int `json:"max_sessions"`
	MaxEntities int `json:"max_entities"`
}

func LoadContentPackage(manifestPath string) (ContentPackage, error) {
	resolved, err := ResolveContentPath(manifestPath)
	if err != nil {
		return ContentPackage{}, err
	}
	payload, err := os.ReadFile(resolved)
	if err != nil {
		return ContentPackage{}, fmt.Errorf("read content package: %w", err)
	}
	var manifest packageManifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return ContentPackage{}, fmt.Errorf("decode content package: %w", err)
	}
	if strings.TrimSpace(manifest.PackageID) == "" {
		return ContentPackage{}, fmt.Errorf("content package_id is required")
	}
	root := filepath.Dir(resolved)
	pkg := ContentPackage{
		PackageID:   manifest.PackageID,
		DisplayName: manifest.DisplayName,
		Version:     manifest.Version,
		Zones:       map[string]ZoneSpec{},
		ZoneOrder:   []string{},
	}
	for _, zonePath := range manifest.Zones {
		zoneFile := filepath.Join(root, zonePath)
		zonePayload, err := os.ReadFile(zoneFile)
		if err != nil {
			return ContentPackage{}, fmt.Errorf("read zone %s: %w", zonePath, err)
		}
		var zone ZoneSpec
		if err := json.Unmarshal(zonePayload, &zone); err != nil {
			return ContentPackage{}, fmt.Errorf("decode zone %s: %w", zonePath, err)
		}
		normalizeZoneSpec(&zone)
		if err := validateZone(zone); err != nil {
			return ContentPackage{}, fmt.Errorf("validate zone %s: %w", zonePath, err)
		}
		if _, exists := pkg.Zones[zone.ZoneID]; exists {
			return ContentPackage{}, fmt.Errorf("duplicate zone id %s", zone.ZoneID)
		}
		pkg.Zones[zone.ZoneID] = zone
		pkg.ZoneOrder = append(pkg.ZoneOrder, zone.ZoneID)
	}
	if len(pkg.ZoneOrder) == 0 {
		return ContentPackage{}, fmt.Errorf("content package must contain at least one zone")
	}
	if err := validateTransitions(pkg); err != nil {
		return ContentPackage{}, err
	}
	return pkg, nil
}

func normalizeZoneSpec(zone *ZoneSpec) {
	if zone == nil {
		return
	}
	for index := range zone.EntryPoints {
		if zone.EntryPoints[index].EntryID == "" {
			zone.EntryPoints[index].EntryID = zone.EntryPoints[index].EntryPointID
		}
	}
	for index := range zone.TransitionGates {
		if zone.TransitionGates[index].EntryPointIDOnArrival == "" {
			zone.TransitionGates[index].EntryPointIDOnArrival = zone.TransitionGates[index].EntryPointIDOnArrivalVerbose
		}
	}
}

func ResolveContentPath(manifestPath string) (string, error) {
	requested := strings.TrimSpace(manifestPath)
	if requested == "" {
		requested = DefaultContentPath
	}
	if filepath.IsAbs(requested) {
		if _, err := os.Stat(requested); err != nil {
			return "", err
		}
		return filepath.Clean(requested), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Clean(filepath.Join(cwd, requested))
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			return "", fmt.Errorf("content package %s was not found from working directory parents", requested)
		}
		cwd = parent
	}
}

func validateZone(zone ZoneSpec) error {
	if zone.ZoneID == "" {
		return fmt.Errorf("zone_id is required")
	}
	if zone.DisplayName == "" {
		return fmt.Errorf("display_name is required for zone %s", zone.ZoneID)
	}
	if zone.Bounds.MaxX <= zone.Bounds.MinX || zone.Bounds.MaxY <= zone.Bounds.MinY || zone.Bounds.MaxZ < zone.Bounds.MinZ {
		return fmt.Errorf("invalid bounds for zone %s", zone.ZoneID)
	}
	if len(zone.EntryPoints) == 0 {
		return fmt.Errorf("zone %s must define at least one entry point", zone.ZoneID)
	}
	seenEntries := map[string]struct{}{}
	for _, entry := range zone.EntryPoints {
		if entry.EntryID == "" {
			return fmt.Errorf("zone %s has an entry point without entry_id", zone.ZoneID)
		}
		if _, exists := seenEntries[entry.EntryID]; exists {
			return fmt.Errorf("zone %s has duplicate entry point %s", zone.ZoneID, entry.EntryID)
		}
		seenEntries[entry.EntryID] = struct{}{}
		if !zone.Bounds.Contains(entry.Position) {
			return fmt.Errorf("entry point %s is outside zone %s", entry.EntryID, zone.ZoneID)
		}
	}
	return nil
}

func validateTransitions(pkg ContentPackage) error {
	for _, zoneID := range pkg.ZoneOrder {
		zone := pkg.Zones[zoneID]
		seenTransitions := map[string]struct{}{}
		for _, gate := range zone.TransitionGates {
			if gate.Disabled {
				continue
			}
			if gate.TransitionID == "" {
				return fmt.Errorf("zone %s has transition without transition_id", zoneID)
			}
			if _, exists := seenTransitions[gate.TransitionID]; exists {
				return fmt.Errorf("zone %s has duplicate transition %s", zoneID, gate.TransitionID)
			}
			seenTransitions[gate.TransitionID] = struct{}{}
			destination, ok := pkg.Zones[gate.ToZoneID]
			if !ok {
				return fmt.Errorf("transition %s references missing zone %s", gate.TransitionID, gate.ToZoneID)
			}
			if _, ok := destination.EntryPoint(gate.EntryPointIDOnArrival); !ok {
				return fmt.Errorf("transition %s references missing arrival entry %s", gate.TransitionID, gate.EntryPointIDOnArrival)
			}
		}
	}
	return nil
}

func (zone ZoneSpec) EntryPoint(entryID string) (EntryPoint, bool) {
	for _, entry := range zone.EntryPoints {
		if entry.EntryID == entryID {
			return entry, true
		}
	}
	return EntryPoint{}, false
}

func (zone ZoneSpec) DefaultEntryPoint() EntryPoint {
	if entry, ok := zone.EntryPoint("default"); ok {
		return entry
	}
	return zone.EntryPoints[0]
}

func (zone ZoneSpec) GateTo(targetZoneID string) (TransitionGate, bool) {
	for _, gate := range zone.TransitionGates {
		if gate.Disabled {
			continue
		}
		if gate.ToZoneID == targetZoneID {
			return gate, true
		}
	}
	return TransitionGate{}, false
}

func (bounds Bounds) Contains(point Point) bool {
	return point.X >= bounds.MinX && point.X <= bounds.MaxX &&
		point.Y >= bounds.MinY && point.Y <= bounds.MaxY &&
		point.Z >= bounds.MinZ && point.Z <= bounds.MaxZ
}

func SortedZoneIDs(zones map[string]ZoneSpec) []string {
	ids := make([]string, 0, len(zones))
	for id := range zones {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
