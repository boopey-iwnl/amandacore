package sqlstore

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func TestTransactionalInventoryMoveSwapAndRollback(t *testing.T) {
	store := newTestStore(t)
	character := createTransactionalStateCharacter(t, store, "InventoryRunner")

	updated, err := store.MoveInventorySlot(character.ID, 0, 5, filestore.MutationOptions{MutationKey: "move-ration-to-5"})
	if err != nil {
		t.Fatalf("failed to move inventory slot: %v", err)
	}
	if updated.Inventory[0].ItemID != "" || updated.Inventory[5].ItemID != "camp_ration" {
		t.Fatalf("expected ration moved to slot 5, got slot0=%#v slot5=%#v", updated.Inventory[0], updated.Inventory[5])
	}

	updated, err = store.MoveInventorySlot(character.ID, 5, 1, filestore.MutationOptions{MutationKey: "swap-ration-linen"})
	if err != nil {
		t.Fatalf("failed to swap occupied inventory slots: %v", err)
	}
	if updated.Inventory[1].ItemID != "camp_ration" || updated.Inventory[5].ItemID != "linen_wrap" {
		t.Fatalf("expected occupied-slot swap, got slot1=%#v slot5=%#v", updated.Inventory[1], updated.Inventory[5])
	}

	if _, err := store.MoveInventorySlot(character.ID, 0, 2, filestore.MutationOptions{MutationKey: "invalid-empty-source"}); !errors.Is(err, filestore.ErrInvalidInventoryMove) {
		t.Fatalf("expected invalid inventory move error, got %v", err)
	}
	reloaded, err := store.GetCharacterByID(character.ID)
	if err != nil {
		t.Fatalf("failed to reload character: %v", err)
	}
	if reloaded.Inventory[1].ItemID != "camp_ration" || reloaded.Inventory[5].ItemID != "linen_wrap" {
		t.Fatalf("invalid move should roll back state, got slot1=%#v slot5=%#v", reloaded.Inventory[1], reloaded.Inventory[5])
	}
}

func TestTransactionalQuestRewardIdempotencyAndRollback(t *testing.T) {
	store := newTestStore(t)
	blocked := createTransactionalStateCharacter(t, store, "BlockedReward")

	fullInventory := make([]platform.CharacterInventorySlot, platform.InventorySlotCount)
	for index := range fullInventory {
		fullInventory[index] = platform.CharacterInventorySlot{
			SlotIndex:   index,
			ItemID:      fmt.Sprintf("filler_%02d", index),
			DisplayName: "Filler",
			StackCount:  1,
		}
	}
	if _, err := store.UpdateCharacterInventory(blocked.ID, fullInventory); err != nil {
		t.Fatalf("failed to fill inventory: %v", err)
	}
	if _, err := store.AcceptQuestProgress(blocked.ID, platform.CharacterQuestProgress{
		QuestID:      "quest_blocked_reward",
		State:        "completed",
		CurrentCount: 1,
		TargetCount:  1,
	}, filestore.MutationOptions{MutationKey: "blocked-accept"}); err != nil {
		t.Fatalf("failed to accept blocked reward quest: %v", err)
	}
	if _, err := store.CompleteQuestWithReward(blocked.ID, filestore.QuestRewardMutation{
		QuestID:             "quest_blocked_reward",
		CurrencyCopperDelta: 25,
		RewardItems: []filestore.InventoryItemGrant{{
			ItemID:      "blocked_reward_item",
			DisplayName: "Blocked Reward",
			Quantity:    1,
			MaxStack:    1,
			Stackable:   false,
		}},
	}, filestore.MutationOptions{MutationKey: "blocked-reward"}); !errors.Is(err, filestore.ErrInventoryFull) {
		t.Fatalf("expected inventory-full rollback, got %v", err)
	}
	reloadedBlocked, err := store.GetCharacterByID(blocked.ID)
	if err != nil {
		t.Fatalf("failed to reload blocked character: %v", err)
	}
	if reloadedBlocked.CurrencyCopper != platform.StarterCurrencyCopper {
		t.Fatalf("expected currency rollback to %d, got %d", platform.StarterCurrencyCopper, reloadedBlocked.CurrencyCopper)
	}
	if reloadedBlocked.Quests["quest_blocked_reward"].RewardGrantedAt != 0 {
		t.Fatalf("expected quest reward rollback, got %#v", reloadedBlocked.Quests["quest_blocked_reward"])
	}

	rewarded := createTransactionalStateCharacter(t, store, "RewardRunner")
	if _, err := store.AcceptQuestProgress(rewarded.ID, platform.CharacterQuestProgress{
		QuestID:      "quest_idempotent_reward",
		State:        "completed",
		CurrentCount: 1,
		TargetCount:  1,
	}, filestore.MutationOptions{MutationKey: "reward-accept"}); err != nil {
		t.Fatalf("failed to accept reward quest: %v", err)
	}
	reward := filestore.QuestRewardMutation{
		QuestID:             "quest_idempotent_reward",
		ExperienceDelta:     15,
		CurrencyCopperDelta: 40,
		RewardItems: []filestore.InventoryItemGrant{{
			ItemID:      "reward_token",
			DisplayName: "Reward Token",
			Quantity:    2,
			MaxStack:    10,
			Stackable:   true,
		}},
	}
	first, err := store.CompleteQuestWithReward(rewarded.ID, reward, filestore.MutationOptions{MutationKey: "reward-once"})
	if err != nil {
		t.Fatalf("failed to apply quest reward: %v", err)
	}
	second, err := store.CompleteQuestWithReward(rewarded.ID, reward, filestore.MutationOptions{MutationKey: "reward-once"})
	if err != nil {
		t.Fatalf("failed to replay quest reward idempotently: %v", err)
	}
	if first.CurrencyCopper != second.CurrencyCopper || countInventoryItem(second.Inventory, "reward_token") != 2 {
		t.Fatalf("expected idempotent reward replay without duplication, first=%#v second=%#v", first.Inventory, second.Inventory)
	}
	if second.CurrencyCopper != platform.StarterCurrencyCopper+40 {
		t.Fatalf("expected currency reward once, got %d", second.CurrencyCopper)
	}
	if second.Quests["quest_idempotent_reward"].State != "reward_granted" {
		t.Fatalf("expected reward-granted quest state, got %#v", second.Quests["quest_idempotent_reward"])
	}
}

func TestTransactionalQuestProgressAndAbilityActionBarRoundTrips(t *testing.T) {
	store := newTestStore(t)
	character := createTransactionalStateCharacter(t, store, "ActionQuestRunner")

	if _, err := store.AcceptQuestProgress(character.ID, platform.CharacterQuestProgress{
		QuestID:      "quest_sql_progress",
		State:        "active",
		CurrentCount: 0,
		TargetCount:  3,
	}, filestore.MutationOptions{MutationKey: "accept-progress"}); err != nil {
		t.Fatalf("failed to accept quest: %v", err)
	}
	if _, err := store.UpdateQuestProgress(character.ID, platform.CharacterQuestProgress{
		QuestID:      "quest_sql_progress",
		State:        "active",
		CurrentCount: 2,
		TargetCount:  3,
		ObjectiveProgress: map[string]platform.CharacterQuestObjectiveProgress{
			"objective_sql": {NodeID: "objective_sql", Current: 2, Target: 3},
		},
	}, filestore.MutationOptions{MutationKey: "progress-two"}); err != nil {
		t.Fatalf("failed to update quest progress: %v", err)
	}

	updated, err := store.GrantLearnedAbility(character.ID, platform.DrivingBlowAbilityID, filestore.MutationOptions{MutationKey: "grant-driving-blow"})
	if err != nil {
		t.Fatalf("failed to grant learned ability: %v", err)
	}
	updated, err = store.GrantLearnedAbility(character.ID, platform.DrivingBlowAbilityID, filestore.MutationOptions{MutationKey: "grant-driving-blow-again"})
	if err != nil {
		t.Fatalf("failed duplicate learned ability handling: %v", err)
	}
	if countString(updated.LearnedAbilityIDs, platform.DrivingBlowAbilityID) != 1 {
		t.Fatalf("expected learned ability once, got %#v", updated.LearnedAbilityIDs)
	}

	if _, err := store.AssignActionBarSlot(character.ID, 7, platform.DrivingBlowAbilityID, filestore.MutationOptions{MutationKey: "assign-slot-7"}); err != nil {
		t.Fatalf("failed to assign action-bar slot: %v", err)
	}
	if _, err := store.MoveActionBarSlot(character.ID, 7, 8, filestore.MutationOptions{MutationKey: "move-slot-7-8"}); err != nil {
		t.Fatalf("failed to move action-bar slot: %v", err)
	}
	if _, err := store.ClearActionBarSlot(character.ID, 8, filestore.MutationOptions{MutationKey: "clear-slot-8"}); err != nil {
		t.Fatalf("failed to clear action-bar slot: %v", err)
	}
	if _, err := store.AssignActionBarSlot(character.ID, 9, platform.RallyingCallAbilityID, filestore.MutationOptions{MutationKey: "assign-unknown"}); !errors.Is(err, filestore.ErrAbilityNotLearned) {
		t.Fatalf("expected unknown ability assignment rejection, got %v", err)
	}

	reloaded, err := store.GetCharacterByID(character.ID)
	if err != nil {
		t.Fatalf("failed to reload character: %v", err)
	}
	if reloaded.Quests["quest_sql_progress"].ObjectiveProgress["objective_sql"].Current != 2 {
		t.Fatalf("expected quest objective progress round trip, got %#v", reloaded.Quests)
	}
	if reloaded.ActionBarSlots[8].AbilityID != "" {
		t.Fatalf("expected cleared action-bar slot, got %#v", reloaded.ActionBarSlots[8])
	}
}

func TestTransactionalPositionSnapshotsAndReconnectRestore(t *testing.T) {
	store := newTestStore(t)
	character := createTransactionalStateCharacter(t, store, "ReconnectRunner")

	if _, err := store.UpdateCharacterState(character.ID, "stonewake_vale", 22, 33, 1.5); err != nil {
		t.Fatalf("failed to update character position: %v", err)
	}
	if _, err := store.UpdateCharacterState(character.ID, "stonewake_vale", 44, 55, 2.5); err != nil {
		t.Fatalf("failed to update character position second time: %v", err)
	}
	snapshots, err := store.GetCharacterPositionSnapshots(character.ID, 10)
	if err != nil {
		t.Fatalf("failed to list position snapshots: %v", err)
	}
	if len(snapshots) < 2 {
		t.Fatalf("expected at least two position snapshots, got %#v", snapshots)
	}

	if _, err := store.GrantInventoryItem(character.ID, filestore.InventoryItemGrant{
		ItemID:      "reconnect_token",
		DisplayName: "Reconnect Token",
		Quantity:    1,
		MaxStack:    5,
		Stackable:   true,
	}, filestore.MutationOptions{MutationKey: "grant-reconnect-token"}); err != nil {
		t.Fatalf("failed to grant reconnect inventory: %v", err)
	}
	if _, err := store.GrantLearnedAbility(character.ID, platform.DrivingBlowAbilityID, filestore.MutationOptions{MutationKey: "grant-reconnect-ability"}); err != nil {
		t.Fatalf("failed to grant reconnect ability: %v", err)
	}
	if _, err := store.AssignActionBarSlot(character.ID, 10, platform.DrivingBlowAbilityID, filestore.MutationOptions{MutationKey: "assign-reconnect-action"}); err != nil {
		t.Fatalf("failed to assign reconnect action slot: %v", err)
	}
	if _, err := store.AcceptQuestProgress(character.ID, platform.CharacterQuestProgress{
		QuestID:      "quest_reconnect",
		State:        "active",
		CurrentCount: 1,
		TargetCount:  2,
	}, filestore.MutationOptions{MutationKey: "accept-reconnect-quest"}); err != nil {
		t.Fatalf("failed to accept reconnect quest: %v", err)
	}

	recovery, err := store.LoadSessionRecoveryState(character.ID)
	if err != nil {
		t.Fatalf("failed to load recovery state: %v", err)
	}
	if recovery.X != 44 || recovery.Y != 55 || recovery.Z != 2.5 {
		t.Fatalf("expected restored position, got %#v", recovery)
	}
	if countInventoryItem(recovery.Inventory, "reconnect_token") != 1 {
		t.Fatalf("expected restored inventory, got %#v", recovery.Inventory)
	}
	if !containsString(recovery.LearnedAbilityIDs, platform.DrivingBlowAbilityID) {
		t.Fatalf("expected restored learned ability, got %#v", recovery.LearnedAbilityIDs)
	}
	if recovery.ActionBarSlots[10].AbilityID != platform.DrivingBlowAbilityID {
		t.Fatalf("expected restored action bar, got %#v", recovery.ActionBarSlots[10])
	}
	if recovery.Quests["quest_reconnect"].CurrentCount != 1 {
		t.Fatalf("expected restored quest progress, got %#v", recovery.Quests)
	}
}

func TestConcurrentInventoryGrantsDoNotLoseState(t *testing.T) {
	store := newTestStore(t)
	character := createTransactionalStateCharacter(t, store, "ConcurrentRunner")

	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for index := 0; index < 16; index++ {
		index := index
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.GrantInventoryItem(character.ID, filestore.InventoryItemGrant{
				ItemID:      "concurrent_token",
				DisplayName: "Concurrent Token",
				Quantity:    1,
				MaxStack:    32,
				Stackable:   true,
			}, filestore.MutationOptions{MutationKey: fmt.Sprintf("concurrent-grant-%02d", index)})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent grant failed: %v", err)
		}
	}

	reloaded, err := store.GetCharacterByID(character.ID)
	if err != nil {
		t.Fatalf("failed to reload concurrent character: %v", err)
	}
	if got := countInventoryItem(reloaded.Inventory, "concurrent_token"); got != 16 {
		t.Fatalf("expected 16 concurrent tokens without loss/duplication, got %d in %#v", got, reloaded.Inventory)
	}
}

func createTransactionalStateCharacter(t *testing.T, store *Store, displayName string) platform.Character {
	t.Helper()
	realm, err := SeedDevRealm(store)
	if err != nil {
		t.Fatalf("failed to seed realm: %v", err)
	}
	account, err := SeedTestAccount(store, "acct_"+displayName, "secret")
	if err != nil {
		t.Fatalf("failed to seed account: %v", err)
	}
	character, err := store.CreateCharacter(
		account.ID,
		realm.ID,
		displayName,
		platform.DefaultRaceID,
		platform.DefaultClassID,
		platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatalf("failed to create character: %v", err)
	}
	return character
}

func countInventoryItem(inventory []platform.CharacterInventorySlot, itemID string) int {
	total := 0
	for _, slot := range platform.NormalizeInventorySlots(inventory) {
		if slot.ItemID == itemID {
			total += slot.StackCount
		}
	}
	return total
}

func countString(values []string, expected string) int {
	count := 0
	for _, value := range values {
		if value == expected {
			count++
		}
	}
	return count
}
