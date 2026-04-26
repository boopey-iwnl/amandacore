package main

import (
	"testing"
	"time"
)

func TestRunSimulationTinyScenario(t *testing.T) {
	report, err := RunSimulation(SimulationOptions{
		Clients:             5,
		Duration:            2 * time.Second,
		CommandsPerSecond:   5,
		ReconnectPercentage: 20,
		RealmID:             "test-realm",
		ZoneID:              "test-zone",
	})
	if err != nil {
		t.Fatalf("simulation failed: %v", err)
	}
	if report.ClientsAttached != 5 {
		t.Fatalf("expected 5 attached clients, got %d", report.ClientsAttached)
	}
	if report.TotalCommandsSent == 0 {
		t.Fatalf("expected commands to be sent")
	}
	if report.TotalCommandsAccepted == 0 {
		t.Fatalf("expected commands to be accepted")
	}
	if report.ReconnectAttempts != 1 || report.ReconnectSuccesses != 1 {
		t.Fatalf("expected one reconnect attempt/success, got %d/%d", report.ReconnectAttempts, report.ReconnectSuccesses)
	}
	if report.Errors != 0 {
		t.Fatalf("expected no loadsim errors, got %d", report.Errors)
	}
}
