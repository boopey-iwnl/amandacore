package sqlstore

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func (s *Store) CreateCharacter(accountID string, realmID string, displayName string, raceID string, classID string, archetypeID string) (platform.Character, error) {
	archetypeID, raceID, classID = platform.NormalizeCharacterIdentity(archetypeID, raceID, classID)
	now := s.now().Unix()
	character := platform.NormalizeCharacter(platform.Character{
		ID:              randomID("char"),
		AccountID:       accountID,
		RealmID:         realmID,
		DisplayName:     displayName,
		RaceID:          raceID,
		ClassID:         classID,
		ArchetypeID:     archetypeID,
		Level:           1,
		Experience:      0,
		CurrencyCopper:  platform.StarterCurrencyCopper,
		ZoneID:          platform.DefaultStarterZoneID,
		PositionX:       platform.DefaultStarterSpawnX,
		PositionY:       platform.DefaultStarterSpawnY,
		PositionZ:       platform.DefaultStarterSpawnZ,
		Inventory:       platform.DefaultStarterInventory(),
		Equipment:       platform.DefaultEquipmentSlots(),
		Professions:     []platform.CharacterProfessionState{},
		Talents:         map[string]int{},
		Quests:          map[string]platform.CharacterQuestProgress{},
		KillCredits:     map[string]platform.CharacterKillCredit{},
		TrackedQuestIDs: []string{},
		LastSeenAt:      now,
	})

	if err := s.WithTransaction("sqlstore.character_create", func(tx *Tx) error {
		return tx.CreateCharacter(character)
	}); err != nil {
		return platform.Character{}, err
	}
	return character, nil
}

func (tx *Tx) CreateCharacter(character platform.Character) error {
	if character.ID == "" {
		character.ID = randomID("char")
	}
	if character.LastSeenAt == 0 {
		character.LastSeenAt = tx.store.now().Unix()
	}
	character = platform.NormalizeCharacter(character)

	equipmentJSON, err := encodeJSON(character.Equipment)
	if err != nil {
		return err
	}
	professionsJSON, err := encodeJSON(character.Professions)
	if err != nil {
		return err
	}
	talentsJSON, err := encodeJSON(character.Talents)
	if err != nil {
		return err
	}
	killCreditsJSON, err := encodeJSON(character.KillCredits)
	if err != nil {
		return err
	}
	trackedQuestIDsJSON, err := encodeJSON(character.TrackedQuestIDs)
	if err != nil {
		return err
	}
	pvpStatsJSON, err := encodeJSON(character.PvPStats)
	if err != nil {
		return err
	}
	bindPointJSON, err := encodeJSON(character.BindPoint)
	if err != nil {
		return err
	}
	travelStateJSON, err := encodeJSON(character.TravelState)
	if err != nil {
		return err
	}
	mountStateJSON, err := encodeJSON(character.MountState)
	if err != nil {
		return err
	}

	_, err = tx.tx.Exec(
		`INSERT INTO ac_characters (
			id, account_id, realm_id, display_name, normalized_display_name, race_id, class_id, archetype_id,
			level, experience, currency_copper, zone_id, position_x, position_y, position_z,
			equipment_json, professions_json, talents_json, kill_credits_json, tracked_quest_ids_json,
			pvp_stats_json, bind_point_json, travel_state_json, mount_state_json, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		character.ID,
		character.AccountID,
		character.RealmID,
		character.DisplayName,
		normalize(character.DisplayName),
		character.RaceID,
		character.ClassID,
		character.ArchetypeID,
		character.Level,
		character.Experience,
		character.CurrencyCopper,
		character.ZoneID,
		character.PositionX,
		character.PositionY,
		character.PositionZ,
		equipmentJSON,
		professionsJSON,
		talentsJSON,
		killCreditsJSON,
		trackedQuestIDsJSON,
		pvpStatsJSON,
		bindPointJSON,
		travelStateJSON,
		mountStateJSON,
		character.LastSeenAt)
	if err != nil {
		if isConstraintError(err) {
			return filestore.ErrCharacterNameExists
		}
		return err
	}
	return saveCharacterCollections(tx.tx, character)
}

func (s *Store) ListCharacters(accountID string, realmID string) ([]platform.Character, error) {
	rows, err := s.db.Query(
		`SELECT id FROM ac_characters WHERE account_id = ? AND (? = '' OR realm_id = ?) ORDER BY display_name`,
		accountID,
		realmID,
		realmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	characters := make([]platform.Character, 0, len(ids))
	for _, id := range ids {
		character, err := s.GetCharacterByID(id)
		if err != nil {
			return nil, err
		}
		characters = append(characters, *character)
	}
	return characters, nil
}

func (s *Store) GetCharacterByID(characterID string) (*platform.Character, error) {
	return s.getCharacter(`WHERE id = ?`, characterID)
}

func (s *Store) GetCharacterByName(realmID string, displayName string) (*platform.Character, error) {
	return s.getCharacter(`WHERE realm_id = ? AND normalized_display_name = ?`, realmID, normalize(displayName))
}

func (s *Store) UpdateCharacterState(characterID string, zoneID string, x float64, y float64, z float64) (*platform.Character, error) {
	now := s.now().Unix()
	var updated platform.Character
	if err := s.WithTransaction("sqlstore.character_position", func(tx *Tx) error {
		character, version, err := tx.loadCharacterForMutation(characterID)
		if err != nil {
			return err
		}
		character.ZoneID = zoneID
		character.PositionX = x
		character.PositionY = y
		character.PositionZ = z
		character.LastSeenAt = now
		if err := tx.saveCharacterWithVersion(platform.NormalizeCharacter(character), version); err != nil {
			return err
		}
		if err := tx.insertPositionSnapshot(filestore.CharacterPositionSnapshot{
			SnapshotID:       randomID("pos"),
			CharacterID:      characterID,
			ZoneID:           zoneID,
			X:                x,
			Y:                y,
			Z:                z,
			CapturedAt:       now,
			Reason:           "character_state_update",
			CharacterVersion: version + 1,
		}); err != nil {
			return err
		}
		updated = platform.NormalizeCharacter(character)
		return nil
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *Store) GetCharacterPositionSnapshots(characterID string, limit int) ([]filestore.CharacterPositionSnapshot, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	rows, err := s.db.Query(
		`SELECT snapshot_id, character_id, world_session_token, zone_id, position_x, position_y, position_z, captured_at, reason, character_version
		FROM ac_character_position_snapshots
		WHERE character_id = ?
		ORDER BY captured_at DESC, snapshot_id DESC
		LIMIT ?`,
		characterID,
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []filestore.CharacterPositionSnapshot
	for rows.Next() {
		var snapshot filestore.CharacterPositionSnapshot
		if err := rows.Scan(
			&snapshot.SnapshotID,
			&snapshot.CharacterID,
			&snapshot.WorldSessionToken,
			&snapshot.ZoneID,
			&snapshot.X,
			&snapshot.Y,
			&snapshot.Z,
			&snapshot.CapturedAt,
			&snapshot.Reason,
			&snapshot.CharacterVersion); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (tx *Tx) insertPositionSnapshot(snapshot filestore.CharacterPositionSnapshot) error {
	if snapshot.SnapshotID == "" {
		snapshot.SnapshotID = randomID("pos")
	}
	_, err := tx.tx.Exec(
		`INSERT INTO ac_character_position_snapshots (
			snapshot_id, character_id, world_session_token, zone_id, position_x, position_y, position_z,
			captured_at, reason, character_version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.SnapshotID,
		snapshot.CharacterID,
		snapshot.WorldSessionToken,
		snapshot.ZoneID,
		snapshot.X,
		snapshot.Y,
		snapshot.Z,
		snapshot.CapturedAt,
		snapshot.Reason,
		snapshot.CharacterVersion)
	return err
}

func (s *Store) GetCharacterInventory(characterID string) ([]platform.CharacterInventorySlot, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	return append([]platform.CharacterInventorySlot(nil), character.Inventory...), nil
}

func (s *Store) UpdateCharacterInventory(characterID string, inventory []platform.CharacterInventorySlot) (*platform.Character, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	character.Inventory = append([]platform.CharacterInventorySlot(nil), inventory...)
	character.LastSeenAt = s.now().Unix()
	if err := s.WithTransaction("sqlstore.inventory_update", func(tx *Tx) error {
		return saveCharacterCollections(tx.tx, platform.NormalizeCharacter(*character))
	}); err != nil {
		return nil, err
	}
	return s.GetCharacterByID(characterID)
}

func (s *Store) UpdateCharacterEconomy(characterID string, currencyCopper int, inventory []platform.CharacterInventorySlot, equipment []platform.CharacterEquipmentSlot) (*platform.Character, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	character.CurrencyCopper = currencyCopper
	character.Inventory = append([]platform.CharacterInventorySlot(nil), inventory...)
	character.Equipment = append([]platform.CharacterEquipmentSlot(nil), equipment...)
	character.LastSeenAt = s.now().Unix()
	if err := s.saveCharacter(*character); err != nil {
		return nil, err
	}
	return s.GetCharacterByID(characterID)
}

func (s *Store) GetCharacterQuestProgress(characterID string) (map[string]platform.CharacterQuestProgress, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	return cloneQuestMap(character.Quests), nil
}

func (s *Store) UpdateCharacterQuestProgress(characterID string, quests map[string]platform.CharacterQuestProgress) (*platform.Character, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	character.Quests = cloneQuestMap(quests)
	character.LastSeenAt = s.now().Unix()
	if err := s.saveCharacter(*character); err != nil {
		return nil, err
	}
	return s.GetCharacterByID(characterID)
}

func (s *Store) UpdateCharacterTrackedQuests(characterID string, trackedQuestIDs []string) (*platform.Character, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	character.TrackedQuestIDs = append([]string(nil), trackedQuestIDs...)
	character.LastSeenAt = s.now().Unix()
	if err := s.saveCharacter(*character); err != nil {
		return nil, err
	}
	return s.GetCharacterByID(characterID)
}

func (s *Store) GetLearnedAbilities(characterID string) ([]string, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), character.LearnedAbilityIDs...), nil
}

func (s *Store) UpdateLearnedAbilities(characterID string, learnedAbilityIDs []string) (*platform.Character, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	character.LearnedAbilityIDs = append([]string(nil), learnedAbilityIDs...)
	character.LastSeenAt = s.now().Unix()
	if err := s.saveCharacter(*character); err != nil {
		return nil, err
	}
	return s.GetCharacterByID(characterID)
}

func (s *Store) GetActionBarSlots(characterID string) ([]platform.CharacterActionBarSlot, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	return append([]platform.CharacterActionBarSlot(nil), character.ActionBarSlots...), nil
}

func (s *Store) UpdateActionBarSlots(characterID string, actionBarSlots []platform.CharacterActionBarSlot) (*platform.Character, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	character.ActionBarSlots = append([]platform.CharacterActionBarSlot(nil), actionBarSlots...)
	character.LastSeenAt = s.now().Unix()
	if err := s.saveCharacter(*character); err != nil {
		return nil, err
	}
	return s.GetCharacterByID(characterID)
}

func (s *Store) UpdateCharacterProgression(
	characterID string,
	experience int,
	currencyCopper int,
	inventory []platform.CharacterInventorySlot,
	learnedAbilityIDs []string,
	actionBarSlots []platform.CharacterActionBarSlot,
	quests map[string]platform.CharacterQuestProgress,
) (*platform.Character, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return nil, err
	}
	character.Experience = experience
	character.CurrencyCopper = currencyCopper
	character.Inventory = append([]platform.CharacterInventorySlot(nil), inventory...)
	character.LearnedAbilityIDs = append([]string(nil), learnedAbilityIDs...)
	character.ActionBarSlots = append([]platform.CharacterActionBarSlot(nil), actionBarSlots...)
	character.Quests = cloneQuestMap(quests)
	character.LastSeenAt = s.now().Unix()
	if err := s.saveCharacter(*character); err != nil {
		return nil, err
	}
	return s.GetCharacterByID(characterID)
}

func (s *Store) saveCharacter(character platform.Character) error {
	return s.WithTransaction("sqlstore.character_save", func(tx *Tx) error {
		if _, err := tx.tx.Exec(
			`UPDATE ac_characters SET
				level = ?, experience = ?, currency_copper = ?, zone_id = ?, position_x = ?, position_y = ?, position_z = ?,
				equipment_json = ?, professions_json = ?, talents_json = ?, kill_credits_json = ?, tracked_quest_ids_json = ?,
				pvp_stats_json = ?, bind_point_json = ?, travel_state_json = ?, mount_state_json = ?, last_seen_at = ?
			WHERE id = ?`,
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
			character.ID); err != nil {
			return err
		}
		return saveCharacterCollections(tx.tx, platform.NormalizeCharacter(character))
	})
}

func (s *Store) getCharacter(where string, args ...any) (*platform.Character, error) {
	row := s.db.QueryRow(characterSelectSQL()+" "+where, args...)
	character, err := scanCharacter(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("character not found")
	}
	if err != nil {
		return nil, err
	}
	if err := s.loadCharacterCollections(&character); err != nil {
		return nil, err
	}
	character = platform.NormalizeCharacter(character)
	return &character, nil
}

func characterSelectSQL() string {
	return `SELECT
		id, account_id, realm_id, display_name, race_id, class_id, archetype_id, level, experience, currency_copper,
		zone_id, position_x, position_y, position_z, equipment_json, professions_json, talents_json, kill_credits_json,
		tracked_quest_ids_json, pvp_stats_json, bind_point_json, travel_state_json, mount_state_json, last_seen_at
	FROM ac_characters`
}

func scanCharacter(scanner rowScanner) (platform.Character, error) {
	var character platform.Character
	var equipmentJSON, professionsJSON, talentsJSON, killCreditsJSON, trackedQuestIDsJSON string
	var pvpStatsJSON, bindPointJSON, travelStateJSON, mountStateJSON string
	if err := scanner.Scan(
		&character.ID,
		&character.AccountID,
		&character.RealmID,
		&character.DisplayName,
		&character.RaceID,
		&character.ClassID,
		&character.ArchetypeID,
		&character.Level,
		&character.Experience,
		&character.CurrencyCopper,
		&character.ZoneID,
		&character.PositionX,
		&character.PositionY,
		&character.PositionZ,
		&equipmentJSON,
		&professionsJSON,
		&talentsJSON,
		&killCreditsJSON,
		&trackedQuestIDsJSON,
		&pvpStatsJSON,
		&bindPointJSON,
		&travelStateJSON,
		&mountStateJSON,
		&character.LastSeenAt); err != nil {
		return platform.Character{}, err
	}
	if err := decodeJSON(equipmentJSON, &character.Equipment); err != nil {
		return platform.Character{}, err
	}
	if err := decodeJSON(professionsJSON, &character.Professions); err != nil {
		return platform.Character{}, err
	}
	if err := decodeJSON(talentsJSON, &character.Talents); err != nil {
		return platform.Character{}, err
	}
	if err := decodeJSON(killCreditsJSON, &character.KillCredits); err != nil {
		return platform.Character{}, err
	}
	if err := decodeJSON(trackedQuestIDsJSON, &character.TrackedQuestIDs); err != nil {
		return platform.Character{}, err
	}
	if err := decodeJSON(pvpStatsJSON, &character.PvPStats); err != nil {
		return platform.Character{}, err
	}
	if err := decodeJSON(bindPointJSON, &character.BindPoint); err != nil {
		return platform.Character{}, err
	}
	if err := decodeJSON(travelStateJSON, &character.TravelState); err != nil {
		return platform.Character{}, err
	}
	if err := decodeJSON(mountStateJSON, &character.MountState); err != nil {
		return platform.Character{}, err
	}
	return character, nil
}

func (s *Store) loadCharacterCollections(character *platform.Character) error {
	inventory, err := s.loadInventory(character.ID)
	if err != nil {
		return err
	}
	quests, err := s.loadQuests(character.ID)
	if err != nil {
		return err
	}
	learnedAbilityIDs, err := s.loadLearnedAbilities(character.ID)
	if err != nil {
		return err
	}
	actionBarSlots, err := s.loadActionBarSlots(character.ID)
	if err != nil {
		return err
	}
	character.Inventory = inventory
	character.Quests = quests
	character.LearnedAbilityIDs = learnedAbilityIDs
	character.ActionBarSlots = actionBarSlots
	return nil
}

func (s *Store) loadInventory(characterID string) ([]platform.CharacterInventorySlot, error) {
	rows, err := s.db.Query(
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

func (s *Store) loadQuests(characterID string) (map[string]platform.CharacterQuestProgress, error) {
	rows, err := s.db.Query(
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

func (s *Store) loadLearnedAbilities(characterID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT ability_id FROM ac_learned_abilities WHERE character_id = ? ORDER BY ability_id`, characterID)
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
	sort.Strings(abilities)
	return abilities, rows.Err()
}

func (s *Store) loadActionBarSlots(characterID string) ([]platform.CharacterActionBarSlot, error) {
	rows, err := s.db.Query(
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

func saveCharacterCollections(tx *sql.Tx, character platform.Character) error {
	if _, err := tx.Exec(`DELETE FROM ac_character_inventory WHERE character_id = ?`, character.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM ac_character_quests WHERE character_id = ?`, character.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM ac_learned_abilities WHERE character_id = ?`, character.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM ac_action_bar_slots WHERE character_id = ?`, character.ID); err != nil {
		return err
	}

	for _, slot := range character.Inventory {
		if _, err := tx.Exec(
			`INSERT INTO ac_character_inventory (
				character_id, slot_index, item_id, display_name, stack_count, slot_version, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			character.ID,
			slot.SlotIndex,
			slot.ItemID,
			slot.DisplayName,
			slot.StackCount,
			1,
			character.LastSeenAt); err != nil {
			return err
		}
	}
	for _, quest := range character.Quests {
		objectivesJSON, err := encodeJSON(quest.ObjectiveProgress)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO ac_character_quests (
				character_id, quest_id, state, current_count, target_count, accepted_at, completed_at,
				reward_granted_at, updated_at, objective_progress_json, progress_version, updated_at_row
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			character.ID,
			quest.QuestID,
			quest.State,
			quest.CurrentCount,
			quest.TargetCount,
			quest.AcceptedAt,
			quest.CompletedAt,
			quest.RewardGrantedAt,
			quest.UpdatedAt,
			objectivesJSON,
			1,
			character.LastSeenAt); err != nil {
			return err
		}
	}
	for _, abilityID := range character.LearnedAbilityIDs {
		if _, err := tx.Exec(
			`INSERT INTO ac_learned_abilities (character_id, ability_id, learned_at, updated_at) VALUES (?, ?, ?, ?)`,
			character.ID,
			abilityID,
			character.LastSeenAt,
			character.LastSeenAt); err != nil {
			return err
		}
	}
	for _, slot := range character.ActionBarSlots {
		if _, err := tx.Exec(
			`INSERT INTO ac_action_bar_slots (character_id, slot_index, ability_id, slot_version, updated_at) VALUES (?, ?, ?, ?, ?)`,
			character.ID,
			slot.SlotIndex,
			slot.AbilityID,
			1,
			character.LastSeenAt); err != nil {
			return err
		}
	}
	return nil
}

func cloneQuestMap(source map[string]platform.CharacterQuestProgress) map[string]platform.CharacterQuestProgress {
	cloned := make(map[string]platform.CharacterQuestProgress, len(source))
	for questID, quest := range source {
		if quest.ObjectiveProgress != nil {
			objectives := make(map[string]platform.CharacterQuestObjectiveProgress, len(quest.ObjectiveProgress))
			for objectiveID, objective := range quest.ObjectiveProgress {
				objectives[objectiveID] = objective
			}
			quest.ObjectiveProgress = objectives
		}
		cloned[questID] = quest
	}
	return cloned
}

func mustJSON(value any) string {
	payload, err := encodeJSON(value)
	if err != nil {
		return "null"
	}
	return payload
}
