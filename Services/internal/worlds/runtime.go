package worlds

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"amandacore/services/internal/simcore"
)

const (
	defaultWorldRuntimeTickInterval = 50 * time.Millisecond
	defaultSlowTickThreshold        = 100 * time.Millisecond
	defaultCommandQueueLimit        = 4096
	defaultMoveMaxStep              = 12.0
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
	MovementRules     MovementRules
	Clock             TickClock
}

type MovementRules struct {
	MaxStepDistance float64
	Bounds          RuntimeBounds
	ServerZ         float64
	ControlZ        bool
}

type RuntimeBounds struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

type MovementValidationResult struct {
	Accepted      bool
	Corrected     bool
	Rejected      bool
	ReasonCode    string
	From          simcore.Vector3
	Requested     simcore.Vector3
	Authoritative simcore.Vector3
}

type SimulationTick struct {
	ID        simcore.TickID `json:"tickId"`
	StartedAt time.Time      `json:"startedAt"`
	Interval  time.Duration  `json:"interval"`
}

type CommandRejection struct {
	CommandID   simcore.CommandID       `json:"commandId"`
	SessionID   simcore.SessionID       `json:"sessionId"`
	CharacterID simcore.CharacterID     `json:"characterId"`
	Reason      simcore.RejectionReason `json:"reason"`
	Message     string                  `json:"message,omitempty"`
}

type TickResult struct {
	Tick                 SimulationTick        `json:"tick"`
	CompletedAt          time.Time             `json:"completedAt"`
	Duration             time.Duration         `json:"duration"`
	Slow                 bool                  `json:"slow"`
	QueueDepthBeforeTick int                   `json:"queueDepthBeforeTick"`
	CommandsProcessed    int                   `json:"commandsProcessed"`
	CommandsRejected     int                   `json:"commandsRejected"`
	ProcessedCommands    []simcore.CommandKind `json:"processedCommands"`
	Rejections           []CommandRejection    `json:"rejections"`
	Events               []simcore.DomainEvent `json:"-"`
	Diffs                []simcore.StateDiff   `json:"-"`
	DirtyCharacters      []DirtyCharacterState `json:"dirtyCharacters"`
}

type RuntimeMetrics struct {
	TicksProcessed     int64   `json:"ticksProcessed"`
	CommandsAccepted   int64   `json:"commandsAccepted"`
	CommandsRejected   int64   `json:"commandsRejected"`
	MaxQueueDepth      int     `json:"maxQueueDepth"`
	LastQueueDepth     int     `json:"lastQueueDepth"`
	LastTickDurationMs float64 `json:"lastTickDurationMs"`
	MaxTickDurationMs  float64 `json:"maxTickDurationMs"`
	ActiveEntities     int     `json:"activeEntities"`
	ActiveSessions     int     `json:"activeSessions"`
	DirtyEntities      int64   `json:"dirtyEntities"`
}

type WorldRuntime struct {
	mutex             sync.Mutex
	tickInterval      time.Duration
	slowTickThreshold time.Duration
	commandQueueLimit int
	clock             TickClock
	gateway           *SessionGateway
	persistence       *PersistenceHandoff
	movementRules     MovementRules
	nextSequence      uint64
	nextTickID        simcore.TickID
	commandQueue      []simcore.CommandEnvelope
	zones             map[simcore.ZoneID]*ZoneRuntime
	instances         map[simcore.ZoneID]*InstanceRuntime
	metrics           RuntimeMetrics
}

func NewWorldRuntime(config WorldRuntimeConfig, gateway *SessionGateway, persistence *PersistenceHandoff) *WorldRuntime {
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
	if config.MovementRules.MaxStepDistance <= 0 {
		config.MovementRules.MaxStepDistance = defaultMoveMaxStep
	}
	if config.MovementRules.Bounds.MaxX <= config.MovementRules.Bounds.MinX {
		config.MovementRules.Bounds = RuntimeBounds{MinX: 0, MinY: 0, MaxX: starterZoneMaxX, MaxY: starterZoneMaxY}
	}
	if gateway == nil {
		gateway = NewSessionGateway()
	}
	if persistence == nil {
		persistence = NewPersistenceHandoff(nil)
	}

	return &WorldRuntime{
		tickInterval:      config.TickInterval,
		slowTickThreshold: config.SlowTickThreshold,
		commandQueueLimit: config.CommandQueueLimit,
		clock:             config.Clock,
		gateway:           gateway,
		persistence:       persistence,
		movementRules:     config.MovementRules,
		zones:             map[simcore.ZoneID]*ZoneRuntime{},
		instances:         map[simcore.ZoneID]*InstanceRuntime{},
	}
}

func (r *WorldRuntime) Gateway() *SessionGateway {
	return r.gateway
}

func (r *WorldRuntime) Enqueue(envelope simcore.CommandEnvelope) (simcore.CommandEnvelope, error) {
	if envelope.Payload == nil {
		return simcore.CommandEnvelope{}, errors.New("command payload is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.commandQueue) >= r.commandQueueLimit {
		r.metrics.CommandsRejected++
		return simcore.CommandEnvelope{}, fmt.Errorf("%s", simcore.RejectionQueueFull)
	}

	r.nextSequence++
	if envelope.CommandID == "" {
		envelope.CommandID = simcore.CommandID(fmt.Sprintf("cmd_%012d", r.nextSequence))
	}
	if envelope.ServerReceiveTime.IsZero() {
		envelope.ServerReceiveTime = r.clock.Now()
	}
	if envelope.EnqueueTick == 0 {
		envelope.EnqueueTick = r.nextTickID + 1
	}
	if envelope.IntendedTick == 0 {
		envelope.IntendedTick = envelope.EnqueueTick
	}
	if envelope.CharacterID == "" {
		envelope.CharacterID = envelope.Payload.CommandActorID()
	}
	r.commandQueue = append(r.commandQueue, envelope)
	if len(r.commandQueue) > r.metrics.MaxQueueDepth {
		r.metrics.MaxQueueDepth = len(r.commandQueue)
	}
	r.metrics.LastQueueDepth = len(r.commandQueue)
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
	r.metrics.LastQueueDepth = 0
	r.mutex.Unlock()

	simcore.SortCommandEnvelopes(queue)

	result := TickResult{
		Tick:                 tick,
		QueueDepthBeforeTick: len(queue),
		ProcessedCommands:    make([]simcore.CommandKind, 0, len(queue)),
		Events:               make([]simcore.DomainEvent, 0, len(queue)),
		Diffs:                []simcore.StateDiff{{TickID: tick.ID}},
	}

	for _, envelope := range queue {
		result.CommandsProcessed++
		result.ProcessedCommands = append(result.ProcessedCommands, envelope.Payload.CommandKind())

		session, validation := r.gateway.ValidateCommand(envelope)
		if !validation.Accepted {
			result.addRejection(envelope, validation)
			continue
		}

		events, deltas, dirty, validation := r.applyCommand(tick.ID, envelope, session, now)
		if !validation.Accepted {
			result.addRejection(envelope, validation)
			continue
		}
		result.Events = append(result.Events, events...)
		if len(deltas) > 0 {
			result.Diffs[0].Deltas = append(result.Diffs[0].Deltas, deltas...)
		}
		if dirty.CharacterID != "" {
			result.DirtyCharacters = append(result.DirtyCharacters, dirty)
		}
	}

	result.CompletedAt = r.clock.Now()
	result.Duration = result.CompletedAt.Sub(result.Tick.StartedAt)
	result.Slow = result.Duration > r.slowTickThreshold
	if len(result.Diffs) == 1 && len(result.Diffs[0].Deltas) == 0 {
		result.Diffs = nil
	}

	r.recordTickMetrics(result)
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

func (r *WorldRuntime) RegisterOrUpdateEntity(entity RuntimeEntity) error {
	zoneID := entity.ZoneID
	if zoneID == "" {
		zoneID = simcore.ZoneID(defaultZoneID)
		entity.ZoneID = zoneID
	}

	zone, ok := r.Zone(zoneID)
	if !ok {
		created, err := NewZoneRuntime(ZoneDefinition{ID: zoneID, DisplayName: string(zoneID)})
		if err != nil {
			return err
		}
		if err := r.RegisterZone(created); err != nil {
			return err
		}
		zone = created
	}
	return zone.RegisterEntity(entity)
}

func (r *WorldRuntime) SnapshotMetrics() RuntimeMetrics {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	metrics := r.metrics
	metrics.ActiveSessions = r.gateway.ActiveSessionCount()
	metrics.ActiveEntities = 0
	for _, zone := range r.zones {
		metrics.ActiveEntities += zone.Entities.Count()
	}
	return metrics
}

func (r *WorldRuntime) applyCommand(tickID simcore.TickID, envelope simcore.CommandEnvelope, session GatewaySession, now time.Time) ([]simcore.DomainEvent, []simcore.StateDelta, DirtyCharacterState, simcore.CommandValidation) {
	switch command := envelope.Payload.(type) {
	case simcore.MoveIntentCommand:
		return r.applyMoveIntent(tickID, envelope, session, command, now)
	case simcore.UseAbilityIntentCommand:
		event := simcore.CombatIntentSubmittedEvent{CharacterID: session.CharacterID, AbilityID: command.AbilityID, TargetID: command.TargetID}
		return []simcore.DomainEvent{event}, []simcore.StateDelta{simcore.EventDelta{Event: event}}, DirtyCharacterState{}, simcore.CommandValidation{Accepted: true}
	case simcore.DisconnectIntentCommand:
		_, _ = r.gateway.Disconnect(envelope.SessionID, command.Reason, now)
		event := simcore.PlayerDisconnectedEvent{CharacterID: session.CharacterID, ZoneID: session.ZoneID, Reason: command.Reason}
		return []simcore.DomainEvent{event}, []simcore.StateDelta{
			simcore.EventDelta{Event: event},
		}, DirtyCharacterState{}, simcore.CommandValidation{Accepted: true}
	case simcore.ReconnectIntentCommand:
		updated, _ := r.gateway.CompleteReconnect(envelope.SessionID, session.AuthoritativePosition, now)
		event := simcore.PlayerReconnectedEvent{CharacterID: updated.CharacterID, ZoneID: updated.ZoneID, Position: updated.AuthoritativePosition, Reason: command.Reason}
		return []simcore.DomainEvent{event}, []simcore.StateDelta{simcore.EventDelta{Event: event}}, DirtyCharacterState{}, simcore.CommandValidation{Accepted: true}
	default:
		return nil, nil, DirtyCharacterState{}, simcore.CommandValidation{Accepted: true}
	}
}

func (r *WorldRuntime) applyMoveIntent(tickID simcore.TickID, envelope simcore.CommandEnvelope, session GatewaySession, command simcore.MoveIntentCommand, now time.Time) ([]simcore.DomainEvent, []simcore.StateDelta, DirtyCharacterState, simcore.CommandValidation) {
	entity := RuntimeEntity{
		ID:       simcore.EntityID(session.CharacterID),
		Kind:     "player",
		ZoneID:   session.ZoneID,
		Position: session.AuthoritativePosition,
	}
	if zone, ok := r.Zone(session.ZoneID); ok {
		if existing, found := zone.LookupEntity(simcore.EntityID(session.CharacterID)); found {
			entity = existing
		}
	}

	validation := ValidateMovement(entity.Position, command.Delta, r.movementRules)
	if validation.Rejected {
		return nil, nil, DirtyCharacterState{}, simcore.CommandValidation{
			Accepted: false,
			Reason:   simcore.RejectionInvalidMovement,
			Message:  validation.ReasonCode,
		}
	}

	entity.Position = validation.Authoritative
	entity.LastUpdatedAt = now
	if err := r.RegisterOrUpdateEntity(entity); err != nil {
		return nil, nil, DirtyCharacterState{}, simcore.CommandValidation{
			Accepted: false,
			Reason:   simcore.RejectionInvalidPayload,
			Message:  err.Error(),
		}
	}
	_, _ = r.gateway.UpdatePosition(envelope.SessionID, session.ZoneID, validation.Authoritative, now)

	dirty := DirtyCharacterState{
		CharacterID: session.CharacterID,
		ZoneID:      session.ZoneID,
		Position:    validation.Authoritative,
		Reason:      "movement",
		MarkedAt:    now,
	}
	r.persistence.MarkCharacterDirty(dirty.CharacterID, dirty.ZoneID, dirty.Position, dirty.Reason, now)

	moved := simcore.PlayerMovedEvent{
		CharacterID: session.CharacterID,
		ZoneID:      session.ZoneID,
		From:        validation.From,
		To:          validation.Authoritative,
	}
	events := []simcore.DomainEvent{moved}
	deltas := []simcore.StateDelta{
		simcore.PositionDelta{
			EntityID: simcore.EntityID(session.CharacterID),
			ZoneID:   session.ZoneID,
			From:     validation.From,
			To:       validation.Authoritative,
		},
	}

	if validation.Corrected {
		corrected := simcore.PlayerCorrectedEvent{
			CharacterID: session.CharacterID,
			ZoneID:      session.ZoneID,
			Requested:   validation.Requested,
			Corrected:   validation.Authoritative,
			Reason:      validation.ReasonCode,
		}
		events = append(events, corrected)
		deltas = append(deltas, simcore.CorrectionDelta{
			EntityID:              simcore.EntityID(session.CharacterID),
			ZoneID:                session.ZoneID,
			RequestedPosition:     validation.Requested,
			AuthoritativePosition: validation.Authoritative,
			ReasonCode:            validation.ReasonCode,
			ServerTick:            tickID,
		})
	}

	return events, deltas, dirty, simcore.CommandValidation{Accepted: true}
}

func (r *WorldRuntime) recordTickMetrics(result TickResult) {
	durationMs := float64(result.Duration.Microseconds()) / 1000.0

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.metrics.TicksProcessed++
	r.metrics.CommandsAccepted += int64(result.CommandsProcessed - result.CommandsRejected)
	r.metrics.CommandsRejected += int64(result.CommandsRejected)
	r.metrics.LastTickDurationMs = durationMs
	if durationMs > r.metrics.MaxTickDurationMs {
		r.metrics.MaxTickDurationMs = durationMs
	}
	r.metrics.DirtyEntities += int64(len(result.DirtyCharacters))
}

func (r *TickResult) addRejection(envelope simcore.CommandEnvelope, validation simcore.CommandValidation) {
	r.CommandsRejected++
	rejection := CommandRejection{
		CommandID:   envelope.CommandID,
		SessionID:   envelope.SessionID,
		CharacterID: envelope.CharacterID,
		Reason:      validation.Reason,
		Message:     validation.Message,
	}
	r.Rejections = append(r.Rejections, rejection)
	r.Events = append(r.Events, simcore.CommandRejectedEvent{
		CharacterID: envelope.CharacterID,
		SessionID:   envelope.SessionID,
		CommandID:   envelope.CommandID,
		Reason:      validation.Reason,
		Message:     validation.Message,
	})
}

func ValidateMovement(current simcore.Vector3, delta simcore.Vector3, rules MovementRules) MovementValidationResult {
	if rules.MaxStepDistance <= 0 {
		rules.MaxStepDistance = defaultMoveMaxStep
	}
	if rules.Bounds.MaxX <= rules.Bounds.MinX {
		rules.Bounds = RuntimeBounds{MinX: 0, MinY: 0, MaxX: starterZoneMaxX, MaxY: starterZoneMaxY}
	}

	result := MovementValidationResult{
		Accepted:      true,
		From:          current,
		Requested:     simcore.Vector3{X: current.X + delta.X, Y: current.Y + delta.Y, Z: current.Z + delta.Z},
		Authoritative: simcore.Vector3{X: current.X + delta.X, Y: current.Y + delta.Y, Z: current.Z + delta.Z},
	}
	if invalidFloat(delta.X) || invalidFloat(delta.Y) || invalidFloat(delta.Z) {
		result.Accepted = false
		result.Rejected = true
		result.ReasonCode = "invalid_number"
		result.Authoritative = current
		return result
	}

	distance := math.Hypot(delta.X, delta.Y)
	if distance > rules.MaxStepDistance {
		scale := rules.MaxStepDistance / distance
		result.Authoritative.X = current.X + (delta.X * scale)
		result.Authoritative.Y = current.Y + (delta.Y * scale)
		result.Corrected = true
		result.ReasonCode = "speed_limited"
	}

	clampedX := clampFloat(result.Authoritative.X, rules.Bounds.MinX, rules.Bounds.MaxX)
	clampedY := clampFloat(result.Authoritative.Y, rules.Bounds.MinY, rules.Bounds.MaxY)
	if clampedX != result.Authoritative.X || clampedY != result.Authoritative.Y {
		result.Authoritative.X = clampedX
		result.Authoritative.Y = clampedY
		result.Corrected = true
		if result.ReasonCode == "" {
			result.ReasonCode = "bounds_clamped"
		}
	}
	if rules.ControlZ {
		if result.Authoritative.Z != rules.ServerZ {
			result.Corrected = true
			if result.ReasonCode == "" {
				result.ReasonCode = "z_server_controlled"
			}
		}
		result.Authoritative.Z = rules.ServerZ
	}
	if result.ReasonCode == "" {
		result.ReasonCode = "accepted"
	}
	return result
}

func invalidFloat(value float64) bool {
	return math.IsNaN(value) || math.IsInf(value, 0)
}

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

type ZoneBounds struct {
	Min simcore.Vector3 `json:"min"`
	Max simcore.Vector3 `json:"max"`
}

type ZoneDefinition struct {
	ID          simcore.ZoneID `json:"zoneId"`
	DisplayName string         `json:"displayName"`
	Bounds      ZoneBounds     `json:"bounds"`
}

type RuntimeEntity struct {
	ID            simcore.EntityID `json:"entityId"`
	Kind          string           `json:"kind"`
	ZoneID        simcore.ZoneID   `json:"zoneId"`
	Position      simcore.Vector3  `json:"position"`
	LastUpdatedAt time.Time        `json:"lastUpdatedAt"`
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
	if entity.ZoneID == "" {
		entity.ZoneID = z.Definition.ID
	}
	return z.Entities.Register(entity)
}

func (z *ZoneRuntime) LookupEntity(id simcore.EntityID) (RuntimeEntity, bool) {
	return z.Entities.Lookup(id)
}

type InstanceRuntime struct {
	ID        simcore.ZoneID `json:"instanceId"`
	Zone      *ZoneRuntime   `json:"-"`
	CreatedAt time.Time      `json:"createdAt"`
}
