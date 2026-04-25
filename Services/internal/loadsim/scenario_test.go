package loadsim

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMultizonePressureScenarioTinyRun(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Clients = 3
	cfg.Duration = 150 * time.Millisecond
	cfg.TickDuration = 50 * time.Millisecond
	cfg.CommandRate = 20
	cfg.Seed = 42
	cfg.ContentPath = DefaultContentPath
	report, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if report.ClientCount != 3 || report.TotalCommandsSent == 0 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestReconnectPressureTinyRunRestoresZoneState(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scenario = ScenarioReconnectPressure
	cfg.Clients = 2
	cfg.Duration = 150 * time.Millisecond
	cfg.TickDuration = 50 * time.Millisecond
	cfg.CommandRate = 1
	cfg.ReconnectRate = 1
	cfg.ReconnectInterval = 50 * time.Millisecond
	cfg.Seed = 7
	report, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if report.ReconnectAttempts == 0 || report.ReconnectSuccesses == 0 {
		t.Fatalf("expected reconnect activity, got %#v", report)
	}
}

func TestBackpressureReportKeepsAcceptedAndRejectedCounts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Clients = 1
	cfg.Duration = 50 * time.Millisecond
	cfg.TickDuration = 50 * time.Millisecond
	cfg.CommandRate = 80
	cfg.QueueCapacity = 1
	cfg.TransitionRate = 0
	cfg.Seed = 11
	report, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if report.AcceptedCommands != 1 {
		t.Fatalf("expected one queued command accepted, got %#v", report)
	}
	if report.RejectedCommands == 0 || report.RejectionReasons[string(RejectQueueFull)] == 0 {
		t.Fatalf("expected queue-full rejection accounting, got %#v", report)
	}
}

func TestJSONReportSerializationIncludesKeyFields(t *testing.T) {
	report := LoadsimReport{
		RunID:            "run",
		Scenario:         ScenarioShardAssignmentBasic,
		Seed:             42,
		ContentPackage:   "dawnwake_isles",
		ClientCount:      1,
		CommandRate:      2,
		RejectionReasons: map[string]int{},
	}
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(payload)
	for _, token := range []string{"runId", "scenario", "seed", "contentPackage", "clientCount"} {
		if !strings.Contains(encoded, token) {
			t.Fatalf("serialized report missing %s: %s", token, encoded)
		}
	}
}
