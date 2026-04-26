package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestParseZoneDistributionNormalizesValidWeights(t *testing.T) {
	valid := map[string]bool{"dawnwake_landing": true, "amberglass_fields": true}
	weights, err := parseZoneDistribution("dawnwake_landing=40%, amberglass_fields=60", valid)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(weights) != 2 {
		t.Fatalf("expected 2 weights, got %#v", weights)
	}
	if weights[0].ZoneID != "dawnwake_landing" || weights[0].Fraction < 0.39 || weights[0].Fraction > 0.41 {
		t.Fatalf("unexpected first weight: %#v", weights[0])
	}
	if weights[1].ZoneID != "amberglass_fields" || weights[1].Fraction < 0.59 || weights[1].Fraction > 0.61 {
		t.Fatalf("unexpected second weight: %#v", weights[1])
	}
}

func TestParseZoneDistributionRejectsUnknownZone(t *testing.T) {
	_, err := parseZoneDistribution("missing_zone=100", map[string]bool{"dawnwake_landing": true})
	if err == nil {
		t.Fatalf("expected unknown zone rejection")
	}
}

func TestCalculateTickStatsIncludesPercentiles(t *testing.T) {
	stats := calculateTickStats([]time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
	})
	if stats.Average != 25*time.Millisecond {
		t.Fatalf("unexpected average: %s", stats.Average)
	}
	if stats.P50 != 20*time.Millisecond || stats.P95 != 40*time.Millisecond || stats.P99 != 40*time.Millisecond || stats.Max != 40*time.Millisecond {
		t.Fatalf("unexpected percentiles: %#v", stats)
	}
}

func TestDawnwakePopulationScenarioReportsZoneDistribution(t *testing.T) {
	result := runScenario(options{
		Clients:          6,
		Duration:         time.Second,
		CmdRate:          1,
		Scenario:         scenarioDawnwakePopulation,
		Content:          dawnwakeContentPath(),
		ZoneDistribution: "dawnwake_landing=50,amberglass_fields=50",
		TransitionRate:   1,
	})
	if len(result.Errors) > 0 {
		t.Fatalf("scenario errors: %#v", result.Errors)
	}
	if result.PlayersAttached != 6 {
		t.Fatalf("expected 6 players, got %d", result.PlayersAttached)
	}
	if result.ZonePopulation["dawnwake_landing"] == 0 || result.ZonePopulation["amberglass_fields"] == 0 {
		t.Fatalf("expected both zones populated, got %#v", result.ZonePopulation)
	}
	if result.VisibilityEvaluations != 6 {
		t.Fatalf("expected visibility for each player, got %d", result.VisibilityEvaluations)
	}
	if result.ShardAssignmentCount != 5 {
		t.Fatalf("expected 5 shard assignments, got %d", result.ShardAssignmentCount)
	}
}

func TestDawnwakeCommandPressureReportsBackpressure(t *testing.T) {
	result := runScenario(options{
		Clients:        2,
		Duration:       3 * time.Second,
		CmdRate:        1,
		Scenario:       scenarioDawnwakeCommandPressure,
		Content:        dawnwakeContentPath(),
		TransitionRate: 1,
	})
	if len(result.Errors) > 0 {
		t.Fatalf("scenario errors: %#v", result.Errors)
	}
	if result.CommandsBackpressured == 0 || result.BackpressureEvents == 0 {
		t.Fatalf("expected backpressure, got %#v", result)
	}
	if result.MaxQueueDepth == 0 {
		t.Fatalf("expected max queue depth")
	}
	if result.CommandsByType["move"] == 0 {
		t.Fatalf("expected command type counts, got %#v", result.CommandsByType)
	}
}

func dawnwakeContentPath() string {
	return filepath.Join("..", "..", "..", "Content", "Packs", "dawnwake_isles", "package.json")
}
