package worlds

import (
	"fmt"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

func (s *worldServer) gatherNodeLocked(session *worldSessionState, nodeID string) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}

	node, found := s.findGatheringNodeLocked(nodeID)
	if !found {
		return fmt.Errorf("gathering node is not available")
	}

	definition := node.Definition
	if definition.ZoneID != "" && definition.ZoneID != session.ZoneID {
		return fmt.Errorf("gathering node is not in this zone")
	}
	if !gatheringNodeReadyAt(node, nowMillis()) {
		return fmt.Errorf("gathering node is not ready")
	}
	if !s.sessionMeetsGatheringRequirementLocked(session, definition) {
		return fmt.Errorf("required profession or skill is missing")
	}

	radius := definition.Radius
	if radius <= 0 {
		radius = starterInteractRadius
	}
	if distance2D(session.X, session.Y, definition.X, definition.Y) > radius {
		return fmt.Errorf("move closer to the gathering node")
	}

	inventory := platform.NormalizeInventorySlots(session.Inventory)
	grantedItems := make([]map[string]any, 0, len(definition.Loot))
	for _, loot := range definition.Loot {
		if !loot.Guaranteed {
			continue
		}

		item, found := findItemDefinition(loot.ItemID)
		if !found {
			return fmt.Errorf("gathered item is not defined")
		}

		quantity := loot.MinCount
		if quantity <= 0 {
			quantity = 1
		}
		if loot.MaxCount > 0 && loot.MaxCount < quantity {
			quantity = loot.MaxCount
		}
		if err := addDefinedItemToInventory(&inventory, item, quantity); err != nil {
			return err
		}
		grantedItems = append(grantedItems, map[string]any{
			"itemId":      item.ItemID,
			"displayName": item.DisplayName,
			"quantity":    quantity,
		})
	}
	if len(grantedItems) == 0 {
		return fmt.Errorf("gathering node has no available materials")
	}

	session.Inventory = inventory
	if err := s.persistSessionEconomyLocked(session); err != nil {
		return err
	}

	nowMs := nowMillis()
	respawnDelayMs := definition.RespawnDelayMs
	if respawnDelayMs <= 0 {
		respawnDelayMs = 1000
	}
	node.ReadyAtMs = nowMs + respawnDelayMs

	observability.LogEvent("world-service", "world.gathering_node_gathered", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"nodeId":            definition.ID,
		"nodeTypeId":        definition.NodeTypeID,
		"professionId":      definition.RequiredProfessionID,
		"grantedItems":      grantedItems,
		"readyAt":           node.ReadyAtMs,
		"gatheredAt":        time.Now().Unix(),
	})
	return nil
}

func (s *worldServer) findGatheringNodeLocked(nodeID string) (*gatheringNodeState, bool) {
	s.ensureGatheringNodesLocked()
	node, ok := s.gatheringNodes[nodeID]
	return node, ok
}

func gatheringNodeReadyAt(node *gatheringNodeState, nowMs int64) bool {
	return node != nil && (node.ReadyAtMs == 0 || nowMs >= node.ReadyAtMs)
}

func (s *worldServer) sessionMeetsGatheringRequirementLocked(
	session *worldSessionState,
	node gatheringNodeDefinition,
) bool {
	if node.RequiredProfessionID == "" {
		return true
	}
	for _, profession := range platform.NormalizeProfessionStates(session.Professions) {
		if profession.ProfessionID != node.RequiredProfessionID {
			continue
		}
		requiredSkill := node.RequiredSkill
		if requiredSkill <= 0 {
			requiredSkill = 1
		}
		return profession.SkillValue >= requiredSkill
	}
	return false
}

func buildGatheringNodeEntity(node *gatheringNodeState, nowMs int64) sessionEntity {
	definition := node.Definition
	ready := gatheringNodeReadyAt(node, nowMs)
	return sessionEntity{
		ID:               definition.ID,
		DisplayName:      definition.DisplayName,
		Kind:             gatheringNodeKind,
		GatherNodeTypeID: definition.NodeTypeID,
		ProfessionID:     definition.RequiredProfessionID,
		RequiredSkill:    definition.RequiredSkill,
		Ready:            ready,
		ReadyAt:          node.ReadyAtMs,
		InteractionLabel: definition.InteractionLabel,
		X:                definition.X,
		Y:                definition.Y,
		Z:                definition.Z,
		Health:           1,
		MaxHealth:        1,
		Alive:            true,
		Targetable:       ready,
	}
}
