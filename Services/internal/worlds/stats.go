package worlds

import "amandacore/services/internal/platform"

type playerStats struct {
	Strength          int
	Stamina           int
	Armor             int
	AttackPower       float64
	ArmorReductionPct float64
	MaxHealth         float64
}

func calculatePlayerStats(level int, equipment []platform.CharacterEquipmentSlot, talents map[string]int) playerStats {
	if level <= 0 {
		level = 1
	}

	stats := playerStats{
		Strength: level,
		Stamina:  level,
		Armor:    0,
	}

	for _, slot := range platform.NormalizeEquipmentSlots(equipment) {
		if slot.ItemID == "" {
			continue
		}
		item, found := findItemDefinition(slot.ItemID)
		if !found {
			continue
		}
		stats.Strength += item.Strength
		stats.Stamina += item.Stamina
		stats.Armor += item.Armor
	}

	if rank := talents[stoutFrameTalentID]; rank > 0 {
		stats.Stamina += rank
	}
	if rank := talents[balancedGripTalentID]; rank > 0 {
		stats.Strength += rank
	}

	stats.AttackPower = float64(level*2) + float64(stats.Strength)*1.5 + weaponPower(equipment)
	stats.MaxHealth = 80.0 + float64(level*8) + float64(stats.Stamina*5)
	stats.ArmorReductionPct = armorReduction(stats.Armor, level)
	return stats
}

func weaponPower(equipment []platform.CharacterEquipmentSlot) float64 {
	for _, slot := range platform.NormalizeEquipmentSlots(equipment) {
		if slot.Slot != platform.EquipmentSlotMainHand || slot.ItemID == "" {
			continue
		}
		item, found := findItemDefinition(slot.ItemID)
		if !found || item.Type != itemTypeWeapon {
			continue
		}
		return 3.0 + float64(item.Strength)
	}
	return 1.0
}

func armorReduction(armor int, attackerLevel int) float64 {
	if armor <= 0 {
		return 0
	}
	if attackerLevel <= 0 {
		attackerLevel = 1
	}
	return clamp(float64(armor)/(float64(armor)+200.0+float64(attackerLevel*30)), 0.0, 0.35)
}

func (s *worldServer) applyDerivedStatsLocked(session *worldSessionState) {
	if session == nil {
		return
	}
	stats := calculatePlayerStats(session.Level, session.Equipment, session.Talents)
	previousMaxHealth := session.MaxHealth
	session.MaxHealth = stats.MaxHealth
	session.MaxResource = playerMaxResource
	if session.Health <= 0 {
		session.Health = session.MaxHealth
	} else if previousMaxHealth > 0 && session.Health >= previousMaxHealth {
		session.Health = session.MaxHealth
	} else if session.Health > session.MaxHealth {
		session.Health = session.MaxHealth
	}
	if session.Resource < 0 {
		session.Resource = 0
	}
	if session.Resource > session.MaxResource {
		session.Resource = session.MaxResource
	}
}

func (s *worldServer) buildStatsResponse(session *worldSessionState) statBlockResponse {
	stats := calculatePlayerStats(session.Level, session.Equipment, session.Talents)
	return statBlockResponse{
		Strength:          stats.Strength,
		Stamina:           stats.Stamina,
		Armor:             stats.Armor,
		AttackPower:       stats.AttackPower,
		ArmorReductionPct: stats.ArmorReductionPct,
	}
}
