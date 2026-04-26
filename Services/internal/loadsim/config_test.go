package loadsim

import (
	"math/rand"
	"testing"
	"time"
)

func TestLoadsimConfigParsesValidFlags(t *testing.T) {
	cfg, err := ParseConfig([]string{
		"--scenario", ScenarioMultizonePressure,
		"--clients", "25",
		"--duration", "15s",
		"--cmd-rate", "3.5",
		"--seed", "42",
		"--tick-ms", "25",
		"--queue-capacity", "32",
		"--shards", "2",
		"--assignment-policy", string(AssignmentHashZone),
	})
	if err != nil {
		t.Fatalf("expected valid flags: %v", err)
	}
	if cfg.Clients != 25 || cfg.Duration != 15*time.Second || cfg.CommandRate != 3.5 || cfg.Seed != 42 {
		t.Fatalf("parsed config mismatch: %#v", cfg)
	}
	if cfg.TickDuration != 25*time.Millisecond || cfg.QueueCapacity != 32 || cfg.ShardCount != 2 {
		t.Fatalf("parsed runtime config mismatch: %#v", cfg)
	}
}

func TestLoadsimConfigRejectsInvalidValues(t *testing.T) {
	cases := [][]string{
		{"--clients", "0"},
		{"--duration", "0s"},
		{"--cmd-rate", "-1"},
		{"--tick-ms", "0"},
		{"--queue-capacity", "0"},
	}
	for _, args := range cases {
		if _, err := ParseConfig(args); err == nil {
			t.Fatalf("expected args %v to fail", args)
		}
	}
}

func TestDeterministicSeedProducesRepeatableDistribution(t *testing.T) {
	zoneIDs := []string{"a", "b", "c"}
	plan := ZoneDistributionPlan{Mode: "weighted", Weights: map[string]int{"a": 1, "b": 2, "c": 3}}
	left, err := AssignClientZones(plan, zoneIDs, 20, rand.New(rand.NewSource(99)))
	if err != nil {
		t.Fatal(err)
	}
	right, err := AssignClientZones(plan, zoneIDs, 20, rand.New(rand.NewSource(99)))
	if err != nil {
		t.Fatal(err)
	}
	for index := range left {
		if left[index] != right[index] {
			t.Fatalf("seeded distribution differs at %d: %v != %v", index, left, right)
		}
	}
}

func TestEvenZoneDistributionAssignsAcrossZones(t *testing.T) {
	assignments, err := AssignClientZones(ZoneDistributionPlan{Mode: "even"}, []string{"a", "b", "c"}, 6, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, zoneID := range assignments {
		counts[zoneID]++
	}
	for _, zoneID := range []string{"a", "b", "c"} {
		if counts[zoneID] != 2 {
			t.Fatalf("expected even assignment for %s, got %v", zoneID, counts)
		}
	}
}

func TestWeightedDistributionRejectsMissingZone(t *testing.T) {
	if _, err := ParseZoneDistribution("weighted:a=1,missing=2", []string{"a"}); err == nil {
		t.Fatalf("expected missing zone rejection")
	}
}
