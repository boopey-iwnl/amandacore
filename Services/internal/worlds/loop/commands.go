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

type StopAutoAttackCommand struct {
	Token  string
	Actor  string
	Reason string
}

func (c StopAutoAttackCommand) Kind() CommandKind    { return CommandStopAutoAttack }
func (c StopAutoAttackCommand) SessionToken() string { return c.Token }
func (c StopAutoAttackCommand) ActorID() string      { return c.Actor }
func (c StopAutoAttackCommand) ReplayPayload() map[string]any {
	return map[string]any{"reason": c.Reason}
}
func (c StopAutoAttackCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	player.AutoAttackActive = false
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), c.Reason), nil
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

type UseAbilityCommand struct {
	Token        string
	Actor        string
	AbilityID    string
	TargetID     string
	Damage       float64
	Heal         float64
	Threat       float64
	ResourceCost float64
}

func (c UseAbilityCommand) Kind() CommandKind    { return CommandUseAbility }
func (c UseAbilityCommand) SessionToken() string { return c.Token }
func (c UseAbilityCommand) ActorID() string      { return c.Actor }
func (c UseAbilityCommand) ReplayPayload() map[string]any {
	return map[string]any{
		"abilityId":    c.AbilityID,
		"targetId":     c.TargetID,
		"damage":       c.Damage,
		"heal":         c.Heal,
		"threat":       c.Threat,
		"resourceCost": c.ResourceCost,
	}
}
func (c UseAbilityCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	targetID := c.TargetID
	if targetID == "" {
		targetID = player.TargetID
	}
	if c.ResourceCost > 0 {
		if player.Resource < c.ResourceCost {
			return CommandResult{}, fmt.Errorf("resource is too low")
		}
		player.Resource -= c.ResourceCost
	}
	if c.Heal > 0 {
		player.Health = minNumber(player.MaxHealth, player.Health+c.Heal)
	}
	if c.Damage > 0 {
		if targetID == "" {
			return CommandResult{}, ErrTargetMissing
		}
		if err := applyDamage(state, player.CharacterID, targetID, c.Damage, c.Threat, context.Tick); err != nil {
			return CommandResult{}, err
		}
	}
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), map[string]any{"abilityId": c.AbilityID, "targetId": targetID}), nil
}

type CancelCastCommand struct {
	Token  string
	Actor  string
	Reason string
}

func (c CancelCastCommand) Kind() CommandKind    { return CommandCancelCast }
func (c CancelCastCommand) SessionToken() string { return c.Token }
func (c CancelCastCommand) ActorID() string      { return c.Actor }
func (c CancelCastCommand) ReplayPayload() map[string]any {
	return map[string]any{"reason": c.Reason}
}
func (c CancelCastCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	if _, ok := state.playersBySession[c.Token]; !ok {
		return CommandResult{}, ErrSessionMissing
	}
	return resultFor(state, context, c.Kind(), c.Reason), nil
}

type ApplyDamageCommand struct {
	Token    string
	Actor    string
	SourceID string
	TargetID string
	Amount   float64
	Threat   float64
	Reason   string
}

func (c ApplyDamageCommand) Kind() CommandKind    { return CommandApplyDamage }
func (c ApplyDamageCommand) SessionToken() string { return c.Token }
func (c ApplyDamageCommand) ActorID() string      { return c.Actor }
func (c ApplyDamageCommand) ReplayPayload() map[string]any {
	return map[string]any{"sourceId": c.SourceID, "targetId": c.TargetID, "amount": c.Amount, "threat": c.Threat, "reason": c.Reason}
}
func (c ApplyDamageCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	sourceID := c.SourceID
	if sourceID == "" {
		sourceID = c.Actor
	}
	if sourceID == "" {
		if player, ok := state.playersBySession[c.Token]; ok {
			sourceID = player.CharacterID
		}
	}
	if err := applyDamage(state, sourceID, c.TargetID, c.Amount, c.Threat, context.Tick); err != nil {
		return CommandResult{}, err
	}
	return resultFor(state, context, c.Kind(), c.TargetID), nil
}

type ApplyHealCommand struct {
	Token    string
	Actor    string
	TargetID string
	Amount   float64
	Reason   string
}

func (c ApplyHealCommand) Kind() CommandKind    { return CommandApplyHeal }
func (c ApplyHealCommand) SessionToken() string { return c.Token }
func (c ApplyHealCommand) ActorID() string      { return c.Actor }
func (c ApplyHealCommand) ReplayPayload() map[string]any {
	return map[string]any{"targetId": c.TargetID, "amount": c.Amount, "reason": c.Reason}
}
func (c ApplyHealCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	if c.Amount <= 0 {
		return resultFor(state, context, c.Kind(), c.TargetID), nil
	}
	targetID := c.TargetID
	if targetID == "" {
		targetID = c.Actor
	}
	if token, ok := state.playerSessionByID[targetID]; ok {
		player := state.playersBySession[token]
		player.Health = minNumber(player.MaxHealth, player.Health+c.Amount)
		if player.Health > 0 {
			player.Alive = true
		}
		state.UpsertPlayer(player)
		return resultFor(state, context, c.Kind(), targetID), nil
	}
	npc, ok := state.npcs[targetID]
	if !ok {
		return CommandResult{}, ErrTargetMissing
	}
	npc.Health = minNumber(npc.MaxHealth, npc.Health+c.Amount)
	if npc.Health > 0 {
		npc.Alive = true
		npc.Targetable = true
	}
	state.UpsertNPC(npc)
	return resultFor(state, context, c.Kind(), targetID), nil
}

type ResolveDeathCommand struct {
	Token       string
	Actor       string
	EntityID    string
	KilledByID  string
	RespawnTick uint64
}

func (c ResolveDeathCommand) Kind() CommandKind    { return CommandResolveDeath }
func (c ResolveDeathCommand) SessionToken() string { return c.Token }
func (c ResolveDeathCommand) ActorID() string      { return c.Actor }
func (c ResolveDeathCommand) ReplayPayload() map[string]any {
	return map[string]any{"entityId": c.EntityID, "killedById": c.KilledByID, "respawnTick": c.RespawnTick}
}
func (c ResolveDeathCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	if err := resolveDeath(state, c.EntityID, c.KilledByID, c.RespawnTick); err != nil {
		return CommandResult{}, err
	}
	return resultFor(state, context, c.Kind(), c.EntityID), nil
}

type RespawnNPCCommand struct {
	Token string
	Actor string
	NPC   NpcState
}

func (c RespawnNPCCommand) Kind() CommandKind    { return CommandRespawnNPC }
func (c RespawnNPCCommand) SessionToken() string { return c.Token }
func (c RespawnNPCCommand) ActorID() string      { return c.Actor }
func (c RespawnNPCCommand) ReplayPayload() map[string]any {
	return map[string]any{
		"npcId":       c.NPC.ID,
		"zoneId":      c.NPC.ZoneID,
		"x":           c.NPC.Position.X,
		"y":           c.NPC.Position.Y,
		"z":           c.NPC.Position.Z,
		"health":      c.NPC.Health,
		"maxHealth":   c.NPC.MaxHealth,
		"displayName": c.NPC.DisplayName,
		"kind":        c.NPC.Kind,
	}
}
func (c RespawnNPCCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	npc := c.NPC
	if npc.ID == "" {
		return CommandResult{}, ErrTargetMissing
	}
	if npc.MaxHealth == 0 {
		npc.MaxHealth = npc.Health
	}
	if npc.Health == 0 && npc.MaxHealth > 0 {
		npc.Health = npc.MaxHealth
	}
	npc.Alive = true
	npc.Targetable = true
	npc.Threat = nil
	npc.RespawnTick = context.Tick
	state.UpsertNPC(npc)
	return resultFor(state, context, c.Kind(), npc.ID), nil
}

type ScheduleRespawnCommand struct {
	Token       string
	Actor       string
	NPCID       string
	RespawnTick uint64
}

func (c ScheduleRespawnCommand) Kind() CommandKind    { return CommandScheduleRespawn }
func (c ScheduleRespawnCommand) SessionToken() string { return c.Token }
func (c ScheduleRespawnCommand) ActorID() string      { return c.Actor }
func (c ScheduleRespawnCommand) ReplayPayload() map[string]any {
	return map[string]any{"npcId": c.NPCID, "respawnTick": c.RespawnTick}
}
func (c ScheduleRespawnCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	npc, ok := state.npcs[c.NPCID]
	if !ok {
		return CommandResult{}, ErrTargetMissing
	}
	npc.RespawnTick = c.RespawnTick
	state.UpsertNPC(npc)
	return resultFor(state, context, c.Kind(), c.NPCID), nil
}

type RequestCombatSnapshotCommand struct {
	Token string
	Actor string
}

func (c RequestCombatSnapshotCommand) Kind() CommandKind    { return CommandRequestCombatSnapshot }
func (c RequestCombatSnapshotCommand) SessionToken() string { return c.Token }
func (c RequestCombatSnapshotCommand) ActorID() string      { return c.Actor }
func (c RequestCombatSnapshotCommand) ReplayPayload() map[string]any {
	return map[string]any{}
}
func (c RequestCombatSnapshotCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	return resultFor(state, context, c.Kind(), nil), nil
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

type AbandonQuestCommand struct {
	Token   string
	Actor   string
	QuestID string
}

func (c AbandonQuestCommand) Kind() CommandKind    { return CommandAbandonQuest }
func (c AbandonQuestCommand) SessionToken() string { return c.Token }
func (c AbandonQuestCommand) ActorID() string      { return c.Actor }
func (c AbandonQuestCommand) ReplayPayload() map[string]any {
	return map[string]any{"questId": c.QuestID}
}
func (c AbandonQuestCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	delete(player.QuestProgress, c.QuestID)
	delete(player.QuestCompleted, c.QuestID)
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

type CompleteQuestCommand struct {
	Token   string
	Actor   string
	QuestID string
}

func (c CompleteQuestCommand) Kind() CommandKind    { return CommandCompleteQuest }
func (c CompleteQuestCommand) SessionToken() string { return c.Token }
func (c CompleteQuestCommand) ActorID() string      { return c.Actor }
func (c CompleteQuestCommand) ReplayPayload() map[string]any {
	return map[string]any{"questId": c.QuestID}
}
func (c CompleteQuestCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	if player.QuestCompleted == nil {
		player.QuestCompleted = map[string]bool{}
	}
	player.QuestCompleted[c.QuestID] = true
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), c.QuestID), nil
}

type ClaimQuestRewardCommand struct {
	Token         string
	Actor         string
	QuestID       string
	ItemIDs       []string
	CurrencyDelta int
	MutationKey   string
}

func (c ClaimQuestRewardCommand) Kind() CommandKind    { return CommandClaimQuestReward }
func (c ClaimQuestRewardCommand) SessionToken() string { return c.Token }
func (c ClaimQuestRewardCommand) ActorID() string      { return c.Actor }
func (c ClaimQuestRewardCommand) ReplayPayload() map[string]any {
	return map[string]any{"questId": c.QuestID, "itemIds": append([]string(nil), c.ItemIDs...), "currencyDelta": c.CurrencyDelta, "mutationKey": c.MutationKey}
}
func (c ClaimQuestRewardCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	key := c.MutationKey
	if key == "" {
		key = "quest:" + c.QuestID
	}
	if player.LootClaims == nil {
		player.LootClaims = map[string]bool{}
	}
	if player.LootClaims[key] {
		return resultFor(state, context, c.Kind(), map[string]any{"questId": c.QuestID, "replayed": true}), nil
	}
	if player.QuestCompleted == nil {
		player.QuestCompleted = map[string]bool{}
	}
	player.QuestCompleted[c.QuestID] = true
	player.CurrencyCopper += c.CurrencyDelta
	if player.InventorySlots == nil {
		player.InventorySlots = map[int]string{}
	}
	for _, itemID := range c.ItemIDs {
		if itemID == "" {
			continue
		}
		slot := firstEmptySlot(player.InventorySlots)
		player.InventorySlots[slot] = itemID
	}
	player.LootClaims[key] = true
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), c.QuestID), nil
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

type GenerateLootCommand struct {
	Token       string
	Actor       string
	ContainerID string
	SourceID    string
	OwnerID     string
	Items       []LootItemState
}

func (c GenerateLootCommand) Kind() CommandKind    { return CommandGenerateLoot }
func (c GenerateLootCommand) SessionToken() string { return c.Token }
func (c GenerateLootCommand) ActorID() string      { return c.Actor }
func (c GenerateLootCommand) ReplayPayload() map[string]any {
	return map[string]any{"containerId": c.ContainerID, "sourceId": c.SourceID, "ownerId": c.OwnerID, "items": cloneLootItems(c.Items)}
}
func (c GenerateLootCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	if c.ContainerID == "" {
		return CommandResult{}, fmt.Errorf("loot container is required")
	}
	state.UpsertLootContainer(LootContainerState{
		ID:               c.ContainerID,
		SourceEntityID:   c.SourceID,
		OwnerCharacterID: c.OwnerID,
		Items:            cloneLootItems(c.Items),
	})
	return resultFor(state, context, c.Kind(), c.ContainerID), nil
}

type OpenLootCommand struct {
	Token       string
	Actor       string
	ContainerID string
}

func (c OpenLootCommand) Kind() CommandKind    { return CommandOpenLoot }
func (c OpenLootCommand) SessionToken() string { return c.Token }
func (c OpenLootCommand) ActorID() string      { return c.Actor }
func (c OpenLootCommand) ReplayPayload() map[string]any {
	return map[string]any{"containerId": c.ContainerID}
}
func (c OpenLootCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	container, ok := state.lootContainers[c.ContainerID]
	if !ok {
		return CommandResult{}, fmt.Errorf("loot is missing")
	}
	if container.OwnerCharacterID != "" && container.OwnerCharacterID != player.CharacterID {
		return CommandResult{}, fmt.Errorf("not loot owner")
	}
	container.OpenedByCharacterID = player.CharacterID
	state.UpsertLootContainer(container)
	return resultFor(state, context, c.Kind(), c.ContainerID), nil
}

type ClaimLootItemCommand struct {
	Token       string
	Actor       string
	ContainerID string
	ItemID      string
	MutationKey string
}

func (c ClaimLootItemCommand) Kind() CommandKind    { return CommandClaimLootItem }
func (c ClaimLootItemCommand) SessionToken() string { return c.Token }
func (c ClaimLootItemCommand) ActorID() string      { return c.Actor }
func (c ClaimLootItemCommand) ReplayPayload() map[string]any {
	return map[string]any{"containerId": c.ContainerID, "itemId": c.ItemID, "mutationKey": c.MutationKey}
}
func (c ClaimLootItemCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	container, ok := state.lootContainers[c.ContainerID]
	if !ok {
		return CommandResult{}, fmt.Errorf("loot is missing")
	}
	if container.OwnerCharacterID != "" && container.OwnerCharacterID != player.CharacterID {
		return CommandResult{}, fmt.Errorf("not loot owner")
	}
	key := c.MutationKey
	if key == "" {
		key = "loot:" + c.ContainerID
	}
	if player.LootClaims == nil {
		player.LootClaims = map[string]bool{}
	}
	if player.LootClaims[key] || container.ClaimedAtTick != 0 {
		return resultFor(state, context, c.Kind(), map[string]any{"containerId": c.ContainerID, "replayed": true}), nil
	}
	if player.InventorySlots == nil {
		player.InventorySlots = map[int]string{}
	}
	for _, item := range container.Items {
		if c.ItemID != "" && item.ItemID != c.ItemID {
			continue
		}
		for quantity := 0; quantity < maxInt(1, item.Quantity); quantity++ {
			player.InventorySlots[firstEmptySlot(player.InventorySlots)] = item.ItemID
		}
	}
	player.LootClaims[key] = true
	container.ClaimedByCharacterID = player.CharacterID
	container.ClaimedAtTick = context.Tick
	state.UpsertPlayer(player)
	state.UpsertLootContainer(container)
	return resultFor(state, context, c.Kind(), c.ContainerID), nil
}

type ClaimCurrencyRewardCommand struct {
	Token       string
	Actor       string
	Amount      int
	MutationKey string
}

func (c ClaimCurrencyRewardCommand) Kind() CommandKind    { return CommandClaimCurrencyReward }
func (c ClaimCurrencyRewardCommand) SessionToken() string { return c.Token }
func (c ClaimCurrencyRewardCommand) ActorID() string      { return c.Actor }
func (c ClaimCurrencyRewardCommand) ReplayPayload() map[string]any {
	return map[string]any{"amount": c.Amount, "mutationKey": c.MutationKey}
}
func (c ClaimCurrencyRewardCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	return ApplyCurrencyDeltaCommand{Token: c.Token, Actor: c.Actor, Amount: c.Amount, MutationKey: c.MutationKey}.Apply(state, context)
}

type CloseLootCommand struct {
	Token       string
	Actor       string
	ContainerID string
}

func (c CloseLootCommand) Kind() CommandKind    { return CommandCloseLoot }
func (c CloseLootCommand) SessionToken() string { return c.Token }
func (c CloseLootCommand) ActorID() string      { return c.Actor }
func (c CloseLootCommand) ReplayPayload() map[string]any {
	return map[string]any{"containerId": c.ContainerID}
}
func (c CloseLootCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	container, ok := state.lootContainers[c.ContainerID]
	if !ok {
		return CommandResult{}, fmt.Errorf("loot is missing")
	}
	container.Closed = true
	state.UpsertLootContainer(container)
	return resultFor(state, context, c.Kind(), c.ContainerID), nil
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

type ApplyQuestRewardCommand struct {
	Token         string
	Actor         string
	QuestID       string
	ItemIDs       []string
	CurrencyDelta int
	MutationKey   string
}

func (c ApplyQuestRewardCommand) Kind() CommandKind    { return CommandApplyQuestReward }
func (c ApplyQuestRewardCommand) SessionToken() string { return c.Token }
func (c ApplyQuestRewardCommand) ActorID() string      { return c.Actor }
func (c ApplyQuestRewardCommand) ReplayPayload() map[string]any {
	return map[string]any{"questId": c.QuestID, "itemIds": append([]string(nil), c.ItemIDs...), "currencyDelta": c.CurrencyDelta, "mutationKey": c.MutationKey}
}
func (c ApplyQuestRewardCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	return ClaimQuestRewardCommand{
		Token:         c.Token,
		Actor:         c.Actor,
		QuestID:       c.QuestID,
		ItemIDs:       c.ItemIDs,
		CurrencyDelta: c.CurrencyDelta,
		MutationKey:   c.MutationKey,
	}.Apply(state, context)
}

type ApplyKillLootCommand struct {
	Token       string
	Actor       string
	ContainerID string
	MutationKey string
}

func (c ApplyKillLootCommand) Kind() CommandKind    { return CommandApplyKillLoot }
func (c ApplyKillLootCommand) SessionToken() string { return c.Token }
func (c ApplyKillLootCommand) ActorID() string      { return c.Actor }
func (c ApplyKillLootCommand) ReplayPayload() map[string]any {
	return map[string]any{"containerId": c.ContainerID, "mutationKey": c.MutationKey}
}
func (c ApplyKillLootCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	return ClaimLootItemCommand{Token: c.Token, Actor: c.Actor, ContainerID: c.ContainerID, MutationKey: c.MutationKey}.Apply(state, context)
}

type ApplyCurrencyDeltaCommand struct {
	Token       string
	Actor       string
	Amount      int
	MutationKey string
}

func (c ApplyCurrencyDeltaCommand) Kind() CommandKind    { return CommandApplyCurrencyDelta }
func (c ApplyCurrencyDeltaCommand) SessionToken() string { return c.Token }
func (c ApplyCurrencyDeltaCommand) ActorID() string      { return c.Actor }
func (c ApplyCurrencyDeltaCommand) ReplayPayload() map[string]any {
	return map[string]any{"amount": c.Amount, "mutationKey": c.MutationKey}
}
func (c ApplyCurrencyDeltaCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	key := c.MutationKey
	if key != "" {
		if player.LootClaims == nil {
			player.LootClaims = map[string]bool{}
		}
		if player.LootClaims[key] {
			return resultFor(state, context, c.Kind(), map[string]any{"replayed": true}), nil
		}
		player.LootClaims[key] = true
	}
	player.CurrencyCopper += c.Amount
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), player.CurrencyCopper), nil
}

type ApplyItemGrantCommand struct {
	Token       string
	Actor       string
	ItemID      string
	Quantity    int
	MutationKey string
}

func (c ApplyItemGrantCommand) Kind() CommandKind    { return CommandApplyItemGrant }
func (c ApplyItemGrantCommand) SessionToken() string { return c.Token }
func (c ApplyItemGrantCommand) ActorID() string      { return c.Actor }
func (c ApplyItemGrantCommand) ReplayPayload() map[string]any {
	return map[string]any{"itemId": c.ItemID, "quantity": c.Quantity, "mutationKey": c.MutationKey}
}
func (c ApplyItemGrantCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	player, ok := state.playersBySession[c.Token]
	if !ok {
		return CommandResult{}, ErrSessionMissing
	}
	key := c.MutationKey
	if key != "" {
		if player.LootClaims == nil {
			player.LootClaims = map[string]bool{}
		}
		if player.LootClaims[key] {
			return resultFor(state, context, c.Kind(), map[string]any{"replayed": true}), nil
		}
		player.LootClaims[key] = true
	}
	if player.InventorySlots == nil {
		player.InventorySlots = map[int]string{}
	}
	quantity := maxInt(1, c.Quantity)
	for index := 0; index < quantity; index++ {
		player.InventorySlots[firstEmptySlot(player.InventorySlots)] = c.ItemID
	}
	state.UpsertPlayer(player)
	return resultFor(state, context, c.Kind(), c.ItemID), nil
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

type AddThreatCommand struct {
	Token    string
	Actor    string
	NPCID    string
	TargetID string
	Amount   float64
}

func (c AddThreatCommand) Kind() CommandKind    { return CommandAddThreat }
func (c AddThreatCommand) SessionToken() string { return c.Token }
func (c AddThreatCommand) ActorID() string      { return c.Actor }
func (c AddThreatCommand) ReplayPayload() map[string]any {
	return map[string]any{"npcId": c.NPCID, "targetId": c.TargetID, "amount": c.Amount}
}
func (c AddThreatCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	if err := addThreat(state, c.NPCID, c.TargetID, c.Amount); err != nil {
		return CommandResult{}, err
	}
	return resultFor(state, context, c.Kind(), c.NPCID), nil
}

type DecayThreatCommand struct {
	Token  string
	Actor  string
	NPCID  string
	Amount float64
}

func (c DecayThreatCommand) Kind() CommandKind    { return CommandDecayThreat }
func (c DecayThreatCommand) SessionToken() string { return c.Token }
func (c DecayThreatCommand) ActorID() string      { return c.Actor }
func (c DecayThreatCommand) ReplayPayload() map[string]any {
	return map[string]any{"npcId": c.NPCID, "amount": c.Amount}
}
func (c DecayThreatCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	npc, ok := state.npcs[c.NPCID]
	if !ok {
		return CommandResult{}, ErrTargetMissing
	}
	for targetID, value := range npc.Threat {
		next := value - c.Amount
		if next <= 0 {
			delete(npc.Threat, targetID)
		} else {
			npc.Threat[targetID] = next
		}
	}
	npc.TargetID = highestThreatTarget(npc.Threat)
	state.UpsertNPC(npc)
	return resultFor(state, context, c.Kind(), c.NPCID), nil
}

type ResetThreatCommand struct {
	Token string
	Actor string
	NPCID string
}

func (c ResetThreatCommand) Kind() CommandKind    { return CommandResetThreat }
func (c ResetThreatCommand) SessionToken() string { return c.Token }
func (c ResetThreatCommand) ActorID() string      { return c.Actor }
func (c ResetThreatCommand) ReplayPayload() map[string]any {
	return map[string]any{"npcId": c.NPCID}
}
func (c ResetThreatCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	npc, ok := state.npcs[c.NPCID]
	if !ok {
		return CommandResult{}, ErrTargetMissing
	}
	npc.Threat = nil
	npc.TargetID = ""
	state.UpsertNPC(npc)
	return resultFor(state, context, c.Kind(), c.NPCID), nil
}

type SelectNPCTargetCommand struct {
	Token string
	Actor string
	NPCID string
}

func (c SelectNPCTargetCommand) Kind() CommandKind    { return CommandSelectNPCTarget }
func (c SelectNPCTargetCommand) SessionToken() string { return c.Token }
func (c SelectNPCTargetCommand) ActorID() string      { return c.Actor }
func (c SelectNPCTargetCommand) ReplayPayload() map[string]any {
	return map[string]any{"npcId": c.NPCID}
}
func (c SelectNPCTargetCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	npc, ok := state.npcs[c.NPCID]
	if !ok {
		return CommandResult{}, ErrTargetMissing
	}
	npc.TargetID = highestThreatTarget(npc.Threat)
	state.UpsertNPC(npc)
	return resultFor(state, context, c.Kind(), npc.TargetID), nil
}

type ClearThreatOnDeathCommand struct {
	Token    string
	Actor    string
	EntityID string
}

func (c ClearThreatOnDeathCommand) Kind() CommandKind    { return CommandClearThreatOnDeath }
func (c ClearThreatOnDeathCommand) SessionToken() string { return c.Token }
func (c ClearThreatOnDeathCommand) ActorID() string      { return c.Actor }
func (c ClearThreatOnDeathCommand) ReplayPayload() map[string]any {
	return map[string]any{"entityId": c.EntityID}
}
func (c ClearThreatOnDeathCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	clearThreatForEntity(state, c.EntityID)
	return resultFor(state, context, c.Kind(), c.EntityID), nil
}

type ClearThreatOnLeashCommand struct {
	Token string
	Actor string
	NPCID string
}

func (c ClearThreatOnLeashCommand) Kind() CommandKind    { return CommandClearThreatOnLeash }
func (c ClearThreatOnLeashCommand) SessionToken() string { return c.Token }
func (c ClearThreatOnLeashCommand) ActorID() string      { return c.Actor }
func (c ClearThreatOnLeashCommand) ReplayPayload() map[string]any {
	return map[string]any{"npcId": c.NPCID}
}
func (c ClearThreatOnLeashCommand) Apply(state *ShardState, context CommandContext) (CommandResult, error) {
	return ResetThreatCommand{Token: c.Token, Actor: c.Actor, NPCID: c.NPCID}.Apply(state, context)
}

func applyDamage(state *ShardState, sourceID string, targetID string, amount float64, threat float64, tick uint64) error {
	if amount <= 0 {
		return nil
	}
	if token, ok := state.playerSessionByID[targetID]; ok {
		player := state.playersBySession[token]
		player.Health = maxNumber(0, player.Health-amount)
		if player.Health <= 0 {
			player.Alive = false
			player.AutoAttackActive = false
			clearThreatForEntity(state, player.CharacterID)
		}
		state.UpsertPlayer(player)
		return nil
	}
	npc, ok := state.npcs[targetID]
	if !ok {
		return ErrTargetMissing
	}
	npc.Health = maxNumber(0, npc.Health-amount)
	if sourceID != "" {
		if npc.Threat == nil {
			npc.Threat = map[string]float64{}
		}
		threatAmount := threat
		if threatAmount <= 0 {
			threatAmount = amount
		}
		npc.Threat[sourceID] += threatAmount
		npc.TargetID = highestThreatTarget(npc.Threat)
	}
	if npc.Health <= 0 {
		npc.Alive = false
		npc.Targetable = false
		npc.TargetID = ""
		npc.Threat = nil
		npc.RespawnTick = tick
	}
	state.UpsertNPC(npc)
	return nil
}

func resolveDeath(state *ShardState, entityID string, killedByID string, respawnTick uint64) error {
	if token, ok := state.playerSessionByID[entityID]; ok {
		player := state.playersBySession[token]
		player.Health = 0
		player.Alive = false
		player.AutoAttackActive = false
		state.UpsertPlayer(player)
		clearThreatForEntity(state, entityID)
		return nil
	}
	npc, ok := state.npcs[entityID]
	if !ok {
		return ErrTargetMissing
	}
	npc.Health = 0
	npc.Alive = false
	npc.Targetable = false
	npc.TargetID = ""
	npc.Threat = nil
	npc.RespawnTick = respawnTick
	_ = killedByID
	state.UpsertNPC(npc)
	return nil
}

func addThreat(state *ShardState, npcID string, targetID string, amount float64) error {
	npc, ok := state.npcs[npcID]
	if !ok {
		return ErrTargetMissing
	}
	if targetID == "" {
		return ErrSessionMissing
	}
	if amount <= 0 {
		return nil
	}
	if npc.Threat == nil {
		npc.Threat = map[string]float64{}
	}
	npc.Threat[targetID] += amount
	npc.TargetID = highestThreatTarget(npc.Threat)
	state.UpsertNPC(npc)
	return nil
}

func clearThreatForEntity(state *ShardState, entityID string) {
	if entityID == "" {
		return
	}
	for npcID, npc := range state.npcs {
		if npc.Threat != nil {
			delete(npc.Threat, entityID)
		}
		if npc.TargetID == entityID {
			npc.TargetID = highestThreatTarget(npc.Threat)
		}
		state.npcs[npcID] = npc
	}
}

func highestThreatTarget(threat map[string]float64) string {
	bestID := ""
	bestThreat := 0.0
	for targetID, value := range threat {
		if value > bestThreat || (value == bestThreat && targetID < bestID) {
			bestID = targetID
			bestThreat = value
		}
	}
	return bestID
}

func firstEmptySlot(slots map[int]string) int {
	for index := 0; ; index++ {
		if slots[index] == "" {
			return index
		}
	}
}

func minNumber(left float64, right float64) float64 {
	if left == 0 {
		return right
	}
	if left < right {
		return left
	}
	return right
}

func maxNumber(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
