package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/realms"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

func TestClassProgressionMilestoneSlice(t *testing.T) {
	t.Run("trainer offers unlock by level with ability metadata", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingExperience: 100, startingCopper: 25})
		state := targetWarriorTrainer(t, fixture)
		offers := requireTrainerOffers(t, requireTrainerState(t, state))

		drivingBlow := offers[platform.DrivingBlowAbilityID]
		if drivingBlow == nil || !drivingBlow["canLearn"].(bool) {
			t.Fatalf("expected level 2 Driving Blow to be learnable, got %#v", drivingBlow)
		}
		rallyingCall := offers[platform.RallyingCallAbilityID]
		if rallyingCall == nil || rallyingCall["canLearn"].(bool) {
			t.Fatalf("expected level 4 Rallying Call to remain locked at level 2, got %#v", rallyingCall)
		}
		if rallyingCall["resourceName"].(string) != "Grit" || rallyingCall["cooldownMs"].(float64) <= 0 {
			t.Fatalf("expected trainer offer to include Grit and cooldown metadata, got %#v", rallyingCall)
		}
	})

	t.Run("resource generation spending cooldown and invalid ability use", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingExperience: 100, startingCopper: 25})
		targetWarriorTrainer(t, fixture)

		var state map[string]any
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/trainer/learn", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"trainerId":         "trainer_armsmaster_corin_vale",
			"abilityId":         platform.DrivingBlowAbilityID,
		}, http.StatusOK, &state)

		const resourceTestMobID = "mob_field_boar_01"
		state = fixture.moveNearMob(t, resourceTestMobID)
		state = fixture.targetMobByID(t, resourceTestMobID)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         platform.DrivingBlowAbilityID,
		}, http.StatusBadRequest, nil)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         platform.SteadyStrikeAbilityID,
		}, http.StatusOK, &state)
		if state["resource"].(float64) <= 0 {
			t.Fatalf("expected Steady Strike to generate Grit, got %#v", state["resource"])
		}
		if state["globalCooldownEndsAt"].(float64) <= 0 {
			t.Fatalf("expected global cooldown to be set, got %#v", state["globalCooldownEndsAt"])
		}

		time.Sleep(1600 * time.Millisecond)
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         platform.SteadyStrikeAbilityID,
		}, http.StatusOK, &state)
		time.Sleep(1600 * time.Millisecond)
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         platform.SteadyStrikeAbilityID,
		}, http.StatusOK, &state)
		gritBeforeSpend := state["resource"].(float64)

		time.Sleep(1600 * time.Millisecond)
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/attack/ability", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"abilityId":         platform.DrivingBlowAbilityID,
		}, http.StatusOK, &state)
		if state["resource"].(float64) >= gritBeforeSpend {
			t.Fatalf("expected Driving Blow to spend Grit, got %.1f from %.1f", state["resource"].(float64), gritBeforeSpend)
		}
	})

	t.Run("talent point grant selection and restart persistence", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingExperience: 1400, startingCopper: 25})
		state := fixture.getWorldState(t)
		talents := requireTalentPayload(t, state)
		if talents["pointsGranted"].(float64) != 1 || talents["pointsAvailable"].(float64) != 1 {
			t.Fatalf("expected one available talent point at level 6, got %#v", talents)
		}

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/talent/select", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"talentId":          "balanced_grip",
		}, http.StatusOK, &state)
		talents = requireTalentPayload(t, state)
		if talents["pointsAvailable"].(float64) != 0 {
			t.Fatalf("expected selected talent to spend point, got %#v", talents)
		}
		assertTalentRank(t, talents, "balanced_grip", 1)

		restarted := restartClassProgressionFixture(t, fixture)
		state = restarted.getWorldState(t)
		assertTalentRank(t, requireTalentPayload(t, state), "balanced_grip", 1)
	})

	t.Run("equipment stats alter character combat stats", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.getWorldState(t)
		baseStats := state["stats"].(map[string]any)
		baseAttackPower := baseStats["attackPower"].(float64)

		state = targetQuartermasterMira(t, fixture)
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/vendor/buy", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"vendorId":          vendorQuartermasterMiraID,
			"itemId":            itemWornMilitiaBladeID,
			"stackCount":        1,
		}, http.StatusOK, &state)
		bladeSlot := assertInventoryItemCount(t, state, itemWornMilitiaBladeID, 1)
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/inventory/equip", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"slotIndex":         bladeSlot,
		}, http.StatusOK, &state)

		stats := state["stats"].(map[string]any)
		if stats["attackPower"].(float64) <= baseAttackPower {
			t.Fatalf("expected equipped weapon to increase attack power, got %.1f <= %.1f", stats["attackPower"].(float64), baseAttackPower)
		}
	})
}

func requireTalentPayload(t *testing.T, state map[string]any) map[string]any {
	t.Helper()

	talents, ok := state["talents"].(map[string]any)
	if !ok {
		t.Fatalf("state response missing talents payload: %#v", state["talents"])
	}
	return talents
}

func assertTalentRank(t *testing.T, talents map[string]any, talentID string, expectedRank int) {
	t.Helper()

	entries := talents["talents"].([]any)
	for _, entryValue := range entries {
		entry := entryValue.(map[string]any)
		if entry["id"].(string) != talentID {
			continue
		}
		if int(entry["rank"].(float64)) != expectedRank {
			t.Fatalf("expected %s rank %d, got %#v", talentID, expectedRank, entry)
		}
		return
	}
	t.Fatalf("expected talent %s in %#v", talentID, entries)
}

func restartClassProgressionFixture(t *testing.T, original *combatFixture) *combatFixture {
	t.Helper()

	original.server.Close()
	restartedStore, err := store.NewFileStore(original.storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to reopen store after restart: %v", err)
	}

	mux := http.NewServeMux()
	authn.RegisterRoutes(mux, restartedStore)
	realms.RegisterRoutes(mux, restartedStore)
	worlds.RegisterRoutes(mux, restartedStore)

	serverAfterRestart := httptest.NewServer(mux)
	t.Cleanup(serverAfterRestart.Close)

	var loginResponse map[string]any
	postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/auth/login", nil, map[string]string{
		"username": original.username,
		"password": original.password,
	}, http.StatusOK, &loginResponse)
	accessToken := loginResponse["accessToken"].(string)

	var ticketResponse map[string]any
	postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/world/join-ticket", bearer(accessToken), map[string]string{
		"realmId":     original.realmID,
		"characterId": original.characterID,
	}, http.StatusCreated, &ticketResponse)

	var connectResponse map[string]any
	postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/world/connect", nil, map[string]string{
		"ticketId": ticketResponse["ticketId"].(string),
	}, http.StatusCreated, &connectResponse)

	return &combatFixture{
		server:            serverAfterRestart,
		fileStore:         restartedStore,
		storePath:         original.storePath,
		username:          original.username,
		password:          original.password,
		realmID:           original.realmID,
		characterID:       original.characterID,
		worldSessionToken: connectResponse["worldSessionToken"].(string),
	}
}
