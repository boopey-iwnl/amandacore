package replication

import "testing"

func TestConvergenceRejectsOutOfOrderFramesAndAcceptsNextDelta(t *testing.T) {
	current := Cursor{ShardID: "stonewake.primary", ZoneID: "stonewake_vale", StateVersion: 10, Sequence: 10, Tick: 10}
	stale := NewDeltaFrame(
		Cursor{ShardID: current.ShardID, ZoneID: current.ZoneID, StateVersion: 8, Sequence: 8, Tick: 8},
		Cursor{ShardID: current.ShardID, ZoneID: current.ZoneID, StateVersion: 9, Sequence: 9, Tick: 9},
		DeltaReasonPoll,
		[]ChangedFields{{Domain: "player", EntityID: "char_one", Fields: []string{"position"}, Version: 9}},
		nil,
	)

	next, report := AcceptFrame(current, stale)
	if next != current || !report.RejectedAsStale || report.Converged {
		t.Fatalf("expected stale frame rejection without cursor movement, next=%#v report=%#v", next, report)
	}

	valid := NewDeltaFrame(
		current,
		Cursor{ShardID: current.ShardID, ZoneID: current.ZoneID, StateVersion: 11, Sequence: 11, Tick: 11},
		DeltaReasonCommand,
		[]ChangedFields{{Domain: "player", EntityID: "char_one", Fields: []string{"position"}, Version: 11}},
		nil,
	)
	next, report = AcceptFrame(current, valid)
	if !report.Converged || next.StateVersion != 11 {
		t.Fatalf("expected valid next delta to converge, next=%#v report=%#v", next, report)
	}
}

func FuzzParseCursorNeverPanics(f *testing.F) {
	f.Add("")
	f.Add("42")
	f.Add("stonewake.primary:stonewake_vale:1:2:3")
	f.Add("not:a:valid:cursor")

	f.Fuzz(func(t *testing.T, value string) {
		cursor, err := ParseCursor(value)
		if err == nil && cursor.Token() != "" {
			if _, roundTripErr := ParseCursor(cursor.Token()); roundTripErr != nil {
				t.Fatalf("valid cursor token did not round trip: %q: %v", cursor.Token(), roundTripErr)
			}
		}
	})
}
