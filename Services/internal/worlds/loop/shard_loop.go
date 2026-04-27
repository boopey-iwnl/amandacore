package loop

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	defaultQueueLimit     = 256
	defaultCommandTimeout = 2 * time.Second
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type LoopEvent struct {
	Name         string
	ShardID      string
	ZoneID       string
	CommandID    string
	CommandKind  CommandKind
	SessionToken string
	ActorID      string
	Sequence     uint64
	Tick         uint64
	QueueDepth   int
	Latency      time.Duration
	Err          error
}

type Observer func(LoopEvent)

type ShardLoopConfig struct {
	ShardID        string
	ZoneID         string
	QueueLimit     int
	CommandTimeout time.Duration
	Clock          Clock
	Observer       Observer
}

type commandRequest struct {
	command    WorldCommand
	receivedAt time.Time
	done       chan commandResponse
}

type commandResponse struct {
	result CommandResult
	err    error
}

type ShardLoop struct {
	mu        sync.Mutex
	config    ShardLoopConfig
	state     *ShardState
	queue     chan commandRequest
	stop      chan struct{}
	done      chan struct{}
	started   bool
	stopped   bool
	sequence  uint64
	tick      uint64
	replayLog []ReplayRecord
	metrics   LoopMetrics
}

func NewShardLoop(config ShardLoopConfig) *ShardLoop {
	if config.ShardID == "" {
		config.ShardID = StonewakeShardID
	}
	if config.ZoneID == "" {
		config.ZoneID = StonewakeZoneID
	}
	if config.QueueLimit <= 0 {
		config.QueueLimit = defaultQueueLimit
	}
	if config.CommandTimeout <= 0 {
		config.CommandTimeout = defaultCommandTimeout
	}
	if config.Clock == nil {
		config.Clock = SystemClock{}
	}

	return &ShardLoop{
		config: config,
		state:  NewShardState(config.ShardID, config.ZoneID),
		queue:  make(chan commandRequest, config.QueueLimit),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
		metrics: LoopMetrics{
			ShardID:       config.ShardID,
			ZoneID:        config.ZoneID,
			QueueCapacity: config.QueueLimit,
		},
	}
}

func (l *ShardLoop) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.started {
		return nil
	}
	if l.stopped {
		return ErrLoopStopped
	}

	l.started = true
	l.metrics.Running = true
	go l.run()
	l.emit(LoopEvent{Name: "world.loop_started"})
	return nil
}

func (l *ShardLoop) Stop(ctx context.Context) error {
	l.mu.Lock()
	if !l.started {
		l.mu.Unlock()
		return nil
	}
	if l.stopped {
		done := l.done
		l.mu.Unlock()
		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	l.stopped = true
	close(l.stop)
	done := l.done
	l.mu.Unlock()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *ShardLoop) Submit(ctx context.Context, command WorldCommand) (CommandResult, error) {
	if command == nil {
		return CommandResult{}, ErrCommandRequired
	}
	if ctx == nil {
		ctx = context.Background()
	}

	l.mu.Lock()
	started := l.started
	stopped := l.stopped
	timeout := l.config.CommandTimeout
	l.mu.Unlock()
	if !started {
		return CommandResult{}, ErrLoopNotStarted
	}
	if stopped {
		return CommandResult{}, ErrLoopStopped
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline && timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	request := commandRequest{
		command:    command,
		receivedAt: l.config.Clock.Now(),
		done:       make(chan commandResponse, 1),
	}

	select {
	case l.queue <- request:
		l.recordAccepted()
		l.emit(LoopEvent{
			Name:         "world.command_accepted",
			CommandKind:  command.Kind(),
			SessionToken: command.SessionToken(),
			ActorID:      command.ActorID(),
			QueueDepth:   len(l.queue),
		})
	case <-ctx.Done():
		l.recordTimeout()
		l.emit(LoopEvent{
			Name:         "world.command_timeout",
			CommandKind:  command.Kind(),
			SessionToken: command.SessionToken(),
			ActorID:      command.ActorID(),
			Err:          ctx.Err(),
		})
		return CommandResult{}, fmt.Errorf("%w: %v", ErrCommandTimeout, ctx.Err())
	}

	select {
	case response := <-request.done:
		return response.result, response.err
	case <-ctx.Done():
		l.recordTimeout()
		l.emit(LoopEvent{
			Name:         "world.command_timeout",
			CommandKind:  command.Kind(),
			SessionToken: command.SessionToken(),
			ActorID:      command.ActorID(),
			Err:          ctx.Err(),
		})
		return CommandResult{}, fmt.Errorf("%w: %v", ErrCommandTimeout, ctx.Err())
	}
}

func (l *ShardLoop) Snapshot(ctx context.Context, token string, actorID string) (WorldSnapshot, error) {
	result, err := l.Submit(ctx, RequestSnapshotCommand{Token: token, Actor: actorID})
	if err != nil {
		return WorldSnapshot{}, err
	}
	return result.Snapshot, nil
}

func (l *ShardLoop) Metrics() LoopMetrics {
	l.mu.Lock()
	defer l.mu.Unlock()

	metrics := l.metrics
	metrics.QueueDepth = len(l.queue)
	return metrics
}

func (l *ShardLoop) ReplayLog() []ReplayRecord {
	l.mu.Lock()
	defer l.mu.Unlock()

	records := make([]ReplayRecord, len(l.replayLog))
	copy(records, l.replayLog)
	return records
}

func (l *ShardLoop) run() {
	defer close(l.done)
	for {
		select {
		case request := <-l.queue:
			l.apply(request)
		case <-l.stop:
			l.rejectPending()
			l.mu.Lock()
			l.metrics.Running = false
			l.mu.Unlock()
			l.emit(LoopEvent{Name: "world.loop_stopped"})
			return
		}
	}
}

func (l *ShardLoop) rejectPending() {
	for {
		select {
		case request := <-l.queue:
			request.done <- commandResponse{err: ErrLoopStopped}
		default:
			return
		}
	}
}

func (l *ShardLoop) apply(request commandRequest) {
	startedAt := l.config.Clock.Now()

	l.mu.Lock()
	l.sequence++
	l.tick++
	sequence := l.sequence
	tick := l.tick
	l.state.Tick = tick
	commandID := fmt.Sprintf("%s-%012d", l.config.ShardID, sequence)
	context := CommandContext{
		CommandID: commandID,
		Sequence:  sequence,
		Tick:      tick,
		Now:       startedAt,
	}
	l.mu.Unlock()

	result, err := request.command.Apply(l.state, context)
	if result.CommandID == "" {
		result.CommandID = commandID
	}
	if result.Sequence == 0 {
		result.Sequence = sequence
	}
	if result.Tick == 0 {
		result.Tick = tick
	}
	if result.Kind == "" {
		result.Kind = request.command.Kind()
	}
	if result.Snapshot.ShardID == "" {
		result.Snapshot = l.state.Snapshot()
	}

	latency := l.config.Clock.Now().Sub(startedAt)
	l.mu.Lock()
	l.metrics.LastCommandLatency = latency
	if latency > l.metrics.MaxCommandLatency {
		l.metrics.MaxCommandLatency = latency
	}
	l.metrics.LastAppliedSequence = sequence
	if err != nil {
		l.metrics.CommandsRejected++
	} else {
		l.metrics.CommandsApplied++
		if request.command.Kind() == CommandRequestSnapshot {
			l.metrics.SnapshotsEmitted++
		}
		if request.command.Kind() == CommandReconnectWorldSession {
			l.metrics.ReconnectsRestored++
		}
		record := ReplayRecord{
			Sequence:     sequence,
			Tick:         tick,
			CommandID:    commandID,
			CommandKind:  request.command.Kind(),
			SessionToken: request.command.SessionToken(),
			ActorID:      request.command.ActorID(),
			RecordedAt:   startedAt,
			Payload:      request.command.ReplayPayload(),
		}
		l.replayLog = append(l.replayLog, record)
		l.metrics.ReplayRecords = uint64(len(l.replayLog))
	}
	l.mu.Unlock()

	eventName := "world.command_applied"
	if err != nil {
		eventName = "world.command_rejected"
	}
	l.emit(LoopEvent{
		Name:         eventName,
		CommandID:    commandID,
		CommandKind:  request.command.Kind(),
		SessionToken: request.command.SessionToken(),
		ActorID:      request.command.ActorID(),
		Sequence:     sequence,
		Tick:         tick,
		QueueDepth:   len(l.queue),
		Latency:      latency,
		Err:          err,
	})
	if err == nil && request.command.Kind() == CommandRequestSnapshot {
		l.emit(LoopEvent{
			Name:         "world.snapshot_emitted",
			CommandID:    commandID,
			CommandKind:  request.command.Kind(),
			SessionToken: request.command.SessionToken(),
			ActorID:      request.command.ActorID(),
			Sequence:     sequence,
			Tick:         tick,
		})
	}
	if err == nil && request.command.Kind() == CommandReconnectWorldSession {
		l.emit(LoopEvent{
			Name:         "world.reconnect_restored",
			CommandID:    commandID,
			CommandKind:  request.command.Kind(),
			SessionToken: request.command.SessionToken(),
			ActorID:      request.command.ActorID(),
			Sequence:     sequence,
			Tick:         tick,
		})
	}
	if err == nil {
		l.emit(LoopEvent{
			Name:         "world.replay_recorded",
			CommandID:    commandID,
			CommandKind:  request.command.Kind(),
			SessionToken: request.command.SessionToken(),
			ActorID:      request.command.ActorID(),
			Sequence:     sequence,
			Tick:         tick,
		})
	}

	request.done <- commandResponse{result: result, err: err}
}

func (l *ShardLoop) recordAccepted() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.metrics.CommandsAccepted++
}

func (l *ShardLoop) recordTimeout() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.metrics.CommandTimeouts++
}

func (l *ShardLoop) emit(event LoopEvent) {
	if l.config.Observer == nil {
		return
	}
	if event.ShardID == "" {
		event.ShardID = l.config.ShardID
	}
	if event.ZoneID == "" {
		event.ZoneID = l.config.ZoneID
	}
	l.config.Observer(event)
}

func Replay(initial WorldSnapshot, records []ReplayRecord) (WorldSnapshot, error) {
	state := NewShardState(initial.ShardID, initial.ZoneID)
	state.Tick = initial.Tick
	for _, player := range initial.Players {
		state.UpsertPlayer(player)
	}
	for _, npc := range initial.NPCs {
		state.UpsertNPC(npc)
	}

	for _, record := range records {
		state.Tick = record.Tick
		if err := applyReplayRecord(state, record); err != nil {
			return WorldSnapshot{}, err
		}
	}
	return state.Snapshot(), nil
}

func applyReplayRecord(state *ShardState, record ReplayRecord) error {
	switch record.CommandKind {
	case CommandConnectWorldSession, CommandReconnectWorldSession:
		player, _ := state.PlayerBySession(record.SessionToken)
		player.SessionToken = record.SessionToken
		player.CharacterID = stringValue(record.Payload, "characterId", record.ActorID)
		player.ZoneID = stringValue(record.Payload, "zoneId", state.ZoneID)
		player.Position = Position{
			X: floatValue(record.Payload, "x", player.Position.X),
			Y: floatValue(record.Payload, "y", player.Position.Y),
			Z: floatValue(record.Payload, "z", player.Position.Z),
		}
		player.Connected = true
		player.Alive = true
		state.UpsertPlayer(player)
	case CommandDisconnectWorldSession:
		player, ok := state.playersBySession[record.SessionToken]
		if !ok {
			return ErrSessionMissing
		}
		player.Connected = false
		player.AutoAttackActive = false
		state.UpsertPlayer(player)
	case CommandApplyMovement:
		player, ok := state.playersBySession[record.SessionToken]
		if !ok {
			return ErrSessionMissing
		}
		if _, ok := record.Payload["x"]; ok {
			player.Position = Position{
				X: floatValue(record.Payload, "x", player.Position.X),
				Y: floatValue(record.Payload, "y", player.Position.Y),
				Z: floatValue(record.Payload, "z", player.Position.Z),
			}
		} else {
			player.Position.X += floatValue(record.Payload, "deltaX", 0)
			player.Position.Y += floatValue(record.Payload, "deltaY", 0)
			player.Position.Z += floatValue(record.Payload, "deltaZ", 0)
		}
		state.UpsertPlayer(player)
	case CommandSelectTarget:
		player, ok := state.playersBySession[record.SessionToken]
		if !ok {
			return ErrSessionMissing
		}
		player.TargetID = stringValue(record.Payload, "targetId", "")
		state.UpsertPlayer(player)
	case CommandClearTarget:
		player, ok := state.playersBySession[record.SessionToken]
		if !ok {
			return ErrSessionMissing
		}
		player.TargetID = ""
		player.AutoAttackActive = false
		state.UpsertPlayer(player)
	case CommandStartAutoAttack:
		player, ok := state.playersBySession[record.SessionToken]
		if !ok {
			return ErrSessionMissing
		}
		player.AutoAttackActive = boolValue(record.Payload, "enabled", false)
		state.UpsertPlayer(player)
	case CommandAcceptQuest:
		player, ok := state.playersBySession[record.SessionToken]
		if !ok {
			return ErrSessionMissing
		}
		if player.QuestProgress == nil {
			player.QuestProgress = map[string]int{}
		}
		questID := stringValue(record.Payload, "questId", "")
		if questID != "" {
			player.QuestProgress[questID] = 0
		}
		state.UpsertPlayer(player)
	case CommandProgressQuestObjective:
		player, ok := state.playersBySession[record.SessionToken]
		if !ok {
			return ErrSessionMissing
		}
		if player.QuestProgress == nil {
			player.QuestProgress = map[string]int{}
		}
		questID := stringValue(record.Payload, "questId", "")
		if questID != "" {
			player.QuestProgress[questID] += int(floatValue(record.Payload, "delta", 1))
		}
		state.UpsertPlayer(player)
	case CommandUpdateActionBar:
		player, ok := state.playersBySession[record.SessionToken]
		if !ok {
			return ErrSessionMissing
		}
		if player.ActionBarSlots == nil {
			player.ActionBarSlots = map[int]string{}
		}
		slot := int(floatValue(record.Payload, "slotIndex", 0))
		if boolValue(record.Payload, "clear", false) {
			delete(player.ActionBarSlots, slot)
		} else if abilityID := stringValue(record.Payload, "abilityId", ""); abilityID != "" {
			player.ActionBarSlots[slot] = abilityID
		}
		state.UpsertPlayer(player)
	case CommandMoveInventoryItem:
		player, ok := state.playersBySession[record.SessionToken]
		if !ok {
			return ErrSessionMissing
		}
		from := int(floatValue(record.Payload, "fromSlotIndex", 0))
		to := int(floatValue(record.Payload, "toSlotIndex", 0))
		itemID := player.InventorySlots[from]
		player.InventorySlots[from] = player.InventorySlots[to]
		player.InventorySlots[to] = itemID
		if player.InventorySlots[from] == "" {
			delete(player.InventorySlots, from)
		}
		state.UpsertPlayer(player)
	case CommandCastAbility, CommandUseAbility, CommandInteractNPC, CommandRequestSnapshot:
		return nil
	default:
		return fmt.Errorf("unsupported replay command %s", record.CommandKind)
	}
	return nil
}

func stringValue(source map[string]any, key string, fallback string) string {
	if source == nil {
		return fallback
	}
	if value, ok := source[key].(string); ok {
		return value
	}
	return fallback
}

func floatValue(source map[string]any, key string, fallback float64) float64 {
	if source == nil {
		return fallback
	}
	switch value := source[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case uint64:
		return float64(value)
	default:
		return fallback
	}
}

func boolValue(source map[string]any, key string, fallback bool) bool {
	if source == nil {
		return fallback
	}
	if value, ok := source[key].(bool); ok {
		return value
	}
	return fallback
}

func IsStopped(err error) bool {
	return errors.Is(err, ErrLoopStopped) || errors.Is(err, ErrLoopNotStarted)
}
