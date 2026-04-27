package loop

import "fmt"

type ConnectWorldSessionCommand struct {
	Player PlayerState
}

func (c ConnectWorldSessionCommand) Kind() CommandKind    { return CommandConnectWorldSession }
func (c ConnectWorldSessionCommand) SessionToken() string { return c.Player.SessionToken }
func (c ConnectWorldSessionCommand) ActorID() string      { return c.Player.CharacterID }
func (c ConnectWorldSessionCommand) ReplayPayload() map[string]any {
	return map[string]any{
		"characterId":       c.Player.CharacterID,
		"worldSessionToken": c.Player.SessionToken,
		"zoneId":            c.Player.ZoneID,
		"x":                 c.Player.Position.X,
		"y":                 c.Player.Position.Y,
		"z":                 c.Player.Position.Z,
	}
}
func (c ConnectWorldSessionCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	if c.Player.SessionToken == "" || c.Player.CharacterID == "" {
		return CommandResult{}, ErrSessionMissing
	}
	player := c.Player
	if player.ZoneID == "" {
		player.ZoneID = state.ZoneID
	}
	player.Connected = true
	if player.MaxHealth == 0 {
		player.MaxHealth = player.Health
	}
	if !player.Alive && player.Health > 0 {
		player.Alive = true
	}
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), nil), nil
}

type DisconnectWorldSessionCommand struct {
	Token  string
	Actor  string
	Reason string
}

func (c DisconnectWorldSessionCommand) Kind() CommandKind    { return CommandDisconnectWorldSession }
func (c DisconnectWorldSessionCommand) SessionToken() string { return c.Token }
func (c DisconnectWorldSessionCommand) ActorID() string      { return c.Actor }
func (c DisconnectWorldSessionCommand) ReplayPayload() map[string]any {
	return map[string]any{"reason": c.Reason}
}
func (c DisconnectWorldSessionCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	player.Connected = false
	player.AutoAttackActive = false
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), nil), nil
}

type ReconnectWorldSessionCommand struct {
	Player PlayerState
	Reason string
}

func (c ReconnectWorldSessionCommand) Kind() CommandKind    { return CommandReconnectWorldSession }
func (c ReconnectWorldSessionCommand) SessionToken() string { return c.Player.SessionToken }
func (c ReconnectWorldSessionCommand) ActorID() string      { return c.Player.CharacterID }
func (c ReconnectWorldSessionCommand) ReplayPayload() map[string]any {
	return map[string]any{
		"characterId":       c.Player.CharacterID,
		"worldSessionToken": c.Player.SessionToken,
		"reason":            c.Reason,
		"x":                 c.Player.Position.X,
		"y":                 c.Player.Position.Y,
		"z":                 c.Player.Position.Z,
	}
}
func (c ReconnectWorldSessionCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	if c.Player.SessionToken == "" || c.Player.CharacterID == "" {
		return CommandResult{}, ErrSessionMissing
	}
	player := c.Player
	if existing, ok := state.playersBySession[player.SessionToken]; ok {
		if player.Position == (Position{}) {
			player.Position = existing.Position
		}
		if player.ZoneID == "" {
			player.ZoneID = existing.ZoneID
		}
	}
	player.Connected = true
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), nil), nil
}

type ApplyMovementCommand struct {
	Token        string
	Actor        string
	Delta        Position
	NextPosition *Position
}

func (c ApplyMovementCommand) Kind() CommandKind    { return CommandApplyMovement }
func (c ApplyMovementCommand) SessionToken() string { return c.Token }
func (c ApplyMovementCommand) ActorID() string      { return c.Actor }
func (c ApplyMovementCommand) ReplayPayload() map[string]any {
	payload := map[string]any{"deltaX": c.Delta.X, "deltaY": c.Delta.Y, "deltaZ": c.Delta.Z}
	if c.NextPosition != nil {
		payload["x"] = c.NextPosition.X
		payload["y"] = c.NextPosition.Y
		payload["z"] = c.NextPosition.Z
	}
	return payload
}
func (c ApplyMovementCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	if c.NextPosition != nil {
		player.Position = *c.NextPosition
	} else {
		player.Position.X += c.Delta.X
		player.Position.Y += c.Delta.Y
		player.Position.Z += c.Delta.Z
	}
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), player.Position), nil
}

type SelectTargetCommand struct {
	Token    string
	Actor    string
	TargetID string
}

func (c SelectTargetCommand) Kind() CommandKind    { return CommandSelectTarget }
func (c SelectTargetCommand) SessionToken() string { return c.Token }
func (c SelectTargetCommand) ActorID() string      { return c.Actor }
func (c SelectTargetCommand) ReplayPayload() map[string]any {
	return map[string]any{"targetId": c.TargetID}
}
func (c SelectTargetCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	if c.TargetID == "" {
		player.TargetID = ""
		player.AutoAttackActive = false
		state.UpsertPlayer(player)
		return resultFor(state, context, CommandClearTarget, nil), nil
	}
	if _, ok := state.npcs[c.TargetID]; !ok {
		if _, ok := state.playerSessionByID[c.TargetID]; !ok {
			return CommandResult{}, ErrTargetMissing
		}
	}
	player.TargetID = c.TargetID
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), c.TargetID), nil
}

type ClearTargetCommand struct {
	Token string
	Actor string
}

func (c ClearTargetCommand) Kind() CommandKind    { return CommandClearTarget }
func (c ClearTargetCommand) SessionToken() string { return c.Token }
func (c ClearTargetCommand) ActorID() string      { return c.Actor }
func (c ClearTargetCommand) ReplayPayload() map[string]any {
	return map[string]any{}
}
func (c ClearTargetCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	player.TargetID = ""
	player.AutoAttackActive = false
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), nil), nil
}

type StartAutoAttackCommand struct {
	Token   string
	Actor   string
	Enabled bool
}

func (c StartAutoAttackCommand) Kind() CommandKind    { return CommandStartAutoAttack }
func (c StartAutoAttackCommand) SessionToken() string { return c.Token }
func (c StartAutoAttackCommand) ActorID() string      { return c.Actor }
func (c StartAutoAttackCommand) ReplayPayload() map[string]any {
	return map[string]any{"enabled": c.Enabled}
}
func (c StartAutoAttackCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	if c.Enabled && player.TargetID == "" {
		return CommandResult{}, fmt.Errorf("target is required")
	}
	player.AutoAttackActive = c.Enabled
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), c.Enabled), nil
}

type CastAbilityCommand struct {
	Token     string
	Actor     string
	AbilityID string
}

func (c CastAbilityCommand) Kind() CommandKind    { return CommandCastAbility }
func (c CastAbilityCommand) SessionToken() string { return c.Token }
func (c CastAbilityCommand) ActorID() string      { return c.Actor }
func (c CastAbilityCommand) ReplayPayload() map[string]any {
	return map[string]any{"abilityId": c.AbilityID}
}
func (c CastAbilityCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	if _, ok := state.playersBySession[c.Token]; !ok {
		return CommandResult{}, ErrSessionMissing
	}
	return resultFor(state, context, c.Kind(), c.AbilityID), nil
}

type AcceptQuestCommand struct {
	Token   string
	Actor   string
	QuestID string
}

func (c AcceptQuestCommand) Kind() CommandKind    { return CommandAcceptQuest }
func (c AcceptQuestCommand) SessionToken() string { return c.Token }
func (c AcceptQuestCommand) ActorID() string      { return c.Actor }
func (c AcceptQuestCommand) ReplayPayload() map[string]any {
	return map[string]any{"questId": c.QuestID}
}
func (c AcceptQuestCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	if player.QuestProgress == nil {
		player.QuestProgress = map[string]int{}
	}
	if _, exists := player.QuestProgress[c.QuestID]; !exists {
		player.QuestProgress[c.QuestID] = 0
	}
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), c.QuestID), nil
}

type ProgressQuestObjectiveCommand struct {
	Token       string
	Actor       string
	QuestID     string
	ObjectiveID string
	Delta       int
}

func (c ProgressQuestObjectiveCommand) Kind() CommandKind    { return CommandProgressQuestObjective }
func (c ProgressQuestObjectiveCommand) SessionToken() string { return c.Token }
func (c ProgressQuestObjectiveCommand) ActorID() string      { return c.Actor }
func (c ProgressQuestObjectiveCommand) ReplayPayload() map[string]any {
	return map[string]any{"questId": c.QuestID, "objectiveId": c.ObjectiveID, "delta": c.Delta}
}
func (c ProgressQuestObjectiveCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	if player.QuestProgress == nil {
		player.QuestProgress = map[string]int{}
	}
	delta := c.Delta
	if delta == 0 {
		delta = 1
	}
	player.QuestProgress[c.QuestID] += delta
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), player.QuestProgress[c.QuestID]), nil
}

type InteractNPCCommand struct {
	Token string
	Actor string
	NPCID string
}

func (c InteractNPCCommand) Kind() CommandKind    { return CommandInteractNPC }
func (c InteractNPCCommand) SessionToken() string { return c.Token }
func (c InteractNPCCommand) ActorID() string      { return c.Actor }
func (c InteractNPCCommand) ReplayPayload() map[string]any {
	return map[string]any{"npcId": c.NPCID}
}
func (c InteractNPCCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	if _, ok := state.playersBySession[c.Token]; !ok {
		return CommandResult{}, ErrSessionMissing
	}
	if _, ok := state.npcs[c.NPCID]; !ok {
		return CommandResult{}, ErrTargetMissing
	}
	return resultFor(state, context, c.Kind(), c.NPCID), nil
}

type UpdateActionBarCommand struct {
	Token     string
	Actor     string
	SlotIndex int
	AbilityID string
	Clear     bool
}

func (c UpdateActionBarCommand) Kind() CommandKind    { return CommandUpdateActionBar }
func (c UpdateActionBarCommand) SessionToken() string { return c.Token }
func (c UpdateActionBarCommand) ActorID() string      { return c.Actor }
func (c UpdateActionBarCommand) ReplayPayload() map[string]any {
	return map[string]any{"slotIndex": c.SlotIndex, "abilityId": c.AbilityID, "clear": c.Clear}
}
func (c UpdateActionBarCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	if player.ActionBarSlots == nil {
		player.ActionBarSlots = map[int]string{}
	}
	if c.Clear || c.AbilityID == "" {
		delete(player.ActionBarSlots, c.SlotIndex)
	} else {
		player.ActionBarSlots[c.SlotIndex] = c.AbilityID
	}
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), c.SlotIndex), nil
}

type MoveInventoryItemCommand struct {
	Token         string
	Actor         string
	FromSlotIndex int
	ToSlotIndex   int
}

func (c MoveInventoryItemCommand) Kind() CommandKind    { return CommandMoveInventoryItem }
func (c MoveInventoryItemCommand) SessionToken() string { return c.Token }
func (c MoveInventoryItemCommand) ActorID() string      { return c.Actor }
func (c MoveInventoryItemCommand) ReplayPayload() map[string]any {
	return map[string]any{"fromSlotIndex": c.FromSlotIndex, "toSlotIndex": c.ToSlotIndex}
}
func (c MoveInventoryItemCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	if player.InventorySlots == nil {
		player.InventorySlots = map[int]string{}
	}
	itemID := player.InventorySlots[c.FromSlotIndex]
	if itemID == "" {
		return CommandResult{}, fmt.Errorf("source slot is empty")
	}
	player.InventorySlots[c.FromSlotIndex] = player.InventorySlots[c.ToSlotIndex]
	player.InventorySlots[c.ToSlotIndex] = itemID
	if player.InventorySlots[c.FromSlotIndex] == "" {
		delete(player.InventorySlots, c.FromSlotIndex)
	}
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), c.ToSlotIndex), nil
}

type RequestSnapshotCommand struct {
	Token string
	Actor string
}

func (c RequestSnapshotCommand) Kind() CommandKind    { return CommandRequestSnapshot }
func (c RequestSnapshotCommand) SessionToken() string { return c.Token }
func (c RequestSnapshotCommand) ActorID() string      { return c.Actor }
func (c RequestSnapshotCommand) ReplayPayload() map[string]any {
	return map[string]any{}
}
func (c RequestSnapshotCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	return resultFor(state, context, c.Kind(), nil), nil
}

func resultFor(state *ShardState, context CommandContext, kind CommandKind, payload any) CommandResult {
	return CommandResult{
		CommandID: context.CommandID,
		Sequence:  context.Sequence,
		Kind:      kind,
		Tick:      context.Tick,
		Snapshot:  state.Snapshot(),
		Payload:   payload,
	}
}
