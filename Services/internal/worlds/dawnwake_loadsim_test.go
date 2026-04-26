package worlds

import (
	"testing"
	"time"
)

func TestDawnwakeStreamingLoadsimCompletesTransitionProbe(t *testing.T) {
	report, err := RunDawnwakeStreamingLoadsim(DawnwakeStreamingLoadsimOptions{
		Clients:     2,
		Duration:    100 * time.Millisecond,
		CommandRate: 2,
		Scenario:    "dawnwake-streaming-basic",
		ContentPath: defaultDawnwakePackagePath,
	})
	if err != nil {
		t.Fatalf("expected Dawnwake loadsim to succeed: %v; report=%#v", err, report)
	}
	if len(report.Errors) > 0 || len(report.ValidationErrors) > 0 {
		t.Fatalf("expected clean report, got %#v", report)
	}
	if !report.ContentPackageLoaded {
		t.Fatalf("expected content package to load")
	}
	if report.ContinentID != "dawnwake_isles" {
		t.Fatalf("unexpected continent id %q", report.ContinentID)
	}
	if report.ZonesActivated < 5 {
		t.Fatalf("expected at least 5 activated zones, got %d", report.ZonesActivated)
	}
	if report.TransitionGatesLoaded < 8 {
		t.Fatalf("expected at least 8 transition gates, got %d", report.TransitionGatesLoaded)
	}
	if report.ZoneTransitionsCompleted != 2 {
		t.Fatalf("expected one completed transition per client, got %d", report.ZoneTransitionsCompleted)
	}
	if report.P95TickDurationMs > report.MaxTickDurationMs {
		t.Fatalf("expected p95 tick duration <= max tick duration, got p95=%f max=%f", report.P95TickDurationMs, report.MaxTickDurationMs)
	}
	if report.NPCsSpawned == 0 || report.QuestProvidersRegistered == 0 {
		t.Fatalf("expected activated NPCs and quest providers, got %#v", report)
	}
}

func TestDawnwakeMultizoneShardingLoadsimDistributesClients(t *testing.T) {
	report, err := RunDawnwakeStreamingLoadsim(DawnwakeStreamingLoadsimOptions{
		Clients:         12,
		Duration:        100 * time.Millisecond,
		CommandRate:     4,
		Scenario:        "dawnwake-multizone-sharding-basic",
		ContentPath:     defaultDawnwakePackagePath,
		TransitionLoops: 3,
		Seed:            7,
		ShardCount:      2,
	})
	if err != nil {
		t.Fatalf("expected Dawnwake multizone loadsim to succeed: %v; report=%#v", err, report)
	}
	if len(report.Errors) > 0 || len(report.ValidationErrors) > 0 {
		t.Fatalf("expected clean report, got %#v", report)
	}
	if report.ZoneTransitionsCompleted != 36 {
		t.Fatalf("expected 36 completed transitions, got %d", report.ZoneTransitionsCompleted)
	}
	if report.ZoneTransitionsRejected != 0 || report.RejectedCommands != 0 {
		t.Fatalf("expected no rejected transitions, got %#v", report)
	}
	if len(report.ZonePopulation) < 2 {
		t.Fatalf("expected clients to distribute across zones, got %#v", report.ZonePopulation)
	}
	if len(report.ShardPopulation) != 2 {
		t.Fatalf("expected both shards to receive final population, got %#v", report.ShardPopulation)
	}
	if report.MaxQueueDepth != 36 {
		t.Fatalf("expected max queue depth 36, got %d", report.MaxQueueDepth)
	}
	if report.P95TickDurationMs > report.MaxTickDurationMs {
		t.Fatalf("expected p95 tick duration <= max tick duration, got p95=%f max=%f", report.P95TickDurationMs, report.MaxTickDurationMs)
	}
}

func TestDawnwakeDisabledTransitionIsNotSelected(t *testing.T) {
	registry := loadDawnwakeRegistryForShardingTest(t)
	gate, found := selectEnabledTransition(registry.Zones["kingsfall_harbor"], 0, 0, 1)
	if !found {
		t.Fatalf("expected enabled Kingsfall transition")
	}
	if gate.Disabled {
		t.Fatalf("selected disabled transition %#v", gate)
	}
	if gate.TransitionID == "kingsfall_harbor_route_placeholder" {
		t.Fatalf("selected disabled harbor placeholder transition")
	}
}

func TestDawnwakeContentActivationPreservesZoneTransitions(t *testing.T) {
	server := newWorldServerWithContentPackage(nil, defaultDawnwakePackagePath)

	landing := server.zones["dawnwake_landing"]
	if len(landing.Transitions) == 0 {
		t.Fatalf("expected Dawnwake Landing transition landmarks to activate")
	}
	if landing.Transitions[0].ID != "to_tideglass_shoal" {
		t.Fatalf("unexpected first runtime transition: %#v", landing.Transitions[0])
	}
	if server.contentRegistry == nil || len(server.contentRegistry.Zones["dawnwake_landing"].TransitionGates) == 0 {
		t.Fatalf("expected Dawnwake Landing transition gates in content registry")
	}
	if server.contentRegistry.Zones["dawnwake_landing"].TransitionGates[0].TransitionID != "landing_to_amberglass_road" {
		t.Fatalf("unexpected first content transition gate: %#v", server.contentRegistry.Zones["dawnwake_landing"].TransitionGates[0])
	}
}
