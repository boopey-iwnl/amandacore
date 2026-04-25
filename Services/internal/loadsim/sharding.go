package loadsim

import (
	"fmt"
	"hash/fnv"
	"sort"
	"time"

	"amandacore/services/internal/observability"
)

type ShardID string
type ShardRole string
type AssignmentPolicy string
type RejectionReason string

const (
	ShardRoleLocalWorld         ShardRole = "LocalWorld"
	ShardRoleZoneOwner          ShardRole = "ZoneOwner"
	ShardRoleInstanceOwner      ShardRole = "InstanceOwner"
	ShardRoleGatewayPlaceholder ShardRole = "GatewayPlaceholder"
)

const (
	AssignmentStatic      AssignmentPolicy = "static"
	AssignmentLeastLoaded AssignmentPolicy = "least-loaded"
	AssignmentHashZone    AssignmentPolicy = "hash-zone"
)

const (
	RejectWrongZone                       RejectionReason = "WrongZone"
	RejectEntityNotInZone                 RejectionReason = "EntityNotInZone"
	RejectCharacterZoneMismatch           RejectionReason = "CharacterZoneMismatch"
	RejectShardNotOwner                   RejectionReason = "ShardNotOwner"
	RejectTransitionRequired              RejectionReason = "TransitionRequired"
	RejectCrossZoneInteractionUnsupported RejectionReason = "CrossZoneInteractionUnsupported"
	RejectQueueFull                       RejectionReason = "QueueFull"
)

const (
	EventShardRegistryCreated      = "shard.registry.created"
	EventShardRegistered           = "shard.registered"
	EventShardZoneAssigned         = "shard.zone.assigned"
	EventShardZoneUnassigned       = "shard.zone.unassigned"
	EventShardAssignmentRejected   = "shard.assignment.rejected"
	EventShardRouteResolved        = "shard.route.resolved"
	EventShardRouteRejected        = "shard.route.rejected"
	EventShardLoadSnapshotRecorded = "shard.load_snapshot.recorded"
	EventShardCapacityWarning      = "shard.capacity.warning"
	EventShardBackpressureDetected = "shard.backpressure.detected"
	EventWorldQueueBackpressure    = "world.queue.backpressure"
	EventWorldCommandRejected      = "world.command.rejected"
	EventWorldWrongZoneRejected    = "world.command.wrong_zone_rejected"
	EventWorldIsolationViolation   = "world.zone.isolation_violation_detected"
)

type ShardCapacity struct {
	MaxZones           int `json:"maxZones"`
	MaxSessions        int `json:"maxSessions"`
	MaxEntities        int `json:"maxEntities"`
	MaxCommandsPerTick int `json:"maxCommandsPerTick"`
}

type ShardLoadSnapshot struct {
	ShardID                    ShardID       `json:"shardId"`
	ActiveZones                int           `json:"activeZones"`
	ActiveSessions             int           `json:"activeSessions"`
	ActiveEntities             int           `json:"activeEntities"`
	QueuedCommands             int           `json:"queuedCommands"`
	LastTickDuration           time.Duration `json:"lastTickDuration"`
	RollingAverageTickDuration time.Duration `json:"rollingAverageTickDuration"`
}

type ZoneShardBinding struct {
	ZoneID  string  `json:"zoneId"`
	ShardID ShardID `json:"shardId"`
}

type ShardAssignment struct {
	Bindings []ZoneShardBinding `json:"bindings"`
}

type CommandEnvelope struct {
	CommandID    string `json:"commandId"`
	CharacterID  string `json:"characterId"`
	ZoneID       string `json:"zoneId"`
	Type         string `json:"type"`
	TargetID     string `json:"targetId,omitempty"`
	TargetZoneID string `json:"targetZoneId,omitempty"`
}

type CommandResult struct {
	CommandID           string          `json:"commandId"`
	Accepted            bool            `json:"accepted"`
	Rejected            bool            `json:"rejected"`
	Reason              RejectionReason `json:"reason,omitempty"`
	ZoneID              string          `json:"zoneId,omitempty"`
	ShardID             ShardID         `json:"shardId,omitempty"`
	TransitionRequested bool            `json:"transitionRequested,omitempty"`
	TransitionCompleted bool            `json:"transitionCompleted,omitempty"`
	TransitionRejected  bool            `json:"transitionRejected,omitempty"`
	ToZoneID            string          `json:"toZoneId,omitempty"`
}

type ZoneRuntime struct {
	Zone          ZoneSpec
	ShardID       ShardID
	QueueCapacity int
	queue         []CommandEnvelope
	Sessions      map[string]Point
	Entities      int
	QueueMetrics  QueueMetrics
	LastTick      time.Duration
	tickDurations []time.Duration
}

type ShardRuntime struct {
	ShardID  ShardID
	Role     ShardRole
	Capacity ShardCapacity
	Zones    map[string]*ZoneRuntime
}

type ShardRegistry struct {
	Shards map[ShardID]*ShardRuntime
	Order  []ShardID
}

type ShardRouter struct {
	Registry      *ShardRegistry
	ZoneBindings  map[string]ShardID
	CharacterZone map[string]string
	Content       ContentPackage
}

func NewShardRegistry() *ShardRegistry {
	observability.LogEvent("loadsim", EventShardRegistryCreated, map[string]any{})
	return &ShardRegistry{Shards: map[ShardID]*ShardRuntime{}, Order: []ShardID{}}
}

func (r *ShardRegistry) Register(shard *ShardRuntime) error {
	if shard == nil || shard.ShardID == "" {
		return fmt.Errorf("shard id is required")
	}
	if _, exists := r.Shards[shard.ShardID]; exists {
		return fmt.Errorf("shard %s is already registered", shard.ShardID)
	}
	if shard.Zones == nil {
		shard.Zones = map[string]*ZoneRuntime{}
	}
	r.Shards[shard.ShardID] = shard
	r.Order = append(r.Order, shard.ShardID)
	sort.Slice(r.Order, func(left int, right int) bool { return r.Order[left] < r.Order[right] })
	observability.LogEvent("loadsim", EventShardRegistered, map[string]any{"shardId": shard.ShardID, "role": shard.Role})
	return nil
}

func NewShardRuntime(id ShardID, capacity ShardCapacity) *ShardRuntime {
	if capacity.MaxZones <= 0 {
		capacity.MaxZones = 64
	}
	if capacity.MaxSessions <= 0 {
		capacity.MaxSessions = 10000
	}
	if capacity.MaxEntities <= 0 {
		capacity.MaxEntities = 100000
	}
	if capacity.MaxCommandsPerTick <= 0 {
		capacity.MaxCommandsPerTick = 100000
	}
	return &ShardRuntime{ShardID: id, Role: ShardRoleZoneOwner, Capacity: capacity, Zones: map[string]*ZoneRuntime{}}
}

func BuildShardRouter(content ContentPackage, shardCount int, policy AssignmentPolicy, queueCapacity int) (*ShardRouter, error) {
	if shardCount <= 0 {
		return nil, fmt.Errorf("shard count must be greater than zero")
	}
	registry := NewShardRegistry()
	for index := 0; index < shardCount; index++ {
		id := ShardID(fmt.Sprintf("local-shard-%02d", index+1))
		if err := registry.Register(NewShardRuntime(id, ShardCapacity{MaxZones: len(content.ZoneOrder), MaxSessions: 10000, MaxEntities: 100000, MaxCommandsPerTick: queueCapacity})); err != nil {
			return nil, err
		}
	}
	router := &ShardRouter{
		Registry:      registry,
		ZoneBindings:  map[string]ShardID{},
		CharacterZone: map[string]string{},
		Content:       content,
	}
	for _, zoneID := range content.ZoneOrder {
		shard, err := router.chooseShard(zoneID, policy)
		if err != nil {
			observability.LogEvent("loadsim", EventShardAssignmentRejected, map[string]any{"zoneId": zoneID, "reason": err.Error()})
			return nil, err
		}
		zoneRuntime := &ZoneRuntime{
			Zone:          content.Zones[zoneID],
			ShardID:       shard.ShardID,
			QueueCapacity: queueCapacity,
			Sessions:      map[string]Point{},
			Entities:      len(content.Zones[zoneID].TransitionGates) + len(content.Zones[zoneID].EntryPoints),
		}
		shard.Zones[zoneID] = zoneRuntime
		router.ZoneBindings[zoneID] = shard.ShardID
		observability.LogEvent("loadsim", EventShardZoneAssigned, map[string]any{"zoneId": zoneID, "shardId": shard.ShardID})
	}
	return router, nil
}

func (r *ShardRouter) chooseShard(zoneID string, policy AssignmentPolicy) (*ShardRuntime, error) {
	if len(r.Registry.Order) == 0 {
		return nil, fmt.Errorf("no shards registered")
	}
	switch policy {
	case AssignmentStatic:
		index := len(r.ZoneBindings) % len(r.Registry.Order)
		return r.Registry.Shards[r.Registry.Order[index]], nil
	case AssignmentHashZone:
		hash := fnv.New32a()
		_, _ = hash.Write([]byte(zoneID))
		index := int(hash.Sum32()) % len(r.Registry.Order)
		return r.Registry.Shards[r.Registry.Order[index]], nil
	case AssignmentLeastLoaded:
		var selected *ShardRuntime
		for _, shardID := range r.Registry.Order {
			shard := r.Registry.Shards[shardID]
			if selected == nil || len(shard.Zones) < len(selected.Zones) {
				selected = shard
			}
		}
		if selected == nil {
			return nil, fmt.Errorf("no shard selected")
		}
		if len(selected.Zones) >= selected.Capacity.MaxZones {
			return nil, fmt.Errorf("least-loaded shard %s is at max_zones capacity", selected.ShardID)
		}
		return selected, nil
	default:
		return nil, fmt.Errorf("unsupported assignment policy %s", policy)
	}
}

func (r *ShardRouter) RegisterCharacter(characterID string, zoneID string, position Point) error {
	zoneRuntime, result := r.Resolve(zoneID)
	if result.Rejected {
		return fmt.Errorf("zone %s is not assigned", zoneID)
	}
	r.CharacterZone[characterID] = zoneID
	zoneRuntime.Sessions[characterID] = position
	return nil
}

func (r *ShardRouter) Submit(command CommandEnvelope) CommandResult {
	ownerZone := r.CharacterZone[command.CharacterID]
	if ownerZone == "" {
		return r.reject(command, RejectEntityNotInZone)
	}
	if command.ZoneID != "" && command.ZoneID != ownerZone {
		observability.LogEvent("loadsim", EventWorldWrongZoneRejected, map[string]any{"characterId": command.CharacterID, "commandZone": command.ZoneID, "ownerZone": ownerZone})
		return r.reject(command, RejectCharacterZoneMismatch)
	}
	zoneRuntime, result := r.Resolve(ownerZone)
	if result.Rejected {
		return result
	}
	command.ZoneID = ownerZone
	result = zoneRuntime.Enqueue(command)
	if result.Rejected {
		if result.Reason == RejectQueueFull {
			observability.LogEvent("loadsim", EventShardBackpressureDetected, map[string]any{"zoneId": ownerZone, "shardId": zoneRuntime.ShardID})
		}
		return result
	}
	return result
}

func (r *ShardRouter) Resolve(zoneID string) (*ZoneRuntime, CommandResult) {
	shardID := r.ZoneBindings[zoneID]
	if shardID == "" {
		return nil, CommandResult{Rejected: true, Reason: RejectShardNotOwner, ZoneID: zoneID}
	}
	shard := r.Registry.Shards[shardID]
	if shard == nil {
		return nil, CommandResult{Rejected: true, Reason: RejectShardNotOwner, ZoneID: zoneID, ShardID: shardID}
	}
	zoneRuntime := shard.Zones[zoneID]
	if zoneRuntime == nil {
		return nil, CommandResult{Rejected: true, Reason: RejectShardNotOwner, ZoneID: zoneID, ShardID: shardID}
	}
	return zoneRuntime, CommandResult{Accepted: true, ZoneID: zoneID, ShardID: shardID}
}

func (r *ShardRouter) TickAll() []CommandResult {
	results := []CommandResult{}
	for _, shardID := range r.Registry.Order {
		shard := r.Registry.Shards[shardID]
		for _, zoneID := range sortedRuntimeZoneIDs(shard.Zones) {
			zoneRuntime := shard.Zones[zoneID]
			tickResults := zoneRuntime.Tick()
			for _, result := range tickResults {
				if result.TransitionRequested && !result.Rejected {
					result = r.applyTransition(result)
				}
				results = append(results, result)
			}
		}
	}
	return results
}

func (r *ShardRouter) applyTransition(result CommandResult) CommandResult {
	fromZone := result.ZoneID
	toZone := result.ToZoneID
	characterID := result.CommandIDCharacter()
	if characterID == "" {
		result.Rejected = true
		result.Reason = RejectEntityNotInZone
		result.TransitionRejected = true
		return result
	}
	sourceRuntime, _ := r.Resolve(fromZone)
	destRuntime, destResult := r.Resolve(toZone)
	if sourceRuntime == nil || destResult.Rejected {
		result.Rejected = true
		result.Reason = RejectTransitionRequired
		result.TransitionRejected = true
		return result
	}
	position := sourceRuntime.Sessions[characterID]
	delete(sourceRuntime.Sessions, characterID)
	entry := destRuntime.Zone.DefaultEntryPoint()
	if entry.EntryID != "" {
		position = entry.Position
	}
	destRuntime.Sessions[characterID] = position
	r.CharacterZone[characterID] = toZone
	result.TransitionCompleted = true
	return result
}

func (r *ShardRouter) LoadSnapshots() []ShardLoadSnapshot {
	snapshots := []ShardLoadSnapshot{}
	for _, shardID := range r.Registry.Order {
		shard := r.Registry.Shards[shardID]
		snapshot := ShardLoadSnapshot{ShardID: shardID, ActiveZones: len(shard.Zones)}
		var totalTick time.Duration
		var tickSamples int
		for _, zone := range shard.Zones {
			snapshot.ActiveSessions += len(zone.Sessions)
			snapshot.ActiveEntities += zone.Entities + len(zone.Sessions)
			snapshot.QueuedCommands += len(zone.queue)
			if zone.LastTick > snapshot.LastTickDuration {
				snapshot.LastTickDuration = zone.LastTick
			}
			for _, tick := range zone.tickDurations {
				totalTick += tick
				tickSamples++
			}
		}
		if tickSamples > 0 {
			snapshot.RollingAverageTickDuration = totalTick / time.Duration(tickSamples)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}

func (r *ShardRouter) reject(command CommandEnvelope, reason RejectionReason) CommandResult {
	observability.LogEvent("loadsim", EventShardRouteRejected, map[string]any{"characterId": command.CharacterID, "zoneId": command.ZoneID, "reason": string(reason)})
	return CommandResult{CommandID: command.CommandID, Rejected: true, Reason: reason, ZoneID: command.ZoneID}
}

func (z *ZoneRuntime) Enqueue(command CommandEnvelope) CommandResult {
	if command.ZoneID != z.Zone.ZoneID {
		observability.LogEvent("loadsim", EventWorldIsolationViolation, map[string]any{"commandZone": command.ZoneID, "ownerZone": z.Zone.ZoneID})
		return CommandResult{CommandID: command.CommandID, Rejected: true, Reason: RejectWrongZone, ZoneID: z.Zone.ZoneID, ShardID: z.ShardID}
	}
	if len(z.queue) >= z.QueueCapacity {
		addQueueSample(&z.QueueMetrics, len(z.queue))
		observability.LogEvent("loadsim", EventWorldQueueBackpressure, map[string]any{"zoneId": z.Zone.ZoneID, "queueDepth": len(z.queue), "capacity": z.QueueCapacity})
		return CommandResult{CommandID: command.CommandID, Rejected: true, Reason: RejectQueueFull, ZoneID: z.Zone.ZoneID, ShardID: z.ShardID}
	}
	z.queue = append(z.queue, command)
	addQueueSample(&z.QueueMetrics, len(z.queue))
	return CommandResult{CommandID: command.CommandID, Accepted: true, ZoneID: z.Zone.ZoneID, ShardID: z.ShardID}
}

func (z *ZoneRuntime) Tick() []CommandResult {
	started := time.Now()
	pending := z.queue
	z.queue = nil
	results := make([]CommandResult, 0, len(pending))
	for _, command := range pending {
		result := CommandResult{CommandID: command.CommandID, Accepted: true, ZoneID: z.Zone.ZoneID, ShardID: z.ShardID}
		switch command.Type {
		case "combat", "loot":
			if command.TargetZoneID != "" && command.TargetZoneID != z.Zone.ZoneID {
				result.Accepted = false
				result.Rejected = true
				result.Reason = RejectCrossZoneInteractionUnsupported
			}
		case "transition":
			if _, ok := z.Zone.GateTo(command.TargetZoneID); !ok {
				result.Accepted = false
				result.Rejected = true
				result.Reason = RejectTransitionRequired
				result.TransitionRejected = true
			} else {
				result.TransitionRequested = true
				result.ToZoneID = command.TargetZoneID
			}
		}
		results = append(results, result)
	}
	z.LastTick = time.Since(started)
	z.tickDurations = append(z.tickDurations, z.LastTick)
	addQueueSample(&z.QueueMetrics, len(z.queue))
	return results
}

func sortedRuntimeZoneIDs(zones map[string]*ZoneRuntime) []string {
	ids := make([]string, 0, len(zones))
	for id := range zones {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r CommandResult) CommandIDCharacter() string {
	for i := 0; i < len(r.CommandID); i++ {
		if r.CommandID[i] == ':' {
			return r.CommandID[:i]
		}
	}
	return ""
}
