package e2e

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/characters"
	"amandacore/services/internal/realms"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

type socialFixture struct {
	server    *httptest.Server
	storePath string
	realmID   string
	alice     socialPlayer
	bob       socialPlayer
	cara      socialPlayer
}

type socialPlayer struct {
	username          string
	password          string
	characterID       string
	displayName       string
	accessToken       string
	worldSessionToken string
}

func TestSocialSliceChatFriendsAndParty(t *testing.T) {
	fixture := newSocialFixture(t)

	t.Run("system and say chat", func(t *testing.T) {
		state := fixture.socialState(t, fixture.alice.worldSessionToken)
		if len(chatMessages(state)) == 0 {
			t.Fatalf("expected initial system chat message, got %#v", state)
		}

		fixture.postSocial(t, "/v1/world/chat/send", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"channel":           "say",
			"messageText":       "Hail from Alice",
		}, http.StatusOK, nil)

		bobState := fixture.socialState(t, fixture.bob.worldSessionToken)
		message := findChatMessage(t, bobState, "say", "Hail from Alice")
		if message["senderDisplayName"] != "Alice" {
			t.Fatalf("expected Alice say sender, got %#v", message)
		}
	})

	t.Run("whisper and invalid chat target", func(t *testing.T) {
		fixture.postSocial(t, "/v1/world/chat/send", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"channel":           "whisper",
			"targetName":        "Bob",
			"messageText":       "Quiet words",
		}, http.StatusOK, nil)

		bobState := fixture.socialState(t, fixture.bob.worldSessionToken)
		message := findChatMessage(t, bobState, "whisper", "Quiet words")
		if message["targetCharacterId"] != fixture.bob.characterID {
			t.Fatalf("expected Bob as whisper target, got %#v", message)
		}

		fixture.postSocial(t, "/v1/world/chat/send", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"channel":           "whisper",
			"targetName":        "Missing",
			"messageText":       "No one hears this",
		}, http.StatusBadRequest, nil)
	})

	t.Run("friends persist and status changes", func(t *testing.T) {
		fixture.postSocial(t, "/v1/world/friends/add", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"name":              "Bob",
		}, http.StatusOK, nil)
		fixture.postSocial(t, "/v1/world/friends/add", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"name":              "Bob",
		}, http.StatusBadRequest, nil)
		fixture.postSocial(t, "/v1/world/friends/add", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"name":              "Missing",
		}, http.StatusBadRequest, nil)

		state := fixture.socialState(t, fixture.alice.worldSessionToken)
		friend := findFriend(t, state, "Bob")
		if online, _ := friend["online"].(bool); !online {
			t.Fatalf("expected Bob to be online, got %#v", friend)
		}

		fixture.postSocial(t, "/v1/world/disconnect", map[string]any{
			"worldSessionToken": fixture.bob.worldSessionToken,
		}, http.StatusOK, nil)

		state = fixture.socialState(t, fixture.alice.worldSessionToken)
		friend = findFriend(t, state, "Bob")
		if online, _ := friend["online"].(bool); online {
			t.Fatalf("expected Bob to be offline after disconnect, got %#v", friend)
		}

		fixture.postSocial(t, "/v1/world/reconnect", map[string]any{
			"worldSessionToken": fixture.bob.worldSessionToken,
		}, http.StatusOK, nil)

		fixture.postSocial(t, "/v1/world/friends/remove", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"name":              "Bob",
		}, http.StatusOK, nil)
		state = fixture.socialState(t, fixture.alice.worldSessionToken)
		if hasFriend(state, "Bob") {
			t.Fatalf("expected Bob to be removed from Alice's friends, got %#v", state["friends"])
		}

		fixture.postSocial(t, "/v1/world/friends/add", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"name":              "Bob",
		}, http.StatusOK, nil)
	})

	t.Run("party invite decline accept chat leave and reconnect", func(t *testing.T) {
		fixture.postSocial(t, "/v1/world/chat/send", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"channel":           "party",
			"messageText":       "Solo party test",
		}, http.StatusBadRequest, nil)

		fixture.postSocial(t, "/v1/world/party/invite", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"targetName":        "Bob",
		}, http.StatusOK, nil)

		bobState := fixture.socialState(t, fixture.bob.worldSessionToken)
		inviteID := firstInviteID(t, bobState)
		fixture.postSocial(t, "/v1/world/party/decline", map[string]any{
			"worldSessionToken": fixture.bob.worldSessionToken,
			"inviteId":          inviteID,
		}, http.StatusOK, nil)
		bobState = fixture.socialState(t, fixture.bob.worldSessionToken)
		if len(partyInvites(bobState)) != 0 {
			t.Fatalf("expected declined invite to clear, got %#v", bobState["partyInvites"])
		}

		fixture.postSocial(t, "/v1/world/party/invite", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"targetName":        "Bob",
		}, http.StatusOK, nil)
		bobState = fixture.socialState(t, fixture.bob.worldSessionToken)
		inviteID = firstInviteID(t, bobState)
		fixture.postSocial(t, "/v1/world/party/accept", map[string]any{
			"worldSessionToken": fixture.bob.worldSessionToken,
			"inviteId":          inviteID,
		}, http.StatusOK, nil)

		aliceState := fixture.socialState(t, fixture.alice.worldSessionToken)
		assertPartyMembers(t, aliceState, "Alice", "Bob")

		fixture.postSocial(t, "/v1/world/chat/send", map[string]any{
			"worldSessionToken": fixture.alice.worldSessionToken,
			"channel":           "party",
			"messageText":       "Party words",
		}, http.StatusOK, nil)
		bobState = fixture.socialState(t, fixture.bob.worldSessionToken)
		findChatMessage(t, bobState, "party", "Party words")

		fixture.postSocial(t, "/v1/world/disconnect", map[string]any{
			"worldSessionToken": fixture.bob.worldSessionToken,
		}, http.StatusOK, nil)
		fixture.postSocial(t, "/v1/world/reconnect", map[string]any{
			"worldSessionToken": fixture.bob.worldSessionToken,
		}, http.StatusOK, nil)
		bobState = fixture.socialState(t, fixture.bob.worldSessionToken)
		assertPartyMembers(t, bobState, "Alice", "Bob")

		fixture.postSocial(t, "/v1/world/party/leave", map[string]any{
			"worldSessionToken": fixture.bob.worldSessionToken,
		}, http.StatusOK, nil)
		bobState = fixture.socialState(t, fixture.bob.worldSessionToken)
		if bobState["party"] != nil {
			t.Fatalf("expected Bob to be solo after leaving, got %#v", bobState["party"])
		}
		aliceState = fixture.socialState(t, fixture.alice.worldSessionToken)
		if aliceState["party"] != nil {
			t.Fatalf("expected two-player party to disband after Bob leaves, got %#v", aliceState["party"])
		}
	})
}

func TestSocialSliceRestartRestoresFriendsAndParty(t *testing.T) {
	fixture := newSocialFixture(t)
	fixture.postSocial(t, "/v1/world/friends/add", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"name":              "Bob",
	}, http.StatusOK, nil)
	fixture.postSocial(t, "/v1/world/party/invite", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"targetName":        "Bob",
	}, http.StatusOK, nil)
	bobState := fixture.socialState(t, fixture.bob.worldSessionToken)
	fixture.postSocial(t, "/v1/world/party/accept", map[string]any{
		"worldSessionToken": fixture.bob.worldSessionToken,
		"inviteId":          firstInviteID(t, bobState),
	}, http.StatusOK, nil)

	restartedStore, err := store.NewFileStore(fixture.storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to restart store: %v", err)
	}
	restartedServer := newSocialServer(t, restartedStore)
	defer restartedServer.Close()

	alice := connectExistingSocialPlayer(t, restartedServer, fixture.realmID, fixture.alice)
	state := socialStateForServer(t, restartedServer, alice.worldSessionToken)
	if friend := findFriend(t, state, "Bob"); friend["online"].(bool) {
		t.Fatalf("expected Bob friend status to be offline after restart before Bob connects, got %#v", friend)
	}
	assertPartyMembers(t, state, "Alice", "Bob")
}

func newSocialFixture(t *testing.T) *socialFixture {
	t.Helper()

	storePath := filepath.Join(t.TempDir(), "social-state.json")
	fileStore, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	server := newSocialServer(t, fileStore)

	var realmResponse map[string][]map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/realms", nil, http.StatusOK, &realmResponse)
	realmID := realmResponse["realms"][0]["id"].(string)

	fixture := &socialFixture{
		server:    server,
		storePath: storePath,
		realmID:   realmID,
		alice:     createAndConnectSocialPlayer(t, server, realmID, "alice_user", "alice_pass", "Alice"),
		bob:       createAndConnectSocialPlayer(t, server, realmID, "bob_user", "bob_pass", "Bob"),
		cara:      createSocialPlayer(t, server, realmID, "cara_user", "cara_pass", "Cara"),
	}
	t.Cleanup(server.Close)
	return fixture
}

func newSocialServer(t *testing.T, fileStore *store.FileStore) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	authn.RegisterRoutes(mux, fileStore)
	realms.RegisterRoutes(mux, fileStore)
	characters.RegisterRoutes(mux, fileStore)
	worlds.RegisterRoutes(mux, fileStore)
	return httptest.NewServer(mux)
}

func createAndConnectSocialPlayer(t *testing.T, server *httptest.Server, realmID string, username string, password string, displayName string) socialPlayer {
	t.Helper()

	player := createSocialPlayer(t, server, realmID, username, password, displayName)
	return connectExistingSocialPlayer(t, server, realmID, player)
}

func createSocialPlayer(t *testing.T, server *httptest.Server, realmID string, username string, password string, displayName string) socialPlayer {
	t.Helper()

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

	var characterResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/characters", bearer(accessToken), map[string]string{
		"realmId":     realmID,
		"displayName": displayName,
		"raceId":      "human",
		"classId":     "warrior",
		"archetypeId": "wayfarer_warden",
	}, http.StatusCreated, &characterResponse)

	return socialPlayer{
		username:    username,
		password:    password,
		characterID: characterResponse["id"].(string),
		displayName: displayName,
		accessToken: accessToken,
	}
}

func connectExistingSocialPlayer(t *testing.T, server *httptest.Server, realmID string, player socialPlayer) socialPlayer {
	t.Helper()

	var loginResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/auth/login", nil, map[string]string{
		"username": player.username,
		"password": player.password,
	}, http.StatusOK, &loginResponse)
	player.accessToken = loginResponse["accessToken"].(string)

	var ticketResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/world/join-ticket", bearer(player.accessToken), map[string]string{
		"realmId":     realmID,
		"characterId": player.characterID,
	}, http.StatusCreated, &ticketResponse)

	var connectResponse map[string]any
	postJSON(t, server.Client(), server.URL+"/v1/world/connect", nil, map[string]string{
		"ticketId": ticketResponse["ticketId"].(string),
	}, http.StatusCreated, &connectResponse)
	player.worldSessionToken = connectResponse["worldSessionToken"].(string)
	return player
}

func (f *socialFixture) postSocial(t *testing.T, path string, payload any, expectedStatus int, target any) {
	t.Helper()
	postJSON(t, f.server.Client(), f.server.URL+path, nil, payload, expectedStatus, target)
}

func (f *socialFixture) socialState(t *testing.T, worldSessionToken string) map[string]any {
	t.Helper()
	return socialStateForServer(t, f.server, worldSessionToken)
}

func socialStateForServer(t *testing.T, server *httptest.Server, worldSessionToken string) map[string]any {
	t.Helper()

	var state map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/world/social/state?worldSessionToken="+worldSessionToken, nil, http.StatusOK, &state)
	return state
}

func chatMessages(state map[string]any) []any {
	messages, _ := state["chatMessages"].([]any)
	return messages
}

func partyInvites(state map[string]any) []any {
	invites, _ := state["partyInvites"].([]any)
	return invites
}

func findChatMessage(t *testing.T, state map[string]any, channel string, text string) map[string]any {
	t.Helper()
	for _, value := range chatMessages(state) {
		message, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if message["channel"] == channel && message["messageText"] == text {
			return message
		}
	}
	t.Fatalf("chat message %s/%q not found in %#v", channel, text, state["chatMessages"])
	return nil
}

func findFriend(t *testing.T, state map[string]any, displayName string) map[string]any {
	t.Helper()
	friends, _ := state["friends"].([]any)
	for _, value := range friends {
		friend, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if friend["displayName"] == displayName {
			return friend
		}
	}
	t.Fatalf("friend %s not found in %#v", displayName, state["friends"])
	return nil
}

func hasFriend(state map[string]any, displayName string) bool {
	friends, _ := state["friends"].([]any)
	for _, value := range friends {
		friend, ok := value.(map[string]any)
		if ok && friend["displayName"] == displayName {
			return true
		}
	}
	return false
}

func firstInviteID(t *testing.T, state map[string]any) string {
	t.Helper()
	invites := partyInvites(state)
	if len(invites) == 0 {
		t.Fatalf("expected pending party invite, got %#v", state["partyInvites"])
	}
	invite := invites[0].(map[string]any)
	return invite["inviteId"].(string)
}

func assertPartyMembers(t *testing.T, state map[string]any, expectedNames ...string) {
	t.Helper()
	party, ok := state["party"].(map[string]any)
	if !ok {
		t.Fatalf("expected party state, got %#v", state["party"])
	}
	members, ok := party["members"].([]any)
	if !ok {
		t.Fatalf("expected party members, got %#v", party)
	}
	if len(members) != len(expectedNames) {
		t.Fatalf("expected %d party members, got %#v", len(expectedNames), members)
	}

	for _, expectedName := range expectedNames {
		found := false
		for _, value := range members {
			member := value.(map[string]any)
			if member["displayName"] == expectedName {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("party member %s not found in %#v", expectedName, members)
		}
	}
}
