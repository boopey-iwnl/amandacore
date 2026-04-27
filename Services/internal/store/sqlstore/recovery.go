package sqlstore

import (
	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func (s *Store) LoadSessionRecoveryState(characterID string) (filestore.SessionRecoveryState, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return filestore.SessionRecoveryState{}, err
	}
	return filestore.SessionRecoveryState{
		CharacterID:       character.ID,
		AccountID:         character.AccountID,
		RealmID:           character.RealmID,
		DisplayName:       character.DisplayName,
		ZoneID:            character.ZoneID,
		X:                 character.PositionX,
		Y:                 character.PositionY,
		Z:                 character.PositionZ,
		Level:             character.Level,
		Experience:        character.Experience,
		CurrencyCopper:    character.CurrencyCopper,
		Inventory:         append([]platform.CharacterInventorySlot(nil), character.Inventory...),
		LearnedAbilityIDs: append([]string(nil), character.LearnedAbilityIDs...),
		ActionBarSlots:    append([]platform.CharacterActionBarSlot(nil), character.ActionBarSlots...),
		Quests:            cloneQuestMap(character.Quests),
		TrackedQuestIDs:   append([]string(nil), character.TrackedQuestIDs...),
		LastSeenAt:        character.LastSeenAt,
	}, nil
}
