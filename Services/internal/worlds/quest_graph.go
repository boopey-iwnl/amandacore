package worlds

import (
	"fmt"
	"time"

	"amandacore/services/internal/platform"
)

type objectiveEvent struct {
	Kind              string
	NpcArchetypeID    string
	SourceEntityID    string
	ItemID            string
	Quantity          int
	CharacterID       string
	OccurredAtSeconds int64
}

func questHasObjectiveGraph(quest questDefinition) bool {
	return len(quest.ObjectiveGraph.Nodes) > 0
}

func (s *worldServer) defaultGraphQuestProgress(quest questDefinition) platform.CharacterQuestProgress {
	progress := platform.CharacterQuestProgress{
		QuestID:     quest.ID,
		State:       questStateNotStarted,
		TargetCount: len(terminalQuestObjectiveNodes(quest)),
	}
	return s.normalizeGraphQuestProgress(quest, progress)
}

func (s *worldServer) normalizeGraphQuestProgress(quest questDefinition, progress platform.CharacterQuestProgress) platform.CharacterQuestProgress {
	if progress.QuestID == "" {
		progress.QuestID = quest.ID
	}
	if progress.State == "" {
		progress.State = questStateNotStarted
	}
	if progress.ObjectiveProgress == nil {
		progress.ObjectiveProgress = map[string]platform.CharacterQuestObjectiveProgress{}
	}
	completed := 0
	for _, node := range quest.ObjectiveGraph.Nodes {
		if node.NodeID == "" {
			continue
		}
		target := node.TargetCount
		if target <= 0 {
			target = 1
		}
		nodeProgress := progress.ObjectiveProgress[node.NodeID]
		nodeProgress.NodeID = node.NodeID
		nodeProgress.Target = target
		if nodeProgress.Current < 0 {
			nodeProgress.Current = 0
		}
		if nodeProgress.Current > target {
			nodeProgress.Current = target
		}
		if nodeProgress.Current >= target || nodeProgress.Completed {
			nodeProgress.Completed = true
			nodeProgress.Current = target
			if nodeProgress.CompletedAt == 0 {
				nodeProgress.CompletedAt = progress.CompletedAt
			}
			completed++
		}
		progress.ObjectiveProgress[node.NodeID] = nodeProgress
	}
	progress.CurrentCount = completed
	progress.TargetCount = len(quest.ObjectiveGraph.Nodes)
	if progress.RewardGrantedAt != 0 {
		progress.State = questStateRewardGranted
		return progress
	}
	if s.questGraphReadyToComplete(quest, progress) && progress.State != questStateNotStarted {
		progress.State = questStateReady
	}
	return progress
}

func terminalQuestObjectiveNodes(quest questDefinition) []questObjectiveNode {
	terminal := []questObjectiveNode{}
	for _, node := range quest.ObjectiveGraph.Nodes {
		if node.Terminal {
			terminal = append(terminal, node)
		}
	}
	if len(terminal) == 0 {
		return quest.ObjectiveGraph.Nodes
	}
	return terminal
}

func (s *worldServer) questGraphReadyToComplete(quest questDefinition, progress platform.CharacterQuestProgress) bool {
	if !questHasObjectiveGraph(quest) {
		return false
	}
	progress = s.normalizeGraphQuestProgressNoReady(quest, progress)
	for _, node := range terminalQuestObjectiveNodes(quest) {
		nodeProgress := progress.ObjectiveProgress[node.NodeID]
		if !nodeProgress.Completed {
			return false
		}
	}
	return true
}

func (s *worldServer) normalizeGraphQuestProgressNoReady(quest questDefinition, progress platform.CharacterQuestProgress) platform.CharacterQuestProgress {
	if progress.QuestID == "" {
		progress.QuestID = quest.ID
	}
	if progress.ObjectiveProgress == nil {
		progress.ObjectiveProgress = map[string]platform.CharacterQuestObjectiveProgress{}
	}
	for _, node := range quest.ObjectiveGraph.Nodes {
		target := node.TargetCount
		if target <= 0 {
			target = 1
		}
		nodeProgress := progress.ObjectiveProgress[node.NodeID]
		nodeProgress.NodeID = node.NodeID
		nodeProgress.Target = target
		if nodeProgress.Current >= target || nodeProgress.Completed {
			nodeProgress.Current = target
			nodeProgress.Completed = true
		}
		progress.ObjectiveProgress[node.NodeID] = nodeProgress
	}
	return progress
}

func (s *worldServer) applyQuestKillCreditEventLocked(session *worldSessionState, credit KillCredit) error {
	if session == nil || credit.CharacterID == "" {
		return nil
	}
	return s.applyQuestObjectiveEventLocked(session, objectiveEvent{
		Kind:              objectiveKindKillNPC,
		NpcArchetypeID:    credit.NpcArchetypeID,
		SourceEntityID:    credit.SourceEntityID,
		CharacterID:       credit.CharacterID,
		OccurredAtSeconds: credit.TickMs / 1000,
	})
}

func (s *worldServer) applyQuestItemGrantedLocked(session *worldSessionState, itemID string, quantity int) error {
	if session == nil || itemID == "" || quantity <= 0 {
		return nil
	}
	return s.applyQuestObjectiveEventLocked(session, objectiveEvent{
		Kind:              objectiveKindCollectItem,
		ItemID:            itemID,
		Quantity:          quantity,
		CharacterID:       session.CharacterID,
		OccurredAtSeconds: time.Now().Unix(),
	})
}

func (s *worldServer) applyQuestObjectiveEventLocked(session *worldSessionState, event objectiveEvent) error {
	if session == nil {
		return nil
	}
	changed := false
	for _, questID := range s.questOrder {
		quest := s.quests[questID]
		if !questHasObjectiveGraph(quest) {
			continue
		}
		progress := s.normalizeGraphQuestProgress(quest, session.QuestProgress[quest.ID])
		if progress.State != questStateActive && progress.State != questStateReady {
			continue
		}
		if progress.State == questStateReady {
			continue
		}

		questChanged := false
		for _, node := range quest.ObjectiveGraph.Nodes {
			if !s.questObjectiveNodeActive(quest, progress, node) || !objectiveEventMatchesNode(event, node) {
				continue
			}
			nodeProgress := progress.ObjectiveProgress[node.NodeID]
			if nodeProgress.Completed {
				continue
			}
			increment := event.Quantity
			if increment <= 0 {
				increment = 1
			}
			nodeProgress.Current += increment
			if nodeProgress.Current >= nodeProgress.Target {
				nodeProgress.Current = nodeProgress.Target
				nodeProgress.Completed = true
				nodeProgress.CompletedAt = event.OccurredAtSeconds
				s.emitWorldEventLocked(eventQuestObjectiveCompleted, map[string]any{
					"characterId": session.CharacterID,
					"questId":     quest.ID,
					"nodeId":      node.NodeID,
					"kind":        node.Kind,
				}, questDelta(session.CharacterID, quest.ID, diffQuestObjectiveCompleted, map[string]any{
					"nodeId": node.NodeID,
					"kind":   node.Kind,
				}))
			}
			nodeProgress.UpdatedAt = event.OccurredAtSeconds
			progress.ObjectiveProgress[node.NodeID] = nodeProgress
			progress.UpdatedAt = event.OccurredAtSeconds
			questChanged = true
			s.emitWorldEventLocked(eventQuestProgressUpdated, map[string]any{
				"characterId":  session.CharacterID,
				"questId":      quest.ID,
				"nodeId":       node.NodeID,
				"currentCount": nodeProgress.Current,
				"targetCount":  nodeProgress.Target,
			}, questDelta(session.CharacterID, quest.ID, diffQuestProgress, map[string]any{
				"nodeId":       node.NodeID,
				"currentCount": nodeProgress.Current,
				"targetCount":  nodeProgress.Target,
			}))
		}

		if !questChanged {
			continue
		}
		progress = s.normalizeGraphQuestProgress(quest, progress)
		if s.questGraphReadyToComplete(quest, progress) {
			progress.State = questStateReady
			s.emitWorldEventLocked(eventQuestReadyToComplete, map[string]any{
				"characterId": session.CharacterID,
				"questId":     quest.ID,
			}, questDelta(session.CharacterID, quest.ID, diffQuestReady, map[string]any{
				"state": questStateReady,
			}))
		}
		session.QuestProgress[quest.ID] = progress
		changed = true
	}
	if !changed {
		return nil
	}
	if err := s.persistSessionProgressionLocked(session); err != nil {
		return err
	}
	for questID := range session.QuestProgress {
		if questHasObjectiveGraph(s.quests[questID]) {
			s.emitWorldEventLocked(eventQuestPersisted, map[string]any{
				"characterId": session.CharacterID,
				"questId":     questID,
			})
		}
	}
	return nil
}

func (s *worldServer) questObjectiveNodeActive(quest questDefinition, progress platform.CharacterQuestProgress, node questObjectiveNode) bool {
	nodeProgress := progress.ObjectiveProgress[node.NodeID]
	if nodeProgress.Completed {
		return false
	}
	for _, dependencyID := range node.DependsOn {
		dependencyProgress := progress.ObjectiveProgress[dependencyID]
		if !dependencyProgress.Completed {
			return false
		}
	}
	return true
}

func objectiveEventMatchesNode(event objectiveEvent, node questObjectiveNode) bool {
	if event.Kind != node.Kind {
		return false
	}
	switch node.Kind {
	case objectiveKindKillNPC:
		return node.TargetNpcArchetype == "" || node.TargetNpcArchetype == event.NpcArchetypeID
	case objectiveKindCollectItem:
		return node.TargetItemID == "" || node.TargetItemID == event.ItemID
	case objectiveKindInteractWithEntity:
		return node.TargetEntityID == "" || node.TargetEntityID == event.SourceEntityID
	default:
		return false
	}
}

func (s *worldServer) buildQuestObjectiveGraphResponse(quest questDefinition, progress platform.CharacterQuestProgress) []map[string]any {
	if !questHasObjectiveGraph(quest) {
		return []map[string]any{}
	}
	progress = s.normalizeGraphQuestProgress(quest, progress)
	response := make([]map[string]any, 0, len(quest.ObjectiveGraph.Nodes))
	for _, node := range quest.ObjectiveGraph.Nodes {
		nodeProgress := progress.ObjectiveProgress[node.NodeID]
		response = append(response, map[string]any{
			"nodeId":             node.NodeID,
			"kind":               node.Kind,
			"targetNpcArchetype": node.TargetNpcArchetype,
			"targetEntityId":     node.TargetEntityID,
			"targetItemId":       node.TargetItemID,
			"targetCount":        nodeProgress.Target,
			"currentCount":       nodeProgress.Current,
			"completed":          nodeProgress.Completed,
			"active":             s.questObjectiveNodeActive(quest, progress, node),
			"dependsOn":          append([]string(nil), node.DependsOn...),
			"terminal":           node.Terminal,
		})
	}
	return response
}

func (s *worldServer) recordKillCreditLocked(session *worldSessionState, mob *mobState, reason KillCreditReason) KillCredit {
	if session == nil || mob == nil {
		s.emitWorldEventLocked(eventProgressionCreditRejected, map[string]any{"reason": "InvalidState"})
		return KillCredit{}
	}
	if s.killCreditLedger.CreditsByCharacter == nil {
		s.killCreditLedger.CreditsByCharacter = map[string]map[string]int{}
	}
	if s.killCreditLedger.CreditsByCharacter[session.CharacterID] == nil {
		s.killCreditLedger.CreditsByCharacter[session.CharacterID] = map[string]int{}
	}
	archetypeID := mob.ArchetypeID
	if archetypeID == "" {
		archetypeID = mob.MobTypeID
	}
	s.killCreditLedger.CreditsByCharacter[session.CharacterID][archetypeID]++
	count := s.killCreditLedger.CreditsByCharacter[session.CharacterID][archetypeID]
	credit := KillCredit{
		CharacterID:    session.CharacterID,
		SourceEntityID: mob.ID,
		NpcArchetypeID: archetypeID,
		ZoneID:         mob.ZoneID,
		InstanceID:     mob.InstanceID,
		TickMs:         nowMillis(),
		Reason:         reason,
	}
	s.killCreditLedger.Entries = append(s.killCreditLedger.Entries, credit)
	s.emitWorldEventLocked(eventProgressionCreditAwarded, map[string]any{
		"characterId":    credit.CharacterID,
		"sourceEntityId": credit.SourceEntityID,
		"npcArchetypeId": credit.NpcArchetypeID,
		"zoneId":         credit.ZoneID,
		"reason":         string(credit.Reason),
		"killCount":      count,
	}, progressionDelta(session.CharacterID, archetypeID, count))
	s.emitWorldEventLocked(eventProgressionCreditSaved, map[string]any{
		"characterId":    credit.CharacterID,
		"npcArchetypeId": credit.NpcArchetypeID,
		"storage":        "in_memory",
	})
	return credit
}

func (s *worldServer) killCreditCountLocked(characterID string, archetypeID string) int {
	if s.killCreditLedger.CreditsByCharacter == nil || s.killCreditLedger.CreditsByCharacter[characterID] == nil {
		return 0
	}
	return s.killCreditLedger.CreditsByCharacter[characterID][archetypeID]
}

func graphObjectiveProgress(progress platform.CharacterQuestProgress, nodeID string) (platform.CharacterQuestObjectiveProgress, error) {
	node, ok := progress.ObjectiveProgress[nodeID]
	if !ok {
		return platform.CharacterQuestObjectiveProgress{}, fmt.Errorf("objective %s missing", nodeID)
	}
	return node, nil
}
