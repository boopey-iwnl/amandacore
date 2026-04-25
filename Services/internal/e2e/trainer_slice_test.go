package e2e

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/characters"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/realms"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

func TestWarriorTrainerSlice(t *testing.T) {
	t.Run("trainer offers are returned for a warrior", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingCopper: 25})
		state := fixture.getWorldState(t)
		trainer := requireTrainerState(t, state)
		if trainer["id"].(string) != "trainer_armsmaster_corin_vale" {
			t.Fatalf("expected warrior trainer id, got %v", trainer["id"])
		}
		assertTrainerEntityVisible(t, state)
		assertQuestGiverEntityVisible(t, state)
		offers := requireTrainerOffers(t, trainer)
		if _, ok := offers["driving_blow"]; !ok {
			t.Fatalf("expected driving_blow trainer offer, got %#v", offers)
		}
		if _, ok := offers[platform.RallyingCallAbilityID]; !ok {
			t.Fatalf("expected rallying_call trainer offer, got %#v", offers)
		}
	})

	t.Run("successful learn of driving blow updates spellbook without auto-placing onto bars", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingExperience: 100, startingCopper: 25})
		fixture.targetFriendlyByID(t, "trainer_armsmaster_corin_vale")

		var response map[string]any
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/trainer/learn", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"trainerId":         "trainer_armsmaster_corin_vale",
			"abilityId":         platform.DrivingBlowAbilityID,
		}, http.StatusOK, &response)

		if response["currencyCopper"].(float64) != 15 {
			t.Fatalf("expected copper to decrease to 15, got %v", response["currencyCopper"])
		}
		assertAbilityLearned(t, response, platform.DrivingBlowAbilityID)
		assertActionBarSlotEmpty(t, response, 3)
	})

	t.Run("trainer learn rejects wrong class", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{
			classID:        "mystic",
			archetypeID:    "mystic",
			startingCopper: 25,
		})
		fixture.targetFriendlyByID(t, "trainer_armsmaster_corin_vale")

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/trainer/learn", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"trainerId":         "trainer_armsmaster_corin_vale",
			"abilityId":         platform.DrivingBlowAbilityID,
		}, http.StatusBadRequest, nil)
	})

	t.Run("trainer learn rejects insufficient funds", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingExperience: 100, startingCopper: 5})
		fixture.targetFriendlyByID(t, "trainer_armsmaster_corin_vale")

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/trainer/learn", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"trainerId":         "trainer_armsmaster_corin_vale",
			"abilityId":         platform.DrivingBlowAbilityID,
		}, http.StatusBadRequest, nil)
	})

	t.Run("trainer learn rejects missing trainer target context", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingExperience: 100, startingCopper: 25})

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/trainer/learn", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"trainerId":         "trainer_armsmaster_corin_vale",
			"abilityId":         platform.DrivingBlowAbilityID,
		}, http.StatusBadRequest, nil)
	})

	t.Run("trainer learn rejects already learned abilities", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingExperience: 100, startingCopper: 25})
		fixture.targetFriendlyByID(t, "trainer_armsmaster_corin_vale")

		var response map[string]any
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/trainer/learn", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"trainerId":         "trainer_armsmaster_corin_vale",
			"abilityId":         platform.DrivingBlowAbilityID,
		}, http.StatusOK, &response)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/trainer/learn", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"trainerId":         "trainer_armsmaster_corin_vale",
			"abilityId":         platform.DrivingBlowAbilityID,
		}, http.StatusBadRequest, nil)
	})

	t.Run("trainer learn rejects abilities below required level", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingCopper: 40})
		fixture.targetFriendlyByID(t, "trainer_armsmaster_corin_vale")

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/trainer/learn", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"trainerId":         "trainer_armsmaster_corin_vale",
			"abilityId":         platform.WarCryAbilityID,
		}, http.StatusBadRequest, nil)
	})

	t.Run("learned trainer abilities persist across reconnect and restart", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingExperience: 100, startingCopper: 25})
		fixture.targetFriendlyByID(t, "trainer_armsmaster_corin_vale")

		var response map[string]any
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/trainer/learn", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"trainerId":         "trainer_armsmaster_corin_vale",
			"abilityId":         platform.DrivingBlowAbilityID,
		}, http.StatusOK, &response)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/disconnect", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
		}, http.StatusOK, nil)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
		}, http.StatusOK, &response)
		assertAbilityLearned(t, response, platform.DrivingBlowAbilityID)
		assertActionBarSlotEmpty(t, response, 3)

		restartedStore, err := store.NewFileStore(fixture.storePath, "test-build", "http://world.local")
		if err != nil {
			t.Fatalf("failed to reopen store after restart: %v", err)
		}

		mux := http.NewServeMux()
		authn.RegisterRoutes(mux, restartedStore)
		realms.RegisterRoutes(mux, restartedStore)
		characters.RegisterRoutes(mux, restartedStore)
		worlds.RegisterRoutes(mux, restartedStore)

		serverAfterRestart := httptest.NewServer(mux)
		defer serverAfterRestart.Close()

		var loginResponse map[string]any
		postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/auth/login", nil, map[string]string{
			"username": fixture.username,
			"password": fixture.password,
		}, http.StatusOK, &loginResponse)
		accessToken := loginResponse["accessToken"].(string)

		var ticketResponse map[string]any
		postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/world/join-ticket", bearer(accessToken), map[string]string{
			"realmId":     fixture.realmID,
			"characterId": fixture.characterID,
		}, http.StatusCreated, &ticketResponse)

		postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/world/connect", nil, map[string]string{
			"ticketId": ticketResponse["ticketId"].(string),
		}, http.StatusCreated, &response)
		assertAbilityLearned(t, response, platform.DrivingBlowAbilityID)
		assertActionBarSlotEmpty(t, response, 3)
	})
}

type trainerFixtureOptions struct {
	classID            string
	archetypeID        string
	startingExperience int
	startingCopper     int
}

func newTrainerFixture(t *testing.T, options trainerFixtureOptions) *combatFixture {
	t.Helper()

	storePath := filepath.Join(t.TempDir(), "trainer-state.json")
	fileStore, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to create trainer store: %v", err)
	}

	mux := http.NewServeMux()
	authn.RegisterRoutes(mux, fileStore)
	realms.RegisterRoutes(mux, fileStore)
	characters.RegisterRoutes(mux, fileStore)
	worlds.RegisterRoutes(mux, fileStore)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	username := "trainer_user"
	password := "trainer_pass"

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

	classID := options.classID
	if classID == "" {
		classID = platform.DefaultClassID
	}
	archetypeID := options.archetypeID
	if archetypeID == "" {
		archetypeID = platform.LegacyWayfarerArchetypeID
	}

	var characterResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/characters", bearer(accessToken), map[string]string{
		"realmId":     realmID,
		"displayName": "TrainerHero",
		"raceId":      platform.DefaultRaceID,
		"classId":     classID,
		"archetypeId": archetypeID,
	}, http.StatusCreated, &characterResponse)
	characterID := characterResponse["id"].(string)

	if options.startingCopper > 0 || options.startingExperience > 0 {
		character, err := fileStore.GetCharacterByID(characterID)
		if err != nil {
			t.Fatalf("failed to reload created character: %v", err)
		}
		if _, err := fileStore.UpdateCharacterProgression(
			characterID,
			options.startingExperience,
			options.startingCopper,
			character.Inventory,
			character.LearnedAbilityIDs,
			character.ActionBarSlots,
			character.Quests,
		); err != nil {
			t.Fatalf("failed to seed copper for trainer fixture: %v", err)
		}
	}

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

func requireTrainerState(t *testing.T, state map[string]any) map[string]any {
	t.Helper()

	trainer, ok := state["trainer"].(map[string]any)
	if !ok {
		t.Fatalf("state response missing trainer payload: %#v", state["trainer"])
	}
	return trainer
}

func requireTrainerOffers(t *testing.T, trainer map[string]any) map[string]map[string]any {
	t.Helper()

	offerValues, ok := trainer["offers"].([]any)
	if !ok {
		t.Fatalf("trainer payload missing offers: %#v", trainer["offers"])
	}

	offers := make(map[string]map[string]any, len(offerValues))
	for _, offerValue := range offerValues {
		offer, ok := offerValue.(map[string]any)
		if !ok {
			continue
		}
		offers[offer["abilityId"].(string)] = offer
	}
	return offers
}

func assertAbilityLearned(t *testing.T, response map[string]any, abilityID string) {
	t.Helper()

	learnedAbilityIds := response["learnedAbilityIds"].([]any)
	found := false
	for _, learnedAbilityID := range learnedAbilityIds {
		if learnedAbilityID.(string) == abilityID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected learned abilities to include %s, got %#v", abilityID, learnedAbilityIds)
	}

	spellbook := response["spellbook"].([]any)
	for _, entryValue := range spellbook {
		entry := entryValue.(map[string]any)
		if entry["id"].(string) != abilityID {
			continue
		}
		if !entry["learned"].(bool) {
			t.Fatalf("expected spellbook entry %s to be learned, got %#v", abilityID, entry)
		}
		return
	}

	t.Fatalf("expected spellbook to include %s", abilityID)
}

func assertActionBarSlotAbility(t *testing.T, response map[string]any, slotIndex int, abilityID string) {
	t.Helper()

	actionBar := response["actionBar"].([]any)
	if len(actionBar) <= slotIndex {
		t.Fatalf("expected action bar to include slot %d, got %d slots", slotIndex, len(actionBar))
	}

	slot := actionBar[slotIndex].(map[string]any)
	if slot["abilityId"].(string) != abilityID {
		t.Fatalf("expected action bar slot %d to be %s, got %#v", slotIndex, abilityID, slot)
	}
}

func assertTrainerEntityVisible(t *testing.T, response map[string]any) {
	t.Helper()

	entities, ok := response["entities"].([]any)
	if !ok {
		t.Fatalf("state response missing entities: %#v", response["entities"])
	}
	for _, entityValue := range entities {
		entity, ok := entityValue.(map[string]any)
		if !ok {
			continue
		}
		if entity["id"] == "trainer_armsmaster_corin_vale" {
			if entity["kind"] != "trainer_npc" {
				t.Fatalf("expected trainer_npc kind for trainer entity, got %#v", entity)
			}
			if entity["targetable"] != true {
				t.Fatalf("expected trainer entity to be targetable, got %#v", entity)
			}
			if entity["displayName"] != "Armsmaster Corin Vale" {
				t.Fatalf("expected trainer display name, got %#v", entity)
			}
			assertEntityService(t, entity, "trainer", "trainer_armsmaster_corin_vale")
			return
		}
	}

	t.Fatalf("expected trainer NPC entity in world state, got %#v", entities)
}

func assertQuestGiverEntityVisible(t *testing.T, response map[string]any) {
	t.Helper()

	entities, ok := response["entities"].([]any)
	if !ok {
		t.Fatalf("state response missing entities: %#v", response["entities"])
	}
	for _, entityValue := range entities {
		entity, ok := entityValue.(map[string]any)
		if !ok {
			continue
		}
		if entity["id"] == "npc_commander_elian_rook" {
			if entity["kind"] != "quest_giver_npc" {
				t.Fatalf("expected quest_giver_npc kind for quest giver entity, got %#v", entity)
			}
			if entity["targetable"] != true {
				t.Fatalf("expected quest giver entity to be targetable, got %#v", entity)
			}
			if entity["displayName"] != "Commander Elian Rook" {
				t.Fatalf("expected quest giver display name, got %#v", entity)
			}
			assertEntityService(t, entity, "quest", "sv_first_muster")
			return
		}
	}

	t.Fatalf("expected quest giver NPC entity in world state, got %#v", entities)
}

func assertEntityService(t *testing.T, entity map[string]any, serviceType string, serviceID string) {
	t.Helper()

	services, ok := entity["npcServices"].([]any)
	if !ok || len(services) == 0 {
		t.Fatalf("expected entity %v to expose npcServices, got %#v", entity["id"], entity["npcServices"])
	}
	for _, serviceValue := range services {
		service, ok := serviceValue.(map[string]any)
		if !ok {
			continue
		}
		if service["type"] == serviceType && service["serviceId"] == serviceID {
			return
		}
	}
	t.Fatalf("expected entity %v to expose service %s/%s, got %#v", entity["id"], serviceType, serviceID, services)
}
