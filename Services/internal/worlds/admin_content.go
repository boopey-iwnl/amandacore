package worlds

type AdminItemDefinition struct {
	ItemID      string `json:"itemId"`
	DisplayName string `json:"displayName"`
	Stackable   bool   `json:"stackable"`
	MaxStack    int    `json:"maxStack"`
}

type AdminSafePosition struct {
	ZoneID string  `json:"zoneId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
}

func FindAdminItemDefinition(itemID string) (AdminItemDefinition, bool) {
	item, found := findItemDefinition(itemID)
	if !found {
		return AdminItemDefinition{}, false
	}
	return AdminItemDefinition{
		ItemID:      item.ItemID,
		DisplayName: item.DisplayName,
		Stackable:   item.Stackable,
		MaxStack:    item.MaxStack,
	}, true
}

func AdminQuestTargetCount(questID string) (int, bool) {
	allQuests := append([]questDefinition{}, stonewakeQuestDefinitions...)
	allQuests = append(allQuests, brindlebrookQuestDefinitions...)
	for _, quest := range allQuests {
		if quest.ID != questID {
			continue
		}
		if quest.TargetCount <= 0 {
			return 1, true
		}
		return quest.TargetCount, true
	}
	return 0, false
}

func StonewakeAdminSafePosition() AdminSafePosition {
	return AdminSafePosition{
		ZoneID: defaultZoneID,
		X:      starterSpawnX,
		Y:      starterSpawnY,
		Z:      starterSpawnZ,
	}
}

func CurrentZoneAdminSafePosition(zoneID string) AdminSafePosition {
	switch zoneID {
	case secondZoneID:
		return AdminSafePosition{
			ZoneID: secondZoneID,
			X:      secondZoneEntryX,
			Y:      secondZoneEntryY,
			Z:      starterSpawnZ,
		}
	case dungeonZoneID:
		return AdminSafePosition{
			ZoneID: defaultZoneID,
			X:      starterSpawnX,
			Y:      starterSpawnY,
			Z:      starterSpawnZ,
		}
	default:
		return StonewakeAdminSafePosition()
	}
}
