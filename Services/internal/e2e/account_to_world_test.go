package e2e

import (
	"bytes"
	"encoding/json"
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

func TestAccountToWorldGoldenPath(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
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
	defer server.Close()

	username := "proof_user"
	password := "proof_pass"

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

	var realmResponse map[string][]map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/realms", nil, http.StatusOK, &realmResponse)
	realmID := realmResponse["realms"][0]["id"].(string)

	var characterResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/characters", bearer(accessToken), map[string]string{
		"realmId":     realmID,
		"displayName": "Runner",
		"raceId":      "human",
		"classId":     "warrior",
		"archetypeId": "wayfarer_warden",
	}, http.StatusCreated, &characterResponse)
	if characterResponse["raceId"].(string) != "human" {
		t.Fatalf("expected created character race human, got %v", characterResponse["raceId"])
	}
	if characterResponse["classId"].(string) != "warrior" {
		t.Fatalf("expected created character class warrior, got %v", characterResponse["classId"])
	}
	learnedAbilityIds := characterResponse["learnedAbilityIds"].([]any)
	if len(learnedAbilityIds) != len(platform.DefaultStartingLearnedAbilityIDs()) {
		t.Fatalf("expected %d starting learned abilities, got %d", len(platform.DefaultStartingLearnedAbilityIDs()), len(learnedAbilityIds))
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
	worldSessionToken := connectResponse["worldSessionToken"].(string)
	if int(connectResponse["currencyCopper"].(float64)) != 125 {
		t.Fatalf("expected new Human Warrior to start with 125 copper, got %v", connectResponse["currencyCopper"])
	}
	assertAbilityPayload(t, connectResponse)

	postJSON(t, server.Client(), server.URL+"/v1/world/action-bar/assign", nil, map[string]any{
		"worldSessionToken": worldSessionToken,
		"slotIndex":         12,
		"abilityId":         "steady_strike",
	}, http.StatusOK, &connectResponse)
	assertActionBarSlotAbility(t, connectResponse, 12, "steady_strike")

	postJSON(t, server.Client(), server.URL+"/v1/world/action-bar/assign", nil, map[string]any{
		"worldSessionToken": worldSessionToken,
		"slotIndex":         12,
		"abilityId":         "not_learned",
	}, http.StatusBadRequest, nil)
	postJSON(t, server.Client(), server.URL+"/v1/world/action-bar/assign", nil, map[string]any{
		"worldSessionToken": worldSessionToken,
		"slotIndex":         48,
		"abilityId":         "steady_strike",
	}, http.StatusBadRequest, nil)

	postJSON(t, server.Client(), server.URL+"/v1/world/action-bar/clear", nil, map[string]any{
		"worldSessionToken": worldSessionToken,
		"slotIndex":         12,
	}, http.StatusOK, &connectResponse)
	assertActionBarSlotEmpty(t, connectResponse, 12)

	postJSON(t, server.Client(), server.URL+"/v1/world/action-bar/move", nil, map[string]any{
		"worldSessionToken": worldSessionToken,
		"fromSlotIndex":     1,
		"toSlotIndex":       12,
	}, http.StatusOK, &connectResponse)
	assertActionBarSlotEmpty(t, connectResponse, 1)
	assertActionBarSlotAbility(t, connectResponse, 12, "steady_strike")

	postJSON(t, server.Client(), server.URL+"/v1/world/inventory/move", nil, map[string]any{
		"worldSessionToken": worldSessionToken,
		"fromSlotIndex":     0,
		"toSlotIndex":       5,
	}, http.StatusOK, &connectResponse)
	assertInventorySlotItem(t, connectResponse, 0, "", 0)
	assertInventorySlotItem(t, connectResponse, 5, "camp_ration", 3)

	postJSON(t, server.Client(), server.URL+"/v1/world/connect", nil, map[string]string{
		"ticketId": ticketResponse["ticketId"].(string),
	}, http.StatusUnauthorized, nil)

	postJSON(t, server.Client(), server.URL+"/v1/world/move", nil, map[string]any{
		"worldSessionToken": worldSessionToken,
		"deltaX":            4,
		"deltaY":            2,
	}, http.StatusOK, &connectResponse)

	postJSON(t, server.Client(), server.URL+"/v1/world/disconnect", nil, map[string]any{
		"worldSessionToken": worldSessionToken,
	}, http.StatusOK, nil)

	postJSON(t, server.Client(), server.URL+"/v1/world/reconnect", nil, map[string]any{
		"worldSessionToken": worldSessionToken,
	}, http.StatusOK, &connectResponse)
	assertAbilityPayloadWithoutDefaultBarLayout(t, connectResponse)
	assertActionBarSlotAbility(t, connectResponse, 12, "steady_strike")
	assertActionBarSlotEmpty(t, connectResponse, 1)
	assertInventorySlotItem(t, connectResponse, 0, "", 0)
	assertInventorySlotItem(t, connectResponse, 5, "camp_ration", 3)

	position := connectResponse["position"].(map[string]any)
	if position["x"].(float64) != 14 {
		t.Fatalf("expected persisted x position 14, got %v", position["x"])
	}
	if position["y"].(float64) != 12 {
		t.Fatalf("expected persisted y position 12, got %v", position["y"])
	}

	fileStoreAfterRestart, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to create restarted store: %v", err)
	}

	muxAfterRestart := http.NewServeMux()
	authn.RegisterRoutes(muxAfterRestart, fileStoreAfterRestart)
	realms.RegisterRoutes(muxAfterRestart, fileStoreAfterRestart)
	characters.RegisterRoutes(muxAfterRestart, fileStoreAfterRestart)
	worlds.RegisterRoutes(muxAfterRestart, fileStoreAfterRestart)

	serverAfterRestart := httptest.NewServer(muxAfterRestart)
	defer serverAfterRestart.Close()

	var loginAfterRestart map[string]any
	postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/auth/login", nil, map[string]string{
		"username": username,
		"password": password,
	}, http.StatusOK, &loginAfterRestart)

	restartedAccessToken := loginAfterRestart["accessToken"].(string)

	var charactersAfterRestart map[string][]map[string]any
	getJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/characters?realmId="+realmID, bearer(restartedAccessToken), http.StatusOK, &charactersAfterRestart)
	restartedCharacterID := charactersAfterRestart["characters"][0]["id"].(string)

	var ticketAfterRestart map[string]any
	postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/world/join-ticket", bearer(restartedAccessToken), map[string]string{
		"realmId":     realmID,
		"characterId": restartedCharacterID,
	}, http.StatusCreated, &ticketAfterRestart)

	var reconnectAfterRestart map[string]any
	postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/world/connect", nil, map[string]string{
		"ticketId": ticketAfterRestart["ticketId"].(string),
	}, http.StatusCreated, &reconnectAfterRestart)
	assertAbilityPayloadWithoutDefaultBarLayout(t, reconnectAfterRestart)
	assertActionBarSlotAbility(t, reconnectAfterRestart, 12, "steady_strike")
	assertActionBarSlotEmpty(t, reconnectAfterRestart, 1)
	assertInventorySlotItem(t, reconnectAfterRestart, 0, "", 0)
	assertInventorySlotItem(t, reconnectAfterRestart, 5, "camp_ration", 3)

	restartedPosition := reconnectAfterRestart["position"].(map[string]any)
	if restartedPosition["x"].(float64) != 14 {
		t.Fatalf("expected restarted x position 14, got %v", restartedPosition["x"])
	}
	if restartedPosition["y"].(float64) != 12 {
		t.Fatalf("expected restarted y position 12, got %v", restartedPosition["y"])
	}
}

func TestLegacyArchetypeCharacterCreationMapsToHumanWarrior(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
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
	defer server.Close()

	postJSON(t, server.Client(), server.URL+"/v1/accounts/register", nil, map[string]string{
		"username": "legacy_user",
		"password": "legacy_pass",
	}, http.StatusCreated, nil)

	var loginResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/auth/login", nil, map[string]string{
		"username": "legacy_user",
		"password": "legacy_pass",
	}, http.StatusOK, &loginResponse)
	accessToken := loginResponse["accessToken"].(string)

	var realmResponse map[string][]map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/realms", nil, http.StatusOK, &realmResponse)
	realmID := realmResponse["realms"][0]["id"].(string)

	var characterResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/characters", bearer(accessToken), map[string]string{
		"realmId":     realmID,
		"displayName": "LegacyRunner",
		"archetypeId": "wayfarer_warden",
	}, http.StatusCreated, &characterResponse)
	if characterResponse["raceId"].(string) != "human" {
		t.Fatalf("expected legacy-created race human, got %v", characterResponse["raceId"])
	}
	if characterResponse["classId"].(string) != "warrior" {
		t.Fatalf("expected legacy-created class warrior, got %v", characterResponse["classId"])
	}
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func assertAbilityPayload(t *testing.T, response map[string]any) {
	t.Helper()

	assertAbilityPayloadWithoutDefaultBarLayout(t, response)

	actionBar, ok := response["actionBar"].([]any)
	if !ok {
		t.Fatalf("world session response missing actionBar: %#v", response["actionBar"])
	}
	if len(actionBar) != 48 {
		t.Fatalf("expected 48 action bar slots, got %d", len(actionBar))
	}

	slot0 := actionBar[0].(map[string]any)
	if slot0["abilityId"].(string) != "auto_attack" {
		t.Fatalf("expected action bar slot 0 to be auto_attack, got %#v", slot0)
	}
	if slot0["buttonLabel"].(string) != "Atk" {
		t.Fatalf("expected action bar slot 0 label Atk, got %#v", slot0)
	}
	slot1 := actionBar[1].(map[string]any)
	if slot1["abilityId"].(string) != "steady_strike" {
		t.Fatalf("expected action bar slot 1 to be steady_strike, got %#v", slot1)
	}
	if slot1["buttonLabel"].(string) != "Strike" {
		t.Fatalf("expected action bar slot 1 label Strike, got %#v", slot1)
	}
	if slot1["category"].(string) != "Warrior" ||
		slot1["abilityType"].(string) != "active" ||
		slot1["passive"].(bool) ||
		!slot1["actionBarAssignable"].(bool) {
		t.Fatalf("expected action bar slot 1 ability metadata, got %#v", slot1)
	}
	slot2 := actionBar[2].(map[string]any)
	if slot2["abilityId"].(string) != "brace" {
		t.Fatalf("expected action bar slot 2 to be brace, got %#v", slot2)
	}
	if slot2["buttonLabel"].(string) != "Brace" {
		t.Fatalf("expected action bar slot 2 label Brace, got %#v", slot2)
	}
}

func assertAbilityPayloadWithoutDefaultBarLayout(t *testing.T, response map[string]any) {
	t.Helper()

	learnedAbilityIds, ok := response["learnedAbilityIds"].([]any)
	if !ok {
		t.Fatalf("world session response missing learnedAbilityIds: %#v", response["learnedAbilityIds"])
	}
	if len(learnedAbilityIds) != len(platform.DefaultStartingLearnedAbilityIDs()) {
		t.Fatalf("expected %d learned abilities in session payload, got %d", len(platform.DefaultStartingLearnedAbilityIDs()), len(learnedAbilityIds))
	}

	spellbook, ok := response["spellbook"].([]any)
	if !ok {
		t.Fatalf("world session response missing spellbook: %#v", response["spellbook"])
	}
	if len(spellbook) < 6 {
		t.Fatalf("expected spellbook preview entries, got %d", len(spellbook))
	}
	spellbook0 := spellbook[0].(map[string]any)
	if spellbook0["displayName"].(string) != "Auto Attack" {
		t.Fatalf("expected first spellbook entry to be Auto Attack, got %#v", spellbook0)
	}
	if spellbook0["category"].(string) != "Warrior" ||
		spellbook0["abilityType"].(string) != "active" ||
		spellbook0["passive"].(bool) ||
		!spellbook0["actionBarAssignable"].(bool) ||
		spellbook0["trainable"].(bool) {
		t.Fatalf("expected spellbook metadata on auto attack, got %#v", spellbook0)
	}
	spellbook1 := spellbook[1].(map[string]any)
	if spellbook1["displayName"].(string) != "Steady Strike" {
		t.Fatalf("expected second spellbook entry to be Steady Strike, got %#v", spellbook1)
	}
	spellbook2 := spellbook[2].(map[string]any)
	if spellbook2["displayName"].(string) != "Brace" {
		t.Fatalf("expected third spellbook entry to be Brace, got %#v", spellbook2)
	}

	trainer, ok := response["trainer"].(map[string]any)
	if !ok {
		t.Fatalf("world session response missing trainer payload: %#v", response["trainer"])
	}
	if trainer["id"].(string) != "trainer_armsmaster_corin_vale" {
		t.Fatalf("expected trainer payload to expose warrior trainer, got %#v", trainer)
	}
}

func assertActionBarSlotEmpty(t *testing.T, response map[string]any, slotIndex int) {
	t.Helper()

	actionBar := response["actionBar"].([]any)
	if len(actionBar) <= slotIndex {
		t.Fatalf("expected action bar to include slot %d, got %d slots", slotIndex, len(actionBar))
	}

	slot := actionBar[slotIndex].(map[string]any)
	if slot["abilityId"].(string) != "" {
		t.Fatalf("expected action bar slot %d to be empty, got %#v", slotIndex, slot)
	}
}

func assertInventorySlotItem(t *testing.T, response map[string]any, slotIndex int, itemID string, stackCount int) {
	t.Helper()

	inventory := response["inventory"].(map[string]any)
	slots := inventory["slots"].([]any)
	if len(slots) <= slotIndex {
		t.Fatalf("expected inventory to include slot %d, got %d slots", slotIndex, len(slots))
	}

	slot := slots[slotIndex].(map[string]any)
	if slot["itemId"].(string) != itemID || int(slot["stackCount"].(float64)) != stackCount {
		t.Fatalf("expected inventory slot %d to be %s x%d, got %#v", slotIndex, itemID, stackCount, slot)
	}
}

func postJSON(t *testing.T, client *http.Client, url string, headers map[string]string, payload any, expectedStatus int, target any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		t.Fatalf("unexpected status for %s: got %d want %d", url, response.StatusCode, expectedStatus)
	}

	if target != nil {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
	}
}

func getJSON(t *testing.T, client *http.Client, url string, headers map[string]string, expectedStatus int, target any) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		t.Fatalf("unexpected status for %s: got %d want %d", url, response.StatusCode, expectedStatus)
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}
