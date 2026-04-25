package platform

type Role string

const (
	RolePlayer        Role = "player"
	RoleModerator     Role = "moderator"
	RoleGameMaster    Role = "game_master"
	RoleAdministrator Role = "administrator"

	InventorySlotCount     = 16
	ActionBarSlotCount     = 48
	StarterCurrencyCopper  = 125
	PrimaryProfessionLimit = 2
	DefaultStarterZoneID   = "stonewake_vale"
	DefaultStarterSpawnX   = 10.0
	DefaultStarterSpawnY   = 10.0
	DefaultStarterSpawnZ   = 0.0

	DefaultRaceID             = "human"
	DefaultClassID            = "warrior"
	LegacyWayfarerArchetypeID = "wayfarer_warden"

	AutoAttackAbilityID      = "auto_attack"
	SteadyStrikeAbilityID    = "steady_strike"
	BraceAbilityID           = "brace"
	DrivingBlowAbilityID     = "driving_blow"
	RallyingCallAbilityID    = "rallying_call"
	WarCryAbilityID          = RallyingCallAbilityID
	HamperingStrikeAbilityID = "hampering_strike"
	GuardedFormAbilityID     = "guarded_form"
	OverhandCutAbilityID     = "overhand_cut"
	IronResolveAbilityID     = "iron_resolve"

	ProfessionOrekeepingID = "orekeeping"
	ProfessionForgecraftID = "forgecraft"
	ProfessionFieldAidID   = "field_aid"
)

var EquipmentSlots = []string{
	EquipmentSlotMainHand,
	EquipmentSlotChest,
	EquipmentSlotHands,
	EquipmentSlotLegs,
	EquipmentSlotFeet,
}

const (
	EquipmentSlotMainHand = "main_hand"
	EquipmentSlotChest    = "chest"
	EquipmentSlotHands    = "hands"
	EquipmentSlotLegs     = "legs"
	EquipmentSlotFeet     = "feet"
)

type CharacterInventorySlot struct {
	SlotIndex   int    `json:"slotIndex"`
	ItemID      string `json:"itemId"`
	DisplayName string `json:"displayName"`
	StackCount  int    `json:"stackCount"`
}

type CharacterEquipmentSlot struct {
	Slot        string `json:"slot"`
	ItemID      string `json:"itemId"`
	DisplayName string `json:"displayName"`
}

type CharacterProfessionState struct {
	ProfessionID   string   `json:"professionId"`
	SkillValue     int      `json:"skillValue"`
	RankID         string   `json:"rankId"`
	KnownRecipeIDs []string `json:"knownRecipeIds"`
	LearnedAt      int64    `json:"learnedAt"`
	UpdatedAt      int64    `json:"updatedAt"`
}

type CharacterActionBarSlot struct {
	SlotIndex int    `json:"slotIndex"`
	AbilityID string `json:"abilityId"`
}

type CharacterTalentRank struct {
	TalentID string `json:"talentId"`
	Rank     int    `json:"rank"`
}

type ChatMessage struct {
	MessageID         string `json:"messageId"`
	Channel           string `json:"channel"`
	SenderCharacterID string `json:"senderCharacterId"`
	SenderDisplayName string `json:"senderDisplayName"`
	TargetCharacterID string `json:"targetCharacterId,omitempty"`
	PartyID           string `json:"partyId,omitempty"`
	GuildID           string `json:"guildId,omitempty"`
	ZoneID            string `json:"zoneId,omitempty"`
	MessageText       string `json:"messageText"`
	Timestamp         int64  `json:"timestamp"`
}

type FriendRelationship struct {
	OwnerCharacterID  string `json:"ownerCharacterId"`
	FriendCharacterID string `json:"friendCharacterId"`
	FriendDisplayName string `json:"friendDisplayName"`
	CreatedAt         int64  `json:"createdAt"`
}

type Party struct {
	ID                 string   `json:"partyId"`
	LeaderCharacterID  string   `json:"leaderCharacterId"`
	MemberCharacterIDs []string `json:"memberCharacterIds"`
	CreatedAt          int64    `json:"createdAt"`
	UpdatedAt          int64    `json:"updatedAt"`
}

const (
	GuildRankLeader  = "leader"
	GuildRankOfficer = "officer"
	GuildRankMember  = "member"
	GuildRankRecruit = "recruit"

	GuildPermissionInviteMember  = "invite_member"
	GuildPermissionRemoveMember  = "remove_member"
	GuildPermissionPromoteMember = "promote_member"
	GuildPermissionDemoteMember  = "demote_member"
	GuildPermissionEditMOTD      = "edit_motd"
	GuildPermissionDisbandGuild  = "disband_guild"
)

type GuildRank struct {
	RankID      string   `json:"rankId"`
	DisplayName string   `json:"displayName"`
	Priority    int      `json:"priority"`
	Permissions []string `json:"permissions"`
}

type GuildMember struct {
	CharacterID  string `json:"characterId"`
	DisplayName  string `json:"displayName"`
	RaceID       string `json:"raceId"`
	ClassID      string `json:"classId"`
	Level        int    `json:"level"`
	RankID       string `json:"rankId"`
	JoinedAt     int64  `json:"joinedAt"`
	LastOnlineAt int64  `json:"lastOnlineAt"`
}

type Guild struct {
	ID                   string        `json:"guildId"`
	RealmID              string        `json:"realmId"`
	GuildName            string        `json:"guildName"`
	CreatedAt            int64         `json:"createdAt"`
	UpdatedAt            int64         `json:"updatedAt"`
	CreatedByCharacterID string        `json:"createdByCharacterId"`
	LeaderCharacterID    string        `json:"leaderCharacterId"`
	MessageOfTheDay      string        `json:"messageOfTheDay,omitempty"`
	Ranks                []GuildRank   `json:"ranks"`
	Members              []GuildMember `json:"members"`
}

type GuildInvite struct {
	InviteID           string `json:"inviteId"`
	GuildID            string `json:"guildId"`
	GuildName          string `json:"guildName"`
	InviterCharacterID string `json:"inviterCharacterId"`
	TargetCharacterID  string `json:"targetCharacterId"`
	CreatedAt          int64  `json:"createdAt"`
	ExpiresAt          int64  `json:"expiresAt"`
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

type CharacterPvPStats struct {
	CharacterID   string `json:"characterId"`
	DuelsWon      int    `json:"duelsWon"`
	DuelsLost     int    `json:"duelsLost"`
	HonorPoints   int    `json:"honorPoints"`
	LastDuelWonAt int64  `json:"lastDuelWonAt"`
	UpdatedAt     int64  `json:"updatedAt"`
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
	Equipment         []CharacterEquipmentSlot          `json:"equipment"`
	Professions       []CharacterProfessionState        `json:"professions"`
	LearnedAbilityIDs []string                          `json:"learnedAbilityIds"`
	ActionBarSlots    []CharacterActionBarSlot          `json:"actionBarSlots"`
	Talents           map[string]int                    `json:"talents"`
	Quests            map[string]CharacterQuestProgress `json:"quests"`
	TrackedQuestIDs   []string                          `json:"trackedQuestIds"`
	PvPStats          CharacterPvPStats                 `json:"pvpStats"`
	LastSeenAt        int64                             `json:"lastSeenAt"`
}

type BuildManifest struct {
	ID                         string   `json:"id"`
	Channel                    string   `json:"channel"`
	DisplayVersion             string   `json:"displayVersion"`
	ClientVersion              string   `json:"clientVersion"`
	ServerVersion              string   `json:"serverVersion"`
	ContentVersion             string   `json:"contentVersion"`
	ProtocolVersion            string   `json:"protocolVersion"`
	APIVersion                 string   `json:"apiVersion"`
	CompatibleClientVersions   []string `json:"compatibleClientVersions"`
	CompatibleServerVersions   []string `json:"compatibleServerVersions"`
	CompatibleProtocolVersions []string `json:"compatibleProtocolVersions"`
	RequiredServices           []string `json:"requiredServices"`
	LauncherNews               string   `json:"launcherNews"`
	AllowedForLogin            bool     `json:"allowedForLogin"`
	WorldEndpointHint          string   `json:"worldEndpointHint"`
	GeneratedAtUTC             string   `json:"generatedAtUtc"`
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

func DefaultEquipmentSlots() []CharacterEquipmentSlot {
	slots := make([]CharacterEquipmentSlot, len(EquipmentSlots))
	for index, slot := range EquipmentSlots {
		slots[index].Slot = slot
	}
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
	character.Equipment = NormalizeEquipmentSlots(character.Equipment)
	character.Professions = NormalizeProfessionStates(character.Professions)
	character.LearnedAbilityIDs = NormalizeLearnedAbilityIDs(character.LearnedAbilityIDs)
	character.ActionBarSlots = NormalizeActionBarSlots(character.ActionBarSlots, character.LearnedAbilityIDs)
	character.Talents = NormalizeTalentRanks(character.Talents)
	if computedLevel := LevelForExperience(character.Experience); character.Level < computedLevel {
		character.Level = computedLevel
	}
	if character.Level <= 0 {
		character.Level = 1
	}
	if character.Quests == nil {
		character.Quests = map[string]CharacterQuestProgress{}
	}
	character.TrackedQuestIDs = NormalizeStringIDs(character.TrackedQuestIDs)
	character.PvPStats = NormalizeCharacterPvPStats(character.ID, character.PvPStats)
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

func NormalizeCharacterPvPStats(characterID string, stats CharacterPvPStats) CharacterPvPStats {
	if stats.CharacterID == "" {
		stats.CharacterID = characterID
	}
	if stats.DuelsWon < 0 {
		stats.DuelsWon = 0
	}
	if stats.DuelsLost < 0 {
		stats.DuelsLost = 0
	}
	if stats.HonorPoints < 0 {
		stats.HonorPoints = 0
	}
	return stats
}

func DefaultGuildRanks() []GuildRank {
	return []GuildRank{
		{
			RankID:      GuildRankLeader,
			DisplayName: "Leader",
			Priority:    0,
			Permissions: []string{
				GuildPermissionInviteMember,
				GuildPermissionRemoveMember,
				GuildPermissionPromoteMember,
				GuildPermissionDemoteMember,
				GuildPermissionEditMOTD,
				GuildPermissionDisbandGuild,
			},
		},
		{
			RankID:      GuildRankOfficer,
			DisplayName: "Officer",
			Priority:    1,
			Permissions: []string{
				GuildPermissionInviteMember,
				GuildPermissionRemoveMember,
				GuildPermissionPromoteMember,
				GuildPermissionDemoteMember,
				GuildPermissionEditMOTD,
			},
		},
		{
			RankID:      GuildRankMember,
			DisplayName: "Member",
			Priority:    2,
			Permissions: []string{},
		},
		{
			RankID:      GuildRankRecruit,
			DisplayName: "Recruit",
			Priority:    3,
			Permissions: []string{},
		},
	}
}

func NormalizeStringIDs(source []string) []string {
	if len(source) == 0 {
		return []string{}
	}

	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(source))
	for _, id := range source {
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized
}

func LevelForExperience(experience int) int {
	switch {
	case experience >= 4900:
		return 12
	case experience >= 4000:
		return 11
	case experience >= 3200:
		return 10
	case experience >= 2500:
		return 9
	case experience >= 1900:
		return 8
	case experience >= 1400:
		return 7
	case experience >= 1000:
		return 6
	case experience >= 700:
		return 5
	case experience >= 450:
		return 4
	case experience >= 250:
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

func NormalizeTalentRanks(source map[string]int) map[string]int {
	normalized := map[string]int{}
	for talentID, rank := range source {
		if talentID == "" || rank <= 0 {
			continue
		}
		normalized[talentID] = rank
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

func NormalizeEquipmentSlots(source []CharacterEquipmentSlot) []CharacterEquipmentSlot {
	slots := DefaultEquipmentSlots()
	slotIndexByName := map[string]int{}
	for index, slot := range slots {
		slotIndexByName[slot.Slot] = index
	}

	for _, sourceSlot := range source {
		index, ok := slotIndexByName[sourceSlot.Slot]
		if !ok {
			continue
		}

		normalizedSlot := CharacterEquipmentSlot{
			Slot:        sourceSlot.Slot,
			ItemID:      sourceSlot.ItemID,
			DisplayName: sourceSlot.DisplayName,
		}
		if normalizedSlot.ItemID == "" {
			normalizedSlot.DisplayName = ""
		}
		slots[index] = normalizedSlot
	}

	return slots
}

func NormalizeProfessionStates(source []CharacterProfessionState) []CharacterProfessionState {
	if len(source) == 0 {
		return []CharacterProfessionState{}
	}

	seenProfessions := map[string]struct{}{}
	normalized := make([]CharacterProfessionState, 0, len(source))
	for _, profession := range source {
		if profession.ProfessionID == "" {
			continue
		}
		if _, exists := seenProfessions[profession.ProfessionID]; exists {
			continue
		}
		seenProfessions[profession.ProfessionID] = struct{}{}

		if profession.SkillValue < 0 {
			profession.SkillValue = 0
		}
		if profession.RankID == "" {
			profession.RankID = "novice"
		}
		profession.KnownRecipeIDs = NormalizeKnownRecipeIDs(profession.KnownRecipeIDs)
		normalized = append(normalized, profession)
	}

	if len(normalized) == 0 {
		return []CharacterProfessionState{}
	}
	return normalized
}

func NormalizeKnownRecipeIDs(source []string) []string {
	if len(source) == 0 {
		return []string{}
	}

	seenRecipes := map[string]struct{}{}
	normalized := make([]string, 0, len(source))
	for _, recipeID := range source {
		if recipeID == "" {
			continue
		}
		if _, exists := seenRecipes[recipeID]; exists {
			continue
		}
		seenRecipes[recipeID] = struct{}{}
		normalized = append(normalized, recipeID)
	}
	return normalized
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
