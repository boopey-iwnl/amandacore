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
	housingEntranceEntityID = "obj_stonewake_room_door"
	housingZoneID           = "house_personal_room"
)

func TestHousingAccessStorageAndReconnectSlice(t *testing.T) {
	fixture := newCombatFixture(t)

	state := fixture.getWorldState(t)
	entrance := findVisibleEntityByID(t, state, housingEntranceEntityID)
	assertEntityService(t, entrance, "housing", "personal_room")

	status := fixture.housingStatus(t)
	if status["unlocked"] != true || status["housingSpaceId"].(string) == "" {
		t.Fatalf("expected lazy housing entitlement, got %#v", status)
	}

	state = fixture.enterHousing(t)
	if state["zoneId"].(string) != housingZoneID {
		t.Fatalf("expected to enter housing zone, got %#v", state["zoneId"])
	}
	assertHousingInState(t, state, true)
	findVisibleEntityByID(t, state, "obj_personal_storage_chest")
	findVisibleEntityByID(t, state, "obj_personal_room_exit")

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/housing/storage/deposit", nil, map[string]any{
		"worldSessionToken":  fixture.worldSessionToken,
		"inventorySlotIndex": 0,
		"storageSlotIndex":   0,
		"stackCount":         2,
	}, http.StatusOK, &state)
	assertInventoryItemCount(t, state, "camp_ration", 1)
	assertHousingStorageItemCount(t, state, "camp_ration", 2)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/housing/storage/withdraw", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"storageSlotIndex":  0,
		"stackCount":        1,
	}, http.StatusOK, &state)
	assertInventoryItemCount(t, state, "camp_ration", 2)
	assertHousingStorageItemCount(t, state, "camp_ration", 1)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/housing/storage/move", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"fromSlotIndex":     0,
		"toSlotIndex":       3,
	}, http.StatusOK, &state)
	assertHousingStorageSlot(t, state, 3, "camp_ration", 1)

	fillCharacterInventoryForHousingTest(t, fixture)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
	}, http.StatusOK, &state)
	if state["zoneId"].(string) == housingZoneID {
		t.Fatalf("expected reconnect from housing to return outside, got %#v", state["zoneId"])
	}

	state = fixture.enterHousing(t)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/housing/storage/withdraw", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"storageSlotIndex":  3,
		"stackCount":        1,
	}, http.StatusBadRequest, nil)
	state = fixture.getWorldState(t)
	assertHousingStorageItemCount(t, state, "camp_ration", 1)
	assertInventoryItemCount(t, state, "camp_ration", 0)

	restarted := restartHousingFixture(t, fixture)
	state = restarted.enterHousing(t)
	assertHousingStorageItemCount(t, state, "camp_ration", 1)

	state = restarted.leaveHousing(t)
	if state["zoneId"].(string) != "stonewake_vale" {
		t.Fatalf("expected housing exit to return outdoors, got %#v", state["zoneId"])
	}
}

func TestHousingDecorationPlacementAndPersistence(t *testing.T) {
	fixture := newCombatFixture(t)
	state := fixture.enterHousing(t)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/housing/decorations/place", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"decorationId":      "simple_cot",
		"x":                 12,
		"y":                 10,
		"z":                 0,
	}, http.StatusOK, &state)
	placements := housingPlacements(t, state)
	if len(placements) != 1 {
		t.Fatalf("expected one placed decoration, got %#v", placements)
	}
	firstPlacementID := placements[0].(map[string]any)["placementId"].(string)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/housing/decorations/place", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"decorationId":      "small_table",
		"x":                 100,
		"y":                 100,
		"z":                 0,
	}, http.StatusBadRequest, nil)
	state = fixture.getWorldState(t)
	if len(housingPlacements(t, state)) != 1 {
		t.Fatalf("outside-bounds placement changed decoration state: %#v", housingPlacements(t, state))
	}

	for index := 0; index < platform.HousingDecorationLimit-1; index++ {
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/housing/decorations/place", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"decorationId":      "supply_crate",
			"x":                 14 + index,
			"y":                 12,
			"z":                 0,
		}, http.StatusOK, &state)
	}
	if len(housingPlacements(t, state)) != platform.HousingDecorationLimit {
		t.Fatalf("expected decoration limit count, got %#v", housingPlacements(t, state))
	}

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/housing/decorations/place", nil, map[string]any{
		"worldSessionToken": fixture.worldSessionToken,
		"decorationId":      "wall_torch",
		"x":                 20,
		"y":                 14,
		"z":                 0,
	}, http.StatusBadRequest, nil)

	restarted := restartHousingFixture(t, fixture)
	state = restarted.enterHousing(t)
	if len(housingPlacements(t, state)) != platform.HousingDecorationLimit {
		t.Fatalf("expected decorations to persist after restart, got %#v", housingPlacements(t, state))
	}

	postJSON(t, restarted.server.Client(), restarted.server.URL+"/v1/world/housing/decorations/remove", nil, map[string]any{
		"worldSessionToken": restarted.worldSessionToken,
		"placementId":       firstPlacementID,
	}, http.StatusOK, &state)
	if len(housingPlacements(t, state)) != platform.HousingDecorationLimit-1 {
		t.Fatalf("expected remove to reduce placed decorations, got %#v", housingPlacements(t, state))
	}
}

func (f *combatFixture) housingStatus(t *testing.T) map[string]any {
	t.Helper()

	var status map[string]any
	getJSON(t, f.server.Client(), f.server.URL+"/v1/world/housing/status?worldSessionToken="+f.worldSessionToken, nil, http.StatusOK, &status)
	return status
}

func (f *combatFixture) enterHousing(t *testing.T) map[string]any {
	t.Helper()

	state := f.getWorldState(t)
	if state["zoneId"].(string) != "stonewake_vale" {
		return state
	}
	entrance := findVisibleEntityByID(t, state, housingEntranceEntityID)
	f.moveToPosition(t, entrance["x"].(float64), entrance["y"].(float64))

	var entered map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/housing/enter", nil, map[string]any{
		"worldSessionToken": f.worldSessionToken,
	}, http.StatusOK, &entered)
	return entered
}

func (f *combatFixture) leaveHousing(t *testing.T) map[string]any {
	t.Helper()

	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/housing/leave", nil, map[string]any{
		"worldSessionToken": f.worldSessionToken,
	}, http.StatusOK, &state)
	return state
}

func restartHousingFixture(t *testing.T, original *combatFixture) *combatFixture {
	t.Helper()

	original.server.Close()
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
		fileStore:         restartedStore,
		storePath:         original.storePath,
		username:          original.username,
		password:          original.password,
		realmID:           original.realmID,
		characterID:       original.characterID,
		worldSessionToken: connectResponse["worldSessionToken"].(string),
	}
}

func fillCharacterInventoryForHousingTest(t *testing.T, fixture *combatFixture) {
	t.Helper()

	slots := make([]platform.CharacterInventorySlot, platform.InventorySlotCount)
	for slotIndex := range slots {
		slots[slotIndex] = platform.CharacterInventorySlot{
			SlotIndex:   slotIndex,
			ItemID:      "field_boots",
			DisplayName: "Field Boots",
			StackCount:  1,
		}
	}
	if _, err := fixture.fileStore.UpdateCharacterInventory(fixture.characterID, slots); err != nil {
		t.Fatalf("failed to fill character inventory: %v", err)
	}
}

func assertHousingInState(t *testing.T, state map[string]any, expected bool) {
	t.Helper()

	housing, ok := state["housing"].(map[string]any)
	if !ok {
		t.Fatalf("state missing housing payload: %#v", state["housing"])
	}
	if housing["inHousing"].(bool) != expected {
		t.Fatalf("expected inHousing=%v, got %#v", expected, housing)
	}
}

func assertHousingStorageItemCount(t *testing.T, state map[string]any, itemID string, expectedCount int) {
	t.Helper()

	total := 0
	for _, slotValue := range housingStorageSlots(t, state) {
		slot := slotValue.(map[string]any)
		if slot["itemId"].(string) == itemID {
			total += int(slot["stackCount"].(float64))
		}
	}
	if total != expectedCount {
		t.Fatalf("expected housing storage to contain %s x%d, got x%d in %#v", itemID, expectedCount, total, housingStorageSlots(t, state))
	}
}

func assertHousingStorageSlot(t *testing.T, state map[string]any, slotIndex int, itemID string, stackCount int) {
	t.Helper()

	slots := housingStorageSlots(t, state)
	slot := slots[slotIndex].(map[string]any)
	if slot["itemId"].(string) != itemID || int(slot["stackCount"].(float64)) != stackCount {
		t.Fatalf("expected housing storage slot %d to hold %s x%d, got %#v", slotIndex, itemID, stackCount, slot)
	}
}

func housingStorageSlots(t *testing.T, state map[string]any) []any {
	t.Helper()

	housing := state["housing"].(map[string]any)
	storage := housing["storage"].(map[string]any)
	slots, ok := storage["slots"].([]any)
	if !ok {
		t.Fatalf("housing storage missing slots: %#v", storage)
	}
	return slots
}

func housingPlacements(t *testing.T, state map[string]any) []any {
	t.Helper()

	housing := state["housing"].(map[string]any)
	decorations := housing["decorations"].(map[string]any)
	placed, ok := decorations["placed"].([]any)
	if !ok {
		t.Fatalf("housing decorations missing placed list: %#v", decorations)
	}
	return placed
}
