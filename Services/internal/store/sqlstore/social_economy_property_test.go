package sqlstore

import (
	"fmt"
	"testing"
	"testing/quick"

	filestore "amandacore/services/internal/store"
)

func TestCurrencyLedgerBalanceMatchesAppliedMutations(t *testing.T) {
	check := func(raw [5]uint8) bool {
		store := newTestStore(t)
		character := seedCharacters(t, store, "LedgerInvariant")[0]

		startingBalance := character.CurrencyCopper
		expected := startingBalance
		for index, value := range raw {
			delta := int(value%9) + 1
			expected += delta
			entry, err := store.AppendCurrencyMutation(filestore.CurrencyLedgerEntry{
				CharacterID: character.ID,
				DeltaCopper: delta,
				Reason:      "property_grant",
				Operation:   "test.currency_property",
				MutationKey: fmt.Sprintf("ledger-property-%d", index),
			})
			if err != nil {
				t.Logf("append mutation failed: %v", err)
				return false
			}
			if entry.BalanceAfter != expected {
				t.Logf("expected ledger balance %d, got %d", expected, entry.BalanceAfter)
				return false
			}
		}

		loaded, err := store.GetCharacterByID(character.ID)
		if err != nil {
			t.Logf("load character failed: %v", err)
			return false
		}
		if loaded.CurrencyCopper != expected {
			t.Logf("expected persisted currency %d, got %d", expected, loaded.CurrencyCopper)
			return false
		}
		return true
	}

	if err := quick.Check(check, &quick.Config{MaxCount: 10}); err != nil {
		t.Fatal(err)
	}
}
