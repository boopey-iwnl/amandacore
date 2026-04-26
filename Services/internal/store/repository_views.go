package store

import "amandacore/services/internal/platform"

func (s *FileStore) GetCharacterInventory(characterID string) ([]platform.CharacterInventorySlot, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	return cloneInventorySlots(character.Inventory), nil
}

func (s *FileStore) GetCharacterQuestProgress(characterID string) (map[string]platform.CharacterQuestProgress, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	return cloneQuestProgress(character.Quests), nil
}

func (s *FileStore) UpdateCharacterQuestProgress(characterID string, quests map[string]platform.CharacterQuestProgress) (*platform.Character, error) {
	return s.UpdateCharacterAtomically("repository.quest_update", characterID, func(character *platform.Character) error {
		character.Quests = cloneQuestProgress(quests)
		return nil
	})
}

func (s *FileStore) GetLearnedAbilities(characterID string) ([]string, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), character.LearnedAbilityIDs...), nil
}

func (s *FileStore) UpdateLearnedAbilities(characterID string, learnedAbilityIDs []string) (*platform.Character, error) {
	return s.UpdateCharacterAtomically("repository.ability_update", characterID, func(character *platform.Character) error {
		character.LearnedAbilityIDs = append([]string(nil), learnedAbilityIDs...)
		return nil
	})
}

func (s *FileStore) GetActionBarSlots(characterID string) ([]platform.CharacterActionBarSlot, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	return cloneActionBarSlots(character.ActionBarSlots, character.LearnedAbilityIDs), nil
}

func (s *FileStore) UpdateActionBarSlots(characterID string, actionBarSlots []platform.CharacterActionBarSlot) (*platform.Character, error) {
	return s.UpdateCharacterAtomically("repository.action_bar_update", characterID, func(character *platform.Character) error {
		character.ActionBarSlots = cloneActionBarSlots(actionBarSlots, character.LearnedAbilityIDs)
		return nil
	})
}

func cloneQuestProgress(source map[string]platform.CharacterQuestProgress) map[string]platform.CharacterQuestProgress {
	if source == nil {
		return map[string]platform.CharacterQuestProgress{}
	}

	cloned := make(map[string]platform.CharacterQuestProgress, len(source))
	for questID, progress := range source {
		if progress.ObjectiveProgress != nil {
			objectives := make(map[string]platform.CharacterQuestObjectiveProgress, len(progress.ObjectiveProgress))
			for objectiveID, objective := range progress.ObjectiveProgress {
				objectives[objectiveID] = objective
			}
			progress.ObjectiveProgress = objectives
		}
		cloned[questID] = progress
	}
	return cloned
}
