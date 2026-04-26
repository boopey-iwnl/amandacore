package loadsim

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"amandacore/services/internal/observability"
)

const (
	EventLoadsimRunStarted              = "loadsim.run_started"
	EventLoadsimRunCompleted            = "loadsim.run_completed"
	EventLoadsimRunFailed               = "loadsim.run_failed"
	EventLoadsimScenarioStarted         = "loadsim.scenario_started"
	EventLoadsimScenarioCompleted       = "loadsim.scenario_completed"
	EventLoadsimReportWritten           = "loadsim.report_written"
	EventLoadsimZoneDistributionApplied = "loadsim.zone_distribution_applied"
	EventLoadsimClientSpawned           = "loadsim.client_spawned"
	EventLoadsimCommandSent             = "loadsim.command_sent"
	EventLoadsimCommandRejected         = "loadsim.command_rejected"
	EventLoadsimTransitionRequested     = "loadsim.transition_requested"
	EventLoadsimTransitionCompleted     = "loadsim.transition_completed"
	EventLoadsimTransitionRejected      = "loadsim.transition_rejected"
	EventLoadsimReconnectAttempted      = "loadsim.reconnect_attempted"
	EventLoadsimReconnectCompleted      = "loadsim.reconnect_completed"
)

type SimClient struct {
	CharacterID       string `json:"characterId"`
	CurrentZoneID     string `json:"currentZoneId"`
	Connected         bool   `json:"connected"`
	lastReconnectTick int
}

type LoadsimReport struct {
	RunID                string                     `json:"runId"`
	Scenario             string                     `json:"scenario"`
	Seed                 int64                      `json:"seed"`
	ContentPackage       string                     `json:"contentPackage"`
	Duration             string                     `json:"duration"`
	ClientCount          int                        `json:"clientCount"`
	CommandRate          float64                    `json:"commandRate"`
	ZonesActivated       []string                   `json:"zonesActivated"`
	ShardCount           int                        `json:"shardCount"`
	ClientsPerZone       map[string]int             `json:"clientsPerZone"`
	ZonesPerShard        map[string][]string        `json:"zonesPerShard"`
	TotalCommandsSent    int                        `json:"totalCommandsSent"`
	AcceptedCommands     int                        `json:"acceptedCommands"`
	RejectedCommands     int                        `json:"rejectedCommands"`
	RejectionReasons     map[string]int             `json:"rejectionReasons"`
	TransitionsRequested int                        `json:"transitionsRequested"`
	TransitionsCompleted int                        `json:"transitionsCompleted"`
	TransitionsRejected  int                        `json:"transitionsRejected"`
	TransitionRoutes     map[string]int             `json:"transitionRoutes"`
	ReconnectAttempts    int                        `json:"reconnectAttempts"`
	ReconnectSuccesses   int                        `json:"reconnectSuccesses"`
	ReconnectFailures    int                        `json:"reconnectFailures"`
	TickMetrics          TickMetrics                `json:"tickMetrics"`
	QueueMetrics         QueueMetrics               `json:"queueMetrics"`
	ZoneReports          map[string]ZoneLoadReport  `json:"perZone"`
	ShardReports         map[string]ShardLoadReport `json:"perShard"`
	Errors               []string                   `json:"errors"`
}

type ZoneLoadReport struct {
	ActiveClients int            `json:"activeClients"`
	Commands      CommandMetrics `json:"commands"`
	Queue         QueueMetrics   `json:"queue"`
	Tick          TickMetrics    `json:"tick"`
}

type ShardLoadReport struct {
	Zones        []string          `json:"zones"`
	LoadSnapshot ShardLoadSnapshot `json:"loadSnapshot"`
}

func Run(ctx context.Context, cfg LoadsimConfig) (LoadsimReport, error) {
	if err := cfg.Validate(); err != nil {
		return LoadsimReport{}, err
	}
	runID := time.Now().UTC().Format("20060102-150405")
	observability.LogEvent("loadsim", EventLoadsimRunStarted, map[string]any{"runId": runID, "scenario": cfg.Scenario, "seed": cfg.Seed})
	observability.LogEvent("loadsim", EventLoadsimScenarioStarted, map[string]any{"runId": runID, "scenario": cfg.Scenario})

	content, err := LoadContentPackage(cfg.ContentPath)
	if err != nil {
		observability.LogEvent("loadsim", EventLoadsimRunFailed, map[string]any{"runId": runID, "error": err.Error()})
		return LoadsimReport{}, err
	}
	rng := rand.New(rand.NewSource(cfg.Seed))
	distribution, err := ParseZoneDistribution(cfg.ZoneDistribution, content.ZoneOrder)
	if err != nil {
		return LoadsimReport{}, err
	}
	assignments, err := AssignClientZones(distribution, content.ZoneOrder, cfg.Clients, rng)
	if err != nil {
		return LoadsimReport{}, err
	}
	observability.LogEvent("loadsim", EventLoadsimZoneDistributionApplied, map[string]any{"runId": runID, "mode": distribution.Mode})

	router, err := BuildShardRouter(content, cfg.ShardCount, AssignmentPolicy(cfg.AssignmentPolicy), cfg.QueueCapacity)
	if err != nil {
		return LoadsimReport{}, err
	}

	clients := make([]*SimClient, 0, cfg.Clients)
	report := LoadsimReport{
		RunID:            runID,
		Scenario:         cfg.Scenario,
		Seed:             cfg.Seed,
		ContentPackage:   content.PackageID,
		Duration:         cfg.Duration.String(),
		ClientCount:      cfg.Clients,
		CommandRate:      cfg.CommandRate,
		ZonesActivated:   append([]string(nil), content.ZoneOrder...),
		ShardCount:       cfg.ShardCount,
		ClientsPerZone:   map[string]int{},
		ZonesPerShard:    zonesPerShard(router),
		RejectionReasons: map[string]int{},
		TransitionRoutes: map[string]int{},
		ZoneReports:      map[string]ZoneLoadReport{},
		ShardReports:     map[string]ShardLoadReport{},
	}
	for index, zoneID := range assignments {
		client := &SimClient{
			CharacterID:   fmt.Sprintf("client_%05d", index+1),
			CurrentZoneID: zoneID,
			Connected:     true,
		}
		entry := content.Zones[zoneID].DefaultEntryPoint()
		if err := router.RegisterCharacter(client.CharacterID, zoneID, entry.Position); err != nil {
			return LoadsimReport{}, err
		}
		report.ClientsPerZone[zoneID]++
		clients = append(clients, client)
		if cfg.Verbose {
			observability.LogEvent("loadsim", EventLoadsimClientSpawned, map[string]any{"runId": runID, "characterId": client.CharacterID, "zoneId": zoneID})
		}
	}

	tickCount := int(cfg.Duration / cfg.TickDuration)
	if tickCount <= 0 {
		tickCount = 1
	}
	commandAccumulator := make([]float64, len(clients))
	reconnectEvery := int(cfg.ReconnectInterval / cfg.TickDuration)
	if reconnectEvery <= 0 {
		reconnectEvery = 1
	}

	for tick := 0; tick < tickCount; tick++ {
		select {
		case <-ctx.Done():
			report.Errors = append(report.Errors, ctx.Err().Error())
			return finalizeReport(report, router), ctx.Err()
		default:
		}
		for index, client := range clients {
			if !client.Connected {
				continue
			}
			if cfg.Scenario == ScenarioReconnectPressure && cfg.ReconnectRate > 0 && tick-client.lastReconnectTick >= reconnectEvery && rng.Float64() < cfg.ReconnectRate {
				report.ReconnectAttempts++
				client.lastReconnectTick = tick
				if cfg.Verbose {
					observability.LogEvent("loadsim", EventLoadsimReconnectAttempted, map[string]any{"runId": runID, "characterId": client.CharacterID})
				}
				if _, ok := router.CharacterZone[client.CharacterID]; ok {
					report.ReconnectSuccesses++
					if cfg.Verbose {
						observability.LogEvent("loadsim", EventLoadsimReconnectCompleted, map[string]any{"runId": runID, "characterId": client.CharacterID, "zoneId": client.CurrentZoneID})
					}
				} else {
					report.ReconnectFailures++
				}
			}
			commandAccumulator[index] += cfg.CommandRate * cfg.TickDuration.Seconds()
			for commandAccumulator[index] >= 1.0 {
				commandAccumulator[index]--
				command := nextCommand(client, content, cfg, rng, tick)
				report.TotalCommandsSent++
				if cfg.Verbose {
					observability.LogEvent("loadsim", EventLoadsimCommandSent, map[string]any{"runId": runID, "characterId": client.CharacterID, "type": command.Type, "zoneId": client.CurrentZoneID})
				}
				result := router.Submit(command)
				recordSubmitResult(&report, result)
			}
		}
		for _, result := range router.TickAll() {
			recordTickResult(&report, router, clients, result)
		}
	}

	report = finalizeReport(report, router)
	observability.LogEvent("loadsim", EventLoadsimScenarioCompleted, map[string]any{"runId": runID, "scenario": cfg.Scenario})
	observability.LogEvent("loadsim", EventLoadsimRunCompleted, map[string]any{"runId": runID, "scenario": cfg.Scenario})
	return report, nil
}

func nextCommand(client *SimClient, content ContentPackage, cfg LoadsimConfig, rng *rand.Rand, tick int) CommandEnvelope {
	command := CommandEnvelope{
		CommandID:   fmt.Sprintf("%s:%08d", client.CharacterID, tick),
		CharacterID: client.CharacterID,
		ZoneID:      client.CurrentZoneID,
		Type:        "move",
	}
	zone := content.Zones[client.CurrentZoneID]
	transitionChance := cfg.TransitionRate
	if cfg.Scenario == ScenarioDawnwakeTraversal || cfg.Scenario == ScenarioMultizonePressure || cfg.ZoneDistribution == "transition-heavy" {
		transitionChance = maxFloat64(transitionChance, 0.25)
	}
	if len(zone.TransitionGates) > 0 && rng.Float64() < transitionChance {
		gate := zone.TransitionGates[rng.Intn(len(zone.TransitionGates))]
		command.Type = "transition"
		command.TargetZoneID = gate.ToZoneID
		return command
	}
	if cfg.CombatRate > 0 && rng.Float64() < cfg.CombatRate {
		command.Type = "combat"
		command.TargetID = "pressure_target"
		if len(content.ZoneOrder) > 1 && rng.Float64() < 0.25 {
			command.TargetZoneID = differentZone(content.ZoneOrder, client.CurrentZoneID, rng)
		} else {
			command.TargetZoneID = client.CurrentZoneID
		}
	}
	if cfg.Scenario == ScenarioCombatBasic {
		command.Type = "combat"
		command.TargetZoneID = client.CurrentZoneID
	}
	return command
}

func recordSubmitResult(report *LoadsimReport, result CommandResult) {
	if result.Rejected {
		report.RejectedCommands++
		report.RejectionReasons[string(result.Reason)]++
		observability.LogEvent("loadsim", EventLoadsimCommandRejected, map[string]any{"runId": report.RunID, "reason": string(result.Reason), "zoneId": result.ZoneID})
		return
	}
	if result.Accepted {
		report.AcceptedCommands++
	}
}

func recordTickResult(report *LoadsimReport, router *ShardRouter, clients []*SimClient, result CommandResult) {
	if result.Rejected {
		report.RejectedCommands++
		report.RejectionReasons[string(result.Reason)]++
		if result.TransitionRejected {
			report.TransitionsRejected++
			observability.LogEvent("loadsim", EventLoadsimTransitionRejected, map[string]any{"runId": report.RunID, "zoneId": result.ZoneID, "reason": string(result.Reason)})
		}
		return
	}
	if result.TransitionRequested {
		report.TransitionsRequested++
	}
	if result.TransitionCompleted {
		report.TransitionsCompleted++
		route := result.ZoneID + "->" + result.ToZoneID
		report.TransitionRoutes[route]++
		characterID := result.CommandIDCharacter()
		for _, client := range clients {
			if client.CharacterID == characterID {
				report.ClientsPerZone[client.CurrentZoneID]--
				client.CurrentZoneID = router.CharacterZone[characterID]
				report.ClientsPerZone[client.CurrentZoneID]++
				break
			}
		}
	}
}

func finalizeReport(report LoadsimReport, router *ShardRouter) LoadsimReport {
	var queue QueueMetrics
	var ticks []time.Duration
	for _, shardID := range router.Registry.Order {
		shard := router.Registry.Shards[shardID]
		zoneIDs := sortedRuntimeZoneIDs(shard.Zones)
		snapshot := routerSnapshotForShard(router, shardID)
		report.ShardReports[string(shardID)] = ShardLoadReport{Zones: zoneIDs, LoadSnapshot: snapshot}
		for _, zoneID := range zoneIDs {
			zone := shard.Zones[zoneID]
			zoneTick := SummarizeTickDurations(zone.tickDurations)
			report.ZoneReports[zoneID] = ZoneLoadReport{
				ActiveClients: len(zone.Sessions),
				Queue:         zone.QueueMetrics,
				Tick:          zoneTick,
				Commands: CommandMetrics{
					Sent:             0,
					Accepted:         0,
					Rejected:         0,
					RejectionReasons: map[string]int{},
				},
			}
			if zone.QueueMetrics.MaxDepth > queue.MaxDepth {
				queue.MaxDepth = zone.QueueMetrics.MaxDepth
			}
			if zone.QueueMetrics.Samples > 0 {
				queue.AverageDepth = ((queue.AverageDepth * float64(queue.Samples)) + (zone.QueueMetrics.AverageDepth * float64(zone.QueueMetrics.Samples))) / float64(queue.Samples+zone.QueueMetrics.Samples)
				queue.Samples += zone.QueueMetrics.Samples
			}
			ticks = append(ticks, zone.tickDurations...)
		}
	}
	report.QueueMetrics = queue
	report.TickMetrics = SummarizeTickDurations(ticks)
	return report
}

func routerSnapshotForShard(router *ShardRouter, shardID ShardID) ShardLoadSnapshot {
	for _, snapshot := range router.LoadSnapshots() {
		if snapshot.ShardID == shardID {
			return snapshot
		}
	}
	return ShardLoadSnapshot{ShardID: shardID}
}

func WriteJSONReport(path string, report LoadsimReport) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return err
	}
	observability.LogEvent("loadsim", EventLoadsimReportWritten, map[string]any{"runId": report.RunID, "path": path})
	return nil
}

func RenderTextReport(report LoadsimReport) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "AmandaCore loadsim report\n")
	fmt.Fprintf(&builder, "run_id=%s scenario=%s seed=%d content=%s\n", report.RunID, report.Scenario, report.Seed, report.ContentPackage)
	fmt.Fprintf(&builder, "clients=%d duration=%s cmd_rate=%.2f shards=%d zones=%s\n", report.ClientCount, report.Duration, report.CommandRate, report.ShardCount, strings.Join(report.ZonesActivated, ","))
	fmt.Fprintf(&builder, "commands sent=%d accepted=%d rejected=%d\n", report.TotalCommandsSent, report.AcceptedCommands, report.RejectedCommands)
	fmt.Fprintf(&builder, "transitions requested=%d completed=%d rejected=%d reconnects=%d/%d\n", report.TransitionsRequested, report.TransitionsCompleted, report.TransitionsRejected, report.ReconnectSuccesses, report.ReconnectAttempts)
	fmt.Fprintf(&builder, "ticks count=%d avg=%s max=%s p50=%s p95=%s p99=%s\n", report.TickMetrics.Count, report.TickMetrics.AverageDuration, report.TickMetrics.MaxDuration, report.TickMetrics.P50, report.TickMetrics.P95, report.TickMetrics.P99)
	fmt.Fprintf(&builder, "queue max=%d avg=%.2f\n", report.QueueMetrics.MaxDepth, report.QueueMetrics.AverageDepth)
	if len(report.RejectionReasons) > 0 {
		fmt.Fprintf(&builder, "rejections=%v\n", report.RejectionReasons)
	}
	fmt.Fprintf(&builder, "clients_per_zone=%v\n", report.ClientsPerZone)
	fmt.Fprintf(&builder, "zones_per_shard=%v\n", report.ZonesPerShard)
	return builder.String()
}

func zonesPerShard(router *ShardRouter) map[string][]string {
	result := map[string][]string{}
	for _, shardID := range router.Registry.Order {
		result[string(shardID)] = sortedRuntimeZoneIDs(router.Registry.Shards[shardID].Zones)
	}
	return result
}

func differentZone(zoneOrder []string, current string, rng *rand.Rand) string {
	candidates := []string{}
	for _, zoneID := range zoneOrder {
		if zoneID != current {
			candidates = append(candidates, zoneID)
		}
	}
	if len(candidates) == 0 {
		return current
	}
	return candidates[rng.Intn(len(candidates))]
}

func maxFloat64(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func SortedReportZoneIDs(report LoadsimReport) []string {
	ids := make([]string, 0, len(report.ZoneReports))
	for id := range report.ZoneReports {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
