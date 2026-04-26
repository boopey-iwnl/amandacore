package worlds

import (
	"fmt"
	"sort"
	"time"

	"amandacore/services/internal/observability"
)

type ShardCoordinatorConfig struct {
	ShardCount      int
	QueueDepthLimit int
}

type ShardRuntimeHandle struct {
	ShardID string
	ZoneID  string
	Runtime *ZoneRuntime
}

type ShardCommandResult struct {
	CharacterID   string
	CommandID     string
	ShardID       string
	ZoneID        string
	Accepted      bool
	Backpressured bool
	QueueDepth    int
	MaxQueueDepth int
}

type ShardRuntime struct {
	ShardID               string
	Zones                 map[string]*ZoneRuntime
	QueueDepth            int
	MaxQueueDepth         int
	CommandsAccepted      int
	CommandsProcessed     int
	CommandsBackpressured int
	TickDurations         []time.Duration
}

type ShardRuntimeMetrics struct {
	ShardID               string
	ZoneIDs               []string
	QueueDepth            int
	MaxQueueDepth         int
	CommandsAccepted      int
	CommandsProcessed     int
	CommandsBackpressured int
	Tick                  TickDurationSummary
}

type TickDurationSummary struct {
	Count   int
	Average time.Duration
	P50     time.Duration
	P95     time.Duration
	P99     time.Duration
	Max     time.Duration
}

type ShardCoordinatorSnapshot struct {
	Assignments       []ShardAssignment
	ShardMetrics      []ShardRuntimeMetrics
	ZonePopulation    map[string]int
	ShardPopulation   map[string]int
	RouteFailures     int
	CommandsAccepted  int
	CommandsProcessed int
	BackpressureCount int
	MaxQueueDepth     int
	Tick              TickDurationSummary
}

type InProcessShardCoordinator struct {
	Continent       *ContinentRuntime
	Assignments     map[string]ShardAssignment
	Shards          map[string]*ShardRuntime
	QueueDepthLimit int
	RouteFailures   int
}

func NewInProcessShardCoordinator(continent *ContinentRuntime, config ShardCoordinatorConfig) *InProcessShardCoordinator {
	shardCount := config.ShardCount
	if shardCount <= 0 {
		shardCount = len(continent.Zones)
	}
	if shardCount <= 0 {
		shardCount = 1
	}
	queueLimit := config.QueueDepthLimit
	if queueLimit <= 0 {
		queueLimit = 1024
	}
	coordinator := &InProcessShardCoordinator{
		Continent:       continent,
		Assignments:     map[string]ShardAssignment{},
		Shards:          map[string]*ShardRuntime{},
		QueueDepthLimit: queueLimit,
	}
	for index := 0; index < shardCount; index++ {
		shardID := fmt.Sprintf("shard-%02d", index+1)
		coordinator.Shards[shardID] = &ShardRuntime{
			ShardID: shardID,
			Zones:   map[string]*ZoneRuntime{},
		}
	}
	zoneIDs := append([]string(nil), continent.Definition.Zones...)
	if len(zoneIDs) == 0 {
		for zoneID := range continent.Zones {
			zoneIDs = append(zoneIDs, zoneID)
		}
		sort.Strings(zoneIDs)
	}
	for index, zoneID := range zoneIDs {
		shardID := fmt.Sprintf("shard-%02d", index%shardCount+1)
		assignment := ShardAssignment{ShardID: shardID, ZoneID: zoneID}
		coordinator.Assignments[zoneID] = assignment
		if runtime := continent.Zones[zoneID]; runtime != nil {
			runtime.ShardID = ShardID(shardID)
			coordinator.Shards[shardID].Zones[zoneID] = runtime
		}
		observability.LogEvent("world-service", observability.EventWorldShardAssigned, map[string]any{
			"continentId": continent.Definition.ContinentID,
			"zoneId":      zoneID,
			"shardId":     shardID,
		})
		observability.LogEvent("world-service", observability.EventWorldShardZoneBound, map[string]any{
			"continentId": continent.Definition.ContinentID,
			"zoneId":      zoneID,
			"shardId":     shardID,
		})
	}
	continent.Shards = &ShardRuntimeIndex{ZoneToShard: map[string]string{}, ShardToZones: map[string][]string{}}
	for zoneID, assignment := range coordinator.Assignments {
		continent.Shards.ZoneToShard[zoneID] = assignment.ShardID
		continent.Shards.ShardToZones[assignment.ShardID] = append(continent.Shards.ShardToZones[assignment.ShardID], zoneID)
	}
	return coordinator
}

func (c *InProcessShardCoordinator) RouteCommand(command WorldCommand) (ShardRuntimeHandle, error) {
	if c == nil || c.Continent == nil {
		return ShardRuntimeHandle{}, fmt.Errorf("shard coordinator is not initialized")
	}
	zoneHandle, err := c.Continent.RouteCommand(command)
	if err != nil {
		c.RouteFailures++
		return ShardRuntimeHandle{}, err
	}
	assignment, found := c.Assignments[zoneHandle.ZoneID]
	if !found {
		c.RouteFailures++
		return ShardRuntimeHandle{}, fmt.Errorf("zone %s is not assigned to a shard", zoneHandle.ZoneID)
	}
	return ShardRuntimeHandle{ShardID: assignment.ShardID, ZoneID: zoneHandle.ZoneID, Runtime: zoneHandle.Runtime}, nil
}

func (c *InProcessShardCoordinator) TryEnqueueCommand(command WorldCommand) (ShardCommandResult, error) {
	handle, err := c.RouteCommand(command)
	if err != nil {
		return ShardCommandResult{CharacterID: command.CharacterID, CommandID: command.CommandID}, err
	}
	shard := c.Shards[handle.ShardID]
	result := ShardCommandResult{
		CharacterID: command.CharacterID,
		CommandID:   command.CommandID,
		ShardID:     handle.ShardID,
		ZoneID:      handle.ZoneID,
		QueueDepth:  shard.QueueDepth,
	}
	if shard.QueueDepth >= c.QueueDepthLimit {
		shard.CommandsBackpressured++
		result.Backpressured = true
		result.MaxQueueDepth = shard.MaxQueueDepth
		observability.LogEvent("world-service", observability.EventWorldZoneCommandBackpressured, map[string]any{
			"characterId": command.CharacterID,
			"commandId":   command.CommandID,
			"commandName": command.Name,
			"zoneId":      handle.ZoneID,
			"shardId":     handle.ShardID,
			"queueDepth":  shard.QueueDepth,
			"queueCap":    c.QueueDepthLimit,
		})
		return result, nil
	}
	shard.QueueDepth++
	shard.CommandsAccepted++
	if shard.QueueDepth > shard.MaxQueueDepth {
		shard.MaxQueueDepth = shard.QueueDepth
	}
	result.Accepted = true
	result.QueueDepth = shard.QueueDepth
	result.MaxQueueDepth = shard.MaxQueueDepth
	observability.LogEvent("world-service", observability.EventWorldZoneCommandEnqueued, map[string]any{
		"characterId": command.CharacterID,
		"commandId":   command.CommandID,
		"commandName": command.Name,
		"zoneId":      handle.ZoneID,
		"shardId":     handle.ShardID,
		"queueDepth":  shard.QueueDepth,
		"queueCap":    c.QueueDepthLimit,
	})
	observability.LogEvent("world-service", observability.EventWorldZoneQueueDepthSampled, map[string]any{
		"zoneId":       handle.ZoneID,
		"shardId":      handle.ShardID,
		"queueDepth":   shard.QueueDepth,
		"queueMax":     shard.MaxQueueDepth,
		"queueCap":     c.QueueDepthLimit,
		"sampleReason": "enqueue",
	})
	return result, nil
}

func (c *InProcessShardCoordinator) CompleteCommand(result ShardCommandResult, tick SimulationTick) error {
	if !result.Accepted {
		return nil
	}
	shard, found := c.Shards[result.ShardID]
	if !found {
		return fmt.Errorf("shard %s is not active", result.ShardID)
	}
	if shard.QueueDepth > 0 {
		shard.QueueDepth--
	}
	shard.CommandsProcessed++
	if tick.Duration > 0 {
		shard.TickDurations = append(shard.TickDurations, tick.Duration)
	}
	if tick.QueueDepth > shard.MaxQueueDepth {
		shard.MaxQueueDepth = tick.QueueDepth
	}
	observability.LogEvent("world-service", observability.EventWorldZoneCommandDequeued, map[string]any{
		"characterId": result.CharacterID,
		"commandId":   result.CommandID,
		"zoneId":      result.ZoneID,
		"shardId":     result.ShardID,
		"queueDepth":  shard.QueueDepth,
	})
	observability.LogEvent("world-service", observability.EventWorldZoneQueueDepthSampled, map[string]any{
		"zoneId":       result.ZoneID,
		"shardId":      result.ShardID,
		"queueDepth":   shard.QueueDepth,
		"queueMax":     shard.MaxQueueDepth,
		"queueCap":     c.QueueDepthLimit,
		"sampleReason": "dequeue",
	})
	return nil
}

func (c *InProcessShardCoordinator) Snapshot() ShardCoordinatorSnapshot {
	snapshot := ShardCoordinatorSnapshot{
		Assignments:     c.sortedAssignments(),
		ZonePopulation:  c.ZonePopulation(),
		ShardPopulation: c.ShardPopulation(),
	}
	allTickDurations := []time.Duration{}
	for _, shardID := range c.sortedShardIDs() {
		shard := c.Shards[shardID]
		metrics := ShardRuntimeMetrics{
			ShardID:               shardID,
			ZoneIDs:               sortedZoneIDs(shard.Zones),
			QueueDepth:            shard.QueueDepth,
			MaxQueueDepth:         shard.MaxQueueDepth,
			CommandsAccepted:      shard.CommandsAccepted,
			CommandsProcessed:     shard.CommandsProcessed,
			CommandsBackpressured: shard.CommandsBackpressured,
			Tick:                  SummarizeTickDurations(shard.TickDurations),
		}
		snapshot.ShardMetrics = append(snapshot.ShardMetrics, metrics)
		snapshot.CommandsAccepted += shard.CommandsAccepted
		snapshot.CommandsProcessed += shard.CommandsProcessed
		snapshot.BackpressureCount += shard.CommandsBackpressured
		if shard.MaxQueueDepth > snapshot.MaxQueueDepth {
			snapshot.MaxQueueDepth = shard.MaxQueueDepth
		}
		allTickDurations = append(allTickDurations, shard.TickDurations...)
	}
	snapshot.Tick = SummarizeTickDurations(allTickDurations)
	snapshot.RouteFailures = c.RouteFailures
	return snapshot
}

func (c *InProcessShardCoordinator) ZonePopulation() map[string]int {
	population := map[string]int{}
	if c == nil || c.Continent == nil {
		return population
	}
	for _, zoneID := range c.Continent.Definition.Zones {
		population[zoneID] = 0
	}
	for _, state := range c.Continent.Characters {
		population[state.ZoneID]++
	}
	return population
}

func (c *InProcessShardCoordinator) ShardPopulation() map[string]int {
	population := map[string]int{}
	if c == nil || c.Continent == nil {
		return population
	}
	for shardID := range c.Shards {
		population[shardID] = 0
	}
	for _, state := range c.Continent.Characters {
		if assignment, found := c.Assignments[state.ZoneID]; found {
			population[assignment.ShardID]++
		}
	}
	return population
}

func (c *InProcessShardCoordinator) ValidateSingleZoneOwnership() error {
	if c == nil || c.Continent == nil {
		return fmt.Errorf("shard coordinator is not initialized")
	}
	for characterID, state := range c.Continent.Characters {
		ownerCount := 0
		entityCount := 0
		for zoneID, zoneRuntime := range c.Continent.Zones {
			if _, found := zoneRuntime.Characters[characterID]; found {
				ownerCount++
				if zoneID != state.ZoneID {
					return fmt.Errorf("character %s is owned by %s but indexed in %s", characterID, zoneID, state.ZoneID)
				}
			}
			if entity, found := zoneRuntime.Entities.Entities[playerEntityID(characterID)]; found {
				entityCount++
				if entity.ZoneID != state.ZoneID || zoneID != state.ZoneID {
					return fmt.Errorf("character %s has entity ownership mismatch in zone %s", characterID, zoneID)
				}
			}
		}
		if ownerCount != 1 {
			return fmt.Errorf("character %s has %d zone owners", characterID, ownerCount)
		}
		if entityCount != 1 {
			return fmt.Errorf("character %s has %d active player entities", characterID, entityCount)
		}
		if c.Continent.EntityZoneIndex[playerEntityID(characterID)] != state.ZoneID {
			return fmt.Errorf("character %s has stale entity zone index", characterID)
		}
		if _, found := c.Assignments[state.ZoneID]; !found {
			return fmt.Errorf("character %s is in unassigned zone %s", characterID, state.ZoneID)
		}
	}
	return nil
}

func SummarizeTickDurations(durations []time.Duration) TickDurationSummary {
	if len(durations) == 0 {
		return TickDurationSummary{}
	}
	sorted := append([]time.Duration(nil), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	total := time.Duration(0)
	for _, duration := range sorted {
		total += duration
	}
	return TickDurationSummary{
		Count:   len(sorted),
		Average: total / time.Duration(len(sorted)),
		P50:     percentileDuration(sorted, 50),
		P95:     percentileDuration(sorted, 95),
		P99:     percentileDuration(sorted, 99),
		Max:     sorted[len(sorted)-1],
	}
}

func (c *InProcessShardCoordinator) sortedAssignments() []ShardAssignment {
	assignments := make([]ShardAssignment, 0, len(c.Assignments))
	for _, assignment := range c.Assignments {
		assignments = append(assignments, assignment)
	}
	sort.Slice(assignments, func(i, j int) bool {
		return assignments[i].ZoneID < assignments[j].ZoneID
	})
	return assignments
}

func (c *InProcessShardCoordinator) sortedShardIDs() []string {
	shardIDs := make([]string, 0, len(c.Shards))
	for shardID := range c.Shards {
		shardIDs = append(shardIDs, shardID)
	}
	sort.Slice(shardIDs, func(i, j int) bool { return shardIDs[i] < shardIDs[j] })
	return shardIDs
}

func sortedZoneIDs(zones map[string]*ZoneRuntime) []string {
	zoneIDs := make([]string, 0, len(zones))
	for zoneID := range zones {
		zoneIDs = append(zoneIDs, zoneID)
	}
	sort.Strings(zoneIDs)
	return zoneIDs
}

func percentileDuration(sorted []time.Duration, percentile int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := (percentile*len(sorted) + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(sorted) {
		index = len(sorted)
	}
	return sorted[index-1]
}
