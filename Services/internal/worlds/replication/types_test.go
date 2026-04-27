package replication

import "testing"

func TestCursorTokenRoundTrip(t *testing.T) {
	cursor := Cursor{
		ShardID:      "stonewake_vale.primary",
		ZoneID:       "stonewake_vale",
		StateVersion: 7,
		Sequence:     11,
		Tick:         13,
	}

	parsed, err := ParseCursor(cursor.Token())
	if err != nil {
		t.Fatalf("parse cursor failed: %v", err)
	}
	if parsed != cursor {
		t.Fatalf("cursor mismatch: got %#v want %#v", parsed, cursor)
	}
}

func TestParseNumericCursor(t *testing.T) {
	cursor, err := ParseCursor("42")
	if err != nil {
		t.Fatalf("parse numeric cursor failed: %v", err)
	}
	if cursor.StateVersion != 42 {
		t.Fatalf("expected state version 42, got %#v", cursor)
	}
}

func TestAcceptFrameRejectsStaleDelta(t *testing.T) {
	current := Cursor{StateVersion: 5}
	frame := NewDeltaFrame(Cursor{StateVersion: 2}, Cursor{StateVersion: 4}, DeltaReasonPoll, []ChangedFields{{
		Domain:   "player",
		EntityID: "char_one",
		Fields:   []string{"position"},
		Version:  4,
	}}, nil)

	next, report := AcceptFrame(current, frame)
	if next != current {
		t.Fatalf("stale frame should not advance cursor: got %#v want %#v", next, current)
	}
	if !report.RejectedAsStale || report.Converged {
		t.Fatalf("expected stale rejection report, got %#v", report)
	}
}

func TestAcceptFrameRequiresResyncForFutureDelta(t *testing.T) {
	current := Cursor{StateVersion: 2}
	frame := NewDeltaFrame(Cursor{StateVersion: 5}, Cursor{StateVersion: 6}, DeltaReasonPoll, []ChangedFields{{
		Domain:   "player",
		EntityID: "char_one",
		Fields:   []string{"position"},
		Version:  6,
	}}, nil)

	next, report := AcceptFrame(current, frame)
	if next != current {
		t.Fatalf("future frame should not advance cursor: got %#v want %#v", next, current)
	}
	if !report.RejectedAsFuture || report.Converged {
		t.Fatalf("expected future rejection report, got %#v", report)
	}
}
