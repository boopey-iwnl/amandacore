package statecutover

import (
	"strings"
	"testing"
	"time"
)

func TestAnalyzeCountsLegacyStateRows(t *testing.T) {
	now := time.Unix(2000, 0)
	report, err := Analyze([]byte(`{
		"accounts": {"acct_1": {"id": "acct_1"}},
		"realms": {"realm_1": {"id": "realm_1"}},
		"characters": {
			"char_1": {
				"id": "char_1",
				"inventory": [{"slotIndex": 0}, {"slotIndex": 1}],
				"quests": {"quest_1": {"state": "active"}},
				"actionBarSlots": [{"slotIndex": 0}]
			}
		},
		"sessions": {"sess_1": {"id": "sess_1"}},
		"worldJoinTickets": {
			"ticket_active": {"ticketId": "ticket_active", "expiresAt": 3000},
			"ticket_expired": {"ticketId": "ticket_expired", "expiresAt": 1000}
		},
		"friends": {"friend_1": {}},
		"guilds": {"guild_1": {}},
		"auctions": {"auction_1": {}},
		"mail": {"mail_1": {}}
	}`), Options{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if report.Counts.Accounts != 1 || report.Counts.Characters != 1 || report.Counts.InventorySlots != 2 {
		t.Fatalf("unexpected core counts: %#v", report.Counts)
	}
	if report.Counts.WorldJoinTickets != 2 || report.Counts.ExpiredWorldTickets != 1 || report.Counts.ImportableWorldTickets != 1 {
		t.Fatalf("unexpected ticket counts: %#v", report.Counts)
	}
	if report.Rows["action_bar_slots"] != 1 || report.Rows["quest_states"] != 1 || report.Rows["auctions"] != 1 {
		t.Fatalf("unexpected row report: %#v", report.Rows)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected excluded ticket warning")
	}
}

func TestAnalyzeAllowsExpiredTicketsWhenExplicit(t *testing.T) {
	report, err := Analyze([]byte(`{
		"worldJoinTickets": {
			"ticket_expired": {"ticketId": "ticket_expired", "expiresAt": 1000}
		}
	}`), Options{Now: time.Unix(2000, 0), IncludeExpiredTickets: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Counts.ImportableWorldTickets != 1 {
		t.Fatalf("expected expired ticket to become importable, got %#v", report.Counts)
	}
}

func TestAnalyzeRejectsInvalidState(t *testing.T) {
	_, err := Analyze([]byte(`[]`), Options{})
	if err == nil || !strings.Contains(err.Error(), "JSON object") {
		t.Fatalf("expected invalid state rejection, got %v", err)
	}
}
