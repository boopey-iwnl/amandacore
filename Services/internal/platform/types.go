package platform

type Role string

const (
	RolePlayer        Role = "player"
	RoleModerator     Role = "moderator"
	RoleGameMaster    Role = "game_master"
	RoleAdministrator Role = "administrator"

	InventorySlotCount    = 16
	ActionBarSlotCount    = 48
	StarterCurrencyCopper = 125
	DefaultStarterZoneID  = "stonewake_vale"
	DefaultStarterSpawnX  = 10.0
	DefaultStarterSpawnY  = 10.0
	DefaultStarterSpawnZ  = 0.0

	DefaultRaceID             = "human"
	DefaultClassID            = "warrior"
	LegacyWayfarerArchetypeID = "wayfarer_warden"

	AutoAttackAbilityID      = "auto_attack"
	SteadyStrikeAbilityID    = "steady_strike"
	BraceAbilityID           = "brace"
	DrivingBlowAbilityID     = "driving_blow"
	WarCryAbilityID          = "war_cry"
	HamperingStrikeAbilityID = "hampering_strike"
)

type CharacterInventorySlot struct {
	SlotIndex   int    `json:"slotIndex"`
	ItemID      string `json:"itemId"`
	DisplayName string `json:"displayName"`
	StackCount  int    `json:"stackCount"`
}

type CharacterActionBarSlot struct {
	SlotIndex int    `json:"slotIndex"`
	AbilityID string `json:"abilityId"`
}

type Account struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
	Roles        []Role `json:"roles"`
	Banned       bool   `json:"banned"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
}

type Session struct {
	ID               string `json:"id"`
	AccountID        string `json:"accountId"`
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	AccessExpiresAt  int64  `json:"accessExpiresAt"`
	RefreshExpiresAt int64  `json:"refreshExpiresAt"`
	CreatedAt        int64  `json:"createdAt"`
}

type Realm struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	Region         string `json:"region"`
	Endpoint       string `json:"endpoint"`
	SupportedBuild string `json:"supportedBuild"`
	OnlinePlayers  int    `json:"onlinePlayers"`
	Online         bool   `json:"online"`
}

type CharacterQuestProgress struct {
	QuestID         string `json:"questId"`
	State           string `json:"state"`
	CurrentCount    int    `json:"currentCount"`
	TargetCount     int    `json:"targetCount"`
	AcceptedAt      int64  `json:"acceptedAt"`
	CompletedAt     int64  `json:"completedAt"`
	RewardGrantedAt int64  `json:"rewardGrantedAt"`
	UpdatedAt       int64  `json:"updatedAt"`
}

type Character struct {
	ID                string                            `json:"id"`
	AccountID         string                            `json:"accountId"`
	RealmID           string                            `json:"realmId"`
	DisplayName       string                            `json:"displayName"`
	RaceID            string                            `json:"raceId"`
	ClassID           string                            `json:"classId"`
	ArchetypeID       string                            `json:"archetypeId"`
	Level             int                               `json:"level"`
	Experience        int                               `json:"experience"`
	CurrencyCopper    int                               `json:"currencyCopper"`
	ZoneID            string                            `json:"zoneId"`
	PositionX         float64                           `json:"positionX"`
	PositionY         float64                           `json:"positionY"`
	PositionZ         float64                           `json:"positionZ"`
	Inventory         []CharacterInventorySlot          `json:"inventory"`
	LearnedAbilityIDs []string                          `json:"learnedAbilityIds"`
	ActionBarSlots    []CharacterActionBarSlot          `json:"actionBarSlots"`
	Quests            map[string]CharacterQuestProgress `json:"quests"`
	LastSeenAt        int64                             `json:"lastSeenAt"`
}

type BuildManifest struct {
	ID                string   `json:"id"`
	Channel           string   `json:"channel"`
	DisplayVersion    string   `json:"displayVersion"`
	RequiredServices  []string `json:"requiredServices"`
	LauncherNews      string   `json:"launcherNews"`
	AllowedForLogin   bool     `json:"allowedForLogin"`
	WorldEndpointHint string   `json:"worldEndpointHint"`
}

type WorldJoinTicket struct {
	TicketID      string `json:"ticketId"`
	SessionID     string `json:"sessionId"`
	AccountID     string `json:"accountId"`
	CharacterID   string `json:"characterId"`
	RealmID       string `json:"realmId"`
	WorldEndpoint string `json:"worldEndpoint"`
	ExpiresAt     int64  `json:"expiresAt"`
	ConsumedAt    int64  `json:"consumedAt"`
}

type PasswordResetTicket struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	ExpiresAt int64  `json:"expiresAt"`
}

func DefaultStarterInventory() []CharacterInventorySlot {
	slots := make([]CharacterInventorySlot, InventorySlotCount)
	for slotIndex := range slots {
		slots[slotIndex].SlotIndex = slotIndex
	}

	slots[0].ItemID = "camp_ration"
	slots[0].DisplayName = "Camp Ration"
	slots[0].StackCount = 3

	slots[1].ItemID = "linen_wrap"
	slots[1].DisplayName = "Linen Wrap"
	slots[1].StackCount = 2
	return slots
}

func NormalizeCharacterIdentity(archetypeID string, raceID string, classID string) (string, string, string) {
	normalizedArchetypeID := archetypeID
	if normalizedArchetypeID == "" {
		normalizedArchetypeID = LegacyWayfarerArchetypeID
	}

	normalizedRaceID := raceID
	if normalizedRaceID == "" {
		normalizedRaceID = DefaultRaceID
	}

	normalizedClassID := classID
	if normalizedClassID == "" {
		normalizedClassID = DefaultClassID
	}

	if normalizedArchetypeID == LegacyWayfarerArchetypeID {
		normalizedRaceID = DefaultRaceID
		normalizedClassID = DefaultClassID
	}

	return normalizedArchetypeID, normalizedRaceID, normalizedClassID
}

func NormalizeCharacter(character Character) Character {
	character.ArchetypeID, character.RaceID, character.ClassID = NormalizeCharacterIdentity(
		character.ArchetypeID,
		character.RaceID,
		character.ClassID)
	character.Inventory = NormalizeInventorySlots(character.Inventory)
	character.LearnedAbilityIDs = NormalizeLearnedAbilityIDs(character.LearnedAbilityIDs)
	character.ActionBarSlots = NormalizeActionBarSlots(character.ActionBarSlots, character.LearnedAbilityIDs)
	if computedLevel := LevelForExperience(character.Experience); character.Level < computedLevel {
		character.Level = computedLevel
	}
	if character.Level <= 0 {
		character.Level = 1
	}
	if character.Quests == nil {
		character.Quests = map[string]CharacterQuestProgress{}
	}
	hasStarterQuestProgress := false
	for questID := range character.Quests {
		if len(questID) >= 3 && questID[:3] == "sv_" {
			hasStarterQuestProgress = true
			break
		}
	}
	legacyDefaultSpawn := character.ZoneID == "west_approach" &&
		character.Level <= 1 &&
		!hasStarterQuestProgress &&
		character.PositionX >= 11.99 && character.PositionX <= 12.01 &&
		character.PositionY >= 11.99 && character.PositionY <= 12.01 &&
		character.PositionZ >= -0.01 && character.PositionZ <= 0.01
	if character.ZoneID == "" || legacyDefaultSpawn {
		character.ZoneID = DefaultStarterZoneID
		character.PositionX = DefaultStarterSpawnX
		character.PositionY = DefaultStarterSpawnY
		character.PositionZ = DefaultStarterSpawnZ
	}
	return character
}

func LevelForExperience(experience int) int {
	switch {
	case experience >= 1200:
		return 6
	case experience >= 850:
		return 5
	case experience >= 550:
		return 4
	case experience >= 300:
		return 3
	case experience >= 100:
		return 2
	default:
		return 1
	}
}

func DefaultStartingLearnedAbilityIDs() []string {
	return []string{
		AutoAttackAbilityID,
		SteadyStrikeAbilityID,
		BraceAbilityID,
	}
}

func NormalizeLearnedAbilityIDs(source []string) []string {
	if len(source) == 0 {
		return DefaultStartingLearnedAbilityIDs()
	}

	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(source))
	for _, abilityID := range source {
		switch abilityID {
		case "ember_bolt":
			abilityID = SteadyStrikeAbilityID
		case "steady_blast":
			abilityID = BraceAbilityID
		}
		if abilityID == "" {
			continue
		}
		if _, exists := seen[abilityID]; exists {
			continue
		}
		seen[abilityID] = struct{}{}
		normalized = append(normalized, abilityID)
	}

	if len(normalized) == 0 {
		return DefaultStartingLearnedAbilityIDs()
	}

	return normalized
}

func DefaultActionBarSlots(learnedAbilityIDs []string) []CharacterActionBarSlot {
	slots := make([]CharacterActionBarSlot, ActionBarSlotCount)
	for slotIndex := range slots {
		slots[slotIndex].SlotIndex = slotIndex
	}

	known := map[string]struct{}{}
	for _, abilityID := range NormalizeLearnedAbilityIDs(learnedAbilityIDs) {
		known[abilityID] = struct{}{}
	}

	defaultSlotByAbility := map[string]int{
		AutoAttackAbilityID:   0,
		SteadyStrikeAbilityID: 1,
		BraceAbilityID:        2,
	}
	for abilityID, slotIndex := range defaultSlotByAbility {
		if _, learned := known[abilityID]; !learned {
			continue
		}
		slots[slotIndex].AbilityID = abilityID
	}

	return slots
}

func NormalizeActionBarSlots(source []CharacterActionBarSlot, learnedAbilityIDs []string) []CharacterActionBarSlot {
	if len(source) == 0 {
		return DefaultActionBarSlots(learnedAbilityIDs)
	}

	known := map[string]struct{}{}
	for _, abilityID := range NormalizeLearnedAbilityIDs(learnedAbilityIDs) {
		known[abilityID] = struct{}{}
	}

	slots := make([]CharacterActionBarSlot, ActionBarSlotCount)
	for slotIndex := range slots {
		slots[slotIndex].SlotIndex = slotIndex
	}

	for _, slot := range source {
		if slot.SlotIndex < 0 || slot.SlotIndex >= ActionBarSlotCount {
			continue
		}

		abilityID := slot.AbilityID
		switch abilityID {
		case "ember_bolt":
			abilityID = SteadyStrikeAbilityID
		case "steady_blast":
			abilityID = BraceAbilityID
		}
		if abilityID == "" {
			continue
		}
		if _, learned := known[abilityID]; !learned {
			continue
		}

		slots[slot.SlotIndex] = CharacterActionBarSlot{
			SlotIndex: slot.SlotIndex,
			AbilityID: abilityID,
		}
	}

	return slots
}

func NormalizeInventorySlots(source []CharacterInventorySlot) []CharacterInventorySlot {
	if len(source) == 0 {
		return DefaultStarterInventory()
	}

	slots := make([]CharacterInventorySlot, InventorySlotCount)
	for slotIndex := range slots {
		slots[slotIndex].SlotIndex = slotIndex
	}

	nextEmptySlot := 0
	for _, slot := range source {
		if slot.StackCount < 0 {
			slot.StackCount = 0
		}

		targetIndex := slot.SlotIndex
		if targetIndex < 0 || targetIndex >= InventorySlotCount {
			for nextEmptySlot < InventorySlotCount && slots[nextEmptySlot].ItemID != "" {
				nextEmptySlot++
			}
			if nextEmptySlot >= InventorySlotCount {
				continue
			}
			targetIndex = nextEmptySlot
		}

		normalizedSlot := CharacterInventorySlot{
			SlotIndex:   targetIndex,
			ItemID:      slot.ItemID,
			DisplayName: slot.DisplayName,
			StackCount:  slot.StackCount,
		}
		if normalizedSlot.StackCount <= 0 {
			normalizedSlot.ItemID = ""
			normalizedSlot.DisplayName = ""
			normalizedSlot.StackCount = 0
		}
		slots[targetIndex] = normalizedSlot
	}

	return slots
}
