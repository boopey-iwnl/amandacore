package observability

import "testing"

func TestStableEventNamesAreNonEmptyAndUnique(t *testing.T) {
	seen := map[string]struct{}{}
	for _, eventName := range StableEventNames() {
		if eventName == "" {
			t.Fatalf("event name must not be empty")
		}
		if _, exists := seen[eventName]; exists {
			t.Fatalf("event name %q is duplicated", eventName)
		}
		seen[eventName] = struct{}{}
	}

	required := []string{
		EventAccountRegistered,
		EventAuthSessionIssued,
		EventCharacterCreated,
		EventCharacterSelected,
		EventWorldJoinTicketIssued,
		EventWorldJoinTicketConsumed,
		EventWorldPlayerSpawned,
		EventWorldCharacterSaved,
		EventWorldReconnected,
		EventWorldTickStarted,
		EventWorldTickCompleted,
		EventWorldCommandEnqueued,
		EventWorldCommandRejected,
		EventWorldZoneLoaded,
		EventWorldEntitySpawned,
		EventCombatIntentSubmitted,
		EventCombatAbilityResolved,
		EventNPCSpawned,
		EventAdminActionRequested,
		EventAdminActionApplied,
		EventPersistenceSnapshotSaved,
	}
	for _, eventName := range required {
		if _, exists := seen[eventName]; !exists {
			t.Fatalf("required event name %q was not returned", eventName)
		}
	}
}
