package e2e

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/characters"
	"amandacore/services/internal/realms"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

const (
	dungeonID          = "dun_tallowdeep_sluice"
	dungeonQuestID     = "bb_tallowdeep_first_descent"
	dungeonQuestGiver  = "npc_bb_stonesetter_barn_ald"
	dungeonZoneID      = "dun_tallowdeep_sluice"
	dungeonOutdoorZone = "brindlebrook_roadlands"
	dungeonBossName    = "Vell Ordrin, Sluice Warden"
	dungeonRewardItem  = "tds_sluiceguard_handwraps"
)

type dungeonFixture struct {
	server    *httptest.Server
	fileStore *store.FileStore
	realmID   string
	alice     socialPlayer
	bob       socialPlayer
	cara      socialPlayer
}

func TestDungeonSlicePartyInstanceCombatQuestRewardAndExit(t *testing.T) {
	fixture := newDungeonFixture(t)

	fixture.prepareDungeonCandidate(t, fixture.alice)
	fixture.prepareDungeonCandidate(t, fixture.bob)
	fixture.prepareDungeonCandidate(t, fixture.cara)
	fixture.acceptDungeonQuest(t, fixture.alice)
	fixture.acceptDungeonQuest(t, fixture.bob)
	fixture.formParty(t)

	fixture.placePlayer(t, fixture.alice, dungeonOutdoorZone, 590, 342)
	fixture.placePlayer(t, fixture.bob, dungeonOutdoorZone, 592, 342)
	fixture.placePlayer(t, fixture.cara, dungeonOutdoorZone, 590, 342)

	var rejected map[string]any
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/dungeon/enter", nil, map[string]any{
		"worldSessionToken": fixture.cara.worldSessionToken,
		"dungeonId":         dungeonID,
	}, http.StatusBadRequest, &rejected)

	aliceState := fixture.enterDungeon(t, fixture.alice)
	bobState := fixture.worldState(t, fixture.bob)
	assertDungeonInstance(t, aliceState, bobState)
	if aliceState["zoneId"] != dungeonZoneID || bobState["zoneId"] != dungeonZoneID {
		t.Fatalf("expected party to enter dungeon zone together, got Alice=%v Bob=%v", aliceState["zoneId"], bobState["zoneId"])
	}
	if countHostileMobs(aliceState) != 7 {
		t.Fatalf("expected 7 private dungeon mobs, got %d", countHostileMobs(aliceState))
	}
	caraState := fixture.worldState(t, fixture.cara)
	if caraState["zoneId"] == dungeonZoneID || countHostileMobs(caraState) == 7 {
		t.Fatalf("expected non-party player to remain outside private instance, got zone=%v mobs=%d", caraState["zoneId"], countHostileMobs(caraState))
	}

	trashID := findHostileMobByDisplayName(t, aliceState, "Sluice Guard")["id"].(string)
	fixture.killDungeonMob(t, trashID, 40, 15, 20*time.Second)
	aliceState = fixture.worldState(t, fixture.alice)
	trash := findHostileMobByID(t, aliceState, trashID)
	if alive, _ := trash["alive"].(bool); alive {
		t.Fatalf("expected trash mob %s to stay dead inside instance", trashID)
	}

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/disconnect", nil, map[string]any{
		"worldSessionToken": fixture.bob.worldSessionToken,
	}, http.StatusOK, nil)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
		"worldSessionToken": fixture.bob.worldSessionToken,
	}, http.StatusOK, &bobState)
	if bobState["zoneId"] != dungeonZoneID {
		t.Fatalf("expected reconnect inside active dungeon instance, got zone=%v", bobState["zoneId"])
	}
	assertDungeonInstance(t, aliceState, bobState)

	bossID := findHostileMobByDisplayName(t, fixture.worldState(t, fixture.alice), dungeonBossName)["id"].(string)
	fixture.killDungeonMob(t, bossID, 146, 34, 45*time.Second)
	aliceState = fixture.worldState(t, fixture.alice)
	if !questReady(t, aliceState, dungeonQuestID) {
		t.Fatalf("expected boss kill to complete dungeon objective, got quests=%#v", aliceState["quests"])
	}
	bobState = fixture.worldState(t, fixture.bob)
	if !questReady(t, bobState, dungeonQuestID) {
		t.Fatalf("expected party credit for Bob on boss kill, got quests=%#v", bobState["quests"])
	}

	aliceXPBefore := int(aliceState["experience"].(float64))
	aliceCopperBefore := int(aliceState["currencyCopper"].(float64))
	aliceState = fixture.exitDungeon(t, fixture.alice)
	if aliceState["zoneId"] != dungeonOutdoorZone {
		t.Fatalf("expected Alice to return outdoors, got zone=%v", aliceState["zoneId"])
	}
	instance := aliceState["instance"].(map[string]any)
	if len(instance) != 0 {
		t.Fatalf("expected no active instance after exit response, got %#v", instance)
	}

	fixture.placePlayer(t, fixture.alice, dungeonOutdoorZone, 550, 324)
	fixture.targetFriendly(t, fixture.alice, dungeonQuestGiver)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"questId":           dungeonQuestID,
	}, http.StatusOK, &aliceState)
	if int(aliceState["experience"].(float64)) != aliceXPBefore+650 {
		t.Fatalf("expected exact dungeon XP reward, got before=%d after=%v", aliceXPBefore, aliceState["experience"])
	}
	if int(aliceState["currencyCopper"].(float64)) != aliceCopperBefore+160 {
		t.Fatalf("expected exact dungeon copper reward, got before=%d after=%v", aliceCopperBefore, aliceState["currencyCopper"])
	}
	if !inventoryHasItem(aliceState, dungeonRewardItem) {
		t.Fatalf("expected dungeon reward item in inventory, got %#v", aliceState["inventory"])
	}
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"questId":           dungeonQuestID,
	}, http.StatusBadRequest, nil)

	_ = fixture.exitDungeon(t, fixture.bob)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/dungeon/reset", nil, map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"dungeonId":         dungeonID,
	}, http.StatusOK, nil)
}

func newDungeonFixture(t *testing.T) *dungeonFixture {
	t.Helper()

	storePath := filepath.Join(t.TempDir(), "dungeon-state.json")
	fileStore, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	mux := http.NewServeMux()
	authn.RegisterRoutes(mux, fileStore)
	realms.RegisterRoutes(mux, fileStore)
	characters.RegisterRoutes(mux, fileStore)
	worlds.RegisterRoutes(mux, fileStore)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var realmResponse map[string][]map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/realms", nil, http.StatusOK, &realmResponse)
	realmID := realmResponse["realms"][0]["id"].(string)

	return &dungeonFixture{
		server:    server,
		fileStore: fileStore,
		realmID:   realmID,
		alice:     createAndConnectSocialPlayer(t, server, realmID, "dungeon_alice", "alice_pass", "Alice"),
		bob:       createAndConnectSocialPlayer(t, server, realmID, "dungeon_bob", "bob_pass", "Bob"),
		cara:      createAndConnectSocialPlayer(t, server, realmID, "dungeon_cara", "cara_pass", "Cara"),
	}
}

func (f *dungeonFixture) prepareDungeonCandidate(t *testing.T, player socialPlayer) {
	t.Helper()

	if _, err := f.fileStore.UpdateCharacterProgression(player.characterID, 4900, 0, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to level dungeon candidate: %v", err)
	}
	f.placePlayer(t, player, dungeonOutdoorZone, 550, 324)
}

func (f *dungeonFixture) placePlayer(t *testing.T, player socialPlayer, zoneID string, x float64, y float64) map[string]any {
	t.Helper()

	if _, err := f.fileStore.UpdateCharacterState(player.characterID, zoneID, x, y, 0); err != nil {
		t.Fatalf("failed to place player %s: %v", player.displayName, err)
	}
	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/reconnect", nil, map[string]any{
		"worldSessionToken": player.worldSessionToken,
	}, http.StatusOK, &state)
	return state
}

func (f *dungeonFixture) worldState(t *testing.T, player socialPlayer) map[string]any {
	t.Helper()
	return getWorldState(t, f.server, player.worldSessionToken)
}

func (f *dungeonFixture) targetFriendly(t *testing.T, player socialPlayer, targetID string) map[string]any {
	t.Helper()

	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/target", nil, map[string]any{
		"worldSessionToken": player.worldSessionToken,
		"targetId":          targetID,
	}, http.StatusOK, &state)
	return state
}

func (f *dungeonFixture) acceptDungeonQuest(t *testing.T, player socialPlayer) {
	t.Helper()

	f.targetFriendly(t, player, dungeonQuestGiver)
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/quest/accept", nil, map[string]any{
		"worldSessionToken": player.worldSessionToken,
		"questId":           dungeonQuestID,
	}, http.StatusOK, nil)
}

func (f *dungeonFixture) formParty(t *testing.T) {
	t.Helper()

	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/party/invite", nil, map[string]any{
		"worldSessionToken": f.alice.worldSessionToken,
		"targetName":        "Bob",
	}, http.StatusOK, nil)
	bobState := socialStateForServer(t, f.server, f.bob.worldSessionToken)
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/party/accept", nil, map[string]any{
		"worldSessionToken": f.bob.worldSessionToken,
		"inviteId":          firstInviteID(t, bobState),
	}, http.StatusOK, nil)
}

func (f *dungeonFixture) enterDungeon(t *testing.T, player socialPlayer) map[string]any {
	t.Helper()

	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/dungeon/enter", nil, map[string]any{
		"worldSessionToken": player.worldSessionToken,
		"dungeonId":         dungeonID,
	}, http.StatusOK, &state)
	return state
}

func (f *dungeonFixture) exitDungeon(t *testing.T, player socialPlayer) map[string]any {
	t.Helper()

	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/dungeon/exit", nil, map[string]any{
		"worldSessionToken": player.worldSessionToken,
	}, http.StatusOK, &state)
	return state
}

func (f *dungeonFixture) movePlayer(t *testing.T, player socialPlayer, x float64, y float64) map[string]any {
	t.Helper()

	state := f.worldState(t, player)
	position := state["position"].(map[string]any)
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/move", nil, map[string]any{
		"worldSessionToken": player.worldSessionToken,
		"deltaX":            x - position["x"].(float64),
		"deltaY":            y - position["y"].(float64),
	}, http.StatusOK, &state)
	return state
}

func (f *dungeonFixture) targetMob(t *testing.T, player socialPlayer, mobID string) map[string]any {
	t.Helper()

	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/target", nil, map[string]any{
		"worldSessionToken": player.worldSessionToken,
		"targetId":          mobID,
	}, http.StatusOK, &state)
	return state
}

func (f *dungeonFixture) killDungeonMob(t *testing.T, mobID string, x float64, y float64, timeout time.Duration) {
	t.Helper()

	f.movePlayer(t, f.alice, x-2, y)
	f.movePlayer(t, f.bob, x+2, y)
	f.targetMob(t, f.alice, mobID)
	f.targetMob(t, f.bob, mobID)
	for _, player := range []socialPlayer{f.alice, f.bob} {
		var state map[string]any
		postJSON(t, f.server.Client(), f.server.URL+"/v1/world/attack/auto", nil, map[string]any{
			"worldSessionToken": player.worldSessionToken,
			"enabled":           true,
		}, http.StatusOK, &state)
	}

	deadline := time.Now().Add(timeout)
	nextAbilityAt := time.Now()
	for time.Now().Before(deadline) {
		state := f.worldState(t, f.alice)
		mob := findHostileMobByID(t, state, mobID)
		if alive, _ := mob["alive"].(bool); !alive {
			return
		}
		if !time.Now().Before(nextAbilityAt) {
			for _, player := range []socialPlayer{f.alice, f.bob} {
				postJSON(t, f.server.Client(), f.server.URL+"/v1/world/attack/ability", nil, map[string]any{
					"worldSessionToken": player.worldSessionToken,
					"abilityId":         "steady_strike",
				}, http.StatusOK, nil)
			}
			nextAbilityAt = time.Now().Add(1700 * time.Millisecond)
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("mob %s did not die before timeout", mobID)
}

func assertDungeonInstance(t *testing.T, left map[string]any, right map[string]any) {
	t.Helper()

	leftInstance := left["instance"].(map[string]any)
	rightInstance := right["instance"].(map[string]any)
	if leftInstance["instanceId"] == "" || leftInstance["instanceId"] != rightInstance["instanceId"] {
		t.Fatalf("expected shared instance, got left=%#v right=%#v", leftInstance, rightInstance)
	}
	if leftInstance["dungeonId"] != dungeonID {
		t.Fatalf("expected dungeon id %s, got %#v", dungeonID, leftInstance)
	}
}

func countHostileMobs(state map[string]any) int {
	entities, _ := state["entities"].([]any)
	count := 0
	for _, value := range entities {
		entity, ok := value.(map[string]any)
		if ok && entity["kind"] == "hostile_mob" {
			count++
		}
	}
	return count
}

func findHostileMobByDisplayName(t *testing.T, state map[string]any, displayName string) map[string]any {
	t.Helper()

	for _, entity := range findHostileMobs(t, state) {
		if entity["displayName"] == displayName {
			return entity
		}
	}
	t.Fatalf("hostile mob %q was not present in state response", displayName)
	return nil
}

func questReady(t *testing.T, state map[string]any, questID string) bool {
	t.Helper()

	quests, ok := state["quests"].([]any)
	if !ok {
		t.Fatalf("state response missing quests: %#v", state["quests"])
	}
	for _, value := range quests {
		quest, ok := value.(map[string]any)
		if !ok || quest["id"] != questID {
			continue
		}
		return quest["state"] == "ready_to_turn_in" || quest["state"] == "completed"
	}
	return false
}

func inventoryHasItem(state map[string]any, itemID string) bool {
	inventory, _ := state["inventory"].(map[string]any)
	slots, _ := inventory["slots"].([]any)
	for _, value := range slots {
		slot, ok := value.(map[string]any)
		if ok && slot["itemId"] == itemID {
			return true
		}
	}
	return false
}
