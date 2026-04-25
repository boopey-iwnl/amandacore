package e2e

import (
	"net/http"
	"testing"

	"amandacore/services/internal/store"
)

func TestGuildSliceCreateInviteRosterRanksChatAndDisband(t *testing.T) {
	fixture := newSocialFixture(t)
	cara := connectExistingSocialPlayer(t, fixture.server, fixture.realmID, fixture.cara)

	fixture.postSocial(t, "/v1/world/guild/create", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"guildName":         "Stonewake Guard",
	}, http.StatusCreated, nil)
	fixture.postSocial(t, "/v1/world/guild/create", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"guildName":         "Stonewake Guard",
	}, http.StatusBadRequest, nil)

	aliceState := fixture.socialState(t, fixture.alice.worldSessionToken)
	guild := guildState(t, aliceState)
	if guild["guildName"] != "Stonewake Guard" {
		t.Fatalf("expected guild name, got %#v", guild)
	}
	assertGuildMembers(t, aliceState, "Alice")
	if len(guild["ranks"].([]any)) != 4 {
		t.Fatalf("expected default guild ranks, got %#v", guild["ranks"])
	}

	fixture.postSocial(t, "/v1/world/guild/invite", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"targetName":        "Bob",
	}, http.StatusOK, nil)
	bobState := fixture.socialState(t, fixture.bob.worldSessionToken)
	inviteID := firstGuildInviteID(t, bobState)
	fixture.postSocial(t, "/v1/world/guild/decline", map[string]any{
		"worldSessionToken": fixture.bob.worldSessionToken,
		"inviteId":          inviteID,
	}, http.StatusOK, nil)
	if len(guildInvites(fixture.socialState(t, fixture.bob.worldSessionToken))) != 0 {
		t.Fatalf("expected declined guild invite to clear")
	}

	fixture.postSocial(t, "/v1/world/guild/invite", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"targetName":        "Bob",
	}, http.StatusOK, nil)
	bobState = fixture.socialState(t, fixture.bob.worldSessionToken)
	fixture.postSocial(t, "/v1/world/guild/accept", map[string]any{
		"worldSessionToken": fixture.bob.worldSessionToken,
		"inviteId":          firstGuildInviteID(t, bobState),
	}, http.StatusOK, nil)
	assertGuildMembers(t, fixture.socialState(t, fixture.alice.worldSessionToken), "Alice", "Bob")

	fixture.postSocial(t, "/v1/world/chat/send", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"channel":           "guild",
		"messageText":       "Guild words",
	}, http.StatusOK, nil)
	findChatMessage(t, fixture.socialState(t, fixture.bob.worldSessionToken), "guild", "Guild words")
	if hasChatMessage(fixture.socialState(t, cara.worldSessionToken), "guild", "Guild words") {
		t.Fatalf("non-member received guild chat")
	}

	fixture.postSocial(t, "/v1/world/guild/disband", map[string]any{
		"worldSessionToken": fixture.bob.worldSessionToken,
	}, http.StatusBadRequest, nil)
	fixture.postSocial(t, "/v1/world/guild/invite", map[string]any{
		"worldSessionToken": fixture.bob.worldSessionToken,
		"targetName":        "Cara",
	}, http.StatusBadRequest, nil)

	fixture.postSocial(t, "/v1/world/guild/promote", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"targetName":        "Bob",
	}, http.StatusOK, nil)
	fixture.postSocial(t, "/v1/world/guild/demote", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"targetName":        "Bob",
	}, http.StatusOK, nil)
	fixture.postSocial(t, "/v1/world/guild/remove", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"targetName":        "Bob",
	}, http.StatusOK, nil)
	assertGuildMembers(t, fixture.socialState(t, fixture.alice.worldSessionToken), "Alice")

	fixture.postSocial(t, "/v1/world/guild/invite", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"targetName":        "Bob",
	}, http.StatusOK, nil)
	bobState = fixture.socialState(t, fixture.bob.worldSessionToken)
	fixture.postSocial(t, "/v1/world/guild/accept", map[string]any{
		"worldSessionToken": fixture.bob.worldSessionToken,
		"inviteId":          firstGuildInviteID(t, bobState),
	}, http.StatusOK, nil)
	fixture.postSocial(t, "/v1/world/guild/leave", map[string]any{
		"worldSessionToken": fixture.bob.worldSessionToken,
	}, http.StatusOK, nil)
	assertGuildMembers(t, fixture.socialState(t, fixture.alice.worldSessionToken), "Alice")

	fixture.postSocial(t, "/v1/world/guild/disband", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
	}, http.StatusOK, nil)
	if fixture.socialState(t, fixture.alice.worldSessionToken)["guild"] != nil {
		t.Fatalf("expected guild to be disbanded")
	}
}

func TestGuildSliceRestartRestoresGuildRoster(t *testing.T) {
	fixture := newSocialFixture(t)
	fixture.postSocial(t, "/v1/world/guild/create", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"guildName":         "Restart Wardens",
	}, http.StatusCreated, nil)
	fixture.postSocial(t, "/v1/world/guild/invite", map[string]any{
		"worldSessionToken": fixture.alice.worldSessionToken,
		"targetName":        "Bob",
	}, http.StatusOK, nil)
	bobState := fixture.socialState(t, fixture.bob.worldSessionToken)
	fixture.postSocial(t, "/v1/world/guild/accept", map[string]any{
		"worldSessionToken": fixture.bob.worldSessionToken,
		"inviteId":          firstGuildInviteID(t, bobState),
	}, http.StatusOK, nil)

	restartedStore, err := store.NewFileStore(fixture.storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to restart store: %v", err)
	}
	restartedServer := newSocialServer(t, restartedStore)
	defer restartedServer.Close()

	alice := connectExistingSocialPlayer(t, restartedServer, fixture.realmID, fixture.alice)
	state := socialStateForServer(t, restartedServer, alice.worldSessionToken)
	assertGuildMembers(t, state, "Alice", "Bob")
	member := findGuildMember(t, state, "Bob")
	if online, _ := member["online"].(bool); online {
		t.Fatalf("expected Bob to be offline before reconnect, got %#v", member)
	}
}

func guildState(t *testing.T, state map[string]any) map[string]any {
	t.Helper()
	guild, ok := state["guild"].(map[string]any)
	if !ok {
		t.Fatalf("expected guild state, got %#v", state["guild"])
	}
	return guild
}

func guildInvites(state map[string]any) []any {
	invites, _ := state["guildInvites"].([]any)
	return invites
}

func firstGuildInviteID(t *testing.T, state map[string]any) string {
	t.Helper()
	invites := guildInvites(state)
	if len(invites) == 0 {
		t.Fatalf("expected pending guild invite, got %#v", state["guildInvites"])
	}
	invite := invites[0].(map[string]any)
	return invite["inviteId"].(string)
}

func assertGuildMembers(t *testing.T, state map[string]any, expectedNames ...string) {
	t.Helper()
	guild := guildState(t, state)
	members, ok := guild["members"].([]any)
	if !ok {
		t.Fatalf("expected guild members, got %#v", guild)
	}
	if len(members) != len(expectedNames) {
		t.Fatalf("expected %d guild members, got %#v", len(expectedNames), members)
	}
	for _, expectedName := range expectedNames {
		findGuildMember(t, state, expectedName)
	}
}

func findGuildMember(t *testing.T, state map[string]any, displayName string) map[string]any {
	t.Helper()
	guild := guildState(t, state)
	members, _ := guild["members"].([]any)
	for _, value := range members {
		member, ok := value.(map[string]any)
		if ok && member["displayName"] == displayName {
			return member
		}
	}
	t.Fatalf("guild member %s not found in %#v", displayName, members)
	return nil
}

func hasChatMessage(state map[string]any, channel string, text string) bool {
	for _, value := range chatMessages(state) {
		message, ok := value.(map[string]any)
		if ok && message["channel"] == channel && message["messageText"] == text {
			return true
		}
	}
	return false
}
