package worlds

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

const groupContentTestRealmID = "sunset-frontier-dev"

func TestSoloKillCreditStillWorksForNonShareableQuest(t *testing.T) {
	server, _, alice, _, _ := newGroupCreditTestServer(t)
	activateTestQuest(t, server, alice, "bb_teeth_in_shallows")

	snapper := testMob("mob_test_snapper", mobBBRiverjawSnapperTypeID, secondZoneID, alice.X, alice.Y)
	if err := server.applyQuestKillCreditLocked(alice, snapper); err != nil {
		t.Fatalf("failed to apply solo kill credit: %v", err)
	}

	assertQuestCount(t, alice, "bb_teeth_in_shallows", 1)
}

func TestNonShareableQuestDoesNotGrantPartyCredit(t *testing.T) {
	server, _, alice, bob, _ := newGroupCreditTestServer(t)
	createTestParty(t, server.store, alice.CharacterID, bob.CharacterID)
	activateTestQuest(t, server, alice, "bb_teeth_in_shallows")
	activateTestQuest(t, server, bob, "bb_teeth_in_shallows")

	snapper := testMob("mob_test_snapper", mobBBRiverjawSnapperTypeID, secondZoneID, alice.X, alice.Y)
	if err := server.applyQuestKillCreditLocked(alice, snapper); err != nil {
		t.Fatalf("failed to apply kill credit: %v", err)
	}

	assertQuestCount(t, alice, "bb_teeth_in_shallows", 1)
	assertQuestCount(t, bob, "bb_teeth_in_shallows", 0)
}

func TestShareableGroupQuestGrantsNearbyPartyCreditAndPersists(t *testing.T) {
	server, storePath, alice, bob, _ := newGroupCreditTestServer(t)
	createTestParty(t, server.store, alice.CharacterID, bob.CharacterID)
	activateTestQuest(t, server, alice, "bb_korrin_at_the_ford")
	activateTestQuest(t, server, bob, "bb_korrin_at_the_ford")

	korrin := server.mobs["mob_bb_korrin_madbrook_01"]
	alice.CurrentTargetID = korrin.ID
	if err := server.applyDamageToMobLocked(alice, korrin, korrin.Health, "test_group_kill"); err != nil {
		t.Fatalf("failed to kill elite: %v", err)
	}

	assertQuestState(t, alice, "bb_korrin_at_the_ford", questStateCompleted)
	assertQuestState(t, bob, "bb_korrin_at_the_ford", questStateCompleted)

	restartedStore, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	bobCharacter, err := restartedStore.GetCharacterByID(bob.CharacterID)
	if err != nil {
		t.Fatalf("failed to reload Bob: %v", err)
	}
	progress := bobCharacter.Quests["bb_korrin_at_the_ford"]
	if progress.State != questStateCompleted || progress.CurrentCount != 1 {
		t.Fatalf("expected persisted Bob credit, got %#v", progress)
	}
}

func TestGroupQuestSummaryReportsNearbyEligiblePartyMembers(t *testing.T) {
	server, _, alice, bob, _ := newGroupCreditTestServer(t)
	createTestParty(t, server.store, alice.CharacterID, bob.CharacterID)
	activateTestQuest(t, server, alice, "bb_korrin_at_the_ford")
	activateTestQuest(t, server, bob, "bb_korrin_at_the_ford")

	summary := findQuestSummary(t, server.buildQuestListResponse(alice), "bb_korrin_at_the_ford")
	if summary["partyNearbyCount"] != 2 || summary["partyEligibleCount"] != 2 {
		t.Fatalf("expected both party members to be nearby and eligible, got %#v", summary)
	}
	if status, _ := summary["partyStatusText"].(string); !strings.Contains(status, "eligible") {
		t.Fatalf("expected party status text to mention eligibility, got %#v", summary["partyStatusText"])
	}
}

func TestPartyFramesExposeGroupCreditStatus(t *testing.T) {
	server, _, alice, bob, cara := newGroupCreditTestServer(t)
	createTestParty(t, server.store, alice.CharacterID, bob.CharacterID, cara.CharacterID)
	activateTestQuest(t, server, alice, "bb_korrin_at_the_ford")
	activateTestQuest(t, server, bob, "bb_korrin_at_the_ford")
	activateTestQuest(t, server, cara, "bb_korrin_at_the_ford")
	cara.X = alice.X + 80.0

	party := server.buildSocialStateLocked(alice, "").Party
	if party == nil {
		t.Fatalf("expected party response")
	}
	bobMember := findPartyMember(t, party, bob.CharacterID)
	if !bobMember.GroupCreditEligible || bobMember.GroupCreditStatus != "eligible" {
		t.Fatalf("expected Bob to be eligible for shared credit, got %#v", bobMember)
	}
	caraMember := findPartyMember(t, party, cara.CharacterID)
	if caraMember.GroupCreditEligible || caraMember.GroupCreditStatus != "out_of_range" {
		t.Fatalf("expected Cara to be out of range, got %#v", caraMember)
	}
}

func TestShareableGroupQuestSkipsOutOfRangePartyMember(t *testing.T) {
	server, _, alice, bob, _ := newGroupCreditTestServer(t)
	createTestParty(t, server.store, alice.CharacterID, bob.CharacterID)
	activateTestQuest(t, server, alice, "bb_korrin_at_the_ford")
	activateTestQuest(t, server, bob, "bb_korrin_at_the_ford")
	bob.X = alice.X + 80.0

	korrin := testMob("mob_test_korrin", mobBBKorrinMadbrookTypeID, secondZoneID, alice.X, alice.Y)
	if err := server.applyQuestKillCreditLocked(alice, korrin); err != nil {
		t.Fatalf("failed to apply group kill credit: %v", err)
	}

	assertQuestState(t, alice, "bb_korrin_at_the_ford", questStateCompleted)
	assertQuestCount(t, bob, "bb_korrin_at_the_ford", 0)
	assertSystemMessageContains(t, server, bob, "No shared credit for Korrin at the Ford: too far away.")
}

func TestShareableGroupQuestSkipsDisconnectedPartyMember(t *testing.T) {
	server, _, alice, bob, _ := newGroupCreditTestServer(t)
	createTestParty(t, server.store, alice.CharacterID, bob.CharacterID)
	activateTestQuest(t, server, alice, "bb_korrin_at_the_ford")
	activateTestQuest(t, server, bob, "bb_korrin_at_the_ford")
	bob.Connected = false

	korrin := testMob("mob_test_korrin", mobBBKorrinMadbrookTypeID, secondZoneID, alice.X, alice.Y)
	if err := server.applyQuestKillCreditLocked(alice, korrin); err != nil {
		t.Fatalf("failed to apply group kill credit: %v", err)
	}

	assertQuestState(t, alice, "bb_korrin_at_the_ford", questStateCompleted)
	assertQuestCount(t, bob, "bb_korrin_at_the_ford", 0)
}

func TestGroupQuestRewardIsExactOncePerCharacter(t *testing.T) {
	server, _, alice, _, _ := newGroupCreditTestServer(t)
	quest := server.quests["bb_korrin_at_the_ford"]
	progress := completedTestQuestProgress(quest)
	alice.QuestProgress[quest.ID] = progress
	alice.CurrentTargetID = quest.TurnInNPCID
	alice.X = 310.0
	alice.Y = 160.0

	startXP := alice.Experience
	startCopper := alice.CurrencyCopper
	if err := server.completeOrTurnInQuestLocked(alice, quest, progress); err != nil {
		t.Fatalf("failed to grant first reward: %v", err)
	}
	if alice.Experience != startXP+quest.RewardXP || alice.CurrencyCopper != startCopper+quest.RewardCopper {
		t.Fatalf("expected first reward to apply, xp %d copper %d", alice.Experience, alice.CurrencyCopper)
	}

	grantedProgress := alice.QuestProgress[quest.ID]
	if err := server.completeOrTurnInQuestLocked(alice, quest, grantedProgress); err == nil {
		t.Fatalf("expected second turn-in to fail")
	}
	if alice.Experience != startXP+quest.RewardXP || alice.CurrencyCopper != startCopper+quest.RewardCopper {
		t.Fatalf("expected second reward to be blocked, xp %d copper %d", alice.Experience, alice.CurrencyCopper)
	}
}

func TestOutdoorEliteEncounterDefinitionLeashAndRespawn(t *testing.T) {
	server := newWorldServer(nil)
	korrin := server.mobs["mob_bb_korrin_madbrook_01"]
	if korrin == nil {
		t.Fatalf("expected Korrin elite spawn")
	}
	if !korrin.Elite || korrin.Classification != "elite" {
		t.Fatalf("expected Korrin to be marked elite, got elite=%t classification=%q", korrin.Elite, korrin.Classification)
	}
	if korrin.MaxHealth < 350 || korrin.AttackDamage < 18 || korrin.LeashRadius < 36 {
		t.Fatalf("unexpected Korrin tuning: %#v", korrin)
	}

	session := &worldSessionState{
		Token:         "world_elite_test",
		CharacterID:   "char_elite_test",
		RealmID:       groupContentTestRealmID,
		ZoneID:        secondZoneID,
		X:             korrin.SpawnX + korrin.LeashRadius + 1.0,
		Y:             korrin.SpawnY,
		Connected:     true,
		Alive:         true,
		Health:        200.0,
		MaxHealth:     200.0,
		QuestProgress: map[string]platform.CharacterQuestProgress{},
	}
	server.sessionsByToken = map[string]*worldSessionState{session.Token: session}
	server.sessionTokenByChar = map[string]string{session.CharacterID: session.Token}

	korrin.CurrentTargetCharacter = session.CharacterID
	korrin.AIState = mobAIStateChasing
	server.advanceMobLocked(korrin, 0.25, 1_000)
	if korrin.AIState != mobAIStateEvading {
		t.Fatalf("expected elite to evade after leash break, got %s", korrin.AIState)
	}

	korrin.Health = 10.0
	korrin.Targetable = true
	korrin.Alive = true
	korrin.AIState = mobAIStateIdle
	session.X = korrin.X
	session.Y = korrin.Y
	session.CurrentTargetID = korrin.ID
	if err := server.applyDamageToMobLocked(session, korrin, 10.0, "test_elite_death"); err != nil {
		t.Fatalf("failed to kill elite: %v", err)
	}
	if korrin.Alive || korrin.RespawnAtMs == 0 {
		t.Fatalf("expected elite death to schedule respawn")
	}

	server.advanceMobLocked(korrin, 0.25, korrin.RespawnAtMs)
	if !korrin.Alive || !korrin.Targetable || korrin.Health != korrin.MaxHealth {
		t.Fatalf("expected elite to respawn cleanly, got alive=%t targetable=%t health=%.1f/%.1f", korrin.Alive, korrin.Targetable, korrin.Health, korrin.MaxHealth)
	}
}

func newGroupCreditTestServer(t *testing.T) (*worldServer, string, *worldSessionState, *worldSessionState, *worldSessionState) {
	t.Helper()

	storePath := filepath.Join(t.TempDir(), "platform-state.json")
	fileStore, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	aliceCharacter := createTestCharacter(t, fileStore, "group_alice", "Alice")
	bobCharacter := createTestCharacter(t, fileStore, "group_bob", "Bob")
	caraCharacter := createTestCharacter(t, fileStore, "group_cara", "Cara")

	server := newWorldServer(fileStore)
	alice := sessionFromCharacter(aliceCharacter, "world_alice")
	bob := sessionFromCharacter(bobCharacter, "world_bob")
	cara := sessionFromCharacter(caraCharacter, "world_cara")
	server.sessionsByToken = map[string]*worldSessionState{
		alice.Token: alice,
		bob.Token:   bob,
		cara.Token:  cara,
	}
	server.sessionTokenByChar = map[string]string{
		alice.CharacterID: alice.Token,
		bob.CharacterID:   bob.Token,
		cara.CharacterID:  cara.Token,
	}
	return server, storePath, alice, bob, cara
}

func createTestCharacter(t *testing.T, fileStore *store.FileStore, username string, displayName string) platform.Character {
	t.Helper()

	account, err := fileStore.RegisterAccount(username, "secret")
	if err != nil {
		t.Fatalf("failed to register account %s: %v", username, err)
	}
	character, err := fileStore.CreateCharacter(
		account.ID,
		groupContentTestRealmID,
		displayName,
		platform.DefaultRaceID,
		platform.DefaultClassID,
		platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatalf("failed to create character %s: %v", displayName, err)
	}
	if _, err := fileStore.UpdateCharacterState(character.ID, secondZoneID, 366.0, 184.0, 0.0); err != nil {
		t.Fatalf("failed to move character %s: %v", displayName, err)
	}
	updated, err := fileStore.GetCharacterByID(character.ID)
	if err != nil {
		t.Fatalf("failed to reload character %s: %v", displayName, err)
	}
	return *updated
}

func sessionFromCharacter(character platform.Character, token string) *worldSessionState {
	return &worldSessionState{
		Token:             token,
		AccountID:         character.AccountID,
		CharacterID:       character.ID,
		DisplayName:       character.DisplayName,
		ClassID:           character.ClassID,
		ArchetypeID:       character.ArchetypeID,
		Level:             character.Level,
		RealmID:           character.RealmID,
		ZoneID:            secondZoneID,
		X:                 366.0,
		Y:                 184.0,
		Z:                 0.0,
		Connected:         true,
		Alive:             true,
		Health:            180.0,
		MaxHealth:         180.0,
		Resource:          playerMaxResource,
		MaxResource:       playerMaxResource,
		Experience:        character.Experience,
		CurrencyCopper:    character.CurrencyCopper,
		Inventory:         platform.NormalizeInventorySlots(character.Inventory),
		Equipment:         platform.NormalizeEquipmentSlots(character.Equipment),
		Talents:           platform.NormalizeTalentRanks(character.Talents),
		LearnedAbilityIDs: platform.NormalizeLearnedAbilityIDs(character.LearnedAbilityIDs),
		ActionBarSlots:    platform.NormalizeActionBarSlots(character.ActionBarSlots, character.LearnedAbilityIDs),
		QuestProgress:     map[string]platform.CharacterQuestProgress{},
		TrackedQuestIDs:   []string{},
	}
}

func createTestParty(t *testing.T, fileStore *store.FileStore, leaderCharacterID string, memberCharacterIDs ...string) {
	t.Helper()
	members := append([]string{leaderCharacterID}, memberCharacterIDs...)
	if _, err := fileStore.CreateParty(leaderCharacterID, members); err != nil {
		t.Fatalf("failed to create party: %v", err)
	}
}

func activateTestQuest(t *testing.T, server *worldServer, session *worldSessionState, questID string) {
	t.Helper()
	quest, found := server.quests[questID]
	if !found {
		t.Fatalf("quest %s not found", questID)
	}
	session.QuestProgress[questID] = activeTestQuestProgress(quest)
}

func activeTestQuestProgress(quest questDefinition) platform.CharacterQuestProgress {
	now := time.Now().Unix()
	return platform.CharacterQuestProgress{
		QuestID:      quest.ID,
		State:        questStateActive,
		CurrentCount: 0,
		TargetCount:  quest.TargetCount,
		AcceptedAt:   now,
		UpdatedAt:    now,
	}
}

func completedTestQuestProgress(quest questDefinition) platform.CharacterQuestProgress {
	progress := activeTestQuestProgress(quest)
	now := time.Now().Unix()
	progress.State = questStateCompleted
	progress.CurrentCount = quest.TargetCount
	progress.CompletedAt = now
	progress.UpdatedAt = now
	return progress
}

func testMob(id string, mobTypeID string, zoneID string, x float64, y float64) *mobState {
	return &mobState{
		ID:              id,
		MobTypeID:       mobTypeID,
		DisplayName:     "Test Mob",
		Kind:            hostileMobKind,
		ZoneID:          zoneID,
		X:               x,
		Y:               y,
		Z:               0.0,
		SpawnX:          x,
		SpawnY:          y,
		SpawnZ:          0.0,
		Health:          100.0,
		MaxHealth:       100.0,
		AttackRange:     2.75,
		AttackDamage:    1.0,
		AttackCadenceMs: 1_900,
		LeashRadius:     30.0,
		RespawnDelayMs:  1_000,
		Alive:           true,
		Targetable:      true,
		AIState:         mobAIStateIdle,
	}
}

func assertQuestCount(t *testing.T, session *worldSessionState, questID string, expected int) {
	t.Helper()
	progress := session.QuestProgress[questID]
	if progress.CurrentCount != expected {
		t.Fatalf("expected %s count %d for %s, got %#v", questID, expected, session.CharacterID, progress)
	}
}

func assertQuestState(t *testing.T, session *worldSessionState, questID string, expected string) {
	t.Helper()
	progress := session.QuestProgress[questID]
	if progress.State != expected {
		t.Fatalf("expected %s state %s for %s, got %#v", questID, expected, session.CharacterID, progress)
	}
}

func findQuestSummary(t *testing.T, summaries []map[string]any, questID string) map[string]any {
	t.Helper()
	for _, summary := range summaries {
		if summary["id"] == questID {
			return summary
		}
	}
	t.Fatalf("quest summary %s not found", questID)
	return nil
}

func findPartyMember(t *testing.T, party *partyResponse, characterID string) partyMemberResponse {
	t.Helper()
	for _, member := range party.Members {
		if member.CharacterID == characterID {
			return member
		}
	}
	t.Fatalf("party member %s not found in %#v", characterID, party.Members)
	return partyMemberResponse{}
}

func assertSystemMessageContains(t *testing.T, server *worldServer, session *worldSessionState, expected string) {
	t.Helper()
	state := server.buildSocialStateLocked(session, "")
	for _, message := range state.ChatMessages {
		if message.Channel == chatChannelSystem && strings.Contains(message.MessageText, expected) {
			return
		}
	}
	t.Fatalf("expected system message containing %q, got %#v", expected, state.ChatMessages)
}
