package loadsim

import (
	"testing"
	"time"
)

func TestReconnectPressureScenarioValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scenario = ScenarioReconnectPressure
	cfg.Clients = 4
	cfg.Duration = 5 * time.Second
	cfg.CommandRate = 2
	cfg.ReconnectRate = 0.25
	cfg.ReconnectInterval = time.Second
	cfg.QueueCapacity = 64

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected reconnect pressure config to validate: %v", err)
	}
}
