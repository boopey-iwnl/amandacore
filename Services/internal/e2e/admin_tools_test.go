package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"amandacore/services/internal/admin"
	"amandacore/services/internal/authn"
	"amandacore/services/internal/characters"
	"amandacore/services/internal/realms"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

func TestAdminToolsMilestoneSlice(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "admin-state.json")
	fileStore, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	if err := fileStore.EnsureAdminSeed("admin_user", "admin_pass"); err != nil {
		t.Fatalf("failed to seed admin: %v", err)
	}

	server := newAdminMilestoneServer(fileStore)
	defer server.Close()

	var realmResponse map[string][]map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/realms", nil, http.StatusOK, &realmResponse)
	realmID := realmResponse["realms"][0]["id"].(string)

	player := createAndConnectSocialPlayer(t, server, realmID, "admin_target", "target_pass", "AdminTarget")
	adminToken := loginForAdminTest(t, server, "admin_user", "admin_pass")

	var forbidden map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/admin/characters?query=AdminTarget", bearer(player.accessToken), http.StatusForbidden, &forbidden)

	var searchResponse map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/admin/characters?query=AdminTarget", bearer(adminToken), http.StatusOK, &searchResponse)
	characters := searchResponse["characters"].([]any)
	if len(characters) != 1 {
		t.Fatalf("expected one admin character search result, got %#v", searchResponse)
	}

	var details map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/admin/characters/"+player.characterID, bearer(adminToken), http.StatusOK, &details)
	assertNoSensitiveAdminFields(t, details)
	if details["mailSummary"] == nil || details["auctionSummary"] == nil || details["housingSummary"] == nil || details["pvpSummary"] == nil {
		t.Fatalf("expected admin inspection summaries, got %#v", details)
	}

	var worldState map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/world/admin/characters/"+player.characterID+"/teleport", bearer(adminToken), map[string]any{
		"destination": "stonewake_spawn",
		"reason":      "e2e unstuck validation",
		"confirm":     true,
	}, http.StatusOK, &worldState)
	position := worldState["position"].(map[string]any)
	if position["x"].(float64) != 10 || position["y"].(float64) != 10 {
		t.Fatalf("expected teleport to Stonewake spawn, got %#v", position)
	}

	postJSON(t, server.Client(), server.URL+"/v1/world/admin/characters/"+player.characterID+"/items/grant", bearer(adminToken), map[string]any{
		"itemId":   "camp_ration",
		"quantity": 1,
		"reason":   "e2e item grant validation",
		"confirm":  true,
	}, http.StatusOK, &worldState)
	if countInventoryItem(worldState, "camp_ration") != 4 {
		t.Fatalf("expected four camp rations after grant, got %#v", worldState["inventory"])
	}

	postJSON(t, server.Client(), server.URL+"/v1/world/admin/characters/"+player.characterID+"/items/remove", bearer(adminToken), map[string]any{
		"itemId":   "camp_ration",
		"quantity": 1,
		"reason":   "e2e item remove validation",
		"confirm":  true,
	}, http.StatusOK, &worldState)
	if countInventoryItem(worldState, "camp_ration") != 3 {
		t.Fatalf("expected three camp rations after remove, got %#v", worldState["inventory"])
	}

	postJSON(t, server.Client(), server.URL+"/v1/world/admin/characters/"+player.characterID+"/currency/grant", bearer(adminToken), map[string]any{
		"copper":  10,
		"reason":  "e2e currency grant validation",
		"confirm": true,
	}, http.StatusOK, &worldState)
	if int(worldState["currencyCopper"].(float64)) != 135 {
		t.Fatalf("expected 135 copper after grant, got %#v", worldState["currencyCopper"])
	}
	postJSON(t, server.Client(), server.URL+"/v1/world/admin/characters/"+player.characterID+"/currency/remove", bearer(adminToken), map[string]any{
		"copper":  5,
		"reason":  "e2e currency remove validation",
		"confirm": true,
	}, http.StatusOK, &worldState)
	if int(worldState["currencyCopper"].(float64)) != 130 {
		t.Fatalf("expected 130 copper after remove, got %#v", worldState["currencyCopper"])
	}
	postJSON(t, server.Client(), server.URL+"/v1/world/admin/characters/"+player.characterID+"/currency/remove", bearer(adminToken), map[string]any{
		"copper":  999999,
		"reason":  "e2e negative currency rejection",
		"confirm": true,
	}, http.StatusBadRequest, nil)

	var ticketResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/support/tickets", bearer(player.accessToken), map[string]any{
		"characterId": player.characterID,
		"category":    "bug",
		"subject":     "Stuck in terrain",
		"body":        "Character needed unstuck during e2e.",
		"buildId":     "test-build",
	}, http.StatusCreated, &ticketResponse)
	reopenedStore, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	tickets, err := reopenedStore.ListSupportTickets("")
	if err != nil || len(tickets) != 1 {
		t.Fatalf("expected persisted support ticket, tickets=%#v err=%v", tickets, err)
	}

	postJSON(t, server.Client(), server.URL+"/v1/admin/moderation/mute", bearer(adminToken), map[string]any{
		"characterId":     player.characterID,
		"durationSeconds": 300,
		"reason":          "e2e mute validation",
		"confirm":         true,
	}, http.StatusOK, nil)
	postJSON(t, server.Client(), server.URL+"/v1/world/chat/send", nil, map[string]any{
		"worldSessionToken": player.worldSessionToken,
		"channel":           "say",
		"messageText":       "this should be blocked",
	}, http.StatusBadRequest, nil)
	postJSON(t, server.Client(), server.URL+"/v1/admin/moderation/unmute", bearer(adminToken), map[string]any{
		"characterId": player.characterID,
		"reason":      "e2e unmute validation",
		"confirm":     true,
	}, http.StatusOK, nil)
	postJSON(t, server.Client(), server.URL+"/v1/world/chat/send", nil, map[string]any{
		"worldSessionToken": player.worldSessionToken,
		"channel":           "say",
		"messageText":       "this should send",
	}, http.StatusOK, nil)

	postJSON(t, server.Client(), server.URL+"/v1/admin/moderation/suspend", bearer(adminToken), map[string]any{
		"accountId":       playerAccountIDFromCharacter(t, fileStore, player.characterID),
		"durationSeconds": 300,
		"reason":          "e2e suspension validation",
		"confirm":         true,
	}, http.StatusOK, nil)
	postJSON(t, server.Client(), server.URL+"/v1/auth/login", nil, map[string]string{
		"username": player.username,
		"password": player.password,
	}, http.StatusUnauthorized, nil)
	postJSON(t, server.Client(), server.URL+"/v1/admin/moderation/unsuspend", bearer(adminToken), map[string]any{
		"accountId": playerAccountIDFromCharacter(t, fileStore, player.characterID),
		"reason":    "e2e unsuspend validation",
		"confirm":   true,
	}, http.StatusOK, nil)
	loginForAdminTest(t, server, player.username, player.password)

	var auditResponse map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/admin/audit?targetCharacterId="+player.characterID, bearer(adminToken), http.StatusOK, &auditResponse)
	events := auditResponse["events"].([]any)
	if len(events) < 6 {
		t.Fatalf("expected admin audit events, got %#v", auditResponse)
	}
}

func newAdminMilestoneServer(fileStore *store.FileStore) *httptest.Server {
	mux := http.NewServeMux()
	authn.RegisterRoutes(mux, fileStore)
	realms.RegisterRoutes(mux, fileStore)
	characters.RegisterRoutes(mux, fileStore)
	worlds.RegisterRoutes(mux, fileStore)
	admin.RegisterRoutes(mux, fileStore)
	return httptest.NewServer(mux)
}

func loginForAdminTest(t *testing.T, server *httptest.Server, username string, password string) string {
	t.Helper()

	var loginResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/auth/login", nil, map[string]string{
		"username": username,
		"password": password,
	}, http.StatusOK, &loginResponse)
	return loginResponse["accessToken"].(string)
}

func assertNoSensitiveAdminFields(t *testing.T, payload any) {
	t.Helper()

	serialized, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal admin payload: %v", err)
	}
	body := string(serialized)
	for _, forbidden := range []string{"passwordHash", "accessToken", "refreshToken"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("admin payload leaked %s: %s", forbidden, body)
		}
	}
}

func countInventoryItem(state map[string]any, itemID string) int {
	inventory := state["inventory"].(map[string]any)
	slots := inventory["slots"].([]any)
	total := 0
	for _, value := range slots {
		slot := value.(map[string]any)
		if slot["itemId"] == itemID {
			total += int(slot["stackCount"].(float64))
		}
	}
	return total
}

func playerAccountIDFromCharacter(t *testing.T, fileStore *store.FileStore, characterID string) string {
	t.Helper()
	character, err := fileStore.GetCharacterByID(characterID)
	if err != nil {
		t.Fatalf("failed to load character: %v", err)
	}
	return character.AccountID
}
