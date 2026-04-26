package worlds

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

func TestDevItemCatalogDefinitionsResolve(t *testing.T) {
	required := map[string]struct {
		name     string
		kind     string
		quality  string
		maxStack int
	}{
		itemDevGlimmerShardID: {"Glimmer Shard", itemKindCraftingMaterial, itemQualityCommon, 99},
		itemDevStalkerFangID:  {"Stalker Fang", itemKindQuestItem, itemQualityCommon, 20},
		itemDevFieldRationID:  {"Field Ration", itemKindConsumable, itemQualityCommon, 10},
		itemDevCopperTokenID:  {"Copper Token", itemKindCurrencyToken, itemQualityCommon, 999},
	}

	for itemID, expected := range required {
		item, found := findItemDefinition(itemID)
		if !found {
			t.Fatalf("expected dev item %s to resolve", itemID)
		}
		if item.DisplayName != expected.name || item.Kind != expected.kind || item.Quality != expected.quality || item.MaxStack != expected.maxStack {
			t.Fatalf("unexpected item definition for %s: %#v", itemID, item)
		}
	}
}

func TestInventoryGrantStackingMaxStackAndCapacity(t *testing.T) {
	server := newWorldServer(nil)
	session := newProgressionTestSession(server, "inventory_stack")

	if _, err := server.grantItemStackToSessionLocked(session, inventoryGrant{ItemID: itemDevGlimmerShardID, Quantity: 3, Reason: "test"}); err != nil {
		t.Fatalf("grant failed: %v", err)
	}
	if got := inventoryItemCount(session.Inventory, itemDevGlimmerShardID); got != 3 {
		t.Fatalf("expected 3 glimmer shards, got %d", got)
	}

	if _, err := server.grantItemStackToSessionLocked(session, inventoryGrant{ItemID: itemDevGlimmerShardID, Quantity: 117, Reason: "test"}); err != nil {
		t.Fatalf("second grant failed: %v", err)
	}
	if got := inventoryItemCount(session.Inventory, itemDevGlimmerShardID); got != 120 {
		t.Fatalf("expected 120 glimmer shards, got %d", got)
	}
	for _, slot := range session.Inventory {
		if slot.ItemID == itemDevGlimmerShardID && slot.StackCount > 99 {
			t.Fatalf("glimmer stack exceeded max stack: %#v", slot)
		}
	}

	full := make([]platform.CharacterInventorySlot, platform.InventorySlotCount)
	for index := range full {
		full[index] = platform.CharacterInventorySlot{SlotIndex: index, ItemID: itemDevFieldRationID, StackCount: 10}
	}
	session.Inventory = full
	if _, err := server.grantItemStackToSessionLocked(session, inventoryGrant{ItemID: itemDevGlimmerShardID, Quantity: 1, Reason: "test"}); err == nil || err.Error() != "InventoryFull" {
		t.Fatalf("expected InventoryFull, got %v", err)
	}
}

func TestDevLootTableDeterministicGeneration(t *testing.T) {
	table, found := findLootTableDefinition(devIsleStalkerLootTableID)
	if !found {
		t.Fatalf("expected dev stalker loot table")
	}
	if table.SourceArchetypeID != devIsleStalkerArchetypeID {
		t.Fatalf("unexpected source archetype: %s", table.SourceArchetypeID)
	}

	context := LootRollContext{SourceEntityID: devIsleStalkerEntityID, SourceArchetypeID: devIsleStalkerArchetypeID, ZoneID: defaultZoneID, KillerCharacterID: "char_seed"}
	left, err := generateLoot(table, context, newSeededLootRollSource(42))
	if err != nil {
		t.Fatalf("first roll failed: %v", err)
	}
	right, err := generateLoot(table, context, newSeededLootRollSource(42))
	if err != nil {
		t.Fatalf("second roll failed: %v", err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("expected deterministic loot rolls, got %#v and %#v", left, right)
	}
	if !lootRollContains(left, itemDevStalkerFangID) || !lootRollContains(left, itemDevCopperTokenID) {
		t.Fatalf("expected guaranteed fang and copper token, got %#v", left.Items)
	}
}

func TestLootContainerOwnershipRangeClaimAndKillCredit(t *testing.T) {
	server := newWorldServer(nil)
	owner := newProgressionTestSession(server, "loot_owner")
	container := killDevStalkerForProgressionTest(t, server, owner)

	if container.OwnerCharacterID != owner.CharacterID {
		t.Fatalf("expected owner %s, got %s", owner.CharacterID, container.OwnerCharacterID)
	}
	if got := server.killCreditCountLocked(owner.CharacterID, devIsleStalkerArchetypeID); got != 1 {
		t.Fatalf("expected one kill credit, got %d", got)
	}

	nonOwner := newProgressionTestSession(server, "loot_other")
	nonOwner.ZoneID = owner.ZoneID
	nonOwner.X = container.X
	nonOwner.Y = container.Y
	if err := server.claimLootLocked(nonOwner, container.LootContainerID); err == nil || err.Error() != "NotLootOwner" {
		t.Fatalf("expected NotLootOwner, got %v", err)
	}

	owner.X = container.X + lootInteractionRange + 1
	if err := server.claimLootLocked(owner, container.LootContainerID); err == nil || err.Error() != "OutOfRange" {
		t.Fatalf("expected OutOfRange, got %v", err)
	}

	owner.X = container.X
	owner.Y = container.Y
	if _, err := server.inspectLootLocked(owner, container.LootContainerID); err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	if err := server.claimLootLocked(owner, container.LootContainerID); err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if inventoryItemCount(owner.Inventory, itemDevStalkerFangID) < 1 {
		t.Fatalf("expected claimed fang in inventory")
	}
	if err := server.claimLootLocked(owner, container.LootContainerID); err == nil || err.Error() != "LootAlreadyClaimed" {
		t.Fatalf("expected LootAlreadyClaimed, got %v", err)
	}
}

func TestDevQuestGraphProgressionAndRewards(t *testing.T) {
	server := newWorldServer(nil)
	session := newProgressionTestSession(server, "quest_graph")
	quest, found := server.quests[devFirstHuntQuestID]
	if !found {
		t.Fatalf("expected dev quest %s", devFirstHuntQuestID)
	}

	if err := server.acceptQuestLocked(session, devFirstHuntQuestID); err != nil {
		t.Fatalf("accept failed: %v", err)
	}
	if err := server.acceptQuestLocked(session, devFirstHuntQuestID); err == nil || !strings.Contains(err.Error(), "already accepted") {
		t.Fatalf("expected duplicate accept rejection, got %v", err)
	}

	progress := session.QuestProgress[devFirstHuntQuestID]
	if !server.questObjectiveNodeActive(quest, progress, quest.ObjectiveGraph.Nodes[0]) {
		t.Fatalf("expected kill node to be active")
	}
	if server.questObjectiveNodeActive(quest, progress, quest.ObjectiveGraph.Nodes[1]) {
		t.Fatalf("expected collect node to wait on kill dependency")
	}

	container := killDevStalkerForProgressionTest(t, server, session)
	progress = session.QuestProgress[devFirstHuntQuestID]
	killProgress, err := graphObjectiveProgress(progress, devFirstHuntKillNodeID)
	if err != nil || !killProgress.Completed {
		t.Fatalf("expected kill objective complete, progress=%#v err=%v", killProgress, err)
	}
	if !server.questObjectiveNodeActive(quest, progress, quest.ObjectiveGraph.Nodes[1]) {
		t.Fatalf("expected collect node active after kill")
	}
	if progress.State == questStateReady {
		t.Fatalf("quest should not be ready before fang collection")
	}

	if err := server.claimLootLocked(session, container.LootContainerID); err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	progress = session.QuestProgress[devFirstHuntQuestID]
	collectProgress, err := graphObjectiveProgress(progress, devFirstHuntCollectNodeID)
	if err != nil || !collectProgress.Completed {
		t.Fatalf("expected collect objective complete, progress=%#v err=%v", collectProgress, err)
	}
	if progress.State != questStateReady {
		t.Fatalf("expected quest ready, got %s", progress.State)
	}

	if err := server.completeQuestLocked(session, devFirstHuntQuestID); err != nil {
		t.Fatalf("complete failed: %v", err)
	}
	if got := session.QuestProgress[devFirstHuntQuestID].State; got != questStateRewardGranted {
		t.Fatalf("expected reward granted state, got %s", got)
	}
	if inventoryItemCount(session.Inventory, itemDevCopperTokenID) < 5 {
		t.Fatalf("expected copper token quest reward")
	}
	if err := server.completeQuestLocked(session, devFirstHuntQuestID); err == nil || !strings.Contains(err.Error(), "already completed") {
		t.Fatalf("expected duplicate completion rejection, got %v", err)
	}
}

func TestProgressionPersistenceStoresQuestAndInventory(t *testing.T) {
	fileStore, err := store.NewFileStore(filepath.Join(t.TempDir(), "platform-state.json"), "test-build", "http://localhost:8085")
	if err != nil {
		t.Fatalf("store create failed: %v", err)
	}
	account, err := fileStore.RegisterAccount("progression_persist", "secret")
	if err != nil {
		t.Fatalf("account create failed: %v", err)
	}
	character, err := fileStore.CreateCharacter(account.ID, "sunset-frontier-dev", "Persistor", platform.DefaultRaceID, platform.DefaultClassID, platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatalf("character create failed: %v", err)
	}

	server := newWorldServer(fileStore)
	session := newProgressionTestSession(server, character.ID)
	server.applyCharacterProgressionLocked(session, &character)

	if err := server.acceptQuestLocked(session, devFirstHuntQuestID); err != nil {
		t.Fatalf("accept failed: %v", err)
	}
	container := killDevStalkerForProgressionTest(t, server, session)
	if err := server.claimLootLocked(session, container.LootContainerID); err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if err := server.completeQuestLocked(session, devFirstHuntQuestID); err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	reloaded, err := fileStore.GetCharacterByID(character.ID)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if got := reloaded.Quests[devFirstHuntQuestID].State; got != questStateRewardGranted {
		t.Fatalf("expected persisted quest reward state, got %s", got)
	}
	if inventoryItemCount(reloaded.Inventory, itemDevStalkerFangID) < 1 {
		t.Fatalf("expected persisted looted fang")
	}
	if inventoryItemCount(reloaded.Inventory, itemDevCopperTokenID) < 5 {
		t.Fatalf("expected persisted reward copper tokens")
	}
}

func newProgressionTestSession(server *worldServer, characterID string) *worldSessionState {
	session := &worldSessionState{
		Token:             "token_" + characterID,
		AccountID:         "account_" + characterID,
		CharacterID:       characterID,
		DisplayName:       characterID,
		ClassID:           platform.DefaultClassID,
		Level:             1,
		RealmID:           "sunset-frontier-dev",
		ZoneID:            defaultZoneID,
		X:                 starterSpawnX,
		Y:                 starterSpawnY,
		Z:                 starterSpawnZ,
		Connected:         true,
		Health:            playerMaxHealth,
		MaxHealth:         playerMaxHealth,
		Resource:          0,
		MaxResource:       playerMaxResource,
		Alive:             true,
		Inventory:         platform.NormalizeInventorySlots(nil),
		LearnedAbilityIDs: platform.DefaultStartingLearnedAbilityIDs(),
		ActionBarSlots:    platform.DefaultActionBarSlots(platform.DefaultStartingLearnedAbilityIDs()),
		QuestProgress:     map[string]platform.CharacterQuestProgress{},
	}
	server.sessionsByToken[session.Token] = session
	server.sessionTokenByChar[session.CharacterID] = session.Token
	return session
}

func killDevStalkerForProgressionTest(t *testing.T, server *worldServer, session *worldSessionState) *lootContainerState {
	t.Helper()

	mob := server.mobs[devIsleStalkerEntityID]
	if mob == nil {
		t.Fatalf("dev stalker missing")
	}
	session.ZoneID = mob.ZoneID
	session.InstanceID = mob.InstanceID
	session.X = mob.X
	session.Y = mob.Y
	session.Z = mob.Z
	if err := server.applyDamageToMobLocked(session, mob, mob.Health+1, "test"); err != nil {
		t.Fatalf("kill failed: %v", err)
	}
	for _, containerID := range server.lootContainerOrder {
		container := server.lootContainers[containerID]
		if container != nil && container.SourceEntityID == mob.ID && container.OwnerCharacterID == session.CharacterID {
			return container
		}
	}
	t.Fatalf("loot container missing")
	return nil
}

func lootRollContains(result LootRollResult, itemID string) bool {
	for _, item := range result.Items {
		if item.ItemID == itemID && item.Quantity > 0 {
			return true
		}
	}
	return false
}
