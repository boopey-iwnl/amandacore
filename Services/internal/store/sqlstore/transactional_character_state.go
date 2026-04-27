package sqlstore

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func (s *Store) MoveInventorySlot(characterID string, fromSlotIndex int, toSlotIndex int, options filestore.MutationOptions) (*platform.Character, error) {
	if fromSlotIndex < 0 || fromSlotIndex >= platform.InventorySlotCount ||
		toSlotIndex < 0 || toSlotIndex >= platform.InventorySlotCount {
		return nil, filestore.ErrInvalidInventoryMove
	}
	if fromSlotIndex == toSlotIndex {
		return s.GetCharacterByID(characterID)
	}
	return s.mutateCharacterState("inventory.move", characterID, options, func(character *platform.Character) error {
		inventory := platform.NormalizeInventorySlots(character.Inventory)
		fromSlot := inventory[fromSlotIndex]
		toSlot := inventory[toSlotIndex]
		if fromSlot.ItemID == "" || fromSlot.StackCount <= 0 {
			return filestore.ErrInvalidInventoryMove
		}
		inventory[fromSlotIndex] = platform.CharacterInventorySlot{
			SlotIndex:   fromSlotIndex,
			ItemID:      toSlot.ItemID,
			DisplayName: toSlot.DisplayName,
			StackCount:  toSlot.StackCount,
		}
		inventory[toSlotIndex] = platform.CharacterInventorySlot{
			SlotIndex:   toSlotIndex,
			ItemID:      fromSlot.ItemID,
			DisplayName: fromSlot.DisplayName,
			StackCount:  fromSlot.StackCount,
		}
		character.Inventory = inventory
		return nil
	})
}

func (s *Store) GrantInventoryItem(characterID string, grant filestore.InventoryItemGrant, options filestore.MutationOptions) (*platform.Character, error) {
	return s.mutateCharacterState("inventory.grant", characterID, options, func(character *platform.Character) error {
		inventory := platform.NormalizeInventorySlots(character.Inventory)
		if err := grantItemToInventory(&inventory, grant); err != nil {
			return err
		}
		character.Inventory = inventory
		return nil
	})
}

func (s *Store) AcceptQuestProgress(characterID string, progress platform.CharacterQuestProgress, options filestore.MutationOptions) (*platform.Character, error) {
	if strings.TrimSpace(progress.QuestID) == "" {
		return nil, filestore.ErrQuestProgressUnavailable
	}
	return s.mutateCharacterState("quest.accept", characterID, options, func(character *platform.Character) error {
		quests := cloneQuestMap(character.Quests)
		if existing, ok := quests[progress.QuestID]; ok && existing.State != "" && existing.State != "not_started" {
			return filestore.ErrQuestAlreadyAccepted
		}
		if progress.State == "" {
			progress.State = "active"
		}
		if progress.ObjectiveProgress == nil {
			progress.ObjectiveProgress = map[string]platform.CharacterQuestObjectiveProgress{}
		}
		quests[progress.QuestID] = progress
		character.Quests = quests
		character.TrackedQuestIDs = appendTrackedQuest(character.TrackedQuestIDs, progress.QuestID)
		return nil
	})
}

func (s *Store) UpdateQuestProgress(characterID string, progress platform.CharacterQuestProgress, options filestore.MutationOptions) (*platform.Character, error) {
	if strings.TrimSpace(progress.QuestID) == "" {
		return nil, filestore.ErrQuestProgressUnavailable
	}
	return s.mutateCharacterState("quest.progress", characterID, options, func(character *platform.Character) error {
		quests := cloneQuestMap(character.Quests)
		if _, ok := quests[progress.QuestID]; !ok {
			return filestore.ErrQuestProgressUnavailable
		}
		if progress.ObjectiveProgress == nil {
			progress.ObjectiveProgress = map[string]platform.CharacterQuestObjectiveProgress{}
		}
		quests[progress.QuestID] = progress
		character.Quests = quests
		return nil
	})
}

func (s *Store) CompleteQuestWithReward(characterID string, reward filestore.QuestRewardMutation, options filestore.MutationOptions) (*platform.Character, error) {
	if strings.TrimSpace(reward.QuestID) == "" {
		return nil, filestore.ErrQuestProgressUnavailable
	}
	return s.mutateCharacterState("quest.reward", characterID, options, func(character *platform.Character) error {
		quests := cloneQuestMap(character.Quests)
		progress, ok := quests[reward.QuestID]
		if !ok {
			return filestore.ErrQuestProgressUnavailable
		}
		if progress.RewardGrantedAt != 0 || progress.State == "reward_granted" {
			return filestore.ErrQuestRewardAlreadyGiven
		}

		nextInventory := platform.NormalizeInventorySlots(character.Inventory)
		for _, item := range reward.RewardItems {
			if err := grantItemToInventory(&nextInventory, item); err != nil {
				return err
			}
		}

		now := s.now().Unix()
		progress.State = "reward_granted"
		progress.CurrentCount = progress.TargetCount
		if progress.CompletedAt == 0 {
			progress.CompletedAt = now
		}
		progress.RewardGrantedAt = now
		progress.UpdatedAt = now
		quests[reward.QuestID] = progress

		character.Experience += reward.ExperienceDelta
		character.CurrencyCopper += reward.CurrencyCopperDelta
		if character.CurrencyCopper < 0 {
			character.CurrencyCopper = 0
		}
		character.Inventory = nextInventory
		character.Quests = quests
		return nil
	})
}

func (s *Store) GrantLearnedAbility(characterID string, abilityID string, options filestore.MutationOptions) (*platform.Character, error) {
	if strings.TrimSpace(abilityID) == "" {
		return nil, filestore.ErrAbilityNotLearned
	}
	return s.mutateCharacterState("ability.grant", characterID, options, func(character *platform.Character) error {
		character.LearnedAbilityIDs = platform.NormalizeLearnedAbilityIDs(append(character.LearnedAbilityIDs, abilityID))
		character.ActionBarSlots = platform.NormalizeActionBarSlots(character.ActionBarSlots, character.LearnedAbilityIDs)
		return nil
	})
}

func (s *Store) AssignActionBarSlot(characterID string, slotIndex int, abilityID string, options filestore.MutationOptions) (*platform.Character, error) {
	if slotIndex < 0 || slotIndex >= platform.ActionBarSlotCount {
		return nil, fmt.Errorf("action bar slot is out of range")
	}
	return s.mutateCharacterState("actionbar.assign", characterID, options, func(character *platform.Character) error {
		if !characterKnowsAbility(character, abilityID) {
			return filestore.ErrAbilityNotLearned
		}
		slots := platform.NormalizeActionBarSlots(character.ActionBarSlots, character.LearnedAbilityIDs)
		slots[slotIndex] = platform.CharacterActionBarSlot{SlotIndex: slotIndex, AbilityID: abilityID}
		character.ActionBarSlots = slots
		return nil
	})
}

func (s *Store) MoveActionBarSlot(characterID string, fromSlotIndex int, toSlotIndex int, options filestore.MutationOptions) (*platform.Character, error) {
	if fromSlotIndex < 0 || fromSlotIndex >= platform.ActionBarSlotCount ||
		toSlotIndex < 0 || toSlotIndex >= platform.ActionBarSlotCount {
		return nil, fmt.Errorf("action bar slot is out of range")
	}
	if fromSlotIndex == toSlotIndex {
		return s.GetCharacterByID(characterID)
	}
	return s.mutateCharacterState("actionbar.move", characterID, options, func(character *platform.Character) error {
		slots := platform.NormalizeActionBarSlots(character.ActionBarSlots, character.LearnedAbilityIDs)
		fromSlot := slots[fromSlotIndex]
		toSlot := slots[toSlotIndex]
		if fromSlot.AbilityID == "" {
			return filestore.ErrAbilityNotLearned
		}
		slots[fromSlotIndex] = platform.CharacterActionBarSlot{SlotIndex: fromSlotIndex, AbilityID: toSlot.AbilityID}
		slots[toSlotIndex] = platform.CharacterActionBarSlot{SlotIndex: toSlotIndex, AbilityID: fromSlot.AbilityID}
		character.ActionBarSlots = slots
		return nil
	})
}

func (s *Store) ClearActionBarSlot(characterID string, slotIndex int, options filestore.MutationOptions) (*platform.Character, error) {
	if slotIndex < 0 || slotIndex >= platform.ActionBarSlotCount {
		return nil, fmt.Errorf("action bar slot is out of range")
	}
	return s.mutateCharacterState("actionbar.clear", characterID, options, func(character *platform.Character) error {
		slots := platform.NormalizeActionBarSlots(character.ActionBarSlots, character.LearnedAbilityIDs)
		slots[slotIndex] = platform.CharacterActionBarSlot{SlotIndex: slotIndex}
		character.ActionBarSlots = slots
		return nil
	})
}

func (s *Store) mutateCharacterState(
	operation string,
	characterID string,
	options filestore.MutationOptions,
	mutate func(*platform.Character) error,
) (*platform.Character, error) {
	if mutate == nil {
		return nil, fmt.Errorf("character state mutator is required")
	}
	var updated platform.Character
	err := s.WithTransaction("sqlstore."+operation, func(tx *Tx) error {
		if options.MutationKey != "" {
			replayed, found, err := tx.replayedCharacterMutation(characterID, operation, options.MutationKey)
			if err != nil {
				return err
			}
			if found {
				updated = replayed
				return nil
			}
		}

		character, version, err := tx.loadCharacterForMutation(characterID)
		if err != nil {
			return err
		}
		if err := mutate(&character); err != nil {
			return err
		}
		character = platform.NormalizeCharacter(character)
		character.LastSeenAt = s.now().Unix()
		if err := tx.saveCharacterWithVersion(character, version); err != nil {
			return err
		}
		if options.MutationKey != "" {
			if err := tx.recordCharacterMutation(character.ID, operation, options.MutationKey, character); err != nil {
				return err
			}
		}
		updated = character
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (tx *Tx) loadCharacterForMutation(characterID string) (platform.Character, int64, error) {
	row := tx.tx.QueryRow(characterSelectSQL()+` WHERE id = ?`, characterID)
	character, err := scanCharacter(row)
	if errors.Is(err, sql.ErrNoRows) {
		return platform.Character{}, 0, fmt.Errorf("character not found")
	}
	if err != nil {
		return platform.Character{}, 0, err
	}
	if err := tx.loadCharacterCollections(&character); err != nil {
		return platform.Character{}, 0, err
	}
	var version int64
	if err := tx.tx.QueryRow(`SELECT state_version FROM ac_characters WHERE id = ?`, characterID).Scan(&version); err != nil {
		return platform.Character{}, 0, err
	}
	return platform.NormalizeCharacter(character), version, nil
}

func (tx *Tx) saveCharacterWithVersion(character platform.Character, expectedVersion int64) error {
	result, err := tx.tx.Exec(
		`UPDATE ac_characters SET
			level = ?, experience = ?, currency_copper = ?, zone_id = ?, position_x = ?, position_y = ?, position_z = ?,
			equipment_json = ?, professions_json = ?, talents_json = ?, kill_credits_json = ?, tracked_quest_ids_json = ?,
			pvp_stats_json = ?, bind_point_json = ?, travel_state_json = ?, mount_state_json = ?, last_seen_at = ?,
			state_version = state_version + 1
		WHERE id = ? AND state_version = ?`,
		character.Level,
		character.Experience,
		character.CurrencyCopper,
		character.ZoneID,
		character.PositionX,
		character.PositionY,
		character.PositionZ,
		mustJSON(character.Equipment),
		mustJSON(character.Professions),
		mustJSON(character.Talents),
		mustJSON(character.KillCredits),
		mustJSON(character.TrackedQuestIDs),
		mustJSON(character.PvPStats),
		mustJSON(character.BindPoint),
		mustJSON(character.TravelState),
		mustJSON(character.MountState),
		character.LastSeenAt,
		character.ID,
		expectedVersion)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return filestore.ErrCharacterStateConflict
	}
	return saveCharacterCollections(tx.tx, character)
}

func (tx *Tx) loadCharacterCollections(character *platform.Character) error {
	inventory, err := tx.loadInventory(character.ID)
	if err != nil {
		return err
	}
	quests, err := tx.loadQuests(character.ID)
	if err != nil {
		return err
	}
	learnedAbilityIDs, err := tx.loadLearnedAbilities(character.ID)
	if err != nil {
		return err
	}
	actionBarSlots, err := tx.loadActionBarSlots(character.ID)
	if err != nil {
		return err
	}
	character.Inventory = inventory
	character.Quests = quests
	character.LearnedAbilityIDs = learnedAbilityIDs
	character.ActionBarSlots = actionBarSlots
	return nil
}

func (tx *Tx) loadInventory(characterID string) ([]platform.CharacterInventorySlot, error) {
	rows, err := tx.tx.Query(
		`SELECT slot_index, item_id, display_name, stack_count FROM ac_character_inventory WHERE character_id = ? ORDER BY slot_index`,
		characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []platform.CharacterInventorySlot
	for rows.Next() {
		var slot platform.CharacterInventorySlot
		if err := rows.Scan(&slot.SlotIndex, &slot.ItemID, &slot.DisplayName, &slot.StackCount); err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, rows.Err()
}

func (tx *Tx) loadQuests(characterID string) (map[string]platform.CharacterQuestProgress, error) {
	rows, err := tx.tx.Query(
		`SELECT quest_id, state, current_count, target_count, accepted_at, completed_at, reward_granted_at, updated_at, objective_progress_json
		FROM ac_character_quests WHERE character_id = ?`,
		characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quests := map[string]platform.CharacterQuestProgress{}
	for rows.Next() {
		var quest platform.CharacterQuestProgress
		var objectivesJSON string
		if err := rows.Scan(
			&quest.QuestID,
			&quest.State,
			&quest.CurrentCount,
			&quest.TargetCount,
			&quest.AcceptedAt,
			&quest.CompletedAt,
			&quest.RewardGrantedAt,
			&quest.UpdatedAt,
			&objectivesJSON); err != nil {
			return nil, err
		}
		if err := decodeJSON(objectivesJSON, &quest.ObjectiveProgress); err != nil {
			return nil, err
		}
		quests[quest.QuestID] = quest
	}
	return quests, rows.Err()
}

func (tx *Tx) loadLearnedAbilities(characterID string) ([]string, error) {
	rows, err := tx.tx.Query(`SELECT ability_id FROM ac_learned_abilities WHERE character_id = ? ORDER BY ability_id`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var abilities []string
	for rows.Next() {
		var abilityID string
		if err := rows.Scan(&abilityID); err != nil {
			return nil, err
		}
		abilities = append(abilities, abilityID)
	}
	return platform.NormalizeLearnedAbilityIDs(abilities), rows.Err()
}

func (tx *Tx) loadActionBarSlots(characterID string) ([]platform.CharacterActionBarSlot, error) {
	rows, err := tx.tx.Query(
		`SELECT slot_index, ability_id FROM ac_action_bar_slots WHERE character_id = ? ORDER BY slot_index`,
		characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []platform.CharacterActionBarSlot
	for rows.Next() {
		var slot platform.CharacterActionBarSlot
		if err := rows.Scan(&slot.SlotIndex, &slot.AbilityID); err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, rows.Err()
}

func (tx *Tx) replayedCharacterMutation(characterID string, operation string, mutationKey string) (platform.Character, bool, error) {
	row := tx.tx.QueryRow(
		`SELECT response_json FROM ac_character_state_mutations WHERE character_id = ? AND operation = ? AND mutation_key = ?`,
		characterID,
		operation,
		mutationKey)
	var responseJSON string
	if err := row.Scan(&responseJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return platform.Character{}, false, nil
		}
		return platform.Character{}, false, err
	}
	var character platform.Character
	if err := decodeJSON(responseJSON, &character); err != nil {
		return platform.Character{}, false, err
	}
	return platform.NormalizeCharacter(character), true, nil
}

func (tx *Tx) recordCharacterMutation(characterID string, operation string, mutationKey string, character platform.Character) error {
	responseJSON, err := encodeJSON(platform.NormalizeCharacter(character))
	if err != nil {
		return err
	}
	_, err = tx.tx.Exec(
		`INSERT INTO ac_character_state_mutations (mutation_id, character_id, operation, mutation_key, response_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		randomID("mutation"),
		characterID,
		operation,
		mutationKey,
		responseJSON,
		tx.store.now().Unix())
	if err != nil && isConstraintError(err) {
		return filestore.ErrIdempotencyConflict
	}
	return err
}

func grantItemToInventory(inventory *[]platform.CharacterInventorySlot, grant filestore.InventoryItemGrant) error {
	if grant.ItemID == "" || grant.Quantity <= 0 {
		return nil
	}
	maxStack := grant.MaxStack
	if maxStack <= 0 {
		maxStack = 1
	}
	if !grant.Stackable {
		maxStack = 1
	}

	slots := platform.NormalizeInventorySlots(*inventory)
	remaining := grant.Quantity
	if grant.Stackable {
		for index := range slots {
			if slots[index].ItemID != grant.ItemID || slots[index].StackCount >= maxStack {
				continue
			}
			added := minInt(remaining, maxStack-slots[index].StackCount)
			slots[index].StackCount += added
			remaining -= added
			if remaining <= 0 {
				*inventory = slots
				return nil
			}
		}
	}

	for index := range slots {
		if slots[index].ItemID != "" && slots[index].StackCount > 0 {
			continue
		}
		added := 1
		if grant.Stackable {
			added = minInt(remaining, maxStack)
		}
		slots[index] = platform.CharacterInventorySlot{
			SlotIndex:   index,
			ItemID:      grant.ItemID,
			DisplayName: grant.DisplayName,
			StackCount:  added,
		}
		remaining -= added
		if remaining <= 0 {
			*inventory = slots
			return nil
		}
	}
	return filestore.ErrInventoryFull
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func characterKnowsAbility(character *platform.Character, abilityID string) bool {
	if character == nil || abilityID == "" {
		return false
	}
	for _, learnedAbilityID := range platform.NormalizeLearnedAbilityIDs(character.LearnedAbilityIDs) {
		if learnedAbilityID == abilityID {
			return true
		}
	}
	return false
}

func appendTrackedQuest(source []string, questID string) []string {
	if questID == "" {
		return append([]string(nil), source...)
	}
	for _, trackedQuestID := range source {
		if trackedQuestID == questID {
			return append([]string(nil), source...)
		}
	}
	next := append([]string(nil), source...)
	return append(next, questID)
}
