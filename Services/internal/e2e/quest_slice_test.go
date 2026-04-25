package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/characters"
	"amandacore/services/internal/realms"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

const milestoneQuestID = "sv_wall_rats"

func TestQuestSlicePersistence(t *testing.T) {
	fixture := newCombatFixture(t)

	state := fixture.getWorldState(t)
	assertQuestSummaryState(t, state, "sv_first_muster", "not_started", 0, 1, 0, 125)
	assertStarterInventory(t, state)

	state = fixture.moveToPosition(t, 26.0, 24.0)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"questId":           "sv_first_muster",
	}, http.StatusBadRequest, nil)
	state = fixture.getWorldState(t)
	assertQuestSummaryState(t, state, "sv_first_muster", "not_started", 0, 1, 0, 125)
	assertStarterInventory(t, state)

	state = fixture.targetFriendlyByID(t, "npc_commander_elian_rook")
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"questId":           "sv_first_muster",
	}, http.StatusOK, &state)
	assertQuestSummaryState(t, state, "sv_first_muster", "active", 0, 1, 0, 125)
	assertStarterInventory(t, state)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"questId":           "sv_first_muster",
	}, http.StatusBadRequest, nil)
	state = fixture.getWorldState(t)
	assertQuestSummaryState(t, state, "sv_first_muster", "active", 0, 1, 0, 125)
	assertStarterInventory(t, state)

	state = fixture.moveToPosition(t, 12.0, 8.0)
	state = fixture.targetFriendlyByID(t, "trainer_armsmaster_corin_vale")
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"questId":           "sv_first_muster",
	}, http.StatusOK, &state)
	assertQuestSummaryState(t, state, "sv_first_muster", "reward_granted", 1, 1, 35, 130)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"questId":           "sv_yard_drills",
	}, http.StatusOK, &state)
	assertQuestSummaryState(t, state, "sv_yard_drills", "active", 0, 3, 35, 130)

	killMobForQuestCredit(t, fixture, "mob_training_dummy_01")
	killMobForQuestCredit(t, fixture, "mob_training_dummy_02")
	killMobForQuestCredit(t, fixture, "mob_training_dummy_03")
	state = fixture.targetFriendlyByID(t, "trainer_armsmaster_corin_vale")
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"questId":           "sv_yard_drills",
	}, http.StatusOK, &state)
	assertQuestSummaryState(t, state, "sv_yard_drills", "reward_granted", 3, 3, 80, 135)

	state = fixture.targetFriendlyByID(t, "npc_commander_elian_rook")
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"questId":           milestoneQuestID,
	}, http.StatusOK, &state)
	assertQuestSummaryState(t, state, milestoneQuestID, "active", 0, 6, 80, 135)

	killMobForQuestCredit(t, fixture, "mob_ditch_rat_01")
	state = fixture.getWorldState(t)
	assertQuestSummaryState(t, state, milestoneQuestID, "active", 1, 6, 80, 135)
	assertStarterInventory(t, state)
	time.Sleep(1200 * time.Millisecond)
	state = fixture.getWorldState(t)
	assertQuestSummaryState(t, state, milestoneQuestID, "active", 1, 6, 80, 135)
	assertStarterInventory(t, state)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/disconnect", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
	}, http.StatusOK, nil)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
	}, http.StatusOK, &state)
	assertQuestSummaryState(t, state, milestoneQuestID, "active", 1, 6, 80, 135)
	assertStarterInventory(t, state)

	killMobForQuestCredit(t, fixture, "mob_ditch_rat_02")
	killMobForQuestCredit(t, fixture, "mob_ditch_rat_03")
	killMobForQuestCredit(t, fixture, "mob_ditch_rat_04")
	killMobForQuestCredit(t, fixture, "mob_ditch_rat_01")
	killMobForQuestCredit(t, fixture, "mob_ditch_rat_02")
	state = fixture.targetFriendlyByID(t, "npc_commander_elian_rook")
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"questId":           milestoneQuestID,
	}, http.StatusOK, &state)
	state = fixture.getWorldState(t)
	assertQuestSummaryState(t, state, milestoneQuestID, "reward_granted", 6, 6, 135, 145)
	assertStarterInventory(t, state)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"questId":           milestoneQuestID,
	}, http.StatusBadRequest, nil)
	state = fixture.getWorldState(t)
	assertQuestSummaryState(t, state, milestoneQuestID, "reward_granted", 6, 6, 135, 145)
	assertStarterInventory(t, state)

	killMobForQuestCredit(t, fixture, "mob_ditch_rat_01")
	state = fixture.getWorldState(t)
	assertQuestSummaryState(t, state, milestoneQuestID, "reward_granted", 6, 6, 135, 145)
	assertStarterInventory(t, state)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/disconnect", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
	}, http.StatusOK, nil)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
	}, http.StatusOK, &state)
	assertQuestSummaryState(t, state, milestoneQuestID, "reward_granted", 6, 6, 135, 145)
	assertStarterInventory(t, state)

	restarted := restartQuestFixture(t, fixture)
	state = restarted.getWorldState(t)
	assertQuestSummaryState(t, state, milestoneQuestID, "reward_granted", 6, 6, 135, 145)
	assertStarterInventory(t, state)
}

func killMobForQuestCredit(t *testing.T, fixture *combatFixture, mobID string) {
	t.Helper()

	state := fixture.getWorldState(t)
	state = topUpWarriorHealth(t, fixture, state, 80.0)
	mob := findHostileMobByID(t, state, mobID)
	state = fixture.moveToPosition(t, mob["x"].(float64)-3.0, mob["y"].(float64)-2.0)
	state = fixture.targetMobByID(t, mobID)
	mob = findHostileMobByID(t, state, mobID)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/auto", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"enabled":           true,
	}, http.StatusOK, &state)

	waitForMobHealthBelow(t, fixture.server, fixture.worldSessionToken, mobID, mob["maxHealth"].(float64), 4*time.Second)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"abilityId":         "steady_strike",
	}, http.StatusOK, &state)

	time.Sleep(1700 * time.Millisecond)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"abilityId":         "steady_strike",
	}, http.StatusOK, &state)

	waitForMobDead(t, fixture.server, fixture.worldSessionToken, mobID, 10*time.Second)
	waitForMobRespawned(t, fixture.server, fixture.worldSessionToken, mobID, 10*time.Second)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/disconnect", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
	}, http.StatusOK, nil)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
	}, http.StatusOK, &state)
}

func topUpWarriorHealth(t *testing.T, fixture *combatFixture, state map[string]any, minimumHealth float64) map[string]any {
	t.Helper()

	health, ok := state["health"].(float64)
	if !ok {
		t.Fatalf("state response missing health payload: %#v", state["health"])
	}

	for health < minimumHealth {
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         "brace",
		}, http.StatusOK, &state)
		time.Sleep(1700 * time.Millisecond)
		health = state["health"].(float64)
	}

	return state
}

func assertQuestSummaryState(t *testing.T, state map[string]any, questID string, expectedState string, expectedCount int, expectedTarget int, expectedExperience int, expectedCurrencyCopper int) {
	t.Helper()

	quest := findQuestSummary(t, state, questID)
	if questID == statePrimaryQuestID(t, state) {
		primary, ok := state["quest"].(map[string]any)
		if !ok {
			t.Fatalf("state response missing quest payload: %#v", state["quest"])
		}
		if primary["id"].(string) != questID {
			t.Fatalf("expected primary quest alias %s, got %v", questID, primary["id"])
		}
	}

	if quest["id"].(string) != questID {
		t.Fatalf("unexpected quest id %v", quest["id"])
	}
	if quest["state"].(string) != expectedState {
		t.Fatalf("expected quest state %s, got %v", expectedState, quest["state"])
	}
	if int(quest["currentCount"].(float64)) != expectedCount {
		t.Fatalf("expected quest count %d, got %v", expectedCount, quest["currentCount"])
	}
	if int(quest["targetCount"].(float64)) != expectedTarget {
		t.Fatalf("expected quest target %d, got %v", expectedTarget, quest["targetCount"])
	}
	if int(state["experience"].(float64)) != expectedExperience {
		t.Fatalf("expected experience %d, got %v", expectedExperience, state["experience"])
	}
	if int(state["currencyCopper"].(float64)) != expectedCurrencyCopper {
		t.Fatalf("expected currency copper %d, got %v", expectedCurrencyCopper, state["currencyCopper"])
	}

	currency, ok := state["currency"].(map[string]any)
	if !ok {
		t.Fatalf("state response missing currency payload: %#v", state["currency"])
	}
	if int(currency["gold"].(float64)) != expectedCurrencyCopper/10000 {
		t.Fatalf("expected gold %d, got %v", expectedCurrencyCopper/10000, currency["gold"])
	}
	if int(currency["silver"].(float64)) != (expectedCurrencyCopper%10000)/100 {
		t.Fatalf("expected silver %d, got %v", (expectedCurrencyCopper%10000)/100, currency["silver"])
	}
	if int(currency["copper"].(float64)) != expectedCurrencyCopper%100 {
		t.Fatalf("expected copper %d, got %v", expectedCurrencyCopper%100, currency["copper"])
	}
}

func findQuestSummary(t *testing.T, state map[string]any, questID string) map[string]any {
	t.Helper()

	quests, ok := state["quests"].([]any)
	if !ok {
		t.Fatalf("state response missing quests payload: %#v", state["quests"])
	}
	for _, questValue := range quests {
		quest, ok := questValue.(map[string]any)
		if !ok {
			continue
		}
		if quest["id"].(string) == questID {
			return quest
		}
	}
	t.Fatalf("quest %s was not present in quest list: %#v", questID, quests)
	return nil
}

func statePrimaryQuestID(t *testing.T, state map[string]any) string {
	t.Helper()

	quest, ok := state["quest"].(map[string]any)
	if !ok {
		t.Fatalf("state response missing quest payload: %#v", state["quest"])
	}
	return quest["id"].(string)
}

func assertStarterInventory(t *testing.T, state map[string]any) {
	t.Helper()

	inventory, ok := state["inventory"].(map[string]any)
	if !ok {
		t.Fatalf("state response missing inventory payload: %#v", state["inventory"])
	}
	if int(inventory["slotCount"].(float64)) != 16 {
		t.Fatalf("expected 16 inventory slots, got %v", inventory["slotCount"])
	}

	slots, ok := inventory["slots"].([]any)
	if !ok {
		t.Fatalf("state response missing inventory slots: %#v", inventory["slots"])
	}
	if len(slots) != 16 {
		t.Fatalf("expected 16 inventory entries, got %d", len(slots))
	}

	slot0 := slots[0].(map[string]any)
	if slot0["itemId"].(string) != "camp_ration" || int(slot0["stackCount"].(float64)) != 3 {
		t.Fatalf("unexpected slot 0 starter item: %#v", slot0)
	}

	slot1 := slots[1].(map[string]any)
	if slot1["itemId"].(string) != "linen_wrap" || int(slot1["stackCount"].(float64)) != 2 {
		t.Fatalf("unexpected slot 1 starter item: %#v", slot1)
	}
}

func restartQuestFixture(t *testing.T, original *combatFixture) *combatFixture {
	t.Helper()

	original.server.Close()

	fileStore, err := store.NewFileStore(original.storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to create restarted store: %v", err)
	}

	mux := http.NewServeMux()
	authn.RegisterRoutes(mux, fileStore)
	realms.RegisterRoutes(mux, fileStore)
	characters.RegisterRoutes(mux, fileStore)
	worlds.RegisterRoutes(mux, fileStore)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var loginResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/auth/login", nil, map[string]string{
		"username": original.username,
		"password": original.password,
	}, http.StatusOK, &loginResponse)
	accessToken := loginResponse["accessToken"].(string)

	var charactersResponse map[string][]map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/characters?realmId="+original.realmID, bearer(accessToken), http.StatusOK, &charactersResponse)
	restartedCharacterID := charactersResponse["characters"][0]["id"].(string)

	var ticketResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/world/join-ticket", bearer(accessToken), map[string]string{
		"realmId":     original.realmID,
		"characterId": restartedCharacterID,
	}, http.StatusCreated, &ticketResponse)

	var connectResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/world/connect", nil, map[string]string{
		"ticketId": ticketResponse["ticketId"].(string),
	}, http.StatusCreated, &connectResponse)

	return &combatFixture{
		server:            server,
		storePath:         original.storePath,
		username:          original.username,
		password:          original.password,
		realmID:           original.realmID,
		characterID:       restartedCharacterID,
		worldSessionToken: connectResponse["worldSessionToken"].(string),
	}
}
