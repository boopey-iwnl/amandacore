package worlds

import (
	"errors"
	"fmt"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
	storepkg "amandacore/services/internal/store"
)

const defaultPartyQuestCreditRadius = 48.0

func (s *worldServer) defaultQuestProgress(quest questDefinition) platform.CharacterQuestProgress {
	return platform.CharacterQuestProgress{
		QuestID:      quest.ID,
		State:        questStateNotStarted,
		CurrentCount: 0,
		TargetCount:  quest.TargetCount,
	}
}

func (s *worldServer) normalizeQuestProgress(quest questDefinition, progress platform.CharacterQuestProgress) platform.CharacterQuestProgress {
	if progress.QuestID == "" {
		progress.QuestID = quest.ID
	}
	if progress.TargetCount <= 0 {
		progress.TargetCount = quest.TargetCount
	}
	if progress.CurrentCount < 0 {
		progress.CurrentCount = 0
	}
	if progress.CurrentCount > progress.TargetCount {
		progress.CurrentCount = progress.TargetCount
	}
	if progress.RewardGrantedAt != 0 {
		progress.State = questStateRewardGranted
		if progress.CompletedAt == 0 {
			progress.CompletedAt = progress.RewardGrantedAt
		}
		progress.CurrentCount = progress.TargetCount
		return progress
	}
	if progress.CompletedAt != 0 || progress.State == questStateCompleted || progress.CurrentCount >= progress.TargetCount {
		progress.State = questStateCompleted
		progress.CurrentCount = progress.TargetCount
		return progress
	}
	if progress.State == "" {
		progress.State = questStateNotStarted
	}
	return progress
}

func (s *worldServer) loadQuestProgressFromCharacter(character *platform.Character) map[string]platform.CharacterQuestProgress {
	progressByQuest := map[string]platform.CharacterQuestProgress{}
	if character != nil && character.Quests != nil {
		for questID, progress := range character.Quests {
			if quest, ok := s.quests[questID]; ok {
				progressByQuest[questID] = s.normalizeQuestProgress(quest, progress)
			} else {
				progressByQuest[questID] = progress
			}
		}
	}

	for _, questID := range s.questOrder {
		if _, exists := progressByQuest[questID]; exists {
			continue
		}
		progressByQuest[questID] = s.defaultQuestProgress(s.quests[questID])
	}
	return progressByQuest
}

func (s *worldServer) applyCharacterProgressionLocked(session *worldSessionState, character *platform.Character) {
	if session == nil || character == nil {
		return
	}

	session.Experience = character.Experience
	session.Level = character.Level
	session.ClassID = character.ClassID
	session.CurrencyCopper = character.CurrencyCopper
	session.Inventory = platform.NormalizeInventorySlots(character.Inventory)
	session.Equipment = platform.NormalizeEquipmentSlots(character.Equipment)
	session.Professions = platform.NormalizeProfessionStates(character.Professions)
	session.Talents = platform.NormalizeTalentRanks(character.Talents)
	session.LearnedAbilityIDs = platform.NormalizeLearnedAbilityIDs(character.LearnedAbilityIDs)
	session.ActionBarSlots = platform.NormalizeActionBarSlots(character.ActionBarSlots, session.LearnedAbilityIDs)
	session.QuestProgress = s.loadQuestProgressFromCharacter(character)
	session.TrackedQuestIDs = s.normalizeTrackedQuestIDsLocked(character.TrackedQuestIDs, session.QuestProgress)
	s.applyDerivedStatsLocked(session)
}

func (s *worldServer) buildCharacterQuestMap(progressByQuest map[string]platform.CharacterQuestProgress) map[string]platform.CharacterQuestProgress {
	result := map[string]platform.CharacterQuestProgress{}
	for questID, progress := range progressByQuest {
		if quest, ok := s.quests[questID]; ok {
			result[questID] = s.normalizeQuestProgress(quest, progress)
		} else {
			result[questID] = progress
		}
	}
	return result
}

func (s *worldServer) persistSessionProgressionLocked(session *worldSessionState) error {
	persistStartedAt := time.Now()
	character, err := s.store.UpdateCharacterProgression(
		session.CharacterID,
		session.Experience,
		session.CurrencyCopper,
		session.Inventory,
		session.LearnedAbilityIDs,
		session.ActionBarSlots,
		s.buildCharacterQuestMap(session.QuestProgress))
	s.recordPersistenceDuration("character_progression", persistStartedAt, err)
	if err != nil {
		return err
	}

	session.Experience = character.Experience
	session.Level = character.Level
	session.CurrencyCopper = character.CurrencyCopper
	session.Inventory = platform.NormalizeInventorySlots(character.Inventory)
	session.Equipment = platform.NormalizeEquipmentSlots(character.Equipment)
	session.Professions = platform.NormalizeProfessionStates(character.Professions)
	session.Talents = platform.NormalizeTalentRanks(character.Talents)
	session.LearnedAbilityIDs = platform.NormalizeLearnedAbilityIDs(character.LearnedAbilityIDs)
	session.ActionBarSlots = platform.NormalizeActionBarSlots(character.ActionBarSlots, session.LearnedAbilityIDs)
	session.QuestProgress = s.loadQuestProgressFromCharacter(character)
	session.TrackedQuestIDs = s.normalizeTrackedQuestIDsLocked(session.TrackedQuestIDs, session.QuestProgress)
	s.applyDerivedStatsLocked(session)
	persistStartedAt = time.Now()
	_, err = s.store.UpdateCharacterTrackedQuests(session.CharacterID, session.TrackedQuestIDs)
	s.recordPersistenceDuration("character_tracked_quests", persistStartedAt, err)
	if err != nil {
		return err
	}
	return nil
}

func (s *worldServer) acceptQuestLocked(session *worldSessionState, questID string) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}

	quest, found := s.quests[questID]
	if !found {
		return fmt.Errorf("quest is not available")
	}

	progress := s.normalizeQuestProgress(quest, session.QuestProgress[quest.ID])
	switch progress.State {
	case questStateNotStarted:
		if err := s.validateQuestStartLocked(session, quest); err != nil {
			return err
		}
		now := time.Now().Unix()
		progress.State = questStateActive
		progress.AcceptedAt = now
		progress.UpdatedAt = now
		session.QuestProgress[quest.ID] = progress
		s.trackQuestLocked(session, quest.ID)
		observability.LogEvent("world-service", "world.quest_accepted", map[string]any{
			"worldSessionToken": session.Token,
			"accountId":         session.AccountID,
			"characterId":       session.CharacterID,
			"questId":           quest.ID,
		})
		return s.persistSessionProgressionLocked(session)
	case questStateActive, questStateCompleted:
		return s.completeOrTurnInQuestLocked(session, quest, progress)
	case questStateRewardGranted:
		return fmt.Errorf("quest is already completed")
	default:
		return fmt.Errorf("quest is not available")
	}
}

func (s *worldServer) validateQuestStartLocked(session *worldSessionState, quest questDefinition) error {
	if session.CurrentTargetID != quest.GiverNPCID {
		return fmt.Errorf("target this quest giver first")
	}
	if !s.friendlyInRangeLocked(session, quest.GiverNPCID) {
		return fmt.Errorf("move closer to the quest giver")
	}
	for _, prerequisiteID := range quest.PrerequisiteIDs {
		prereq, found := s.quests[prerequisiteID]
		if !found {
			return fmt.Errorf("quest prerequisite is unavailable")
		}
		progress := s.normalizeQuestProgress(prereq, session.QuestProgress[prerequisiteID])
		if progress.State != questStateRewardGranted {
			return fmt.Errorf("complete the previous quest first")
		}
	}
	return nil
}

func (s *worldServer) completeOrTurnInQuestLocked(session *worldSessionState, quest questDefinition, progress platform.CharacterQuestProgress) error {
	if progress.State == questStateActive {
		if !s.questObjectiveReadyLocked(session, quest, progress) {
			return fmt.Errorf("quest objective is not complete")
		}
		now := time.Now().Unix()
		progress.State = questStateCompleted
		progress.CurrentCount = progress.TargetCount
		progress.CompletedAt = now
		progress.UpdatedAt = now
		session.QuestProgress[quest.ID] = progress
		observability.LogEvent("world-service", "world.quest_completed", map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"questId":           quest.ID,
		})
	}

	if session.CurrentTargetID != quest.TurnInNPCID {
		if progress.State == questStateCompleted && progress.RewardGrantedAt == 0 {
			return s.persistSessionProgressionLocked(session)
		}
		return fmt.Errorf("return to the quest turn-in")
	}
	if !s.friendlyInRangeLocked(session, quest.TurnInNPCID) {
		return fmt.Errorf("move closer to the quest turn-in")
	}
	if progress.RewardGrantedAt != 0 {
		return fmt.Errorf("quest reward is already granted")
	}

	return s.grantQuestRewardLocked(session, quest, progress)
}

func (s *worldServer) questObjectiveReadyLocked(session *worldSessionState, quest questDefinition, progress platform.CharacterQuestProgress) bool {
	switch quest.ObjectiveType {
	case objectiveTalk:
		return session.CurrentTargetID == quest.TargetEntityID && s.friendlyInRangeLocked(session, quest.TargetEntityID)
	case objectiveKill, objectiveCollect:
		return progress.CurrentCount >= progress.TargetCount
	case objectiveTrainer:
		return s.sessionKnowsAbilityLocked(session, quest.TargetEntityID)
	case objectiveExplore:
		if progress.CurrentCount >= progress.TargetCount {
			return true
		}
		return distance2D(session.X, session.Y, quest.MarkerX, quest.MarkerY) <= starterInteractRadius ||
			(session.CurrentTargetID == quest.TurnInNPCID && s.friendlyInRangeLocked(session, quest.TurnInNPCID))
	case objectiveUse:
		return session.CurrentTargetID == quest.TargetEntityID && s.friendlyInRangeLocked(session, quest.TargetEntityID)
	default:
		return false
	}
}

func (s *worldServer) grantQuestRewardLocked(session *worldSessionState, quest questDefinition, progress platform.CharacterQuestProgress) error {
	now := time.Now().Unix()
	session.Experience += quest.RewardXP
	session.CurrencyCopper += quest.RewardCopper
	for _, item := range quest.RewardItems {
		if err := addItemToInventory(&session.Inventory, item); err != nil {
			return err
		}
	}

	progress.State = questStateRewardGranted
	progress.CurrentCount = progress.TargetCount
	progress.CompletedAt = now
	progress.RewardGrantedAt = now
	progress.UpdatedAt = now
	session.QuestProgress[quest.ID] = progress

	if err := s.persistSessionProgressionLocked(session); err != nil {
		return err
	}

	rewardCurrency := breakdownCurrency(quest.RewardCopper)
	observability.LogEvent("world-service", "world.quest_reward_granted", map[string]any{
		"worldSessionToken":         session.Token,
		"characterId":               session.CharacterID,
		"questId":                   quest.ID,
		"rewardXp":                  quest.RewardXP,
		"rewardCurrencyTotalCopper": quest.RewardCopper,
		"rewardCurrencyGold":        rewardCurrency.Gold,
		"rewardCurrencySilver":      rewardCurrency.Silver,
		"rewardCurrencyCopper":      rewardCurrency.Copper,
		"experience":                session.Experience,
		"level":                     session.Level,
		"currencyTotalCopper":       session.CurrencyCopper,
	})
	return nil
}

func addItemToInventory(inventory *[]platform.CharacterInventorySlot, item itemRewardDefinition) error {
	if item.ItemID == "" || item.StackCount <= 0 {
		return nil
	}

	if definition, found := findItemDefinition(item.ItemID); found {
		return addDefinedItemToInventory(inventory, definition, item.StackCount)
	}

	slots := platform.NormalizeInventorySlots(*inventory)
	for index := range slots {
		if slots[index].ItemID == item.ItemID {
			slots[index].StackCount += item.StackCount
			*inventory = slots
			return nil
		}
	}
	for index := range slots {
		if slots[index].ItemID == "" || slots[index].StackCount <= 0 {
			slots[index] = platform.CharacterInventorySlot{
				SlotIndex:   index,
				ItemID:      item.ItemID,
				DisplayName: item.DisplayName,
				StackCount:  item.StackCount,
			}
			*inventory = slots
			return nil
		}
	}
	return fmt.Errorf("inventory is full")
}

func (s *worldServer) applyQuestKillCreditLocked(killer *worldSessionState, killedMob *mobState) error {
	if killer == nil || killedMob == nil {
		return nil
	}

	changedSessions := map[string]*worldSessionState{}
	now := time.Now().Unix()
	for _, questID := range s.questOrder {
		quest := s.quests[questID]
		if quest.TargetMobType != killedMob.MobTypeID {
			continue
		}

		for _, candidate := range s.killCreditCandidatesLocked(killer, quest, killedMob) {
			progress := s.normalizeQuestProgress(quest, candidate.QuestProgress[quest.ID])
			if progress.State != questStateActive || progress.CurrentCount >= progress.TargetCount {
				continue
			}

			progress.CurrentCount++
			progress.UpdatedAt = now
			if progress.CurrentCount >= progress.TargetCount {
				progress.State = questStateCompleted
				progress.CompletedAt = now
			}
			candidate.QuestProgress[quest.ID] = progress
			changedSessions[candidate.Token] = candidate
			observability.LogEvent("world-service", "world.quest_progressed", map[string]any{
				"worldSessionToken": candidate.Token,
				"characterId":       candidate.CharacterID,
				"questId":           quest.ID,
				"currentCount":      progress.CurrentCount,
				"targetCount":       progress.TargetCount,
				"sharedPartyCredit": quest.PartyShareable && candidate.CharacterID != killer.CharacterID,
				"killedMobId":       killedMob.ID,
			})
		}
	}

	for _, session := range changedSessions {
		if err := s.persistSessionProgressionLocked(session); err != nil {
			return err
		}
	}
	return nil
}

func (s *worldServer) killCreditCandidatesLocked(killer *worldSessionState, quest questDefinition, killedMob *mobState) []*worldSessionState {
	if !quest.PartyShareable {
		return []*worldSessionState{killer}
	}

	candidates := make([]*worldSessionState, 0, partySizeLimit)
	seen := map[string]struct{}{}
	addIfEligible := func(candidate *worldSessionState) {
		if candidate == nil {
			return
		}
		if _, exists := seen[candidate.CharacterID]; exists {
			return
		}
		seen[candidate.CharacterID] = struct{}{}
		if s.partyQuestCreditEligibleLocked(killer, candidate, quest, killedMob) {
			candidates = append(candidates, candidate)
		}
	}

	if s.store == nil {
		addIfEligible(killer)
		return candidates
	}

	party, err := s.store.GetPartyForCharacter(killer.CharacterID)
	if err != nil {
		if !errors.Is(err, storepkg.ErrPartyMissing) {
			observability.LogEvent("world-service", "world.party_credit_lookup_failed", map[string]any{
				"characterId": killer.CharacterID,
				"questId":     quest.ID,
				"error":       err.Error(),
			})
		}
		addIfEligible(killer)
		return candidates
	}

	for _, memberID := range party.MemberCharacterIDs {
		addIfEligible(s.findConnectedSessionByCharacterLocked(memberID))
	}
	return candidates
}

func (s *worldServer) partyQuestCreditEligibleLocked(killer *worldSessionState, candidate *worldSessionState, quest questDefinition, killedMob *mobState) bool {
	if killer == nil || candidate == nil || killedMob == nil {
		return false
	}
	if !candidate.Connected || candidate.RealmID != killer.RealmID || candidate.ZoneID != killedMob.ZoneID {
		s.logPartyCreditSkipped(candidate, quest, killedMob, "offline_or_wrong_zone")
		return false
	}
	if distance2D(candidate.X, candidate.Y, killedMob.X, killedMob.Y) > questPartyCreditRadius(quest) {
		s.logPartyCreditSkipped(candidate, quest, killedMob, "out_of_range")
		return false
	}
	progress := s.normalizeQuestProgress(quest, candidate.QuestProgress[quest.ID])
	if progress.State != questStateActive || progress.CurrentCount >= progress.TargetCount {
		s.logPartyCreditSkipped(candidate, quest, killedMob, "quest_not_active")
		return false
	}
	return true
}

func (s *worldServer) logPartyCreditSkipped(candidate *worldSessionState, quest questDefinition, killedMob *mobState, reason string) {
	if candidate == nil || killedMob == nil || !quest.PartyShareable {
		return
	}
	observability.LogEvent("world-service", "world.party_credit_skipped", map[string]any{
		"characterId": candidate.CharacterID,
		"questId":     quest.ID,
		"mobId":       killedMob.ID,
		"reason":      reason,
	})
}

func (s *worldServer) primaryQuestLocked(session *worldSessionState) questDefinition {
	for _, questID := range s.questOrder {
		quest := s.quests[questID]
		progress := s.normalizeQuestProgress(quest, session.QuestProgress[quest.ID])
		if progress.State == questStateActive || progress.State == questStateCompleted {
			return quest
		}
	}
	for _, questID := range s.questOrder {
		quest := s.quests[questID]
		progress := s.normalizeQuestProgress(quest, session.QuestProgress[quest.ID])
		if progress.State == questStateNotStarted && s.prerequisitesMetLocked(session, quest) {
			return quest
		}
	}
	if len(s.questOrder) > 0 {
		return s.quests[s.questOrder[len(s.questOrder)-1]]
	}
	return s.quest
}

func (s *worldServer) normalizeTrackedQuestIDsLocked(source []string, progressByQuest map[string]platform.CharacterQuestProgress) []string {
	normalized := platform.NormalizeStringIDs(source)
	if len(normalized) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(normalized))
	for _, questID := range normalized {
		quest, found := s.quests[questID]
		if !found {
			continue
		}
		progress := s.normalizeQuestProgress(quest, progressByQuest[questID])
		if progress.State == questStateNotStarted || progress.State == questStateRewardGranted {
			continue
		}
		result = append(result, questID)
		if len(result) >= 3 {
			break
		}
	}
	return result
}

func (s *worldServer) questTrackedLocked(session *worldSessionState, questID string) bool {
	for _, trackedQuestID := range session.TrackedQuestIDs {
		if trackedQuestID == questID {
			return true
		}
	}
	return false
}

func (s *worldServer) trackQuestLocked(session *worldSessionState, questID string) {
	if session == nil || questID == "" || s.questTrackedLocked(session, questID) {
		return
	}
	session.TrackedQuestIDs = append(session.TrackedQuestIDs, questID)
	session.TrackedQuestIDs = s.normalizeTrackedQuestIDsLocked(session.TrackedQuestIDs, session.QuestProgress)
}

func (s *worldServer) untrackQuestLocked(session *worldSessionState, questID string) {
	if session == nil || questID == "" {
		return
	}
	next := make([]string, 0, len(session.TrackedQuestIDs))
	for _, trackedQuestID := range session.TrackedQuestIDs {
		if trackedQuestID != questID {
			next = append(next, trackedQuestID)
		}
	}
	session.TrackedQuestIDs = next
}

func (s *worldServer) prerequisitesMetLocked(session *worldSessionState, quest questDefinition) bool {
	for _, prerequisiteID := range quest.PrerequisiteIDs {
		prereq, found := s.quests[prerequisiteID]
		if !found {
			return false
		}
		progress := s.normalizeQuestProgress(prereq, session.QuestProgress[prerequisiteID])
		if progress.State != questStateRewardGranted {
			return false
		}
	}
	return true
}

func (s *worldServer) buildQuestResponse(session *worldSessionState) map[string]any {
	quest := s.primaryQuestLocked(session)
	progress := s.normalizeQuestProgress(quest, session.QuestProgress[quest.ID])
	return s.buildQuestSummary(quest, progress, s.questTrackedLocked(session, quest.ID))
}

func (s *worldServer) buildQuestListResponse(session *worldSessionState) []map[string]any {
	quests := make([]map[string]any, 0, len(s.questOrder))
	for _, questID := range s.questOrder {
		quest := s.quests[questID]
		progress := s.normalizeQuestProgress(quest, session.QuestProgress[quest.ID])
		if progress.State == questStateNotStarted && !s.prerequisitesMetLocked(session, quest) {
			continue
		}
		quests = append(quests, s.buildQuestSummary(quest, progress, s.questTrackedLocked(session, quest.ID)))
	}
	return quests
}

func (s *worldServer) buildQuestSummary(quest questDefinition, progress platform.CharacterQuestProgress, tracked bool) map[string]any {
	objectiveArea := s.objectiveAreaForQuest(quest)
	rewardItems := make([]itemRewardResponse, 0, len(quest.RewardItems))
	for _, item := range quest.RewardItems {
		rewardItems = append(rewardItems, itemRewardResponse{
			ItemID:      item.ItemID,
			DisplayName: item.DisplayName,
			StackCount:  item.StackCount,
		})
	}
	uiTags := []string{}
	if quest.GroupRecommended {
		uiTags = append(uiTags, "Group")
	}

	return map[string]any{
		"id":                   quest.ID,
		"title":                quest.Title,
		"category":             s.questCategory(quest),
		"statusBucket":         questStatusBucket(progress),
		"tracked":              tracked,
		"objectiveType":        quest.ObjectiveType,
		"objectiveText":        quest.ObjectiveText,
		"state":                progress.State,
		"currentCount":         progress.CurrentCount,
		"targetCount":          progress.TargetCount,
		"giverNpcId":           quest.GiverNPCID,
		"turnInNpcId":          quest.TurnInNPCID,
		"levelBand":            quest.LevelBand,
		"rewardXp":             quest.RewardXP,
		"rewardCurrencyCopper": quest.RewardCopper,
		"rewardCurrency":       breakdownCurrency(quest.RewardCopper),
		"rewardItems":          rewardItems,
		"objectiveArea":        objectiveArea,
		"partyShareable":       quest.PartyShareable,
		"groupRecommended":     quest.GroupRecommended,
		"recommendedPlayers":   quest.RecommendedPlayers,
		"partyCreditRadius":    questPartyCreditRadius(quest),
		"uiTags":               uiTags,
	}
}

func (s *worldServer) questCategory(quest questDefinition) string {
	if quest.ID == dungeonQuestTallowdeepID {
		return "Dungeon - Group Recommended"
	}
	if zone, ok := s.zones[quest.ZoneID]; ok && zone.DisplayName != "" {
		return zone.DisplayName
	}
	return "Stonewake Vale"
}

func questStatusBucket(progress platform.CharacterQuestProgress) string {
	switch progress.State {
	case questStateNotStarted:
		return "available"
	case questStateActive:
		return "active"
	case questStateCompleted:
		return "ready_to_turn_in"
	case questStateRewardGranted:
		return "completed"
	default:
		return "available"
	}
}

func (s *worldServer) objectiveAreaForQuest(quest questDefinition) map[string]any {
	if area, ok := s.findNavigationAreaForQuest(quest); ok {
		return map[string]any{
			"areaId":        area.ID,
			"displayName":   area.DisplayName,
			"kind":          area.Kind,
			"centerX":       area.CenterX,
			"centerY":       area.CenterY,
			"radius":        area.Radius,
			"routeHintText": area.RouteHintText,
		}
	}

	if quest.MarkerX != 0 || quest.MarkerY != 0 {
		return map[string]any{
			"areaId":        quest.ID + "_marker",
			"displayName":   quest.Title,
			"kind":          "objective",
			"centerX":       quest.MarkerX,
			"centerY":       quest.MarkerY,
			"radius":        starterInteractRadius,
			"routeHintText": "Follow the road marker toward the objective.",
		}
	}

	return map[string]any{}
}

func (s *worldServer) findNavigationAreaForQuest(quest questDefinition) (navigationAreaDefinition, bool) {
	source := stonewakeNavigationAreas
	if quest.ID == dungeonQuestTallowdeepID {
		source = tallowdeepNavigationAreas
	}
	for _, area := range source {
		for _, questID := range area.QuestIDs {
			if questID == quest.ID {
				return area, true
			}
		}
	}
	return navigationAreaDefinition{}, false
}

func questPartyCreditRadius(quest questDefinition) float64 {
	if quest.PartyCreditRadius > 0 {
		return quest.PartyCreditRadius
	}
	if quest.PartyShareable {
		return defaultPartyQuestCreditRadius
	}
	return 0
}

func breakdownCurrency(totalCopper int) currencyBreakdown {
	if totalCopper < 0 {
		totalCopper = 0
	}

	return currencyBreakdown{
		Gold:   totalCopper / 10000,
		Silver: (totalCopper % 10000) / 100,
		Copper: totalCopper % 100,
	}
}
