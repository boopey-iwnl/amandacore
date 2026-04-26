package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/simcore"
	"amandacore/services/internal/worlds"
)

type SimulationOptions struct {
	Clients             int
	Duration            time.Duration
	CommandsPerSecond   float64
	ReconnectPercentage float64
	RealmID             string
	ZoneID              string
}

type SimulationReport struct {
	ClientsAttached       int     `json:"clientsAttached"`
	TotalCommandsSent     int64   `json:"totalCommandsSent"`
	TotalCommandsAccepted int64   `json:"totalCommandsAccepted"`
	TotalCommandsRejected int64   `json:"totalCommandsRejected"`
	AverageTickDurationMs float64 `json:"averageTickDurationMs"`
	MaxTickDurationMs     float64 `json:"maxTickDurationMs"`
	P95TickDurationMs     float64 `json:"p95TickDurationMs"`
	MaxCommandQueueDepth  int     `json:"maxCommandQueueDepth"`
	ReconnectAttempts     int64   `json:"reconnectAttempts"`
	ReconnectSuccesses    int64   `json:"reconnectSuccesses"`
	PersistenceFlushCount int64   `json:"persistenceFlushCount"`
	Errors                int64   `json:"errors"`
}

type memoryWriter struct {
	saved map[simcore.CharacterID]simcore.Vector3
}

func (w *memoryWriter) UpdateCharacterState(characterID string, zoneID string, x float64, y float64, z float64) (*platform.Character, error) {
	if w.saved == nil {
		w.saved = map[simcore.CharacterID]simcore.Vector3{}
	}
	w.saved[simcore.CharacterID(characterID)] = simcore.Vector3{X: x, Y: y, Z: z}
	return &platform.Character{ID: characterID, ZoneID: zoneID, PositionX: x, PositionY: y, PositionZ: z}, nil
}

func main() {
	opts := parseOptions()
	report, err := RunSimulation(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadsim failed: %v\n", err)
		os.Exit(1)
	}

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode report: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}

func parseOptions() SimulationOptions {
	var durationText string
	opts := SimulationOptions{}
	flag.IntVar(&opts.Clients, "clients", 25, "number of simulated clients")
	flag.StringVar(&durationText, "duration", "10s", "simulation duration, for example 2s, 60s, or 5m")
	flag.Float64Var(&opts.CommandsPerSecond, "cmd-rate", 5, "movement commands per second per client")
	flag.Float64Var(&opts.ReconnectPercentage, "reconnect-percent", 0, "percentage of clients to disconnect/reconnect once during the run")
	flag.StringVar(&opts.RealmID, "realm", "loadsim-realm", "simulated realm id")
	flag.StringVar(&opts.ZoneID, "zone", "loadsim-zone", "simulated zone id")
	flag.Parse()

	var err error
	opts.Duration, err = time.ParseDuration(durationText)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid duration: %v\n", err)
		os.Exit(2)
	}
	return opts
}

func RunSimulation(opts SimulationOptions) (SimulationReport, error) {
	if opts.Clients <= 0 {
		return SimulationReport{}, fmt.Errorf("clients must be greater than zero")
	}
	if opts.Duration <= 0 {
		return SimulationReport{}, fmt.Errorf("duration must be greater than zero")
	}
	if opts.CommandsPerSecond <= 0 {
		return SimulationReport{}, fmt.Errorf("cmd-rate must be greater than zero")
	}
	if opts.RealmID == "" {
		opts.RealmID = "loadsim-realm"
	}
	if opts.ZoneID == "" {
		opts.ZoneID = "loadsim-zone"
	}

	observability.LogEvent("loadsim", observability.EventLoadsimStarted, map[string]any{
		"clients":           opts.Clients,
		"durationSeconds":   opts.Duration.Seconds(),
		"commandsPerSecond": opts.CommandsPerSecond,
	})

	writer := &memoryWriter{}
	gateway := worlds.NewSessionGateway()
	persistence := worlds.NewPersistenceHandoff(writer)
	runtime := worlds.NewWorldRuntime(worlds.WorldRuntimeConfig{
		CommandQueueLimit: opts.Clients * 8,
		MovementRules: worlds.MovementRules{
			MaxStepDistance: 12,
			Bounds:          worlds.RuntimeBounds{MinX: 0, MinY: 0, MaxX: 1000, MaxY: 1000},
			ServerZ:         0.05,
			ControlZ:        true,
		},
	}, gateway, persistence)

	if err := runtime.RegisterZone(mustZone(opts.ZoneID)); err != nil {
		return SimulationReport{}, err
	}

	now := time.Now().UTC()
	for client := 0; client < opts.Clients; client++ {
		characterID := simcore.CharacterID(fmt.Sprintf("loadsim_char_%05d", client))
		sessionID := simcore.SessionID(fmt.Sprintf("loadsim_session_%05d", client))
		position := simcore.Vector3{X: float64(client % 50), Y: float64(client / 50), Z: 0.05}
		if _, err := gateway.Attach(worlds.AttachSessionRequest{
			SessionID:             sessionID,
			AccountID:             simcore.AccountID(fmt.Sprintf("loadsim_account_%05d", client)),
			CharacterID:           characterID,
			RealmID:               simcore.RealmID(opts.RealmID),
			ZoneID:                simcore.ZoneID(opts.ZoneID),
			AuthoritativePosition: position,
			Now:                   now,
		}); err != nil {
			return SimulationReport{}, err
		}
		if err := runtime.RegisterOrUpdateEntity(worlds.RuntimeEntity{
			ID:       simcore.EntityID(characterID),
			Kind:     "player",
			ZoneID:   simcore.ZoneID(opts.ZoneID),
			Position: position,
		}); err != nil {
			return SimulationReport{}, err
		}
	}

	report := SimulationReport{ClientsAttached: opts.Clients}
	tickInterval := 50 * time.Millisecond
	totalTicks := int(math.Ceil(float64(opts.Duration) / float64(tickInterval)))
	if totalTicks <= 0 {
		totalTicks = 1
	}
	commandEveryTicks := int(math.Round(float64(time.Second) / float64(tickInterval) / opts.CommandsPerSecond))
	if commandEveryTicks <= 0 {
		commandEveryTicks = 1
	}
	reconnectCount := int(math.Round(float64(opts.Clients) * (opts.ReconnectPercentage / 100.0)))
	reconnectDone := false
	tickDurations := make([]float64, 0, totalTicks)

	for tick := 0; tick < totalTicks; tick++ {
		tickNow := now.Add(time.Duration(tick) * tickInterval)
		if tick%commandEveryTicks == 0 {
			for client := 0; client < opts.Clients; client++ {
				characterID := simcore.CharacterID(fmt.Sprintf("loadsim_char_%05d", client))
				sessionID := simcore.SessionID(fmt.Sprintf("loadsim_session_%05d", client))
				_, err := runtime.Enqueue(simcore.CommandEnvelope{
					CommandID:         simcore.CommandID(fmt.Sprintf("cmd_%05d_%05d", tick, client)),
					SessionID:         sessionID,
					AccountID:         simcore.AccountID(fmt.Sprintf("loadsim_account_%05d", client)),
					CharacterID:       characterID,
					RealmID:           simcore.RealmID(opts.RealmID),
					ZoneID:            simcore.ZoneID(opts.ZoneID),
					ClientSequence:    uint64(tick),
					ServerReceiveTime: tickNow,
					IntendedTick:      simcore.TickID(tick + 1),
					Payload: simcore.MoveIntentCommand{
						CharacterID: characterID,
						Delta:       simcore.Vector3{X: 1, Y: float64((client % 3) - 1)},
					},
				})
				report.TotalCommandsSent++
				if err != nil {
					report.TotalCommandsRejected++
					report.Errors++
				}
			}
		}

		if !reconnectDone && reconnectCount > 0 && tick >= totalTicks/2 {
			for client := 0; client < reconnectCount; client++ {
				sessionID := simcore.SessionID(fmt.Sprintf("loadsim_session_%05d", client))
				report.ReconnectAttempts++
				session, ok := gateway.Session(sessionID)
				if !ok {
					report.Errors++
					continue
				}
				if _, err := gateway.Disconnect(sessionID, "loadsim_reconnect", tickNow); err != nil {
					report.Errors++
					continue
				}
				if _, err := gateway.CompleteReconnect(sessionID, session.AuthoritativePosition, tickNow); err != nil {
					report.Errors++
					continue
				}
				report.ReconnectSuccesses++
			}
			reconnectDone = true
		}

		result := runtime.RunTick(tickNow)
		report.TotalCommandsAccepted += int64(result.CommandsProcessed - result.CommandsRejected)
		report.TotalCommandsRejected += int64(result.CommandsRejected)
		if result.QueueDepthBeforeTick > report.MaxCommandQueueDepth {
			report.MaxCommandQueueDepth = result.QueueDepthBeforeTick
		}
		tickDurationMs := float64(result.Duration.Microseconds()) / 1000.0
		tickDurations = append(tickDurations, tickDurationMs)
		if tickDurationMs > report.MaxTickDurationMs {
			report.MaxTickDurationMs = tickDurationMs
		}

		flushResults := persistence.FlushDirty(nilContext{})
		report.PersistenceFlushCount += int64(len(flushResults))
		for _, flush := range flushResults {
			if flush.Error != nil {
				report.Errors++
			}
		}
	}

	report.AverageTickDurationMs = average(tickDurations)
	report.P95TickDurationMs = percentile(tickDurations, 0.95)

	observability.LogEvent("loadsim", observability.EventLoadsimCompleted, map[string]any{
		"clientsAttached":       report.ClientsAttached,
		"totalCommandsSent":     report.TotalCommandsSent,
		"totalCommandsAccepted": report.TotalCommandsAccepted,
		"totalCommandsRejected": report.TotalCommandsRejected,
		"errors":                report.Errors,
	})
	return report, nil
}

func mustZone(zoneID string) *worlds.ZoneRuntime {
	zone, err := worlds.NewZoneRuntime(worlds.ZoneDefinition{
		ID:          simcore.ZoneID(zoneID),
		DisplayName: zoneID,
		Bounds: worlds.ZoneBounds{
			Min: simcore.Vector3{X: 0, Y: 0, Z: 0},
			Max: simcore.Vector3{X: 1000, Y: 1000, Z: 0.05},
		},
	})
	if err != nil {
		panic(err)
	}
	return zone
}

type nilContext struct{}

func (nilContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (nilContext) Done() <-chan struct{}       { return nil }
func (nilContext) Err() error                  { return nil }
func (nilContext) Value(key any) any           { return nil }

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func percentile(values []float64, rank float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64{}, values...)
	sort.Float64s(sorted)
	index := int(math.Ceil(rank*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
