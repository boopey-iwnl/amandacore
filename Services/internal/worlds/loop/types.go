// Package loop provides AmandaCore-owned single-writer shard execution
// primitives for world-service mutation paths.
package loop

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"amandacore/services/internal/worlds/replication"
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
	CommandStopAutoAttack         CommandKind = "StopAutoAttack"
	CommandCastAbility            CommandKind = "CastAbility"
	CommandUseAbility             CommandKind = "UseAbility"
	CommandCancelCast             CommandKind = "CancelCast"
	CommandApplyDamage            CommandKind = "ApplyDamage"
	CommandApplyHeal              CommandKind = "ApplyHeal"
	CommandResolveDeath           CommandKind = "ResolveDeath"
	CommandRespawnNPC             CommandKind = "RespawnNpc"
	CommandScheduleRespawn        CommandKind = "ScheduleRespawn"
	CommandRequestCombatSnapshot  CommandKind = "RequestCombatSnapshot"
	CommandAddThreat              CommandKind = "AddThreat"
	CommandDecayThreat            CommandKind = "DecayThreat"
	CommandResetThreat            CommandKind = "ResetThreat"
	CommandSelectNPCTarget        CommandKind = "SelectNpcTarget"
	CommandClearThreatOnDeath     CommandKind = "ClearThreatOnDeath"
	CommandClearThreatOnLeash     CommandKind = "ClearThreatOnLeash"
	CommandAcceptQuest            CommandKind = "AcceptQuest"
	CommandAbandonQuest           CommandKind = "AbandonQuest"
	CommandProgressQuestObjective CommandKind = "ProgressQuestObjective"
	CommandCompleteQuest          CommandKind = "CompleteQuest"
	CommandClaimQuestReward       CommandKind = "ClaimQuestReward"
	CommandInteractNPC            CommandKind = "InteractNpc"
	CommandGenerateLoot           CommandKind = "GenerateLoot"
	CommandOpenLoot               CommandKind = "OpenLoot"
	CommandClaimLootItem          CommandKind = "ClaimLootItem"
	CommandClaimCurrencyReward    CommandKind = "ClaimCurrencyReward"
	CommandCloseLoot              CommandKind = "CloseLoot"
	CommandApplyQuestReward       CommandKind = "ApplyQuestReward"
	CommandApplyKillLoot          CommandKind = "ApplyKillLoot"
	CommandApplyCurrencyDelta     CommandKind = "ApplyCurrencyDelta"
	CommandApplyItemGrant         CommandKind = "ApplyItemGrant"
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
	SessionToken     string          `json:"worldSessionToken"`
	AccountID        string          `json:"accountId,omitempty"`
	CharacterID      string          `json:"characterId"`
	DisplayName      string          `json:"displayName,omitempty"`
	ZoneID           string          `json:"zoneId"`
	Position         Position        `json:"position"`
	Connected        bool            `json:"connected"`
	Health           float64         `json:"health,omitempty"`
	MaxHealth        float64         `json:"maxHealth,omitempty"`
	Resource         float64         `json:"resource,omitempty"`
	MaxResource      float64         `json:"maxResource,omitempty"`
	Alive            bool            `json:"alive"`
	TargetID         string          `json:"targetId,omitempty"`
	AutoAttackActive bool            `json:"autoAttackActive,omitempty"`
	QuestProgress    map[string]int  `json:"questProgress,omitempty"`
	QuestCompleted   map[string]bool `json:"questCompleted,omitempty"`
	LootClaims       map[string]bool `json:"lootClaims,omitempty"`
	InventorySlots   map[int]string  `json:"inventorySlots,omitempty"`
	ActionBarSlots   map[int]string  `json:"actionBarSlots,omitempty"`
	CurrencyCopper   int             `json:"currencyCopper,omitempty"`
}

type NpcState struct {
	ID          string             `json:"id"`
	ZoneID      string             `json:"zoneId"`
	Position    Position           `json:"position"`
	Health      float64            `json:"health,omitempty"`
	MaxHealth   float64            `json:"maxHealth,omitempty"`
	Alive       bool               `json:"alive"`
	Targetable  bool               `json:"targetable"`
	TargetID    string             `json:"targetId,omitempty"`
	DisplayName string             `json:"displayName,omitempty"`
	Kind        string             `json:"kind,omitempty"`
	Threat      map[string]float64 `json:"threat,omitempty"`
	RespawnTick uint64             `json:"respawnTick,omitempty"`
}

type LootItemState struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}

type LootContainerState struct {
	ID                   string          `json:"id"`
	SourceEntityID       string          `json:"sourceEntityId"`
	OwnerCharacterID     string          `json:"ownerCharacterId"`
	Items                []LootItemState `json:"items"`
	OpenedByCharacterID  string          `json:"openedByCharacterId,omitempty"`
	ClaimedByCharacterID string          `json:"claimedByCharacterId,omitempty"`
	ClaimedAtTick        uint64          `json:"claimedAtTick,omitempty"`
	Closed               bool            `json:"closed,omitempty"`
}

type WorldSnapshot struct {
	ShardID        string               `json:"shardId"`
	ZoneID         string               `json:"zoneId"`
	Tick           uint64               `json:"tick"`
	Players        []PlayerState        `json:"players"`
	NPCs           []NpcState           `json:"npcs"`
	LootContainers []LootContainerState `json:"lootContainers,omitempty"`
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
	ShardID                  string        `json:"shardId"`
	ZoneID                   string        `json:"zoneId"`
	QueueDepth               int           `json:"queueDepth"`
	QueueCapacity            int           `json:"queueCapacity"`
	CommandsAccepted         uint64        `json:"commandsAccepted"`
	CommandsApplied          uint64        `json:"commandsApplied"`
	CommandsRejected         uint64        `json:"commandsRejected"`
	GameplayCommandsApplied  uint64        `json:"gameplayCommandsApplied"`
	GameplayCommandsRejected uint64        `json:"gameplayCommandsRejected"`
	CommandTimeouts          uint64        `json:"commandTimeouts"`
	SnapshotsEmitted         uint64        `json:"snapshotsEmitted"`
	ReconnectsRestored       uint64        `json:"reconnectsRestored"`
	ReplayRecords            uint64        `json:"replayRecords"`
	StateVersion             uint64        `json:"stateVersion"`
	ReplicationFrames        uint64        `json:"replicationFrames"`
	ReplicationRetainedFrom  uint64        `json:"replicationRetainedFrom"`
	ReplicationRetainedTo    uint64        `json:"replicationRetainedTo"`
	LastCommandLatency       time.Duration `json:"lastCommandLatency"`
	MaxCommandLatency        time.Duration `json:"maxCommandLatency"`
	LastAppliedSequence      uint64        `json:"lastAppliedSequence"`
	Running                  bool          `json:"running"`
}

type CommandContext struct {
	CommandID string
	Sequence  uint64
	Tick      uint64
	Now       time.Time
}

type CommandResult struct {
	CommandID   string
	Sequence    uint64
	Kind        CommandKind
	Tick        uint64
	Snapshot    WorldSnapshot
	Replication replication.Frame
	Payload     any
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
	lootContainers    map[string]LootContainerState
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
		lootContainers:    map[string]LootContainerState{},
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
	player.QuestCompleted = cloneBoolMap(player.QuestCompleted)
	player.LootClaims = cloneBoolMap(player.LootClaims)
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
	npc.Threat = cloneFloatMap(npc.Threat)
	s.npcs[npc.ID] = npc
}

func (s *ShardState) RemoveNPC(id string) {
	delete(s.npcs, id)
}

func (s *ShardState) NPC(id string) (NpcState, bool) {
	npc, ok := s.npcs[id]
	npc.Threat = cloneFloatMap(npc.Threat)
	return npc, ok
}

func (s *ShardState) UpsertLootContainer(container LootContainerState) {
	if container.ID == "" {
		return
	}
	container.Items = cloneLootItems(container.Items)
	s.lootContainers[container.ID] = container
}

func (s *ShardState) LootContainer(id string) (LootContainerState, bool) {
	container, ok := s.lootContainers[id]
	container.Items = cloneLootItems(container.Items)
	return container, ok
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
		npc.Threat = cloneFloatMap(npc.Threat)
		npcs = append(npcs, npc)
	}
	sort.Slice(npcs, func(left, right int) bool {
		return npcs[left].ID < npcs[right].ID
	})

	lootContainers := make([]LootContainerState, 0, len(s.lootContainers))
	for _, container := range s.lootContainers {
		container.Items = cloneLootItems(container.Items)
		lootContainers = append(lootContainers, container)
	}
	sort.Slice(lootContainers, func(left, right int) bool {
		return lootContainers[left].ID < lootContainers[right].ID
	})

	return WorldSnapshot{
		ShardID:        s.ShardID,
		ZoneID:         s.ZoneID,
		Tick:           s.Tick,
		Players:        players,
		NPCs:           npcs,
		LootContainers: lootContainers,
	}
}

func clonePlayer(player PlayerState) PlayerState {
	player.QuestProgress = cloneIntMap(player.QuestProgress)
	player.QuestCompleted = cloneBoolMap(player.QuestCompleted)
	player.LootClaims = cloneBoolMap(player.LootClaims)
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

func cloneBoolMap(source map[string]bool) map[string]bool {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]bool, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneFloatMap(source map[string]float64) map[string]float64 {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]float64, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneLootItems(source []LootItemState) []LootItemState {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]LootItemState, len(source))
	copy(cloned, source)
	return cloned
}

func changedFields(previous WorldSnapshot, current WorldSnapshot, version uint64) []replication.ChangedFields {
	var changes []replication.ChangedFields
	if previous.ShardID != current.ShardID || previous.ZoneID != current.ZoneID {
		changes = append(changes, replication.ChangedFields{
			Domain:   "world",
			EntityID: current.ShardID,
			Fields:   []string{"shardId", "zoneId"},
			Version:  version,
		})
	}

	previousPlayers := map[string]PlayerState{}
	for _, player := range previous.Players {
		previousPlayers[player.CharacterID] = player
	}
	currentPlayers := map[string]PlayerState{}
	for _, player := range current.Players {
		currentPlayers[player.CharacterID] = player
		fields := changedPlayerFields(previousPlayers[player.CharacterID], player)
		if len(fields) > 0 {
			changes = append(changes, replication.ChangedFields{
				Domain:   "player",
				EntityID: player.CharacterID,
				Fields:   fields,
				Version:  version,
			})
		}
	}
	for id := range previousPlayers {
		if _, ok := currentPlayers[id]; !ok {
			changes = append(changes, replication.ChangedFields{
				Domain:   "player",
				EntityID: id,
				Fields:   []string{"removed"},
				Version:  version,
			})
		}
	}

	previousNPCs := map[string]NpcState{}
	for _, npc := range previous.NPCs {
		previousNPCs[npc.ID] = npc
	}
	currentNPCs := map[string]NpcState{}
	for _, npc := range current.NPCs {
		currentNPCs[npc.ID] = npc
		fields := changedNPCFields(previousNPCs[npc.ID], npc)
		if len(fields) > 0 {
			changes = append(changes, replication.ChangedFields{
				Domain:   "npc",
				EntityID: npc.ID,
				Fields:   fields,
				Version:  version,
			})
		}
	}
	for id := range previousNPCs {
		if _, ok := currentNPCs[id]; !ok {
			changes = append(changes, replication.ChangedFields{
				Domain:   "npc",
				EntityID: id,
				Fields:   []string{"removed"},
				Version:  version,
			})
		}
	}

	previousLoot := map[string]LootContainerState{}
	for _, container := range previous.LootContainers {
		previousLoot[container.ID] = container
	}
	currentLoot := map[string]LootContainerState{}
	for _, container := range current.LootContainers {
		currentLoot[container.ID] = container
		fields := changedLootFields(previousLoot[container.ID], container)
		if len(fields) > 0 {
			changes = append(changes, replication.ChangedFields{
				Domain:   "loot",
				EntityID: container.ID,
				Fields:   fields,
				Version:  version,
			})
		}
	}
	for id := range previousLoot {
		if _, ok := currentLoot[id]; !ok {
			changes = append(changes, replication.ChangedFields{
				Domain:   "loot",
				EntityID: id,
				Fields:   []string{"removed"},
				Version:  version,
			})
		}
	}

	return replication.NormalizeChangedFields(changes)
}

func changedPlayerFields(previous PlayerState, current PlayerState) []string {
	var fields []string
	if previous.SessionToken == "" && previous.CharacterID == "" {
		return []string{"created"}
	}
	if previous.SessionToken != current.SessionToken || previous.AccountID != current.AccountID || previous.DisplayName != current.DisplayName {
		fields = append(fields, "identity")
	}
	if previous.ZoneID != current.ZoneID {
		fields = append(fields, "zone")
	}
	if previous.Position != current.Position {
		fields = append(fields, "position")
	}
	if previous.Connected != current.Connected {
		fields = append(fields, "connection")
	}
	if previous.Health != current.Health || previous.MaxHealth != current.MaxHealth || previous.Resource != current.Resource || previous.MaxResource != current.MaxResource || previous.Alive != current.Alive {
		fields = append(fields, "vitals")
	}
	if previous.TargetID != current.TargetID || previous.AutoAttackActive != current.AutoAttackActive {
		fields = append(fields, "combat")
	}
	if !reflect.DeepEqual(previous.QuestProgress, current.QuestProgress) || !reflect.DeepEqual(previous.QuestCompleted, current.QuestCompleted) {
		fields = append(fields, "quests")
	}
	if !reflect.DeepEqual(previous.LootClaims, current.LootClaims) {
		fields = append(fields, "loot")
	}
	if !reflect.DeepEqual(previous.InventorySlots, current.InventorySlots) || previous.CurrencyCopper != current.CurrencyCopper {
		fields = append(fields, "inventory")
	}
	if !reflect.DeepEqual(previous.ActionBarSlots, current.ActionBarSlots) {
		fields = append(fields, "actionBar")
	}
	sort.Strings(fields)
	return fields
}

func changedNPCFields(previous NpcState, current NpcState) []string {
	var fields []string
	if previous.ID == "" {
		return []string{"created"}
	}
	if previous.ZoneID != current.ZoneID || previous.DisplayName != current.DisplayName || previous.Kind != current.Kind {
		fields = append(fields, "identity")
	}
	if previous.Position != current.Position {
		fields = append(fields, "position")
	}
	if previous.Health != current.Health || previous.MaxHealth != current.MaxHealth || previous.Alive != current.Alive || previous.Targetable != current.Targetable || previous.RespawnTick != current.RespawnTick {
		fields = append(fields, "vitals")
	}
	if previous.TargetID != current.TargetID || !reflect.DeepEqual(previous.Threat, current.Threat) {
		fields = append(fields, "threat")
	}
	sort.Strings(fields)
	return fields
}

func changedLootFields(previous LootContainerState, current LootContainerState) []string {
	var fields []string
	if previous.ID == "" {
		return []string{"created"}
	}
	if previous.SourceEntityID != current.SourceEntityID || previous.OwnerCharacterID != current.OwnerCharacterID {
		fields = append(fields, "ownership")
	}
	if !reflect.DeepEqual(previous.Items, current.Items) {
		fields = append(fields, "items")
	}
	if previous.OpenedByCharacterID != current.OpenedByCharacterID || previous.ClaimedByCharacterID != current.ClaimedByCharacterID || previous.ClaimedAtTick != current.ClaimedAtTick || previous.Closed != current.Closed {
		fields = append(fields, "claim")
	}
	sort.Strings(fields)
	return fields
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
