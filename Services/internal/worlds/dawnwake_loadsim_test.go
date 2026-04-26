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
	if report.ZonesActivated != 5 {
		t.Fatalf("expected 5 activated zones, got %d", report.ZonesActivated)
	}
	if report.TransitionGatesLoaded < 8 {
		t.Fatalf("expected at least 8 transition gates, got %d", report.TransitionGatesLoaded)
	}
	if report.ZoneTransitionsCompleted != 2 {
		t.Fatalf("expected one completed transition per client, got %d", report.ZoneTransitionsCompleted)
	}
	if report.NPCsSpawned == 0 || report.QuestProvidersRegistered == 0 {
		t.Fatalf("expected activated NPCs and quest providers, got %#v", report)
	}
}

func TestDawnwakeContentActivationPreservesZoneTransitions(t *testing.T) {
	server := newWorldServerWithContentPackage(nil, defaultDawnwakePackagePath)

	landing := server.zones["dawnwake_landing"]
	if len(landing.Transitions) == 0 {
		t.Fatalf("expected Dawnwake Landing transition landmarks to activate")
	}
	if landing.Transitions[0].ID != "landing_to_amberglass_road" {
		t.Fatalf("unexpected first transition: %#v", landing.Transitions[0])
	}
}
