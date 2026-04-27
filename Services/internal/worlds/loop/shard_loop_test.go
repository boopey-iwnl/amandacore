package loop

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestShardLoopReplayMatchesFinalSnapshot(t *testing.T) {
	shard := NewShardLoop(ShardLoopConfig{QueueLimit: 16})
	if err := shard.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer stopLoop(t, shard)

	ctx := context.Background()
	_, err := shard.Submit(ctx, ConnectWorldSessionCommand{Player: PlayerState{
		SessionToken: "world_one",
		CharacterID:  "char_one",
		ZoneID:       StonewakeZoneID,
		Position:     Position{X: 10, Y: 10, Z: 0},
		Connected:    true,
		Alive:        true,
		Health:       88,
		MaxHealth:    88,
	}})
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if _, err := shard.Submit(ctx, ApplyMovementCommand{Token: "world_one", Actor: "char_one", Delta: Position{X: 4, Y: 2}}); err != nil {
		t.Fatalf("move failed: %v", err)
	}
	if _, err := shard.Submit(ctx, DisconnectWorldSessionCommand{Token: "world_one", Actor: "char_one", Reason: "test"}); err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}
	snapshot, err := shard.Snapshot(ctx, "world_one", "char_one")
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	replayed, err := Replay(WorldSnapshot{ShardID: StonewakeShardID, ZoneID: StonewakeZoneID}, shard.ReplayLog())
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if len(snapshot.Players) != 1 || len(replayed.Players) != 1 {
		t.Fatalf("expected one player in snapshots, got %d/%d", len(snapshot.Players), len(replayed.Players))
	}
	if snapshot.Players[0].Position != replayed.Players[0].Position {
		t.Fatalf("replay position mismatch: live=%#v replay=%#v", snapshot.Players[0].Position, replayed.Players[0].Position)
	}
	if snapshot.Players[0].Connected != replayed.Players[0].Connected {
		t.Fatalf("replay connected mismatch: live=%v replay=%v", snapshot.Players[0].Connected, replayed.Players[0].Connected)
	}
}

func TestShardLoopConcurrentMovementIsSingleWriterOrdered(t *testing.T) {
	shard := NewShardLoop(ShardLoopConfig{QueueLimit: 128})
	if err := shard.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer stopLoop(t, shard)

	ctx := context.Background()
	if _, err := shard.Submit(ctx, ConnectWorldSessionCommand{Player: PlayerState{
		SessionToken: "world_runner",
		CharacterID:  "runner",
		ZoneID:       StonewakeZoneID,
		Position:     Position{X: 0, Y: 0, Z: 0},
		Connected:    true,
		Alive:        true,
	}}); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	const moves = 40
	var wg sync.WaitGroup
	for i := 0; i < moves; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := shard.Submit(ctx, ApplyMovementCommand{Token: "world_runner", Actor: "runner", Delta: Position{X: 1}}); err != nil {
				t.Errorf("move failed: %v", err)
			}
		}()
	}
	wg.Wait()

	snapshot, err := shard.Snapshot(ctx, "world_runner", "runner")
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if got := snapshot.Players[0].Position.X; got != moves {
		t.Fatalf("expected final x %d, got %.1f", moves, got)
	}
	metrics := shard.Metrics()
	if metrics.CommandsApplied != moves+2 {
		t.Fatalf("expected %d applied commands including connect/snapshot, got %d", moves+2, metrics.CommandsApplied)
	}
}

func TestShardLoopReconnectDuringMovementPreservesPosition(t *testing.T) {
	shard := NewShardLoop(ShardLoopConfig{QueueLimit: 16})
	if err := shard.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer stopLoop(t, shard)

	ctx := context.Background()
	player := PlayerState{
		SessionToken: "world_reconnect",
		CharacterID:  "reconnect",
		ZoneID:       StonewakeZoneID,
		Position:     Position{X: 10, Y: 10},
		Connected:    true,
		Alive:        true,
	}
	if _, err := shard.Submit(ctx, ConnectWorldSessionCommand{Player: player}); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if _, err := shard.Submit(ctx, ApplyMovementCommand{Token: player.SessionToken, Actor: player.CharacterID, Delta: Position{X: 3, Y: -2}}); err != nil {
		t.Fatalf("move failed: %v", err)
	}
	player.Position = Position{X: 13, Y: 8}
	if _, err := shard.Submit(ctx, ReconnectWorldSessionCommand{Player: player, Reason: "test_reconnect"}); err != nil {
		t.Fatalf("reconnect failed: %v", err)
	}

	snapshot, err := shard.Snapshot(ctx, player.SessionToken, player.CharacterID)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if got := snapshot.Players[0].Position; got.X != 13 || got.Y != 8 {
		t.Fatalf("expected reconnect position 13,8 got %#v", got)
	}
	if !snapshot.Players[0].Connected {
		t.Fatalf("expected player connected after reconnect")
	}
}

func TestShardLoopInvalidSessionAndStopBehavior(t *testing.T) {
	shard := NewShardLoop(ShardLoopConfig{QueueLimit: 2})
	if err := shard.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if _, err := shard.Submit(context.Background(), ApplyMovementCommand{Token: "missing", Actor: "missing", Delta: Position{X: 1}}); !errors.Is(err, ErrSessionMissing) {
		t.Fatalf("expected missing session error, got %v", err)
	}

	stopLoop(t, shard)
	if _, err := shard.Submit(context.Background(), RequestSnapshotCommand{}); !errors.Is(err, ErrLoopStopped) {
		t.Fatalf("expected stopped loop error, got %v", err)
	}
}

func TestShardLoopCommandTimeout(t *testing.T) {
	shard := NewShardLoop(ShardLoopConfig{QueueLimit: 1, CommandTimeout: 10 * time.Millisecond})
	if err := shard.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer stopLoop(t, shard)

	block := make(chan struct{})
	_, err := shard.Submit(context.Background(), CommandFunc{
		CommandKind: CommandRequestSnapshot,
		ApplyCommand: func(state *ShardState, context CommandContext) (CommandResult, error) {
			<-block
			return resultFor(state, context, CommandRequestSnapshot, nil), nil
		},
	})
	if err == nil {
		t.Fatalf("expected timeout from blocking command")
	}
	close(block)
	if !errors.Is(err, ErrCommandTimeout) {
		t.Fatalf("expected command timeout, got %v", err)
	}
}

func TestShardLoopCombatQuestLootReplayIsDeterministic(t *testing.T) {
	shard := NewShardLoop(ShardLoopConfig{QueueLimit: 32})
	if err := shard.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer stopLoop(t, shard)

	ctx := context.Background()
	player := PlayerState{
		SessionToken:   "world_combat",
		CharacterID:    "combatant",
		ZoneID:         StonewakeZoneID,
		Position:       Position{X: 10, Y: 10},
		Connected:      true,
		Alive:          true,
		Health:         88,
		MaxHealth:      88,
		InventorySlots: map[int]string{},
	}
	mob := NpcState{
		ID:         "npc_training_rat",
		ZoneID:     StonewakeZoneID,
		Position:   Position{X: 12, Y: 10},
		Health:     20,
		MaxHealth:  20,
		Alive:      true,
		Targetable: true,
		Kind:       "hostile_mob",
	}

	mustSubmit(t, shard, ConnectWorldSessionCommand{Player: player})
	mustSubmit(t, shard, RespawnNPCCommand{NPC: mob})
	mustSubmit(t, shard, AcceptQuestCommand{Token: player.SessionToken, Actor: player.CharacterID, QuestID: "quest_clear_path"})
	mustSubmit(t, shard, SelectTargetCommand{Token: player.SessionToken, Actor: player.CharacterID, TargetID: mob.ID})
	mustSubmit(t, shard, StartAutoAttackCommand{Token: player.SessionToken, Actor: player.CharacterID, Enabled: true})
	mustSubmit(t, shard, UseAbilityCommand{Token: player.SessionToken, Actor: player.CharacterID, AbilityID: "steady_strike", TargetID: mob.ID, Damage: 12})
	mustSubmit(t, shard, ApplyDamageCommand{Token: player.SessionToken, Actor: player.CharacterID, SourceID: player.CharacterID, TargetID: mob.ID, Amount: 8})
	mustSubmit(t, shard, ProgressQuestObjectiveCommand{Token: player.SessionToken, Actor: player.CharacterID, QuestID: "quest_clear_path", ObjectiveID: "kill_rat", Delta: 1})
	mustSubmit(t, shard, CompleteQuestCommand{Token: player.SessionToken, Actor: player.CharacterID, QuestID: "quest_clear_path"})
	mustSubmit(t, shard, GenerateLootCommand{
		Token:       player.SessionToken,
		Actor:       player.CharacterID,
		ContainerID: "loot_rat_001",
		SourceID:    mob.ID,
		OwnerID:     player.CharacterID,
		Items:       []LootItemState{{ItemID: "rat_tail", Quantity: 1}},
	})
	mustSubmit(t, shard, ClaimLootItemCommand{Token: player.SessionToken, Actor: player.CharacterID, ContainerID: "loot_rat_001", MutationKey: "loot-rat-001"})
	mustSubmit(t, shard, ClaimQuestRewardCommand{
		Token:         player.SessionToken,
		Actor:         player.CharacterID,
		QuestID:       "quest_clear_path",
		ItemIDs:       []string{"militia_token"},
		CurrencyDelta: 5,
		MutationKey:   "quest-clear-path-reward",
	})

	snapshot, err := shard.Snapshot(ctx, player.SessionToken, player.CharacterID)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	replayed, err := Replay(WorldSnapshot{ShardID: StonewakeShardID, ZoneID: StonewakeZoneID}, shard.ReplayLog())
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	assertCombatQuestLootSnapshot(t, snapshot)
	assertCombatQuestLootSnapshot(t, replayed)
}

func TestShardLoopDuplicateLootAndQuestRewardsApplyOnce(t *testing.T) {
	shard := NewShardLoop(ShardLoopConfig{QueueLimit: 32})
	if err := shard.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer stopLoop(t, shard)

	player := PlayerState{
		SessionToken:   "world_loot_retry",
		CharacterID:    "looter",
		ZoneID:         StonewakeZoneID,
		Connected:      true,
		Alive:          true,
		InventorySlots: map[int]string{},
	}
	mustSubmit(t, shard, ConnectWorldSessionCommand{Player: player})
	mustSubmit(t, shard, GenerateLootCommand{
		Token:       player.SessionToken,
		Actor:       player.CharacterID,
		ContainerID: "loot_retry",
		SourceID:    "npc_retry",
		OwnerID:     player.CharacterID,
		Items:       []LootItemState{{ItemID: "retry_cloth", Quantity: 1}},
	})

	var wg sync.WaitGroup
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := shard.Submit(context.Background(), ClaimLootItemCommand{
				Token:       player.SessionToken,
				Actor:       player.CharacterID,
				ContainerID: "loot_retry",
				MutationKey: "claim-once",
			}); err != nil {
				t.Errorf("loot claim failed: %v", err)
			}
		}()
	}
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := shard.Submit(context.Background(), ClaimQuestRewardCommand{
				Token:         player.SessionToken,
				Actor:         player.CharacterID,
				QuestID:       "quest_retry",
				ItemIDs:       []string{"quest_badge"},
				CurrencyDelta: 7,
				MutationKey:   "quest-once",
			}); err != nil {
				t.Errorf("quest reward failed: %v", err)
			}
		}()
	}
	wg.Wait()

	snapshot, err := shard.Snapshot(context.Background(), player.SessionToken, player.CharacterID)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if len(snapshot.Players) != 1 {
		t.Fatalf("expected one player, got %d", len(snapshot.Players))
	}
	gotItems := countItems(snapshot.Players[0].InventorySlots)
	if gotItems["retry_cloth"] != 1 {
		t.Fatalf("expected one retry_cloth, got inventory %#v", snapshot.Players[0].InventorySlots)
	}
	if gotItems["quest_badge"] != 1 {
		t.Fatalf("expected one quest_badge, got inventory %#v", snapshot.Players[0].InventorySlots)
	}
	if snapshot.Players[0].CurrencyCopper != 7 {
		t.Fatalf("expected currency reward once, got %d", snapshot.Players[0].CurrencyCopper)
	}
}

func stopLoop(t *testing.T, shard *ShardLoop) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := shard.Stop(ctx); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
}

func mustSubmit(t *testing.T, shard *ShardLoop, command WorldCommand) CommandResult {
	t.Helper()
	result, err := shard.Submit(context.Background(), command)
	if err != nil {
		t.Fatalf("%s failed: %v", command.Kind(), err)
	}
	return result
}

func assertCombatQuestLootSnapshot(t *testing.T, snapshot WorldSnapshot) {
	t.Helper()
	if len(snapshot.Players) != 1 {
		t.Fatalf("expected one player, got %d", len(snapshot.Players))
	}
	player := snapshot.Players[0]
	if player.QuestProgress["quest_clear_path"] != 1 {
		t.Fatalf("expected quest kill count 1, got %#v", player.QuestProgress)
	}
	if !player.QuestCompleted["quest_clear_path"] {
		t.Fatalf("expected quest completed flag, got %#v", player.QuestCompleted)
	}
	if player.CurrencyCopper != 5 {
		t.Fatalf("expected 5 copper reward, got %d", player.CurrencyCopper)
	}
	items := countItems(player.InventorySlots)
	if items["rat_tail"] != 1 || items["militia_token"] != 1 {
		t.Fatalf("expected loot and reward items, got %#v", player.InventorySlots)
	}
	if len(snapshot.NPCs) != 1 || snapshot.NPCs[0].Alive {
		t.Fatalf("expected defeated NPC in snapshot, got %#v", snapshot.NPCs)
	}
	if len(snapshot.LootContainers) != 1 || snapshot.LootContainers[0].ClaimedByCharacterID != player.CharacterID {
		t.Fatalf("expected claimed loot container, got %#v", snapshot.LootContainers)
	}
}

func countItems(slots map[int]string) map[string]int {
	counts := map[string]int{}
	for _, itemID := range slots {
		if itemID != "" {
			counts[itemID]++
		}
	}
	return counts
}
