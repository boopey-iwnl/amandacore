package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/characters"
	"amandacore/services/internal/realms"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

type multiClientTestServer struct {
	server *httptest.Server
}

type provisionedClient struct {
	AccessToken string
	RealmID     string
	CharacterID string
	TicketID    string
	Token       string
	State       map[string]any
}

func TestMultiClientStabilityFoundation(t *testing.T) {
	t.Run("two simultaneous joins see each other in the same zone", func(t *testing.T) {
		fixture := newMultiClientTestServer(t)
		first := fixture.provision(t, 1)
		second := fixture.provision(t, 2)

		var wg sync.WaitGroup
		errors := make(chan error, 2)
		for _, client := range []*provisionedClient{first, second} {
			wg.Add(1)
			go func(client *provisionedClient) {
				defer wg.Done()
				var state map[string]any
				status, err := postJSONStatus(fixture.server.Client(), fixture.server.URL+"/v1/world/connect", nil, map[string]string{
					"ticketId": client.TicketID,
				}, &state)
				if err != nil {
					errors <- err
					return
				}
				if status != http.StatusCreated {
					errors <- fmt.Errorf("connect got status %d, want %d", status, http.StatusCreated)
					return
				}
				client.Token = state["worldSessionToken"].(string)
				client.State = state
			}(client)
		}
		wg.Wait()
		close(errors)
		for err := range errors {
			if err != nil {
				t.Fatal(err)
			}
		}

		first.State = fixture.state(t, first.Token)
		second.State = fixture.state(t, second.Token)
		assertVisiblePlayer(t, first.State, second.CharacterID)
		assertVisiblePlayer(t, second.State, first.CharacterID)
	})

	t.Run("duplicate character connect resumes the existing session", func(t *testing.T) {
		fixture := newMultiClientTestServer(t)
		client := fixture.provisionAndConnect(t, 1)
		secondTicket := fixture.issueTicket(t, client.AccessToken, client.RealmID, client.CharacterID)

		var resumed map[string]any
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/connect", nil, map[string]string{
			"ticketId": secondTicket,
		}, http.StatusOK, &resumed)
		if resumed["worldSessionToken"].(string) != client.Token {
			t.Fatalf("expected duplicate connect to resume token %s, got %s", client.Token, resumed["worldSessionToken"].(string))
		}
	})

	t.Run("movement remains stable under concurrent clients", func(t *testing.T) {
		fixture := newMultiClientTestServer(t)
		clients := []*provisionedClient{
			fixture.provisionAndConnect(t, 1),
			fixture.provisionAndConnect(t, 2),
			fixture.provisionAndConnect(t, 3),
			fixture.provisionAndConnect(t, 4),
			fixture.provisionAndConnect(t, 5),
		}

		var wg sync.WaitGroup
		errors := make(chan error, len(clients)*10)
		for _, client := range clients {
			wg.Add(1)
			go func(client *provisionedClient) {
				defer wg.Done()
				for step := 0; step < 10; step++ {
					var state map[string]any
					status, err := postJSONStatus(fixture.server.Client(), fixture.server.URL+"/v1/world/move", nil, map[string]any{
						"worldSessionToken": client.Token,
						"deltaX":            0.25,
						"deltaY":            0.10,
					}, &state)
					if err != nil {
						errors <- err
						return
					}
					if status != http.StatusOK {
						errors <- fmt.Errorf("move got status %d, want %d", status, http.StatusOK)
						return
					}
				}
			}(client)
		}
		wg.Wait()
		close(errors)
		for err := range errors {
			if err != nil {
				t.Fatal(err)
			}
		}

		for _, client := range clients {
			state := fixture.state(t, client.Token)
			position := state["position"].(map[string]any)
			if position["x"].(float64) <= 10.0 || position["y"].(float64) <= 10.0 {
				t.Fatalf("expected client %s to move from spawn, got %#v", client.CharacterID, position)
			}
		}
	})

	t.Run("disconnect reconnect loop remains stable", func(t *testing.T) {
		fixture := newMultiClientTestServer(t)
		client := fixture.provisionAndConnect(t, 1)

		for attempt := 0; attempt < 5; attempt++ {
			postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/disconnect", nil, map[string]any{
				"worldSessionToken": client.Token,
			}, http.StatusOK, nil)

			var state map[string]any
			postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
				"worldSessionToken": client.Token,
			}, http.StatusOK, &state)
			if !state["alive"].(bool) {
				t.Fatalf("expected client to be alive after reconnect attempt %d", attempt+1)
			}
		}
	})

	t.Run("world metrics endpoint reports sessions and endpoint timings", func(t *testing.T) {
		fixture := newMultiClientTestServer(t)
		_ = fixture.provisionAndConnect(t, 1)

		var metrics map[string]any
		getJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/metrics", nil, http.StatusOK, &metrics)

		sessions := metrics["sessions"].(map[string]any)
		if int(sessions["active"].(float64)) < 1 {
			t.Fatalf("expected active session metrics, got %#v", sessions)
		}
		endpoints := metrics["endpoints"].([]any)
		if len(endpoints) == 0 {
			t.Fatalf("expected endpoint metrics, got %#v", metrics["endpoints"])
		}
	})
}

func newMultiClientTestServer(t *testing.T) *multiClientTestServer {
	t.Helper()

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
	t.Cleanup(server.Close)
	return &multiClientTestServer{server: server}
}

func (f *multiClientTestServer) provision(t *testing.T, index int) *provisionedClient {
	t.Helper()

	username := fmt.Sprintf("multi_user_%d", index)
	password := "multi_pass"
	postJSON(t, f.server.Client(), f.server.URL+"/v1/accounts/register", nil, map[string]string{
		"username": username,
		"password": password,
	}, http.StatusCreated, nil)

	var login map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/auth/login", nil, map[string]string{
		"username": username,
		"password": password,
	}, http.StatusOK, &login)
	accessToken := login["accessToken"].(string)

	var realms map[string][]map[string]any
	getJSON(t, f.server.Client(), f.server.URL+"/v1/realms", nil, http.StatusOK, &realms)
	realmID := realms["realms"][0]["id"].(string)

	var character map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/characters", bearer(accessToken), map[string]string{
		"realmId":     realmID,
		"displayName": fmt.Sprintf("MultiRunner%d", index),
		"raceId":      "human",
		"classId":     "warrior",
		"archetypeId": "wayfarer_warden",
	}, http.StatusCreated, &character)
	characterID := character["id"].(string)

	return &provisionedClient{
		AccessToken: accessToken,
		RealmID:     realmID,
		CharacterID: characterID,
		TicketID:    f.issueTicket(t, accessToken, realmID, characterID),
	}
}

func (f *multiClientTestServer) provisionAndConnect(t *testing.T, index int) *provisionedClient {
	t.Helper()

	client := f.provision(t, index)
	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/connect", nil, map[string]string{
		"ticketId": client.TicketID,
	}, http.StatusCreated, &state)
	client.Token = state["worldSessionToken"].(string)
	client.State = state
	return client
}

func (f *multiClientTestServer) issueTicket(t *testing.T, accessToken string, realmID string, characterID string) string {
	t.Helper()

	var ticket map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/join-ticket", bearer(accessToken), map[string]string{
		"realmId":     realmID,
		"characterId": characterID,
	}, http.StatusCreated, &ticket)
	return ticket["ticketId"].(string)
}

func (f *multiClientTestServer) state(t *testing.T, worldSessionToken string) map[string]any {
	t.Helper()

	var state map[string]any
	getJSON(t, f.server.Client(), f.server.URL+"/v1/world/state?worldSessionToken="+worldSessionToken, nil, http.StatusOK, &state)
	return state
}

func assertVisiblePlayer(t *testing.T, state map[string]any, characterID string) {
	t.Helper()

	entities := state["entities"].([]any)
	for _, value := range entities {
		entity := value.(map[string]any)
		if entity["kind"] == "player" && entity["id"] == characterID {
			return
		}
	}
	t.Fatalf("expected state for %s to include player entity %s", state["characterId"], characterID)
}

func postJSONStatus(client *http.Client, url string, headers map[string]string, payload any, target any) (int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if target != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			return response.StatusCode, err
		}
	}
	return response.StatusCode, nil
}
