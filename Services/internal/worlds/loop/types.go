// Package loop provides AmandaCore-owned single-writer shard execution
// primitives for world-service mutation paths.
package loop

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

const (
	StonewakeZoneID  = "stonewake_vale"
	StonewakeShardID = "stonewake_vale.primary"
)

type CommandKind string

const (
	CommandConnectWorldSession    CommandKind = "ConnectWorldSession"
	CommandDisconnectWorldSession CommandKind = "DisconnectWorldSession"
	CommandReconnectWorldSession  CommandKind = "ReconnectWorldSession"
	CommandApplyMovement          CommandKind = "ApplyMovement"
	CommandSelectTarget           CommandKind = "SelectTarget"
	CommandClearTarget            CommandKind = "ClearTarget"
	CommandStartAutoAttack        CommandKind = "StartAutoAttack"
	CommandCastAbility            CommandKind = "CastAbility"
	CommandUseAbility             CommandKind = "UseAbility"
	CommandAcceptQuest            CommandKind = "AcceptQuest"
	CommandProgressQuestObjective CommandKind = "ProgressQuestObjective"
	CommandInteractNPC            CommandKind = "InteractNpc"
	CommandUpdateActionBar        CommandKind = "UpdateActionBar"
	CommandMoveInventoryItem      CommandKind = "MoveInventoryItem"
	CommandRequestSnapshot        CommandKind = "RequestSnapshot"
)

var (
	ErrLoopNotStarted  = errors.New("world loop is not started")
	ErrLoopStopped     = errors.New("world loop is stopped")
	ErrCommandRequired = errors.New("world command is required")
	ErrCommandTimeout  = errors.New("world command timed out")
	ErrQueueFull       = errors.New("world command queue is full")
	ErrSessionMissing  = errors.New("world session is missing")
	ErrTargetMissing   = errors.New("target is missing")
)

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type PlayerState struct {
	SessionToken     string         `json:"worldSessionToken"`
	AccountID        string         `json:"accountId,omitempty"`
	CharacterID      string         `json:"characterId"`
	DisplayName      string         `json:"displayName,omitempty"`
	ZoneID           string         `json:"zoneId"`
	Position         Position       `json:"position"`
	Connected        bool           `json:"connected"`
	Health           float64        `json:"health,omitempty"`
	MaxHealth        float64        `json:"maxHealth,omitempty"`
	Resource         float64        `json:"resource,omitempty"`
	MaxResource      float64        `json:"maxResource,omitempty"`
	Alive            bool           `json:"alive"`
	TargetID         string         `json:"targetId,omitempty"`
	AutoAttackActive bool           `json:"autoAttackActive,omitempty"`
	QuestProgress    map[string]int `json:"questProgress,omitempty"`
	InventorySlots   map[int]string `json:"inventorySlots,omitempty"`
	ActionBarSlots   map[int]string `json:"actionBarSlots,omitempty"`
}

type NpcState struct {
	ID          string   `json:"id"`
	ZoneID      string   `json:"zoneId"`
	Position    Position `json:"position"`
	Health      float64  `json:"health,omitempty"`
	MaxHealth   float64  `json:"maxHealth,omitempty"`
	Alive       bool     `json:"alive"`
	Targetable  bool     `json:"targetable"`
	TargetID    string   `json:"targetId,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	Kind        string   `json:"kind,omitempty"`
}

type WorldSnapshot struct {
	ShardID string        `json:"shardId"`
	ZoneID  string        `json:"zoneId"`
	Tick    uint64        `json:"tick"`
	Players []PlayerState `json:"players"`
	NPCs    []NpcState    `json:"npcs"`
}

type ReplayRecord struct {
	Sequence     uint64         `json:"sequence"`
	Tick         uint64         `json:"tick"`
	CommandID    string         `json:"commandId"`
	CommandKind  CommandKind    `json:"commandKind"`
	SessionToken string         `json:"worldSessionToken,omitempty"`
	ActorID      string         `json:"actorId,omitempty"`
	RecordedAt   time.Time      `json:"recordedAt"`
	Payload      map[string]any `json:"payload,omitempty"`
}

type LoopMetrics struct {
	ShardID             string        `json:"shardId"`
	ZoneID              string        `json:"zoneId"`
	QueueDepth          int           `json:"queueDepth"`
	QueueCapacity       int           `json:"queueCapacity"`
	CommandsAccepted    uint64        `json:"commandsAccepted"`
	CommandsApplied     uint64        `json:"commandsApplied"`
	CommandsRejected    uint64        `json:"commandsRejected"`
	CommandTimeouts     uint64        `json:"commandTimeouts"`
	SnapshotsEmitted    uint64        `json:"snapshotsEmitted"`
	ReconnectsRestored  uint64        `json:"reconnectsRestored"`
	ReplayRecords       uint64        `json:"replayRecords"`
	LastCommandLatency  time.Duration `json:"lastCommandLatency"`
	MaxCommandLatency   time.Duration `json:"maxCommandLatency"`
	LastAppliedSequence uint64        `json:"lastAppliedSequence"`
	Running             bool          `json:"running"`
}

type CommandContext struct {
	CommandID string
	Sequence  uint64
	Tick      uint64
	Now       time.Time
}

type CommandResult struct {
	CommandID string
	Sequence  uint64
	Kind      CommandKind
	Tick      uint64
	Snapshot  WorldSnapshot
	Payload   any
}

type WorldCommand interface {
	Kind() CommandKind
	SessionToken() string
	ActorID() string
	Apply(state *ShardState, context CommandContext) (CommandResult, error)
	ReplayPayload() map[string]any
}

type CommandFunc struct {
	CommandKind  CommandKind
	Token        string
	Actor        string
	Payload      map[string]any
	ApplyCommand func(state *ShardState, context CommandContext) (CommandResult, error)
}

func (c CommandFunc) Kind() CommandKind {
	return c.CommandKind
}

func (c CommandFunc) SessionToken() string {
	return c.Token
}

func (c CommandFunc) ActorID() string {
	return c.Actor
}

func (c CommandFunc) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	if c.ApplyCommand == nil {
		return CommandResult{}, fmt.Errorf("command apply function is required")
	}
	return c.ApplyCommand(state, context)
}

func (c CommandFunc) ReplayPayload() map[string]any {
	return cloneAnyMap(c.Payload)
}

type ShardState struct {
	ShardID string
	ZoneID  string
	Tick    uint64

	playersBySession  map[string]PlayerState
	playerSessionByID map[string]string
	npcs              map[string]NpcState
}

func NewShardState(shardID string, zoneID string) *ShardState {
	if shardID == "" {
		shardID = StonewakeShardID
	}
	if zoneID == "" {
		zoneID = StonewakeZoneID
	}
	return &ShardState{
		ShardID:           shardID,
		ZoneID:            zoneID,
		playersBySession:  map[string]PlayerState{},
		playerSessionByID: map[string]string{},
		npcs:              map[string]NpcState{},
	}
}

func (s *ShardState) UpsertPlayer(player PlayerState) {
	if player.SessionToken == "" || player.CharacterID == "" {
		return
	}
	if player.ZoneID == "" {
		player.ZoneID = s.ZoneID
	}
	player.QuestProgress = cloneIntMap(player.QuestProgress)
	player.InventorySlots = cloneSlotMap(player.InventorySlots)
	player.ActionBarSlots = cloneSlotMap(player.ActionBarSlots)
	s.playersBySession[player.SessionToken] = player
	s.playerSessionByID[player.CharacterID] = player.SessionToken
}

func (s *ShardState) RemoveSession(sessionToken string) {
	if sessionToken == "" {
		return
	}
	if player, ok := s.playersBySession[sessionToken]; ok {
		delete(s.playerSessionByID, player.CharacterID)
	}
	delete(s.playersBySession, sessionToken)
}

func (s *ShardState) PlayerBySession(sessionToken string) (PlayerState, bool) {
	player, ok := s.playersBySession[sessionToken]
	return clonePlayer(player), ok
}

func (s *ShardState) UpsertNPC(npc NpcState) {
	if npc.ID == "" {
		return
	}
	if npc.ZoneID == "" {
		npc.ZoneID = s.ZoneID
	}
	s.npcs[npc.ID] = npc
}

func (s *ShardState) RemoveNPC(id string) {
	delete(s.npcs, id)
}

func (s *ShardState) Snapshot() WorldSnapshot {
	players := make([]PlayerState, 0, len(s.playersBySession))
	for _, player := range s.playersBySession {
		players = append(players, clonePlayer(player))
	}
	sort.Slice(players, func(left, right int) bool {
		return players[left].SessionToken < players[right].SessionToken
	})

	npcs := make([]NpcState, 0, len(s.npcs))
	for _, npc := range s.npcs {
		npcs = append(npcs, npc)
	}
	sort.Slice(npcs, func(left, right int) bool {
		return npcs[left].ID < npcs[right].ID
	})

	return WorldSnapshot{
		ShardID: s.ShardID,
		ZoneID:  s.ZoneID,
		Tick:    s.Tick,
		Players: players,
		NPCs:    npcs,
	}
}

func clonePlayer(player PlayerState) PlayerState {
	player.QuestProgress = cloneIntMap(player.QuestProgress)
	player.InventorySlots = cloneSlotMap(player.InventorySlots)
	player.ActionBarSlots = cloneSlotMap(player.ActionBarSlots)
	return player
}

func cloneIntMap(source map[string]int) map[string]int {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]int, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneSlotMap(source map[int]string) map[int]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[int]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
