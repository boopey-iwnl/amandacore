package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/characters"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/realms"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

const (
	professionTrainerNPCID = "npc_talla_grayspark"
	professionTrainerID    = "profession_trainer_talla_grayspark"
	professionFieldAlchemy = "field_alchemy"
	professionWildharvest  = "wildharvest"
)

func TestProfessionLearningSlice(t *testing.T) {
	t.Run("learn profession succeeds", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.moveToProfessionTrainer(t)
		assertProfessionTrainerEntityVisible(t, state)
		assertProfessionTrainerOffer(t, state, platform.ProfessionOrekeepingID, true)

		state = fixture.learnProfession(t, platform.ProfessionOrekeepingID, http.StatusOK)
		assertProfessionLearned(t, state, platform.ProfessionOrekeepingID)
	})

	t.Run("duplicate learn rejects", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.moveToProfessionTrainer(t)

		fixture.learnProfession(t, platform.ProfessionOrekeepingID, http.StatusOK)
		fixture.learnProfession(t, platform.ProfessionOrekeepingID, http.StatusBadRequest)
	})

	t.Run("deferred profession rejects", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.moveToProfessionTrainer(t)

		fixture.learnProfession(t, professionFieldAlchemy, http.StatusBadRequest)
		state := fixture.getWorldState(t)
		assertProfessionAbsent(t, state, professionFieldAlchemy)
	})

	t.Run("primary profession cap works", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.moveToProfessionTrainer(t)

		fixture.learnProfession(t, platform.ProfessionOrekeepingID, http.StatusOK)
		state := fixture.learnProfession(t, platform.ProfessionForgecraftID, http.StatusOK)
		assertProfessionLearned(t, state, platform.ProfessionOrekeepingID)
		assertProfessionLearned(t, state, platform.ProfessionForgecraftID)
		assertProfessionOfferRequirement(t, state, professionWildharvest, "Primary profession limit reached.")

		fixture.learnProfession(t, professionWildharvest, http.StatusBadRequest)
		state = fixture.learnProfession(t, platform.ProfessionFieldAidID, http.StatusOK)
		assertProfessionLearned(t, state, platform.ProfessionFieldAidID)
	})

	t.Run("profession state persists across reconnect and restart", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.moveToProfessionTrainer(t)

		fixture.learnProfession(t, platform.ProfessionOrekeepingID, http.StatusOK)
		state := fixture.learnProfession(t, platform.ProfessionFieldAidID, http.StatusOK)
		assertProfessionLearned(t, state, platform.ProfessionOrekeepingID)
		assertProfessionLearned(t, state, platform.ProfessionFieldAidID)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/disconnect", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
		}, http.StatusOK, nil)
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
		}, http.StatusOK, &state)
		assertProfessionLearned(t, state, platform.ProfessionOrekeepingID)
		assertProfessionLearned(t, state, platform.ProfessionFieldAidID)

		restarted := restartProfessionFixture(t, fixture)
		state = restarted.getWorldState(t)
		assertProfessionLearned(t, state, platform.ProfessionOrekeepingID)
		assertProfessionLearned(t, state, platform.ProfessionFieldAidID)
	})
}

func (f *combatFixture) moveToProfessionTrainer(t *testing.T) map[string]any {
	t.Helper()

	f.moveToPosition(t, 58.0, 24.0)
	return f.targetFriendlyByID(t, professionTrainerNPCID)
}

func (f *combatFixture) learnProfession(t *testing.T, professionID string, expectedStatus int) map[string]any {
	t.Helper()

	var state map[string]any
	target := any(&state)
	if expectedStatus >= http.StatusBadRequest {
		target = nil
	}
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/profession/learn", nil, map[string]any{
		"worldSessionToken": f.worldSessionToken,
		"trainerId":         professionTrainerID,
		"professionId":      professionID,
	}, expectedStatus, target)
	return state
}

func assertProfessionTrainerEntityVisible(t *testing.T, state map[string]any) {
	t.Helper()

	entities, ok := state["entities"].([]any)
	if !ok {
		t.Fatalf("state response missing entities: %#v", state["entities"])
	}
	for _, entityValue := range entities {
		entity, ok := entityValue.(map[string]any)
		if !ok {
			continue
		}
		if entity["id"] == professionTrainerNPCID {
			if entity["kind"] != "profession_trainer_npc" {
				t.Fatalf("expected profession_trainer_npc kind, got %#v", entity)
			}
			if entity["targetable"] != true {
				t.Fatalf("expected profession trainer to be targetable, got %#v", entity)
			}
			assertEntityService(t, entity, "profession_trainer", professionTrainerID)
			return
		}
	}
	t.Fatalf("expected profession trainer entity in world state, got %#v", entities)
}

func assertProfessionTrainerOffer(t *testing.T, state map[string]any, professionID string, canLearn bool) {
	t.Helper()

	offer := requireProfessionTrainerOffer(t, state, professionID)
	if offer["canLearn"].(bool) != canLearn {
		t.Fatalf("expected profession %s canLearn=%v, got %#v", professionID, canLearn, offer)
	}
}

func assertProfessionOfferRequirement(t *testing.T, state map[string]any, professionID string, expected string) {
	t.Helper()

	offer := requireProfessionTrainerOffer(t, state, professionID)
	if offer["requirementText"].(string) != expected {
		t.Fatalf("expected profession %s requirement %q, got %#v", professionID, expected, offer)
	}
}

func requireProfessionTrainerOffer(t *testing.T, state map[string]any, professionID string) map[string]any {
	t.Helper()

	trainer, ok := state["professionTrainer"].(map[string]any)
	if !ok {
		t.Fatalf("state response missing professionTrainer payload: %#v", state["professionTrainer"])
	}
	offers, ok := trainer["offers"].([]any)
	if !ok {
		t.Fatalf("professionTrainer missing offers: %#v", trainer)
	}
	for _, offerValue := range offers {
		offer, ok := offerValue.(map[string]any)
		if !ok {
			continue
		}
		if offer["professionId"] == professionID {
			return offer
		}
	}
	t.Fatalf("expected profession trainer offer %s, got %#v", professionID, offers)
	return nil
}

func assertProfessionLearned(t *testing.T, state map[string]any, professionID string) map[string]any {
	t.Helper()

	profession := findLearnedProfession(t, state, professionID)
	if int(profession["skillValue"].(float64)) != 1 {
		t.Fatalf("expected profession %s to start at skill 1, got %#v", professionID, profession)
	}
	if profession["rankId"].(string) != "novice" {
		t.Fatalf("expected profession %s to start at novice rank, got %#v", professionID, profession)
	}
	return profession
}

func assertProfessionAbsent(t *testing.T, state map[string]any, professionID string) {
	t.Helper()

	if profession := findLearnedProfessionOrNil(t, state, professionID); profession != nil {
		t.Fatalf("expected profession %s to be absent, got %#v", professionID, profession)
	}
}

func findLearnedProfession(t *testing.T, state map[string]any, professionID string) map[string]any {
	t.Helper()

	profession := findLearnedProfessionOrNil(t, state, professionID)
	if profession == nil {
		t.Fatalf("expected learned profession %s in state %#v", professionID, state["professions"])
	}
	return profession
}

func findLearnedProfessionOrNil(t *testing.T, state map[string]any, professionID string) map[string]any {
	t.Helper()

	professions, ok := state["professions"].(map[string]any)
	if !ok {
		t.Fatalf("state response missing professions payload: %#v", state["professions"])
	}
	learned, ok := professions["learned"].([]any)
	if !ok {
		t.Fatalf("professions payload missing learned list: %#v", professions)
	}
	for _, professionValue := range learned {
		profession, ok := professionValue.(map[string]any)
		if !ok {
			continue
		}
		if profession["professionId"] == professionID {
			return profession
		}
	}
	return nil
}

func restartProfessionFixture(t *testing.T, original *combatFixture) *combatFixture {
	t.Helper()

	restartedStore, err := store.NewFileStore(original.storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to reopen store after restart: %v", err)
	}

	mux := http.NewServeMux()
	authn.RegisterRoutes(mux, restartedStore)
	realms.RegisterRoutes(mux, restartedStore)
	characters.RegisterRoutes(mux, restartedStore)
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
		storePath:         original.storePath,
		username:          original.username,
		password:          original.password,
		realmID:           original.realmID,
		characterID:       original.characterID,
		worldSessionToken: connectResponse["worldSessionToken"].(string),
	}
}
