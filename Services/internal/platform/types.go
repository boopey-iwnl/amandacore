package platform

type Role string
type Permission string

const (
	RolePlayer        Role = "player"
	RoleTester        Role = "tester"
	RoleSupport       Role = "support"
	RoleGM            Role = "gm"
	RoleAdmin         Role = "admin"
	RoleModerator     Role = "moderator"
	RoleGameMaster    Role = "game_master"
	RoleAdministrator Role = "administrator"

	PermissionViewAccount       Permission = "view_account"
	PermissionViewCharacter     Permission = "view_character"
	PermissionViewInventory     Permission = "view_inventory"
	PermissionViewEconomy       Permission = "view_economy"
	PermissionRepairCharacter   Permission = "repair_character"
	PermissionTeleportCharacter Permission = "teleport_character"
	PermissionGrantItem         Permission = "grant_item"
	PermissionGrantCurrency     Permission = "grant_currency"
	PermissionModifyQuestState  Permission = "modify_quest_state"
	PermissionModerateChat      Permission = "moderate_chat"
	PermissionSuspendAccount    Permission = "suspend_account"
	PermissionViewAuditLog      Permission = "view_audit_log"
	PermissionManageSupport     Permission = "manage_support"

	InventorySlotCount      = 16
	HousingStorageSlotCount = 24
	HousingDecorationLimit  = 8
	ActionBarSlotCount      = 48
	StarterCurrencyCopper   = 125
	PrimaryProfessionLimit  = 2
	DefaultStarterZoneID    = "stonewake_vale"
	DefaultStarterSpawnX    = 10.0
	DefaultStarterSpawnY    = 10.0
	DefaultStarterSpawnZ    = 0.0

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

	DefaultBindLocationID  = "bind_hearthwatch_yard"
	DefaultBindDisplayName = "Hearthwatch Yard"
	DefaultTravelPointID   = "travel_hearthwatch_yard"
	DefaultBindZoneID      = DefaultStarterZoneID
	DefaultBindPositionX   = 13.0
	DefaultBindPositionY   = 10.0
	DefaultBindPositionZ   = 0.0

	AchievementScopeAccount   = "account"
	AchievementScopeCharacter = "character"

	CriteriaCompleteQuest     = "complete_quest"
	CriteriaReachLevel        = "reach_level"
	CriteriaDefeatMobType     = "defeat_mob_type"
	CriteriaDefeatNamedEnemy  = "defeat_named_enemy"
	CriteriaLearnProfession   = "learn_profession"
	CriteriaCraftItem         = "craft_item"
	CriteriaJoinParty         = "join_party"
	CriteriaJoinGuild         = "join_guild"
	CriteriaCompleteDungeon   = "complete_dungeon"
	CriteriaWinDuel           = "win_duel"
	CriteriaVendorBuy         = "vendor_buy"
	CriteriaVendorSell        = "vendor_sell"
	CriteriaEnterHousing      = "enter_housing"
	CriteriaPlaceDecoration   = "place_decoration"
	CriteriaCurrencyThreshold = "currency_threshold"
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

type HousingEntitlement struct {
	CharacterID    string `json:"characterId"`
	HousingSpaceID string `json:"housingSpaceId"`
	TemplateID     string `json:"templateId"`
	Unlocked       bool   `json:"unlocked"`
	CreatedAt      int64  `json:"createdAt"`
}

type HousingSpace struct {
	HousingSpaceID   string  `json:"housingSpaceId"`
	OwnerCharacterID string  `json:"ownerCharacterId"`
	OwnerAccountID   string  `json:"ownerAccountId"`
	TemplateID       string  `json:"templateId"`
	CreatedAt        int64   `json:"createdAt"`
	LastVisitedAt    int64   `json:"lastVisitedAt"`
	ReturnZoneID     string  `json:"returnZoneId"`
	ReturnX          float64 `json:"returnX"`
	ReturnY          float64 `json:"returnY"`
	ReturnZ          float64 `json:"returnZ"`
}

type HousingStorageSlot struct {
	SlotIndex   int    `json:"slotIndex"`
	ItemID      string `json:"itemId"`
	DisplayName string `json:"displayName"`
	StackCount  int    `json:"stackCount"`
}

type DecorationPlacement struct {
	PlacementID    string  `json:"placementId"`
	HousingSpaceID string  `json:"housingSpaceId"`
	DecorationID   string  `json:"decorationId"`
	DisplayName    string  `json:"displayName"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Z              float64 `json:"z"`
	RotationYaw    float64 `json:"rotationYaw"`
	CreatedAt      int64   `json:"createdAt"`
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

const (
	AuctionStateActive   = "active"
	AuctionStateSold     = "sold"
	AuctionStateCanceled = "canceled"
	AuctionStateExpired  = "expired"
)

type AuctionListing struct {
	AuctionID                string `json:"auctionId"`
	RealmID                  string `json:"realmId"`
	SellerCharacterID        string `json:"sellerCharacterId"`
	SellerDisplayName        string `json:"sellerDisplayName"`
	BuyerCharacterID         string `json:"buyerCharacterId,omitempty"`
	ItemID                   string `json:"itemId"`
	ItemDisplayName          string `json:"itemDisplayName"`
	ItemQuality              string `json:"itemQuality"`
	ItemType                 string `json:"itemType"`
	ItemSubtype              string `json:"itemSubtype"`
	ItemStackable            bool   `json:"itemStackable"`
	ItemMaxStack             int    `json:"itemMaxStack"`
	StackCount               int    `json:"stackCount"`
	BuyoutCopper             int    `json:"buyoutCopper"`
	BidCopper                int    `json:"bidCopper,omitempty"`
	CurrentBidCopper         int    `json:"currentBidCopper,omitempty"`
	CurrentBidderCharacterID string `json:"currentBidderCharacterId,omitempty"`
	DepositCopper            int    `json:"depositCopper"`
	CutCopper                int    `json:"cutCopper"`
	CutPercent               int    `json:"cutPercent"`
	CreatedAt                int64  `json:"createdAt"`
	ExpiresAt                int64  `json:"expiresAt"`
	SoldAt                   int64  `json:"soldAt,omitempty"`
	CanceledAt               int64  `json:"canceledAt,omitempty"`
	State                    string `json:"state"`
	SourceInventorySlot      int    `json:"sourceInventorySlot"`
	Version                  int    `json:"version"`
	ItemDeliveredMailID      string `json:"itemDeliveredMailId,omitempty"`
	ProceedsDeliveredMailID  string `json:"proceedsDeliveredMailId,omitempty"`
	ReturnDeliveredMailID    string `json:"returnDeliveredMailId,omitempty"`
}

type MailItemAttachment struct {
	ItemID      string `json:"itemId"`
	DisplayName string `json:"displayName"`
	StackCount  int    `json:"stackCount"`
}

type MailEnvelope struct {
	MailID               string               `json:"mailId"`
	AuctionID            string               `json:"auctionId,omitempty"`
	SenderCharacterID    string               `json:"senderCharacterId,omitempty"`
	SenderDisplayName    string               `json:"senderDisplayName"`
	RecipientCharacterID string               `json:"recipientCharacterId"`
	Subject              string               `json:"subject"`
	Body                 string               `json:"body"`
	ItemAttachments      []MailItemAttachment `json:"itemAttachments"`
	CurrencyCopper       int                  `json:"currencyCopper"`
	CreatedAt            int64                `json:"createdAt"`
	DeliveredAt          int64                `json:"deliveredAt,omitempty"`
}

type EconomyAuditEvent struct {
	EventID           string `json:"eventId"`
	EventType         string `json:"eventType"`
	AuctionID         string `json:"auctionId,omitempty"`
	SellerCharacterID string `json:"sellerCharacterId,omitempty"`
	BuyerCharacterID  string `json:"buyerCharacterId,omitempty"`
	ItemID            string `json:"itemId,omitempty"`
	StackCount        int    `json:"stackCount,omitempty"`
	BuyoutCopper      int    `json:"buyoutCopper,omitempty"`
	DepositCopper     int    `json:"depositCopper,omitempty"`
	CutCopper         int    `json:"cutCopper,omitempty"`
	Reason            string `json:"reason,omitempty"`
	CreatedAt         int64  `json:"createdAt"`
}

type Account struct {
	ID               string `json:"id"`
	Username         string `json:"username"`
	PasswordHash     string `json:"passwordHash"`
	Roles            []Role `json:"roles"`
	Banned           bool   `json:"banned"`
	SuspendedUntil   int64  `json:"suspendedUntil,omitempty"`
	SuspensionReason string `json:"suspensionReason,omitempty"`
	LastLoginAt      int64  `json:"lastLoginAt,omitempty"`
	LastSessionID    string `json:"lastSessionId,omitempty"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
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

type CharacterBindPoint struct {
	CharacterID    string  `json:"characterId"`
	ZoneID         string  `json:"zoneId"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Z              float64 `json:"z"`
	BindLocationID string  `json:"bindLocationId"`
	DisplayName    string  `json:"displayName"`
	SetAt          int64   `json:"setAt"`
}

type CharacterTravelState struct {
	DiscoveredTravelPointIDs []string `json:"discoveredTravelPointIds"`
	LastTravelPointID        string   `json:"lastTravelPointId,omitempty"`
	RecallReadyAt            int64    `json:"recallReadyAt,omitempty"`
}

type CharacterMountState struct {
	UnlockedMountIDs     []string `json:"unlockedMountIds"`
	SelectedMountID      string   `json:"selectedMountId,omitempty"`
	CurrentlyMounted     bool     `json:"currentlyMounted"`
	MountedSince         int64    `json:"mountedSince,omitempty"`
	CurrentSpeedModifier float64  `json:"currentSpeedModifier"`
}

type AchievementCriteria struct {
	Type         string `json:"type"`
	QuestID      string `json:"questId,omitempty"`
	Level        int    `json:"level,omitempty"`
	MobTypeID    string `json:"mobTypeId,omitempty"`
	MobID        string `json:"mobId,omitempty"`
	ProfessionID string `json:"professionId,omitempty"`
	RecipeID     string `json:"recipeId,omitempty"`
	ItemID       string `json:"itemId,omitempty"`
	DungeonID    string `json:"dungeonId,omitempty"`
	TargetValue  int    `json:"targetValue,omitempty"`
}

type AchievementDefinition struct {
	AchievementID            string                `json:"achievementId"`
	DisplayName              string                `json:"displayName"`
	Description              string                `json:"description"`
	Category                 string                `json:"category"`
	Scope                    string                `json:"scope"`
	Criteria                 []AchievementCriteria `json:"criteria"`
	Points                   int                   `json:"points,omitempty"`
	RewardTitleID            string                `json:"rewardTitleId,omitempty"`
	RewardCollectionUnlockID string                `json:"rewardCollectionUnlockId,omitempty"`
	Hidden                   bool                  `json:"hidden,omitempty"`
	FactionRequirementID     string                `json:"factionRequirementId,omitempty"`
	ClassRequirementID       string                `json:"classRequirementId,omitempty"`
	RaceRequirementID        string                `json:"raceRequirementId,omitempty"`
	Version                  int                   `json:"version,omitempty"`
	CreatedAt                int64                 `json:"createdAt,omitempty"`
}

type AchievementProgress struct {
	AccountID               string `json:"accountId"`
	CharacterID             string `json:"characterId,omitempty"`
	AchievementID           string `json:"achievementId"`
	CurrentValue            int    `json:"currentValue"`
	TargetValue             int    `json:"targetValue"`
	Completed               bool   `json:"completed"`
	CompletedAt             int64  `json:"completedAt,omitempty"`
	ContributingCharacterID string `json:"contributingCharacterId,omitempty"`
	UpdatedAt               int64  `json:"updatedAt"`
}

type TitleDefinition struct {
	TitleID             string `json:"titleId"`
	DisplayName         string `json:"displayName"`
	Prefix              string `json:"prefix,omitempty"`
	Suffix              string `json:"suffix,omitempty"`
	SourceAchievementID string `json:"sourceAchievementId,omitempty"`
	AccountWide         bool   `json:"accountWide"`
	Hidden              bool   `json:"hidden,omitempty"`
}

type CollectionUnlock struct {
	UnlockID            string `json:"unlockId"`
	Category            string `json:"category"`
	DisplayName         string `json:"displayName"`
	SourceAchievementID string `json:"sourceAchievementId,omitempty"`
	UnlockedAt          int64  `json:"unlockedAt"`
}

type AccountProgressState struct {
	AccountID                  string                         `json:"accountId"`
	AchievementProgress        map[string]AchievementProgress `json:"achievementProgress"`
	UnlockedTitleIDs           []string                       `json:"unlockedTitleIds"`
	SelectedTitleByCharacterID map[string]string              `json:"selectedTitleByCharacterId"`
	CollectionUnlocks          map[string]CollectionUnlock    `json:"collectionUnlocks"`
	UpdatedAt                  int64                          `json:"updatedAt"`
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
	BindPoint         CharacterBindPoint                `json:"bindPoint"`
	TravelState       CharacterTravelState              `json:"travelState"`
	MountState        CharacterMountState               `json:"mountState"`
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

type AuditEvent struct {
	ID                string         `json:"auditEventId"`
	Timestamp         int64          `json:"timestamp"`
	Action            string         `json:"action"`
	ActorAccountID    string         `json:"actorAccountId"`
	ActorCharacterID  string         `json:"actorCharacterId,omitempty"`
	TargetAccountID   string         `json:"targetAccountId,omitempty"`
	TargetCharacterID string         `json:"targetCharacterId,omitempty"`
	Reason            string         `json:"reason,omitempty"`
	BeforeSummary     map[string]any `json:"beforeSummary,omitempty"`
	AfterSummary      map[string]any `json:"afterSummary,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

type SupportTicketStatus string

const (
	SupportTicketOpen     SupportTicketStatus = "open"
	SupportTicketInReview SupportTicketStatus = "in_review"
	SupportTicketResolved SupportTicketStatus = "resolved"
	SupportTicketClosed   SupportTicketStatus = "closed"
)

type SupportTicketNote struct {
	NoteID          string `json:"noteId"`
	TicketID        string `json:"ticketId"`
	AuthorAccountID string `json:"authorAccountId"`
	Body            string `json:"body"`
	CreatedAt       int64  `json:"createdAt"`
}

type SupportTicket struct {
	TicketID              string              `json:"ticketId"`
	CreatedByCharacterID  string              `json:"createdByCharacterId"`
	CreatedByAccountID    string              `json:"createdByAccountId"`
	Category              string              `json:"category"`
	Subject               string              `json:"subject"`
	Body                  string              `json:"body"`
	Status                SupportTicketStatus `json:"status"`
	AssignedToAdminID     string              `json:"assignedToAdminId,omitempty"`
	CreatedAt             int64               `json:"createdAt"`
	UpdatedAt             int64               `json:"updatedAt"`
	ResolutionNote        string              `json:"resolutionNote,omitempty"`
	AttachedDiagnosticsID string              `json:"attachedDiagnosticsId,omitempty"`
	BuildID               string              `json:"buildId,omitempty"`
	ClientVersion         string              `json:"clientVersion,omitempty"`
	Notes                 []SupportTicketNote `json:"notes,omitempty"`
}

type MuteRecord struct {
	CharacterID      string `json:"characterId"`
	AccountID        string `json:"accountId"`
	MutedByAccountID string `json:"mutedByAccountId"`
	Reason           string `json:"reason"`
	CreatedAt        int64  `json:"createdAt"`
	ExpiresAt        int64  `json:"expiresAt"`
}

func PermissionsForRoles(roles []Role) []Permission {
	seen := map[Permission]struct{}{}
	add := func(permission Permission) {
		if permission == "" {
			return
		}
		seen[permission] = struct{}{}
	}

	for _, role := range roles {
		switch role {
		case RolePlayer:
		case RoleTester:
			add(PermissionViewAccount)
			add(PermissionViewCharacter)
			add(PermissionViewInventory)
			add(PermissionViewEconomy)
		case RoleSupport:
			add(PermissionViewAccount)
			add(PermissionViewCharacter)
			add(PermissionViewInventory)
			add(PermissionViewEconomy)
			add(PermissionManageSupport)
		case RoleModerator:
			add(PermissionViewAccount)
			add(PermissionViewCharacter)
			add(PermissionViewInventory)
			add(PermissionViewEconomy)
			add(PermissionManageSupport)
			add(PermissionModerateChat)
		case RoleGM, RoleGameMaster:
			add(PermissionViewAccount)
			add(PermissionViewCharacter)
			add(PermissionViewInventory)
			add(PermissionViewEconomy)
			add(PermissionManageSupport)
			add(PermissionRepairCharacter)
			add(PermissionTeleportCharacter)
			add(PermissionModifyQuestState)
			add(PermissionModerateChat)
		case RoleAdmin, RoleAdministrator:
			for _, permission := range AllAdminPermissions() {
				add(permission)
			}
		}
	}

	permissions := make([]Permission, 0, len(seen))
	for permission := range seen {
		permissions = append(permissions, permission)
	}
	return permissions
}

func AllAdminPermissions() []Permission {
	return []Permission{
		PermissionViewAccount,
		PermissionViewCharacter,
		PermissionViewInventory,
		PermissionViewEconomy,
		PermissionRepairCharacter,
		PermissionTeleportCharacter,
		PermissionGrantItem,
		PermissionGrantCurrency,
		PermissionModifyQuestState,
		PermissionModerateChat,
		PermissionSuspendAccount,
		PermissionViewAuditLog,
		PermissionManageSupport,
	}
}

func HasPermission(roles []Role, required Permission) bool {
	for _, permission := range PermissionsForRoles(roles) {
		if permission == required {
			return true
		}
	}
	return false
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
	character.BindPoint = NormalizeCharacterBindPoint(character.ID, character.BindPoint)
	character.TravelState = NormalizeCharacterTravelState(character.TravelState)
	character.MountState = NormalizeCharacterMountState(character.MountState)
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

func NormalizeCharacterBindPoint(characterID string, bindPoint CharacterBindPoint) CharacterBindPoint {
	if bindPoint.CharacterID == "" {
		bindPoint.CharacterID = characterID
	}
	if bindPoint.ZoneID == "" {
		bindPoint.ZoneID = DefaultBindZoneID
	}
	if bindPoint.BindLocationID == "" {
		bindPoint.BindLocationID = DefaultBindLocationID
	}
	if bindPoint.DisplayName == "" {
		bindPoint.DisplayName = DefaultBindDisplayName
	}
	if bindPoint.X == 0 && bindPoint.Y == 0 && bindPoint.Z == 0 {
		bindPoint.X = DefaultBindPositionX
		bindPoint.Y = DefaultBindPositionY
		bindPoint.Z = DefaultBindPositionZ
	}
	return bindPoint
}

func NormalizeCharacterTravelState(travelState CharacterTravelState) CharacterTravelState {
	travelState.DiscoveredTravelPointIDs = NormalizeStringIDs(travelState.DiscoveredTravelPointIDs)
	if !containsStringID(travelState.DiscoveredTravelPointIDs, DefaultTravelPointID) {
		travelState.DiscoveredTravelPointIDs = append([]string{DefaultTravelPointID}, travelState.DiscoveredTravelPointIDs...)
	}
	if travelState.LastTravelPointID == "" {
		travelState.LastTravelPointID = DefaultTravelPointID
	}
	if travelState.RecallReadyAt < 0 {
		travelState.RecallReadyAt = 0
	}
	return travelState
}

func NormalizeCharacterMountState(mountState CharacterMountState) CharacterMountState {
	mountState.UnlockedMountIDs = NormalizeStringIDs(mountState.UnlockedMountIDs)
	if mountState.SelectedMountID != "" && !containsStringID(mountState.UnlockedMountIDs, mountState.SelectedMountID) {
		mountState.SelectedMountID = ""
	}
	mountState.CurrentlyMounted = false
	mountState.MountedSince = 0
	mountState.CurrentSpeedModifier = 1.0
	return mountState
}

func containsStringID(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func NormalizeAccountProgress(accountID string, progress AccountProgressState) AccountProgressState {
	if progress.AccountID == "" {
		progress.AccountID = accountID
	}
	if progress.AchievementProgress == nil {
		progress.AchievementProgress = map[string]AchievementProgress{}
	}
	for achievementID, achievementProgress := range progress.AchievementProgress {
		if achievementProgress.AchievementID == "" {
			achievementProgress.AchievementID = achievementID
		}
		if achievementProgress.AccountID == "" {
			achievementProgress.AccountID = progress.AccountID
		}
		if achievementProgress.TargetValue <= 0 {
			achievementProgress.TargetValue = 1
		}
		if achievementProgress.CurrentValue < 0 {
			achievementProgress.CurrentValue = 0
		}
		if achievementProgress.Completed && achievementProgress.CurrentValue < achievementProgress.TargetValue {
			achievementProgress.CurrentValue = achievementProgress.TargetValue
		}
		progress.AchievementProgress[achievementID] = achievementProgress
	}
	progress.UnlockedTitleIDs = NormalizeStringIDs(progress.UnlockedTitleIDs)
	if progress.SelectedTitleByCharacterID == nil {
		progress.SelectedTitleByCharacterID = map[string]string{}
	}
	for characterID, titleID := range progress.SelectedTitleByCharacterID {
		if characterID == "" || titleID == "" {
			delete(progress.SelectedTitleByCharacterID, characterID)
		}
	}
	if progress.CollectionUnlocks == nil {
		progress.CollectionUnlocks = map[string]CollectionUnlock{}
	}
	for unlockID, unlock := range progress.CollectionUnlocks {
		if unlock.UnlockID == "" {
			unlock.UnlockID = unlockID
		}
		if unlock.UnlockID == "" || unlock.Category == "" {
			delete(progress.CollectionUnlocks, unlockID)
			continue
		}
		progress.CollectionUnlocks[unlockID] = unlock
	}
	return progress
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

func NormalizeHousingStorageSlots(source []HousingStorageSlot) []HousingStorageSlot {
	slots := make([]HousingStorageSlot, HousingStorageSlotCount)
	for slotIndex := range slots {
		slots[slotIndex].SlotIndex = slotIndex
	}

	nextEmptySlot := 0
	for _, slot := range source {
		if slot.StackCount < 0 {
			slot.StackCount = 0
		}

		targetIndex := slot.SlotIndex
		if targetIndex < 0 || targetIndex >= HousingStorageSlotCount {
			for nextEmptySlot < HousingStorageSlotCount && slots[nextEmptySlot].ItemID != "" {
				nextEmptySlot++
			}
			if nextEmptySlot >= HousingStorageSlotCount {
				continue
			}
			targetIndex = nextEmptySlot
		}

		normalizedSlot := HousingStorageSlot{
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

func NormalizeDecorationPlacements(source []DecorationPlacement) []DecorationPlacement {
	if len(source) == 0 {
		return []DecorationPlacement{}
	}

	seen := map[string]struct{}{}
	placements := make([]DecorationPlacement, 0, len(source))
	for _, placement := range source {
		if placement.PlacementID == "" || placement.HousingSpaceID == "" || placement.DecorationID == "" {
			continue
		}
		if _, exists := seen[placement.PlacementID]; exists {
			continue
		}
		seen[placement.PlacementID] = struct{}{}
		placements = append(placements, placement)
	}

	return placements
}
