package worlds

import (
	"fmt"
	"math/rand"
	"time"

	"amandacore/services/internal/observability"
)

const (
	devIsleStalkerLootTableID = "dev_isle_stalker_loot"
	lootContainerKind         = "loot_container"
	lootInteractionRange      = 4.0
	defaultLootExpiry         = 2 * time.Minute
)

type LootTableID string
type LootContainerID string

type LootTableDefinition struct {
	LootTableID       string
	SourceArchetypeID string
	Entries           []LootEntry
}

type LootEntry struct {
	ItemID            string
	MinQuantity       int
	MaxQuantity       int
	DropChancePercent float64
	DropWeight        int
	IsGuaranteed      bool
	Tags              []string
}

type LootRollContext struct {
	SourceEntityID    string
	SourceArchetypeID string
	ZoneID            string
	KillerCharacterID string
}

type LootRollResult struct {
	Items []InventoryGrant
}

type lootRollSource interface {
	Float64() float64
	Intn(n int) int
}

type seededLootRollSource struct {
	rng *rand.Rand
}

func newSeededLootRollSource(seed int64) *seededLootRollSource {
	return &seededLootRollSource{rng: rand.New(rand.NewSource(seed))}
}

func (s *seededLootRollSource) Float64() float64 {
	return s.rng.Float64()
}

func (s *seededLootRollSource) Intn(n int) int {
	return s.rng.Intn(n)
}

type LootContainer struct {
	LootContainerID  LootContainerID
	OwnerCharacterID string
	Items            []InventoryGrant
	ExpiresAtMs      int64
}

type LootOwnership struct {
	OwnerCharacterID string
}

type lootContainerItemState struct {
	ItemID      string `json:"itemId"`
	DisplayName string `json:"displayName"`
	Quantity    int    `json:"quantity"`
}

type lootContainerState struct {
	LootContainerID      string
	SourceEntityID       string
	SourceArchetypeID    string
	LootTableID          string
	ZoneID               string
	InstanceID           string
	X                    float64
	Y                    float64
	Z                    float64
	OwnerCharacterID     string
	Items                []lootContainerItemState
	CreatedAtMs          int64
	ExpiresAtMs          int64
	ClaimedAtMs          int64
	ClaimedByCharacterID string
	ExpiredLogged        bool
}

type lootContainerResponse struct {
	LootContainerID   string                   `json:"lootContainerId"`
	SourceEntityID    string                   `json:"sourceEntityId"`
	SourceArchetypeID string                   `json:"sourceArchetypeId"`
	LootTableID       string                   `json:"lootTableId"`
	ZoneID            string                   `json:"zoneId"`
	X                 float64                  `json:"x"`
	Y                 float64                  `json:"y"`
	Z                 float64                  `json:"z"`
	OwnerCharacterID  string                   `json:"ownerCharacterId"`
	Items             []lootContainerItemState `json:"items"`
	ExpiresAtMs       int64                    `json:"expiresAtMs"`
	Claimed           bool                     `json:"claimed"`
	Expired           bool                     `json:"expired"`
}

var devLootTables = map[string]LootTableDefinition{
	devIsleStalkerLootTableID: {
		LootTableID:       devIsleStalkerLootTableID,
		SourceArchetypeID: devIsleStalkerArchetypeID,
		Entries: []LootEntry{
			{ItemID: itemDevStalkerFangID, MinQuantity: 1, MaxQuantity: 1, IsGuaranteed: true, Tags: []string{"quest"}},
			{ItemID: itemDevGlimmerShardID, MinQuantity: 1, MaxQuantity: 2, DropChancePercent: 50, Tags: []string{"material"}},
			{ItemID: itemDevFieldRationID, MinQuantity: 1, MaxQuantity: 1, DropChancePercent: 20, Tags: []string{"consumable"}},
			{ItemID: itemDevCopperTokenID, MinQuantity: 1, MaxQuantity: 3, IsGuaranteed: true, Tags: []string{"currency"}},
		},
	},
}

func findLootTableDefinition(lootTableID string) (LootTableDefinition, bool) {
	table, ok := devLootTables[lootTableID]
	return table, ok
}

func generateLoot(table LootTableDefinition, context LootRollContext, rolls lootRollSource) (LootRollResult, error) {
	if table.LootTableID == "" {
		return LootRollResult{}, fmt.Errorf("loot table is missing")
	}
	if rolls == nil {
		rolls = newSeededLootRollSource(time.Now().UnixNano())
	}

	result := LootRollResult{Items: []InventoryGrant{}}
	for _, entry := range table.Entries {
		if entry.ItemID == "" {
			continue
		}
		if _, found := findItemDefinition(entry.ItemID); !found {
			return LootRollResult{}, fmt.Errorf("loot item %s is not defined", entry.ItemID)
		}
		included := entry.IsGuaranteed
		if !included {
			chance := entry.DropChancePercent
			if chance < 0 {
				chance = 0
			}
			if chance > 100 {
				chance = 100
			}
			included = rolls.Float64()*100 < chance
		}
		if !included {
			continue
		}
		minQuantity := entry.MinQuantity
		if minQuantity <= 0 {
			minQuantity = 1
		}
		maxQuantity := entry.MaxQuantity
		if maxQuantity < minQuantity {
			maxQuantity = minQuantity
		}
		quantity := minQuantity
		if maxQuantity > minQuantity {
			quantity += rolls.Intn(maxQuantity - minQuantity + 1)
		}
		result.Items = append(result.Items, InventoryGrant{
			ItemID:   entry.ItemID,
			Quantity: quantity,
			Reason:   "loot",
		})
	}
	_ = context
	return result, nil
}

func (s *worldServer) createLootContainerForMobDeathLocked(killer *worldSessionState, mob *mobState) (*lootContainerState, error) {
	if killer == nil || mob == nil {
		return nil, nil
	}
	lootTableID := mob.LootTableID
	if lootTableID == "" {
		if mob.ArchetypeID == devIsleStalkerArchetypeID || mob.MobTypeID == devIsleStalkerArchetypeID {
			lootTableID = devIsleStalkerLootTableID
		}
	}
	if lootTableID == "" {
		return nil, nil
	}
	table, found := findLootTableDefinition(lootTableID)
	if !found {
		return nil, fmt.Errorf("loot table %s is not defined", lootTableID)
	}

	archetypeID := mob.ArchetypeID
	if archetypeID == "" {
		archetypeID = mob.MobTypeID
	}
	context := LootRollContext{
		SourceEntityID:    mob.ID,
		SourceArchetypeID: archetypeID,
		ZoneID:            mob.ZoneID,
		KillerCharacterID: killer.CharacterID,
	}
	s.emitWorldEventLocked(eventLootRollStarted, map[string]any{
		"lootTableId":       table.LootTableID,
		"sourceEntityId":    context.SourceEntityID,
		"sourceArchetypeId": context.SourceArchetypeID,
		"characterId":       context.KillerCharacterID,
	})
	seed := int64(len(s.lootContainerOrder)+1)*7919 + nowMillis()
	result, err := generateLoot(table, context, newSeededLootRollSource(seed))
	if err != nil {
		return nil, err
	}
	s.emitWorldEventLocked(eventLootRollCompleted, map[string]any{
		"lootTableId":    table.LootTableID,
		"sourceEntityId": context.SourceEntityID,
		"itemCount":      len(result.Items),
	})
	if len(result.Items) == 0 {
		return nil, nil
	}

	if s.lootContainers == nil {
		s.lootContainers = map[string]*lootContainerState{}
	}
	s.lootContainerCounter++
	containerID := fmt.Sprintf("loot_%06d", s.lootContainerCounter)
	nowMs := nowMillis()
	container := &lootContainerState{
		LootContainerID:   containerID,
		SourceEntityID:    mob.ID,
		SourceArchetypeID: archetypeID,
		LootTableID:       table.LootTableID,
		ZoneID:            mob.ZoneID,
		InstanceID:        mob.InstanceID,
		X:                 mob.X,
		Y:                 mob.Y,
		Z:                 mob.Z,
		OwnerCharacterID:  killer.CharacterID,
		CreatedAtMs:       nowMs,
		ExpiresAtMs:       nowMs + defaultLootExpiry.Milliseconds(),
	}
	for _, grant := range result.Items {
		item, _ := findItemDefinition(grant.ItemID)
		container.Items = append(container.Items, lootContainerItemState{
			ItemID:      grant.ItemID,
			DisplayName: item.DisplayName,
			Quantity:    grant.Quantity,
		})
	}
	s.lootContainers[containerID] = container
	s.lootContainerOrder = append(s.lootContainerOrder, containerID)
	observability.LogEvent("world-service", observability.EventLootGenerated, map[string]any{
		"lootContainerId":   container.LootContainerID,
		"sourceEntityId":    container.SourceEntityID,
		"sourceArchetypeId": container.SourceArchetypeID,
		"ownerCharacterId":  container.OwnerCharacterID,
		"itemCount":         len(container.Items),
	})
	s.emitWorldEventLocked(eventLootContainerCreated, map[string]any{
		"lootContainerId":   container.LootContainerID,
		"sourceEntityId":    container.SourceEntityID,
		"sourceArchetypeId": container.SourceArchetypeID,
		"lootTableId":       container.LootTableID,
		"ownerCharacterId":  container.OwnerCharacterID,
		"expiresAtMs":       container.ExpiresAtMs,
	}, lootContainerCreatedDelta(container))
	return container, nil
}

func (s *worldServer) inspectLootLocked(session *worldSessionState, containerID string) (*lootContainerState, error) {
	s.emitWorldEventLocked(eventLootInspectRequested, map[string]any{
		"characterId":     characterIDOrEmpty(session),
		"lootContainerId": containerID,
	})
	container, err := s.validateLootInteractionLocked(session, containerID)
	if err != nil {
		return nil, err
	}
	s.emitWorldEventLocked(eventLootInspectResolved, map[string]any{
		"characterId":     session.CharacterID,
		"lootContainerId": containerID,
		"itemCount":       len(container.Items),
	})
	return container, nil
}

func (s *worldServer) claimLootLocked(session *worldSessionState, containerID string) error {
	s.emitWorldEventLocked(eventLootClaimRequested, map[string]any{
		"characterId":     characterIDOrEmpty(session),
		"lootContainerId": containerID,
	})
	container, err := s.validateLootInteractionLocked(session, containerID)
	if err != nil {
		s.emitWorldEventLocked(eventLootClaimRejected, map[string]any{
			"characterId":     characterIDOrEmpty(session),
			"lootContainerId": containerID,
			"reason":          lootRejectReason(err),
		})
		observability.LogEvent("world-service", observability.EventLootClaimRejected, map[string]any{
			"characterId":     characterIDOrEmpty(session),
			"lootContainerId": containerID,
			"reason":          lootRejectReason(err),
		})
		return err
	}

	grants := make([]InventoryGrant, 0, len(container.Items))
	for _, item := range container.Items {
		grants = append(grants, InventoryGrant{
			ItemID:   item.ItemID,
			Quantity: item.Quantity,
			Reason:   "loot",
		})
	}
	if err := s.grantInventoryItemsLocked(session, grants, "loot"); err != nil {
		s.emitWorldEventLocked(eventLootClaimRejected, map[string]any{
			"characterId":     session.CharacterID,
			"lootContainerId": containerID,
			"reason":          "InventoryFull",
			"error":           err.Error(),
		}, lootClaimResultDelta(container, false, "InventoryFull"))
		observability.LogEvent("world-service", observability.EventLootClaimRejected, map[string]any{
			"characterId":     session.CharacterID,
			"lootContainerId": containerID,
			"reason":          "InventoryFull",
			"error":           err.Error(),
		})
		return fmt.Errorf("InventoryFull: %w", err)
	}

	nowMs := nowMillis()
	container.ClaimedAtMs = nowMs
	container.ClaimedByCharacterID = session.CharacterID
	s.emitWorldEventLocked(eventLootClaimCompleted, map[string]any{
		"characterId":     session.CharacterID,
		"lootContainerId": containerID,
		"itemCount":       len(container.Items),
	}, lootContainerUpdatedDelta(container), lootClaimResultDelta(container, true, ""))
	observability.LogEvent("world-service", observability.EventLootClaimed, map[string]any{
		"characterId":     session.CharacterID,
		"lootContainerId": containerID,
		"itemCount":       len(container.Items),
	})
	return nil
}

func (s *worldServer) validateLootInteractionLocked(session *worldSessionState, containerID string) (*lootContainerState, error) {
	if session == nil || !session.Connected {
		return nil, fmt.Errorf("SessionInvalid")
	}
	if !session.Alive {
		return nil, fmt.Errorf("CharacterDead")
	}
	container := s.lootContainers[containerID]
	if container == nil {
		return nil, fmt.Errorf("LootMissing")
	}
	if container.ClaimedAtMs != 0 {
		return nil, fmt.Errorf("LootAlreadyClaimed")
	}
	if nowMillis() >= container.ExpiresAtMs {
		return nil, fmt.Errorf("LootExpired")
	}
	if container.OwnerCharacterID != "" && container.OwnerCharacterID != session.CharacterID {
		return nil, fmt.Errorf("NotLootOwner")
	}
	if container.ZoneID != session.ZoneID || container.InstanceID != session.InstanceID {
		return nil, fmt.Errorf("InvalidState")
	}
	if distance2D(session.X, session.Y, container.X, container.Y) > lootInteractionRange {
		return nil, fmt.Errorf("OutOfRange")
	}
	return container, nil
}

func (s *worldServer) cleanupExpiredLootContainersLocked(now time.Time) {
	nowMs := now.UnixMilli()
	for _, containerID := range s.lootContainerOrder {
		container := s.lootContainers[containerID]
		if container == nil || container.ClaimedAtMs != 0 || container.ExpiredLogged || nowMs < container.ExpiresAtMs {
			continue
		}
		container.ExpiredLogged = true
		s.emitWorldEventLocked(eventLootContainerExpired, map[string]any{
			"lootContainerId":  container.LootContainerID,
			"ownerCharacterId": container.OwnerCharacterID,
		}, lootContainerUpdatedDelta(container))
	}
}

func (s *worldServer) buildLootContainersResponseLocked(session *worldSessionState) []lootContainerResponse {
	if session == nil || len(s.lootContainerOrder) == 0 {
		return []lootContainerResponse{}
	}
	nowMs := nowMillis()
	responses := []lootContainerResponse{}
	for _, containerID := range s.lootContainerOrder {
		container := s.lootContainers[containerID]
		if container == nil || container.ZoneID != session.ZoneID || container.InstanceID != session.InstanceID {
			continue
		}
		if container.OwnerCharacterID != "" && container.OwnerCharacterID != session.CharacterID {
			continue
		}
		responses = append(responses, lootContainerResponse{
			LootContainerID:   container.LootContainerID,
			SourceEntityID:    container.SourceEntityID,
			SourceArchetypeID: container.SourceArchetypeID,
			LootTableID:       container.LootTableID,
			ZoneID:            container.ZoneID,
			X:                 container.X,
			Y:                 container.Y,
			Z:                 container.Z,
			OwnerCharacterID:  container.OwnerCharacterID,
			Items:             append([]lootContainerItemState(nil), container.Items...),
			ExpiresAtMs:       container.ExpiresAtMs,
			Claimed:           container.ClaimedAtMs != 0,
			Expired:           nowMs >= container.ExpiresAtMs,
		})
	}
	return responses
}

func lootContainerCreatedDelta(container *lootContainerState) stateDiff {
	if container == nil {
		return stateDiff{}
	}
	return stateDiff{
		Type:     diffLootContainerCreated,
		EntityID: container.LootContainerID,
		Fields: map[string]any{
			"sourceEntityId":    container.SourceEntityID,
			"sourceArchetypeId": container.SourceArchetypeID,
			"lootTableId":       container.LootTableID,
			"ownerCharacterId":  container.OwnerCharacterID,
			"expiresAtMs":       container.ExpiresAtMs,
		},
	}
}

func lootContainerUpdatedDelta(container *lootContainerState) stateDiff {
	if container == nil {
		return stateDiff{}
	}
	return stateDiff{
		Type:     diffLootContainerUpdated,
		EntityID: container.LootContainerID,
		Fields: map[string]any{
			"claimed":     container.ClaimedAtMs != 0,
			"claimedBy":   container.ClaimedByCharacterID,
			"expiresAtMs": container.ExpiresAtMs,
			"expired":     nowMillis() >= container.ExpiresAtMs,
		},
	}
}

func lootClaimResultDelta(container *lootContainerState, accepted bool, reason string) stateDiff {
	if container == nil {
		return stateDiff{}
	}
	return stateDiff{
		Type:     diffLootClaimResult,
		EntityID: container.LootContainerID,
		Fields: map[string]any{
			"accepted": accepted,
			"reason":   reason,
		},
	}
}

func lootRejectReason(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func characterIDOrEmpty(session *worldSessionState) string {
	if session == nil {
		return ""
	}
	return session.CharacterID
}
