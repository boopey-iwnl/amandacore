package worlds

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"amandacore/services/internal/simcore"
)

const (
	defaultWorldRuntimeTickInterval = 50 * time.Millisecond
	defaultSlowTickThreshold        = 100 * time.Millisecond
	defaultCommandQueueLimit        = 1024
)

type TickClock interface {
	Now() time.Time
}

type SystemTickClock struct{}

func (SystemTickClock) Now() time.Time {
	return time.Now().UTC()
}

type WorldRuntimeConfig struct {
	TickInterval      time.Duration
	SlowTickThreshold time.Duration
	CommandQueueLimit int
	Clock             TickClock
}

type SimulationTick struct {
	ID        simcore.TickID `json:"tickId"`
	StartedAt time.Time      `json:"startedAt"`
	Interval  time.Duration  `json:"interval"`
}

type TickResult struct {
	Tick                 SimulationTick        `json:"tick"`
	CompletedAt          time.Time             `json:"completedAt"`
	Duration             time.Duration         `json:"duration"`
	Slow                 bool                  `json:"slow"`
	QueueDepthBeforeTick int                   `json:"queueDepthBeforeTick"`
	CommandsProcessed    int                   `json:"commandsProcessed"`
	ProcessedCommands    []simcore.CommandKind `json:"processedCommands"`
	Events               []simcore.DomainEvent `json:"-"`
}

type WorldRuntime struct {
	mutex             sync.Mutex
	tickInterval      time.Duration
	slowTickThreshold time.Duration
	commandQueueLimit int
	clock             TickClock
	nextSequence      uint64
	nextTickID        simcore.TickID
	commandQueue      []simcore.CommandEnvelope
	zones             map[simcore.ZoneID]*ZoneRuntime
	instances         map[simcore.InstanceID]*InstanceRuntime
}

func NewWorldRuntime(config WorldRuntimeConfig) *WorldRuntime {
	if config.TickInterval <= 0 {
		config.TickInterval = defaultWorldRuntimeTickInterval
	}
	if config.SlowTickThreshold <= 0 {
		config.SlowTickThreshold = defaultSlowTickThreshold
	}
	if config.CommandQueueLimit <= 0 {
		config.CommandQueueLimit = defaultCommandQueueLimit
	}
	if config.Clock == nil {
		config.Clock = SystemTickClock{}
	}

	return &WorldRuntime{
		tickInterval:      config.TickInterval,
		slowTickThreshold: config.SlowTickThreshold,
		commandQueueLimit: config.CommandQueueLimit,
		clock:             config.Clock,
		zones:             map[simcore.ZoneID]*ZoneRuntime{},
		instances:         map[simcore.InstanceID]*InstanceRuntime{},
	}
}

func (r *WorldRuntime) Enqueue(envelope simcore.CommandEnvelope) (simcore.CommandEnvelope, error) {
	if envelope.Command == nil {
		return simcore.CommandEnvelope{}, errors.New("command is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.commandQueue) >= r.commandQueueLimit {
		return simcore.CommandEnvelope{}, fmt.Errorf("command queue limit reached")
	}

	r.nextSequence++
	if envelope.Sequence == 0 {
		envelope.Sequence = r.nextSequence
	}
	if envelope.CommandID == "" {
		envelope.CommandID = simcore.CommandID(fmt.Sprintf("cmd_%012d", envelope.Sequence))
	}
	if envelope.ReceivedAt.IsZero() {
		envelope.ReceivedAt = r.clock.Now()
	}
	if envelope.ActorID == "" {
		envelope.ActorID = envelope.Command.CommandActorID()
	}

	r.commandQueue = append(r.commandQueue, envelope)
	return envelope, nil
}

func (r *WorldRuntime) PendingCommandCount() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return len(r.commandQueue)
}

func (r *WorldRuntime) RunTick(now time.Time) TickResult {
	if now.IsZero() {
		now = r.clock.Now()
	}

	r.mutex.Lock()
	r.nextTickID++
	tick := SimulationTick{
		ID:        r.nextTickID,
		StartedAt: now,
		Interval:  r.tickInterval,
	}
	queue := append([]simcore.CommandEnvelope{}, r.commandQueue...)
	r.commandQueue = r.commandQueue[:0]
	r.mutex.Unlock()

	sort.SliceStable(queue, func(left, right int) bool {
		if !queue[left].ReceivedAt.Equal(queue[right].ReceivedAt) {
			return queue[left].ReceivedAt.Before(queue[right].ReceivedAt)
		}
		if queue[left].Sequence != queue[right].Sequence {
			return queue[left].Sequence < queue[right].Sequence
		}
		return queue[left].CommandID < queue[right].CommandID
	})

	result := TickResult{
		Tick:                 tick,
		QueueDepthBeforeTick: len(queue),
		ProcessedCommands:    make([]simcore.CommandKind, 0, len(queue)),
		Events:               make([]simcore.DomainEvent, 0, len(queue)),
	}

	for _, envelope := range queue {
		result.CommandsProcessed++
		result.ProcessedCommands = append(result.ProcessedCommands, envelope.Command.CommandKind())
		result.Events = append(result.Events, eventsForCommand(envelope)...)
	}

	result.CompletedAt = r.clock.Now()
	result.Duration = result.CompletedAt.Sub(result.Tick.StartedAt)
	result.Slow = result.Duration > r.slowTickThreshold
	return result
}

func (r *WorldRuntime) RegisterZone(zone *ZoneRuntime) error {
	if zone == nil {
		return errors.New("zone runtime is required")
	}
	if zone.Definition.ID == "" {
		return errors.New("zone id is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.zones[zone.Definition.ID] = zone
	return nil
}

func (r *WorldRuntime) Zone(id simcore.ZoneID) (*ZoneRuntime, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	zone, ok := r.zones[id]
	return zone, ok
}

func (r *WorldRuntime) RegisterInstance(instance *InstanceRuntime) error {
	if instance == nil {
		return errors.New("instance runtime is required")
	}
	if instance.ID == "" {
		return errors.New("instance id is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.instances[instance.ID] = instance
	return nil
}

func eventsForCommand(envelope simcore.CommandEnvelope) []simcore.DomainEvent {
	switch command := envelope.Command.(type) {
	case simcore.MoveIntentCommand:
		return []simcore.DomainEvent{
			simcore.PlayerMovedEvent{
				EntityID: command.EntityID,
				ZoneID:   envelope.ZoneID,
				From:     command.From,
				To:       command.To,
			},
		}
	case simcore.AbilityIntentCommand:
		return []simcore.DomainEvent{
			simcore.CombatIntentSubmittedEvent{
				EntityID:  command.EntityID,
				AbilityID: command.AbilityID,
				TargetID:  command.TargetID,
			},
		}
	case simcore.DisconnectIntentCommand:
		return []simcore.DomainEvent{
			simcore.PlayerDisconnectedEvent{
				EntityID: command.EntityID,
				ZoneID:   envelope.ZoneID,
				Reason:   command.Reason,
			},
		}
	case simcore.ReconnectIntentCommand:
		return []simcore.DomainEvent{
			simcore.PlayerReconnectedEvent{
				EntityID: command.EntityID,
				ZoneID:   envelope.ZoneID,
				Reason:   command.Reason,
			},
		}
	default:
		return nil
	}
}

type ZoneBounds struct {
	Min simcore.Vector3 `json:"min"`
	Max simcore.Vector3 `json:"max"`
}

type ZoneDefinition struct {
	ID          simcore.ZoneID `json:"zoneId"`
	DisplayName string         `json:"displayName"`
	ContinentID string         `json:"continentId,omitempty"`
	Bounds      ZoneBounds     `json:"bounds"`
	SpawnPoints []SpawnPoint   `json:"spawnPoints,omitempty"`
}

type SpawnPoint struct {
	ID       string          `json:"spawnPointId"`
	ZoneID   simcore.ZoneID  `json:"zoneId"`
	Position simcore.Vector3 `json:"position"`
	Tags     []string        `json:"tags,omitempty"`
}

type RuntimeEntity struct {
	ID            simcore.EntityID   `json:"entityId"`
	Kind          string             `json:"kind"`
	DisplayName   string             `json:"displayName,omitempty"`
	ZoneID        simcore.ZoneID     `json:"zoneId"`
	InstanceID    simcore.InstanceID `json:"instanceId,omitempty"`
	Position      simcore.Vector3    `json:"position"`
	LastUpdatedAt time.Time          `json:"lastUpdatedAt"`
}

type EntityRegistry struct {
	mutex    sync.RWMutex
	entities map[simcore.EntityID]RuntimeEntity
}

func NewEntityRegistry() *EntityRegistry {
	return &EntityRegistry{entities: map[simcore.EntityID]RuntimeEntity{}}
}

func (r *EntityRegistry) Register(entity RuntimeEntity) error {
	if entity.ID == "" {
		return errors.New("entity id is required")
	}
	if entity.LastUpdatedAt.IsZero() {
		entity.LastUpdatedAt = time.Now().UTC()
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.entities[entity.ID] = entity
	return nil
}

func (r *EntityRegistry) Lookup(id simcore.EntityID) (RuntimeEntity, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entity, ok := r.entities[id]
	return entity, ok
}

func (r *EntityRegistry) Unregister(id simcore.EntityID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.entities, id)
}

func (r *EntityRegistry) Count() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.entities)
}

type ZoneRuntime struct {
	Definition ZoneDefinition  `json:"definition"`
	LoadedAt   time.Time       `json:"loadedAt"`
	Entities   *EntityRegistry `json:"-"`
}

func NewZoneRuntime(definition ZoneDefinition) (*ZoneRuntime, error) {
	if definition.ID == "" {
		return nil, errors.New("zone id is required")
	}
	return &ZoneRuntime{
		Definition: definition,
		LoadedAt:   time.Now().UTC(),
		Entities:   NewEntityRegistry(),
	}, nil
}

func (z *ZoneRuntime) RegisterEntity(entity RuntimeEntity) error {
	if z == nil {
		return errors.New("zone runtime is required")
	}
	if entity.ZoneID == "" {
		entity.ZoneID = z.Definition.ID
	}
	return z.Entities.Register(entity)
}

func (z *ZoneRuntime) LookupEntity(id simcore.EntityID) (RuntimeEntity, bool) {
	if z == nil || z.Entities == nil {
		return RuntimeEntity{}, false
	}
	return z.Entities.Lookup(id)
}

type InstanceRuntime struct {
	ID        simcore.InstanceID `json:"instanceId"`
	Zone      *ZoneRuntime       `json:"-"`
	OwnerID   simcore.EntityID   `json:"ownerId,omitempty"`
	CreatedAt time.Time          `json:"createdAt"`
}

func NewInstanceRuntime(id simcore.InstanceID, zone *ZoneRuntime, ownerID simcore.EntityID) (*InstanceRuntime, error) {
	if id == "" {
		return nil, errors.New("instance id is required")
	}
	if zone == nil {
		return nil, errors.New("zone runtime is required")
	}
	return &InstanceRuntime{
		ID:        id,
		Zone:      zone,
		OwnerID:   ownerID,
		CreatedAt: time.Now().UTC(),
	}, nil
}
