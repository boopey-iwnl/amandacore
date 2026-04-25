package loadsim

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	ScenarioMovementBasic        = "movement-basic"
	ScenarioCombatBasic          = "combat-basic"
	ScenarioAbilityBasic         = "ability-basic"
	ScenarioQuestBasic           = "quest-basic"
	ScenarioDawnwakeTraversal    = "dawnwake-traversal-basic"
	ScenarioMultizonePressure    = "multizone-pressure"
	ScenarioShardAssignmentBasic = "shard-assignment-basic"
	ScenarioReconnectPressure    = "reconnect-pressure"
)

type LoadsimConfig struct {
	Scenario          string        `json:"scenario"`
	Clients           int           `json:"clients"`
	Duration          time.Duration `json:"duration"`
	CommandRate       float64       `json:"commandRate"`
	ContentPath       string        `json:"contentPath"`
	Continent         string        `json:"continent"`
	ZoneDistribution  string        `json:"zoneDistribution"`
	TransitionRate    float64       `json:"transitionRate"`
	CombatRate        float64       `json:"combatRate"`
	AbilityRate       float64       `json:"abilityRate"`
	QuestRate         float64       `json:"questRate"`
	ReconnectRate     float64       `json:"reconnectRate"`
	ReconnectInterval time.Duration `json:"reconnectInterval"`
	Seed              int64         `json:"seed"`
	ReportPath        string        `json:"reportPath"`
	TickDuration      time.Duration `json:"tickDuration"`
	QueueCapacity     int           `json:"queueCapacity"`
	Verbose           bool          `json:"verbose"`
	ShardCount        int           `json:"shards"`
	AssignmentPolicy  string        `json:"assignmentPolicy"`
}

func DefaultConfig() LoadsimConfig {
	return LoadsimConfig{
		Scenario:          ScenarioMultizonePressure,
		Clients:           5,
		Duration:          10 * time.Second,
		CommandRate:       2,
		ContentPath:       DefaultContentPath,
		Continent:         "dawnwake_isles",
		ZoneDistribution:  "even",
		TransitionRate:    0.10,
		CombatRate:        0,
		AbilityRate:       0,
		QuestRate:         0,
		ReconnectRate:     0,
		ReconnectInterval: 10 * time.Second,
		Seed:              time.Now().UTC().UnixNano(),
		TickDuration:      50 * time.Millisecond,
		QueueCapacity:     256,
		ShardCount:        1,
		AssignmentPolicy:  string(AssignmentStatic),
	}
}

func ParseConfig(args []string) (LoadsimConfig, error) {
	cfg := DefaultConfig()
	fs := flag.NewFlagSet("loadsim", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var durationText string
	var tickText string
	var reconnectIntervalText string
	fs.StringVar(&cfg.Scenario, "scenario", cfg.Scenario, "load scenario")
	fs.IntVar(&cfg.Clients, "clients", cfg.Clients, "simulated client count")
	fs.StringVar(&durationText, "duration", cfg.Duration.String(), "run duration")
	fs.Float64Var(&cfg.CommandRate, "cmd-rate", cfg.CommandRate, "commands per simulated client per second")
	fs.StringVar(&cfg.ContentPath, "content", cfg.ContentPath, "content package manifest path")
	fs.StringVar(&cfg.Continent, "continent", cfg.Continent, "continent id")
	fs.StringVar(&cfg.ZoneDistribution, "zone-distribution", cfg.ZoneDistribution, "even, transition-heavy, single:<zone>, or weighted:<zone>=<weight>,...")
	fs.Float64Var(&cfg.TransitionRate, "transition-rate", cfg.TransitionRate, "transition command probability")
	fs.Float64Var(&cfg.CombatRate, "combat-rate", cfg.CombatRate, "cross-zone combat pressure probability")
	fs.Float64Var(&cfg.AbilityRate, "ability-rate", cfg.AbilityRate, "ability command probability")
	fs.Float64Var(&cfg.QuestRate, "quest-rate", cfg.QuestRate, "quest command probability")
	fs.Float64Var(&cfg.ReconnectRate, "reconnect-rate", cfg.ReconnectRate, "reconnect attempt probability")
	fs.StringVar(&reconnectIntervalText, "reconnect-interval", cfg.ReconnectInterval.String(), "minimum reconnect interval")
	fs.Int64Var(&cfg.Seed, "seed", cfg.Seed, "deterministic random seed")
	fs.StringVar(&cfg.ReportPath, "report", cfg.ReportPath, "optional JSON report output path")
	fs.StringVar(&tickText, "tick-ms", strconv.Itoa(int(cfg.TickDuration.Milliseconds())), "simulation tick duration in milliseconds")
	fs.IntVar(&cfg.QueueCapacity, "queue-capacity", cfg.QueueCapacity, "per-zone command queue capacity")
	fs.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "print verbose event output")
	fs.IntVar(&cfg.ShardCount, "shards", cfg.ShardCount, "local shard runtime count")
	fs.StringVar(&cfg.AssignmentPolicy, "assignment-policy", cfg.AssignmentPolicy, "static, least-loaded, or hash-zone")
	if err := fs.Parse(args); err != nil {
		return LoadsimConfig{}, err
	}
	duration, err := time.ParseDuration(durationText)
	if err != nil {
		return LoadsimConfig{}, fmt.Errorf("invalid duration: %w", err)
	}
	cfg.Duration = duration
	if strings.HasSuffix(tickText, "ms") || strings.ContainsAny(tickText, "shm") {
		cfg.TickDuration, err = time.ParseDuration(tickText)
	} else {
		ms, parseErr := strconv.Atoi(tickText)
		if parseErr != nil {
			err = parseErr
		} else {
			cfg.TickDuration = time.Duration(ms) * time.Millisecond
		}
	}
	if err != nil {
		return LoadsimConfig{}, fmt.Errorf("invalid tick-ms: %w", err)
	}
	cfg.ReconnectInterval, err = time.ParseDuration(reconnectIntervalText)
	if err != nil {
		return LoadsimConfig{}, fmt.Errorf("invalid reconnect interval: %w", err)
	}
	return cfg, cfg.Validate()
}

func (cfg LoadsimConfig) Validate() error {
	if cfg.Clients <= 0 {
		return fmt.Errorf("clients must be greater than zero")
	}
	if cfg.Duration <= 0 {
		return fmt.Errorf("duration must be greater than zero")
	}
	if cfg.CommandRate < 0 {
		return fmt.Errorf("cmd-rate cannot be negative")
	}
	if cfg.TickDuration <= 0 {
		return fmt.Errorf("tick-ms must be greater than zero")
	}
	if cfg.QueueCapacity <= 0 {
		return fmt.Errorf("queue-capacity must be greater than zero")
	}
	if cfg.ShardCount <= 0 {
		return fmt.Errorf("shards must be greater than zero")
	}
	if cfg.TransitionRate < 0 || cfg.CombatRate < 0 || cfg.AbilityRate < 0 || cfg.QuestRate < 0 || cfg.ReconnectRate < 0 {
		return fmt.Errorf("rates cannot be negative")
	}
	switch cfg.Scenario {
	case ScenarioMovementBasic, ScenarioCombatBasic, ScenarioAbilityBasic, ScenarioQuestBasic, ScenarioDawnwakeTraversal, ScenarioMultizonePressure, ScenarioShardAssignmentBasic, ScenarioReconnectPressure:
	default:
		return fmt.Errorf("unsupported scenario %q", cfg.Scenario)
	}
	switch AssignmentPolicy(cfg.AssignmentPolicy) {
	case AssignmentStatic, AssignmentLeastLoaded, AssignmentHashZone:
	default:
		return fmt.Errorf("unsupported assignment policy %q", cfg.AssignmentPolicy)
	}
	return nil
}
