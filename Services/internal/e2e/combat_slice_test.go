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

type combatFixture struct {
	server            *httptest.Server
	fileStore         *store.FileStore
	storePath         string
	username          string
	password          string
	realmID           string
	characterID       string
	worldSessionToken string
}

const (
	stonewakeCombatMob1 = "mob_ditch_rat_01"
	stonewakeCombatMob2 = "mob_ditch_rat_02"
	stonewakeDeathMob   = "mob_bram_kettle_01"
)

func TestCombatSliceHardening(t *testing.T) {
	t.Run("multiple hostile mobs are present", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.getWorldState(t)
		mobs := findHostileMobs(t, state)
		if len(mobs) != 37 {
			t.Fatalf("expected 37 Stonewake hostile/training entities, got %d", len(mobs))
		}
	})

	t.Run("world state preserves authoritative mob order", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.getWorldState(t)
		mobs := findHostileMobs(t, state)
		expected := []string{"mob_training_dummy_01", "mob_training_dummy_02", "mob_training_dummy_03", "mob_ditch_rat_01"}
		for index, expectedID := range expected {
			mob := mobs[index]
			if mob["id"].(string) != expectedID {
				t.Fatalf("expected mob order %v, got %v", expected, []string{
					mobs[0]["id"].(string),
					mobs[1]["id"].(string),
					mobs[2]["id"].(string),
					mobs[3]["id"].(string),
				})
			}
		}
	})

	t.Run("cannot auto attack without a target", func(t *testing.T) {
		fixture := newCombatFixture(t)
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/auto", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"enabled":           true,
		}, http.StatusBadRequest, nil)
	})

	t.Run("cannot use steady strike without a target", func(t *testing.T) {
		fixture := newCombatFixture(t)
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         "steady_strike",
		}, http.StatusBadRequest, nil)
	})

	t.Run("out of range target is rejected", func(t *testing.T) {
		fixture := newCombatFixture(t)
		var state map[string]any
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/move", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"deltaX":            -12.0,
			"deltaY":            -12.0,
		}, http.StatusOK, &state)
		state = fixture.getWorldState(t)
		mob := findHostileMobByID(t, state, stonewakeDeathMob)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/target", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"targetId":          mob["id"].(string),
		}, http.StatusBadRequest, nil)
	})

	t.Run("targeting and combat ranges remain planar even when player z differs", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.getWorldState(t)
		mob := findHostileMobByID(t, state, stonewakeCombatMob1)
		_, err := fixture.fileStore.UpdateCharacterState(
			fixture.characterID,
			"stonewake_vale",
			mob["x"].(float64)-3.0,
			mob["y"].(float64)-2.0,
			50.0)
		if err != nil {
			t.Fatalf("failed to update character z for planar regression: %v", err)
		}

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
		}, http.StatusOK, &state)
		position := state["position"].(map[string]any)
		if position["z"].(float64) != 50.0 {
			t.Fatalf("expected reconnect to restore elevated z for planar regression, got %v", position["z"])
		}

		state = fixture.targetMobByID(t, stonewakeCombatMob1)
		if state["currentTargetId"].(string) != stonewakeCombatMob1 {
			t.Fatalf("expected planar target selection to succeed with elevated z, got %v", state["currentTargetId"])
		}

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/auto", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"enabled":           true,
		}, http.StatusOK, &state)
		if !state["autoAttackActive"].(bool) {
			t.Fatalf("expected planar auto attack to activate with elevated z")
		}
	})

	t.Run("gcd blocks ability spam", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.moveNearMob(t, stonewakeCombatMob1)
		state = fixture.targetMobByID(t, stonewakeCombatMob1)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         "steady_strike",
		}, http.StatusOK, &state)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         "steady_strike",
		}, http.StatusBadRequest, nil)
	})

	t.Run("brace can be used without a target and stabilizes the warrior", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.getWorldState(t)
		healthBefore := state["health"].(float64)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         "brace",
		}, http.StatusOK, &state)
		if state["health"].(float64) < healthBefore {
			t.Fatalf("expected brace to avoid lowering health, got %.1f -> %.1f", healthBefore, state["health"].(float64))
		}
	})

	t.Run("kill cycle clears target stops auto attack and respawns", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.moveNearMob(t, stonewakeCombatMob1)
		state = fixture.targetMobByID(t, stonewakeCombatMob1)
		mob := findHostileMobByID(t, state, stonewakeCombatMob1)
		mobID := stonewakeCombatMob1

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
		state = fixture.getWorldState(t)
		if state["currentTargetId"].(string) != "" {
			t.Fatalf("expected target to clear on mob death, got %v", state["currentTargetId"])
		}
		if state["autoAttackActive"].(bool) {
			t.Fatalf("expected auto attack to stop on mob death")
		}

		waitForMobRespawned(t, fixture.server, fixture.worldSessionToken, mobID, 10*time.Second)
	})

	t.Run("target switching between hostile mobs stays authoritative", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.moveToPosition(t, 89.0, 37.0)
		firstMobID := stonewakeCombatMob1
		secondMobID := stonewakeCombatMob2

		state = fixture.targetMobByID(t, firstMobID)
		if state["currentTargetId"].(string) != firstMobID {
			t.Fatalf("expected first target %s, got %v", firstMobID, state["currentTargetId"])
		}

		state = fixture.targetMobByID(t, secondMobID)
		if state["currentTargetId"].(string) != secondMobID {
			t.Fatalf("expected second target %s, got %v", secondMobID, state["currentTargetId"])
		}
	})

	t.Run("auto attack cleanly switches to the new target without continuing on the old target", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.moveToPosition(t, 89.0, 37.0)
		state = fixture.targetMobByID(t, stonewakeCombatMob1)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/auto", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"enabled":           true,
		}, http.StatusOK, &state)

		firstHealth := waitForMobHealthBelow(
			t,
			fixture.server,
			fixture.worldSessionToken,
			stonewakeCombatMob1,
			findHostileMobByID(t, state, stonewakeCombatMob1)["maxHealth"].(float64),
			4*time.Second)
		state = fixture.targetMobByID(t, stonewakeCombatMob2)
		if state["currentTargetId"].(string) != stonewakeCombatMob2 {
			t.Fatalf("expected target to switch to second mob, got %v", state["currentTargetId"])
		}

		time.Sleep(2200 * time.Millisecond)
		state = fixture.getWorldState(t)
		firstMob := findHostileMobByID(t, state, stonewakeCombatMob1)
		secondMob := findHostileMobByID(t, state, stonewakeCombatMob2)
		if firstMob["health"].(float64) != firstHealth {
			t.Fatalf("expected first mob health to stop changing after retarget, got %.1f then %.1f", firstHealth, firstMob["health"].(float64))
		}
		if secondMob["health"].(float64) >= secondMob["maxHealth"].(float64) {
			t.Fatalf("expected second mob to take auto-attack damage after retarget")
		}
	})

	t.Run("steady strike stays bound to the currently selected target", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.moveToPosition(t, 88.0, 37.0)
		state = fixture.targetMobByID(t, stonewakeCombatMob1)
		firstMob := findHostileMobByID(t, state, stonewakeCombatMob1)
		secondMob := findHostileMobByID(t, state, stonewakeCombatMob2)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         "steady_strike",
		}, http.StatusOK, &state)
		state = fixture.targetMobByID(t, stonewakeCombatMob2)
		if state["currentTargetId"].(string) != stonewakeCombatMob2 {
			t.Fatalf("expected current target to switch during cast, got %v", state["currentTargetId"])
		}

		state = fixture.getWorldState(t)
		firstMob = findHostileMobByID(t, state, stonewakeCombatMob1)
		secondMob = findHostileMobByID(t, state, stonewakeCombatMob2)
		if firstMob["health"].(float64) >= firstMob["maxHealth"].(float64) {
			t.Fatalf("expected steady strike damage to land on the original target")
		}
		if secondMob["health"].(float64) != secondMob["maxHealth"].(float64) {
			t.Fatalf("expected retargeted mob to remain undamaged by the original strike")
		}
	})

	t.Run("reconnect during combat clears combat state and revives dead player", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.getWorldState(t)
		mob := findHostileMobByID(t, state, stonewakeDeathMob)
		state = fixture.moveToPosition(t, mob["x"].(float64)-1.0, mob["y"].(float64)-1.0)
		state = fixture.targetMobByID(t, stonewakeDeathMob)

		waitForPlayerDead(t, fixture.server, fixture.worldSessionToken, 40*time.Second)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/disconnect", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
		}, http.StatusOK, nil)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
		}, http.StatusOK, &state)

		if !state["alive"].(bool) {
			t.Fatalf("expected reconnect to revive dead player")
		}
		if state["health"].(float64) != state["maxHealth"].(float64) {
			t.Fatalf("expected reconnect revive to restore full health, got %.1f", state["health"].(float64))
		}
		if state["resource"].(float64) != state["maxResource"].(float64) {
			t.Fatalf("expected reconnect revive to restore full resource, got %.1f", state["resource"].(float64))
		}
		if state["currentTargetId"].(string) != "" {
			t.Fatalf("expected reconnect to clear target, got %v", state["currentTargetId"])
		}
		if state["autoAttackActive"].(bool) {
			t.Fatalf("expected reconnect to clear auto attack")
		}
		if state["castingAbilityId"].(string) != "" {
			t.Fatalf("expected reconnect to clear cast state, got %v", state["castingAbilityId"])
		}

		mob = findHostileMobByID(t, state, stonewakeDeathMob)
		switch mob["aiState"].(string) {
		case "alerted", "chasing", "attacking":
			t.Fatalf("expected reconnect to return mob to a non-engaged state, got %v", mob["aiState"])
		}
	})
}

func newCombatFixture(t *testing.T) *combatFixture {
	t.Helper()

	storePath := filepath.Join(t.TempDir(), "combat-state.json")
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

	username := "combat_user"
	password := "combat_pass"

	postJSON(t, server.Client(), server.URL+"/v1/accounts/register", nil, map[string]string{
		"username": username,
		"password": password,
	}, http.StatusCreated, nil)

	var loginResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/auth/login", nil, map[string]string{
		"username": username,
		"password": password,
	}, http.StatusOK, &loginResponse)
	accessToken := loginResponse["accessToken"].(string)

	var realmsResponse map[string][]map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/realms", nil, http.StatusOK, &realmsResponse)
	realmID := realmsResponse["realms"][0]["id"].(string)

	var characterResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/characters", bearer(accessToken), map[string]string{
		"realmId":     realmID,
		"displayName": "Combator",
		"raceId":      "human",
		"classId":     "warrior",
		"archetypeId": "wayfarer_warden",
	}, http.StatusCreated, &characterResponse)
	if characterResponse["raceId"].(string) != "human" {
		t.Fatalf("expected combat fixture race human, got %v", characterResponse["raceId"])
	}
	if characterResponse["classId"].(string) != "warrior" {
		t.Fatalf("expected combat fixture class warrior, got %v", characterResponse["classId"])
	}
	characterID := characterResponse["id"].(string)

	var ticketResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/world/join-ticket", bearer(accessToken), map[string]string{
		"realmId":     realmID,
		"characterId": characterID,
	}, http.StatusCreated, &ticketResponse)

	var connectResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/world/connect", nil, map[string]string{
		"ticketId": ticketResponse["ticketId"].(string),
	}, http.StatusCreated, &connectResponse)

	return &combatFixture{
		server:            server,
		fileStore:         fileStore,
		storePath:         storePath,
		username:          username,
		password:          password,
		realmID:           realmID,
		characterID:       characterID,
		worldSessionToken: connectResponse["worldSessionToken"].(string),
	}
}

func (f *combatFixture) getWorldState(t *testing.T) map[string]any {
	t.Helper()

	return getWorldState(t, f.server, f.worldSessionToken)
}

func (f *combatFixture) moveToPosition(t *testing.T, targetX float64, targetY float64) map[string]any {
	t.Helper()

	state := f.getWorldState(t)
	position := state["position"].(map[string]any)

	var movedState map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/move", nil, map[string]any{
		"worldSessionToken": f.worldSessionToken,
		"deltaX":            targetX - position["x"].(float64),
		"deltaY":            targetY - position["y"].(float64),
	}, http.StatusOK, &movedState)
	return movedState
}

func (f *combatFixture) moveNearMob(t *testing.T, mobID string) map[string]any {
	t.Helper()

	state := f.getWorldState(t)
	mob := findHostileMobByID(t, state, mobID)
	return f.moveToPosition(t, mob["x"].(float64)-3.0, mob["y"].(float64)-2.0)
}

func (f *combatFixture) targetMobByID(t *testing.T, mobID string) map[string]any {
	t.Helper()

	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/target", nil, map[string]any{
		"worldSessionToken": f.worldSessionToken,
		"targetId":          mobID,
	}, http.StatusOK, &state)
	return state
}

func (f *combatFixture) targetFriendlyByID(t *testing.T, friendlyID string) map[string]any {
	t.Helper()

	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/target", nil, map[string]any{
		"worldSessionToken": f.worldSessionToken,
		"targetId":          friendlyID,
	}, http.StatusOK, &state)
	return state
}

func findHostileMobs(t *testing.T, state map[string]any) []map[string]any {
	t.Helper()

	entities, ok := state["entities"].([]any)
	if !ok {
		t.Fatalf("state response missing entities: %#v", state["entities"])
	}

	mobs := make([]map[string]any, 0)
	for _, entityValue := range entities {
		entity, ok := entityValue.(map[string]any)
		if !ok {
			continue
		}
		if entity["kind"] == "hostile_mob" {
			mobs = append(mobs, entity)
		}
	}

	if len(mobs) == 0 {
		t.Fatalf("hostile mobs were not present in state response")
	}

	return mobs
}

func findHostileMobByID(t *testing.T, state map[string]any, mobID string) map[string]any {
	t.Helper()

	for _, entity := range findHostileMobs(t, state) {
		if entity["id"] == mobID {
			return entity
		}
	}

	t.Fatalf("hostile mob %s was not present in state response", mobID)
	return nil
}

func getWorldState(t *testing.T, server *httptest.Server, worldSessionToken string) map[string]any {
	t.Helper()

	var state map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/world/state?worldSessionToken="+worldSessionToken, nil, http.StatusOK, &state)
	return state
}

func waitForMobHealthBelow(t *testing.T, server *httptest.Server, worldSessionToken string, mobID string, threshold float64, timeout time.Duration) float64 {
	t.Helper()

	deadline := time.Now().Add(timeout)
	lastHealth := threshold
	for time.Now().Before(deadline) {
		state := getWorldState(t, server, worldSessionToken)
		mob := findHostileMobByID(t, state, mobID)
		lastHealth = mob["health"].(float64)
		if lastHealth < threshold {
			return lastHealth
		}
		time.Sleep(250 * time.Millisecond)
	}

	return lastHealth
}

func waitForMobDead(t *testing.T, server *httptest.Server, worldSessionToken string, mobID string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state := getWorldState(t, server, worldSessionToken)
		mob := findHostileMobByID(t, state, mobID)
		if alive, ok := mob["alive"].(bool); ok && !alive {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("mob %s did not die before timeout", mobID)
}

func waitForMobRespawned(t *testing.T, server *httptest.Server, worldSessionToken string, mobID string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state := getWorldState(t, server, worldSessionToken)
		mob := findHostileMobByID(t, state, mobID)
		alive, _ := mob["alive"].(bool)
		health, _ := mob["health"].(float64)
		maxHealth, _ := mob["maxHealth"].(float64)
		if alive && health == maxHealth {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("mob %s did not respawn before timeout", mobID)
}

func waitForPlayerDead(t *testing.T, server *httptest.Server, worldSessionToken string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state := getWorldState(t, server, worldSessionToken)
		alive, _ := state["alive"].(bool)
		if !alive {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("player did not die before timeout")
}
