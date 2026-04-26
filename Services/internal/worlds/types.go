package worlds

import (
	"math"
	"sync"
	"time"

	contentpkg "amandacore/services/internal/content"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

const (
	playerMaxHealth           = 88.0
	playerMaxResource         = 100.0
	playerResourceRegenPerSec = 0.0
	playerTargetRange         = 28.0
	playerAutoAttackRange     = 5.5
	playerAutoAttackDamage    = 8.0
	playerAutoAttackGrit      = 12.0
	playerAutoAttackCadenceMs = int64(1800)
	playerGlobalCooldownMs    = int64(1500)

	hostileMobKind           = "hostile_mob"
	gatheringNodeKind        = "gathering_node"
	trainerNPCKind           = "trainer_npc"
	professionTrainerNPCKind = "profession_trainer_npc"
	questGiverNPCKind        = "quest_giver_npc"
	worldObjectNPCKind       = "quest_object"
	dungeonEntranceKind      = "dungeon_entrance"
	dungeonExitKind          = "dungeon_exit"
	housingEntranceKind      = "housing_entrance"
	housingExitKind          = "housing_exit"
	housingStorageKind       = "housing_storage"
	housingDecorationKind    = "housing_decoration"

	mobAIStateIdle       = "idle"
	mobAIStatePatrolling = "patrolling"
	mobAIStateAlerted    = "alerted"
	mobAIStateChasing    = "chasing"
	mobAIStateAttacking  = "attacking"
	mobAIStateEvading    = "evading"
	mobAIStateReturning  = "returning"
	mobAIStateDead       = "dead"
	mobAIStateRespawning = "respawning"

	defaultZoneID       = "stonewake_vale"
	secondZoneID        = "brindlebrook_roadlands"
	dungeonZoneID       = "dun_tallowdeep_sluice"
	housingZoneID       = "house_personal_room"
	nextZoneID          = secondZoneID
	worldTickMaxSeconds = 0.25

	questStateNotStarted    = "not_started"
	questStateActive        = "active"
	questStateReady         = "ready_to_complete"
	questStateCompleted     = "completed"
	questStateRewardGranted = "reward_granted"

	objectiveTalk    = "talk"
	objectiveKill    = "kill_hostile_mob"
	objectiveCollect = "collect"
	objectiveTrainer = "trainer"
	objectiveExplore = "explore"
	objectiveUse     = "use_location"

	objectiveKindKillNPC               = "KillNpc"
	objectiveKindCollectItem           = "CollectItem"
	objectiveKindInteractWithEntity    = "InteractWithEntity"
	objectiveKindReachLocation         = "ReachLocationPlaceholder"
	objectiveKindUseAbilityPlaceholder = "UseAbilityPlaceholder"

	starterQuestID        = "sv_first_muster"
	legacyEmberQuestID    = "defeat_ember_hounds_01"
	starterSpawnX         = 10.0
	starterSpawnY         = 10.0
	starterSpawnZ         = 0.0
	playableGroundZ       = 0.05
	starterInteractRadius = 5.0
	starterZoneMaxX       = 480.0
	starterZoneMaxY       = 300.0
	secondZoneEntryX      = 34.0
	secondZoneEntryY      = 148.0
	secondZoneMaxX        = 720.0
	secondZoneMaxY        = 420.0
)

type NpcDisposition string

const (
	NpcDispositionNeutral  NpcDisposition = "Neutral"
	NpcDispositionHostile  NpcDisposition = "Hostile"
	NpcDispositionFriendly NpcDisposition = "Friendly"
)

type joinTicketRequest struct {
	RealmID     string `json:"realmId"`
	CharacterID string `json:"characterId"`
}

type connectRequest struct {
	TicketID string `json:"ticketId"`
}

type reconnectRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
}

type disconnectRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
}

type bindSetRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	BindLocationID    string `json:"bindLocationId"`
}

type recallUseRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
}

type travelDiscoverRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	TravelPointID     string `json:"travelPointId"`
}

type travelRouteRequest struct {
	WorldSessionToken  string `json:"worldSessionToken"`
	SourcePointID      string `json:"sourcePointId"`
	DestinationPointID string `json:"destinationPointId"`
}

type mountUnlockRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	MountID           string `json:"mountId"`
}

type mountSelectRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	MountID           string `json:"mountId"`
}

type mountSummonRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	MountID           string `json:"mountId,omitempty"`
}

type mountDismissRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
}

type moveRequest struct {
	WorldSessionToken string  `json:"worldSessionToken"`
	DeltaX            float64 `json:"deltaX"`
	DeltaY            float64 `json:"deltaY"`
}

type dungeonEnterRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	DungeonID         string `json:"dungeonId"`
}

type dungeonExitRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
}

type dungeonResetRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	DungeonID         string `json:"dungeonId"`
}

type housingEnterRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
}

type housingLeaveRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
}

type housingStorageDepositRequest struct {
	WorldSessionToken  string `json:"worldSessionToken"`
	InventorySlotIndex int    `json:"inventorySlotIndex"`
	StorageSlotIndex   *int   `json:"storageSlotIndex,omitempty"`
	StackCount         int    `json:"stackCount"`
}

type housingStorageWithdrawRequest struct {
	WorldSessionToken  string `json:"worldSessionToken"`
	StorageSlotIndex   int    `json:"storageSlotIndex"`
	InventorySlotIndex *int   `json:"inventorySlotIndex,omitempty"`
	StackCount         int    `json:"stackCount"`
}

type housingStorageMoveRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	FromSlotIndex     int    `json:"fromSlotIndex"`
	ToSlotIndex       int    `json:"toSlotIndex"`
}

type decorationPlaceRequest struct {
	WorldSessionToken string  `json:"worldSessionToken"`
	DecorationID      string  `json:"decorationId"`
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	Z                 float64 `json:"z"`
	RotationYaw       float64 `json:"rotationYaw"`
}

type decorationRemoveRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	PlacementID       string `json:"placementId"`
}

type targetRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	TargetID          string `json:"targetId"`
}

type autoAttackRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	Enabled           bool   `json:"enabled"`
}

type abilityRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	AbilityID         string `json:"abilityId"`
}

type duelRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	TargetCharacterID string `json:"targetCharacterId"`
	TargetName        string `json:"targetName"`
}

type duelActionRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	DuelID            string `json:"duelId"`
}

type trainerLearnRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	TrainerID         string `json:"trainerId"`
	AbilityID         string `json:"abilityId"`
}

type talentSelectRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	TalentID          string `json:"talentId"`
}

type professionLearnRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	TrainerID         string `json:"trainerId"`
	ProfessionID      string `json:"professionId"`
}

type gatherRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	NodeID            string `json:"nodeId"`
}

type craftRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	RecipeID          string `json:"recipeId"`
}

type actionBarAssignRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	SlotIndex         int    `json:"slotIndex"`
	AbilityID         string `json:"abilityId"`
}

type actionBarMoveRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	FromSlotIndex     int    `json:"fromSlotIndex"`
	ToSlotIndex       int    `json:"toSlotIndex"`
}

type actionBarClearRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	SlotIndex         int    `json:"slotIndex"`
}

type inventoryMoveRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	FromSlotIndex     int    `json:"fromSlotIndex"`
	ToSlotIndex       int    `json:"toSlotIndex"`
}

type inventoryEquipRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	SlotIndex         int    `json:"slotIndex"`
}

type lootInspectRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	LootContainerID   string `json:"lootContainerId"`
}

type lootClaimRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	LootContainerID   string `json:"lootContainerId"`
}

type vendorBuyRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	VendorID          string `json:"vendorId"`
	ItemID            string `json:"itemId"`
	StackCount        int    `json:"stackCount"`
}

type vendorSellRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	VendorID          string `json:"vendorId"`
	SlotIndex         int    `json:"slotIndex"`
	StackCount        int    `json:"stackCount"`
}

type questAcceptRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	QuestID           string `json:"questId"`
}

type questCompleteRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	QuestID           string `json:"questId"`
}

type questTrackRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	QuestID           string `json:"questId"`
	Tracked           bool   `json:"tracked"`
}

type titleSelectRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	TitleID           string `json:"titleId"`
}

type titleClearRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
}

type npcService struct {
	Type      string `json:"type"`
	ServiceID string `json:"serviceId"`
	Label     string `json:"label"`
}

type sessionEntity struct {
	ID               string       `json:"id"`
	ArchetypeID      string       `json:"archetypeId,omitempty"`
	SpawnPointID     string       `json:"spawnPointId,omitempty"`
	DisplayName      string       `json:"displayName"`
	Kind             string       `json:"kind"`
	MobTypeID        string       `json:"mobTypeId,omitempty"`
	Disposition      string       `json:"disposition,omitempty"`
	Classification   string       `json:"classification,omitempty"`
	Elite            bool         `json:"elite,omitempty"`
	GatherNodeTypeID string       `json:"gatherNodeTypeId,omitempty"`
	ProfessionID     string       `json:"professionId,omitempty"`
	RequiredSkill    int          `json:"requiredSkill,omitempty"`
	Ready            bool         `json:"ready,omitempty"`
	ReadyAt          int64        `json:"readyAt,omitempty"`
	InteractionLabel string       `json:"interactionLabel,omitempty"`
	X                float64      `json:"x"`
	Y                float64      `json:"y"`
	Z                float64      `json:"z"`
	Health           float64      `json:"health"`
	MaxHealth        float64      `json:"maxHealth"`
	Alive            bool         `json:"alive"`
	Targetable       bool         `json:"targetable"`
	IsInCombat       bool         `json:"isInCombat,omitempty"`
	CurrentTargetID  string       `json:"currentTargetEntityId,omitempty"`
	LastDamagedByID  string       `json:"lastDamagedByEntityId,omitempty"`
	RespawnDelayMs   int64        `json:"respawnDelayMs,omitempty"`
	DeathTick        int64        `json:"deathTick,omitempty"`
	RespawnTick      int64        `json:"respawnTick,omitempty"`
	AIState          string       `json:"aiState,omitempty"`
	PvPState         string       `json:"pvpState,omitempty"`
	DuelOpponent     bool         `json:"duelOpponent,omitempty"`
	Services         []npcService `json:"npcServices,omitempty"`
}

type itemRewardDefinition struct {
	ItemID      string
	DisplayName string
	StackCount  int
}

type itemRewardResponse struct {
	ItemID      string `json:"itemId"`
	DisplayName string `json:"displayName"`
	StackCount  int    `json:"stackCount"`
}

type questObjectiveGraph struct {
	Nodes []questObjectiveNode
}

type questObjectiveNode struct {
	NodeID             string
	Kind               string
	TargetNpcArchetype string
	TargetEntityID     string
	TargetItemID       string
	TargetCount        int
	DependsOn          []string
	Terminal           bool
}

type questDefinition struct {
	ID                 string
	ZoneID             string
	Title              string
	Summary            string
	RequiredLevel      int
	ObjectiveType      string
	ObjectiveText      string
	GiverNPCID         string
	TurnInNPCID        string
	TargetEntityID     string
	TargetMobType      string
	TargetItemID       string
	TargetItemName     string
	TargetCount        int
	RewardXP           int
	RewardCopper       int
	RewardItems        []itemRewardDefinition
	PrerequisiteIDs    []string
	LevelBand          string
	MarkerX            float64
	MarkerY            float64
	PartyShareable     bool
	GroupRecommended   bool
	RecommendedPlayers int
	PartyCreditRadius  float64
	ObjectiveGraph     questObjectiveGraph
	AllowDirectAccept  bool
	Tags               []string
}

type navigationAreaDefinition struct {
	ID             string
	DisplayName    string
	Kind           string
	CenterX        float64
	CenterY        float64
	Radius         float64
	RouteHintText  string
	QuestIDs       []string
	TargetMobType  string
	TargetEntityID string
}

type mapRoadDefinition struct {
	ID          string
	DisplayName string
	Points      []mapPointDefinition
}

type mapPointDefinition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type mapLandmarkDefinition struct {
	ID          string
	DisplayName string
	Kind        string
	X           float64
	Y           float64
}

type zoneMapDefinition struct {
	ZoneID      string
	DisplayName string
	MinX        float64
	MinY        float64
	MaxX        float64
	MaxY        float64
	Roads       []mapRoadDefinition
	Landmarks   []mapLandmarkDefinition
}

type friendlyNPCDefinition struct {
	ID          string
	ZoneID      string
	DisplayName string
	Kind        string
	X           float64
	Y           float64
	Z           float64
	AIState     string
	Radius      float64
	Services    []npcService
}

type gatheringLootDefinition struct {
	ItemID     string
	MinCount   int
	MaxCount   int
	Guaranteed bool
}

type gatheringNodeDefinition struct {
	ID                   string
	NodeTypeID           string
	DisplayName          string
	ZoneID               string
	X                    float64
	Y                    float64
	Z                    float64
	Radius               float64
	RequiredProfessionID string
	RequiredSkill        int
	Loot                 []gatheringLootDefinition
	RespawnDelayMs       int64
	InteractionLabel     string
}

type gatheringNodeState struct {
	Definition gatheringNodeDefinition
	ReadyAtMs  int64
}

type currencyBreakdown struct {
	Gold   int `json:"gold"`
	Silver int `json:"silver"`
	Copper int `json:"copper"`
}

type inventoryResponse struct {
	SlotCount int                     `json:"slotCount"`
	Slots     []inventorySlotResponse `json:"slots"`
}

type inventorySlotResponse struct {
	SlotIndex   int    `json:"slotIndex"`
	ItemID      string `json:"itemId"`
	DisplayName string `json:"displayName"`
	StackCount  int    `json:"stackCount"`
	ItemType    string `json:"itemType,omitempty"`
	ItemSubtype string `json:"itemSubtype,omitempty"`
	Quality     string `json:"quality,omitempty"`
	IconKind    string `json:"iconKind,omitempty"`
}

type equipmentResponse struct {
	Slots []platform.CharacterEquipmentSlot `json:"slots"`
}

type statBlockResponse struct {
	Strength          int     `json:"strength"`
	Stamina           int     `json:"stamina"`
	Armor             int     `json:"armor"`
	AttackPower       float64 `json:"attackPower"`
	ArmorReductionPct float64 `json:"armorReductionPct"`
}

type zoneBoundsDefinition struct {
	MinX float64 `json:"minX"`
	MinY float64 `json:"minY"`
	MaxX float64 `json:"maxX"`
	MaxY float64 `json:"maxY"`
}

type zonePointDefinition struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	Type        string  `json:"type"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
}

type zoneRoadDefinition struct {
	ID          string                `json:"id"`
	DisplayName string                `json:"displayName"`
	Points      []zonePointDefinition `json:"points"`
}

type zoneDefinition struct {
	ID          string                `json:"id"`
	DisplayName string                `json:"displayName"`
	LevelBand   string                `json:"levelBand"`
	Bounds      zoneBoundsDefinition  `json:"bounds"`
	Roads       []zoneRoadDefinition  `json:"roads"`
	Landmarks   []zonePointDefinition `json:"landmarks"`
	Transitions []zonePointDefinition `json:"transitions"`
}

type achievementNotification struct {
	AchievementID string `json:"achievementId"`
	DisplayName   string `json:"displayName"`
	CompletedAt   int64  `json:"completedAt"`
}

type worldSessionState struct {
	Token              string
	AccountID          string
	CharacterID        string
	DisplayName        string
	ClassID            string
	Level              int
	RealmID            string
	ZoneID             string
	InstanceID         string
	HousingSpaceID     string
	HousingInstanceID  string
	ReturnZoneID       string
	ReturnX            float64
	ReturnY            float64
	ReturnZ            float64
	X                  float64
	Y                  float64
	Z                  float64
	Connected          bool
	LastSeenAt         int64
	Health             float64
	MaxHealth          float64
	Resource           float64
	MaxResource        float64
	Alive              bool
	CurrentTargetID    string
	AutoAttackActive   bool
	LastAutoAttackAtMs int64
	GlobalCooldownEnds int64
	CastEndsAtMs       int64
	CastingAbilityID   string
	CastingTargetID    string
	AbilityCooldowns   map[string]int64
	Experience         int
	CurrencyCopper     int
	Inventory          []platform.CharacterInventorySlot
	Equipment          []platform.CharacterEquipmentSlot
	Professions        []platform.CharacterProfessionState
	Talents            map[string]int
	LearnedAbilityIDs  []string
	ActionBarSlots     []platform.CharacterActionBarSlot
	QuestProgress      map[string]platform.CharacterQuestProgress
	TrackedQuestIDs    []string
	PvPStats           platform.CharacterPvPStats
	LastDuelResult     *duelResultState
	AccountProgress    platform.AccountProgressState
	BindPoint          platform.CharacterBindPoint
	TravelState        platform.CharacterTravelState
	MountState         platform.CharacterMountState
	CurrentlyTraveling bool
}

type mobState struct {
	ID                     string
	InstanceID             string
	SpawnPointID           string
	MobTypeID              string
	ArchetypeID            string
	DisplayName            string
	Kind                   string
	ZoneID                 string
	Level                  int
	Disposition            string
	LootTableID            string
	X                      float64
	Y                      float64
	Z                      float64
	SpawnX                 float64
	SpawnY                 float64
	SpawnZ                 float64
	Health                 float64
	MaxHealth              float64
	AggroRadius            float64
	AttackRange            float64
	AttackDamage           float64
	AttackCadenceMs        int64
	MoveSpeedPerSec        float64
	LeashRadius            float64
	RespawnDelayMs         int64
	Classification         string
	Elite                  bool
	Alive                  bool
	Targetable             bool
	AIState                string
	CurrentTargetCharacter string
	LastDamagedByCharacter string
	LastAttackAtMs         int64
	DeathTick              int64
	RespawnAtMs            int64
	RespawnTick            int64
}

type mobSpawnDefinition struct {
	ID              string
	SpawnPointID    string
	ZoneID          string
	MobTypeID       string
	ArchetypeID     string
	DisplayName     string
	Level           int
	LootTableID     string
	X               float64
	Y               float64
	Z               float64
	MaxHealth       float64
	AggroRadius     float64
	AttackRange     float64
	AttackDamage    float64
	AttackCadenceMs int64
	MoveSpeedPerSec float64
	LeashRadius     float64
	RespawnDelayMs  int64
	Disposition     string
	Classification  string
	Elite           bool
}

type worldPosition struct {
	ZoneID string
	X      float64
	Y      float64
	Z      float64
}

type dungeonDefinition struct {
	ID               string
	DisplayName      string
	LevelBand        string
	InstanceZoneID   string
	EntranceZoneID   string
	EntranceEntityID string
	ExitEntityID     string
	StartPositions   []worldPosition
	ExitPosition     worldPosition
	ReturnPosition   worldPosition
	MobSpawns        []mobSpawnDefinition
	BossMobTypeID    string
	QuestID          string
	EmptyExpiryMs    int64
	HardExpiryMs     int64
}

type dungeonObjectiveState struct {
	BossDefeated bool  `json:"bossDefeated"`
	UpdatedAtMs  int64 `json:"updatedAt"`
}

type dungeonInstanceState struct {
	InstanceID         string
	DungeonID          string
	PartyID            string
	ZoneID             string
	CreatedAtMs        int64
	ExpiresAtMs        int64
	State              string
	MemberCharacterIDs []string
	PlayersInside      map[string]bool
	ReturnPositions    map[string]worldPosition
	Mobs               map[string]*mobState
	MobOrder           []string
	Objective          dungeonObjectiveState
	BossRewardGranted  map[string]bool
	LastPlayerLeftAtMs int64
}

type KillCreditReason string

const (
	KillCreditReasonKillingBlow KillCreditReason = "killing_blow"
)

type KillCredit struct {
	CharacterID    string           `json:"characterId"`
	SourceEntityID string           `json:"sourceEntityId"`
	NpcArchetypeID string           `json:"npcArchetypeId"`
	ZoneID         string           `json:"zoneId"`
	InstanceID     string           `json:"instanceId,omitempty"`
	TickMs         int64            `json:"tickMs"`
	Reason         KillCreditReason `json:"reason"`
}

type KillCreditLedger struct {
	CreditsByCharacter map[string]map[string]int
	Entries            []KillCredit
}

type worldServer struct {
	store                  *store.FileStore
	metrics                *worldMetrics
	mutex                  sync.Mutex
	sessionsByToken        map[string]*worldSessionState
	sessionTokenByChar     map[string]string
	mobs                   map[string]*mobState
	mobOrder               []string
	duels                  map[string]*duelState
	duelByCharacter        map[string]string
	duelCounter            int64
	dungeonInstances       map[string]*dungeonInstanceState
	instanceByParty        map[string]string
	instanceCounter        int64
	housingInstanceCounter int64
	quests                 map[string]questDefinition
	questOrder             []string
	quest                  questDefinition
	friendlyNPCs           map[string]friendlyNPCDefinition
	friendlyNPCOrder       []string
	gatheringNodes         map[string]*gatheringNodeState
	gatheringNodeOrder     []string
	lootContainers         map[string]*lootContainerState
	lootContainerOrder     []string
	lootContainerCounter   int64
	domainEvents           []DomainEvent
	stateDiffs             []StateDiff
	killCreditLedger       KillCreditLedger
	zones                  map[string]zoneDefinition
	contentRegistry        *contentpkg.RuntimeContentRegistry
	zoneRuntimes           map[string]*ZoneRuntime
	contentMobSpawns       []mobSpawnDefinition
	contentActivation      ContentActivationResult
	chatMessages           []chatEnvelope
	chatSequence           int64
	partyInvites           map[string]partyInviteState
	partyInviteCounter     int64
	lastUpdatedAt          time.Time
}

func newWorldServer(fileStore *store.FileStore) *worldServer {
	return newWorldServerWithContentPackage(fileStore, "")
}

func newWorldServerWithContentPackage(fileStore *store.FileStore, contentPackagePath string) *worldServer {
	server := &worldServer{
		store:              fileStore,
		metrics:            newWorldMetrics(),
		sessionsByToken:    map[string]*worldSessionState{},
		sessionTokenByChar: map[string]string{},
		mobs:               map[string]*mobState{},
		duels:              map[string]*duelState{},
		duelByCharacter:    map[string]string{},
		dungeonInstances:   map[string]*dungeonInstanceState{},
		instanceByParty:    map[string]string{},
		quests:             map[string]questDefinition{},
		friendlyNPCs:       map[string]friendlyNPCDefinition{},
		gatheringNodes:     map[string]*gatheringNodeState{},
		lootContainers:     map[string]*lootContainerState{},
		killCreditLedger: KillCreditLedger{
			CreditsByCharacter: map[string]map[string]int{},
		},
		zones:        map[string]zoneDefinition{},
		zoneRuntimes: map[string]*ZoneRuntime{},
		partyInvites: map[string]partyInviteState{},
	}
	server.loadStarterContentLocked()
	server.loadConfiguredContentPackageLocked(contentPackagePath)
	server.ensureMobsLocked()
	server.ensureGatheringNodesLocked()
	server.emitWorldEventLocked(eventItemCatalogLoaded, map[string]any{"itemCount": len(itemDefinitions)})
	server.emitWorldEventLocked(eventLootTableLoaded, map[string]any{"lootTableCount": len(devLootTables)})
	return server
}

func (s *worldServer) loadStarterContentLocked() {
	allZones := append([]zoneDefinition{}, zoneDefinitions...)
	allZones = append(allZones, dungeonZoneDefinitions...)
	allZones = append(allZones, housingZoneDefinitions...)
	for _, zone := range allZones {
		s.zones[zone.ID] = zone
	}

	allQuests := append([]questDefinition{}, stonewakeQuestDefinitions...)
	allQuests = append(allQuests, brindlebrookQuestDefinitions...)
	allQuests = append(allQuests, dungeonQuestDefinitions...)
	allQuests = append(allQuests, devProgressionQuestDefinitions...)
	s.questOrder = make([]string, 0, len(allQuests))
	for _, quest := range allQuests {
		if quest.ZoneID == "" {
			quest.ZoneID = defaultZoneID
		}
		if quest.TargetCount <= 0 {
			quest.TargetCount = 1
		}
		s.quests[quest.ID] = quest
		s.questOrder = append(s.questOrder, quest.ID)
	}
	if quest, ok := s.quests[starterQuestID]; ok {
		s.quest = quest
	}
	s.emitWorldEventLocked(eventQuestCatalogLoaded, map[string]any{"questCount": len(s.quests)})

	allFriendlyNPCs := append([]friendlyNPCDefinition{}, stonewakeFriendlyNPCs...)
	allFriendlyNPCs = append(allFriendlyNPCs, brindlebrookFriendlyNPCs...)
	allFriendlyNPCs = append(allFriendlyNPCs, dungeonFriendlyNPCs...)
	allFriendlyNPCs = append(allFriendlyNPCs, housingFriendlyNPCs...)
	s.friendlyNPCOrder = make([]string, 0, len(allFriendlyNPCs))
	for _, npc := range allFriendlyNPCs {
		if npc.ZoneID == "" {
			npc.ZoneID = defaultZoneID
		}
		if npc.Radius <= 0 {
			npc.Radius = starterInteractRadius
		}
		npc.Z = clampSpawnGroundZ(npc.Z)
		s.friendlyNPCs[npc.ID] = npc
		s.friendlyNPCOrder = append(s.friendlyNPCOrder, npc.ID)
	}
}

func (s *worldServer) ensureMobsLocked() {
	if len(s.mobs) != 0 {
		return
	}

	allMobSpawns := append([]mobSpawnDefinition{}, stonewakeMobSpawns...)
	allMobSpawns = append(allMobSpawns, brindlebrookMobSpawns...)
	allMobSpawns = append(allMobSpawns, devProgressionMobSpawns...)
	allMobSpawns = append(allMobSpawns, s.contentMobSpawns...)
	s.mobOrder = make([]string, 0, len(allMobSpawns))
	for _, spawn := range allMobSpawns {
		zoneID := spawn.ZoneID
		if zoneID == "" {
			zoneID = defaultZoneID
		}
		archetypeID := spawn.ArchetypeID
		if archetypeID == "" {
			archetypeID = spawn.MobTypeID
		}
		spawnPointID := spawn.SpawnPointID
		if spawnPointID == "" {
			spawnPointID = spawn.ID
		}
		disposition := spawn.Disposition
		if disposition == "" {
			disposition = string(NpcDispositionHostile)
		}
		spawn.Z = clampSpawnGroundZ(spawn.Z)
		s.mobOrder = append(s.mobOrder, spawn.ID)
		mob := &mobState{
			ID:              spawn.ID,
			InstanceID:      "",
			SpawnPointID:    spawnPointID,
			MobTypeID:       spawn.MobTypeID,
			ArchetypeID:     archetypeID,
			DisplayName:     spawn.DisplayName,
			Kind:            hostileMobKind,
			ZoneID:          zoneID,
			Level:           spawn.Level,
			Disposition:     disposition,
			LootTableID:     spawn.LootTableID,
			X:               spawn.X,
			Y:               spawn.Y,
			Z:               spawn.Z,
			SpawnX:          spawn.X,
			SpawnY:          spawn.Y,
			SpawnZ:          spawn.Z,
			Health:          spawn.MaxHealth,
			MaxHealth:       spawn.MaxHealth,
			AggroRadius:     spawn.AggroRadius,
			AttackRange:     spawn.AttackRange,
			AttackDamage:    spawn.AttackDamage,
			AttackCadenceMs: spawn.AttackCadenceMs,
			MoveSpeedPerSec: spawn.MoveSpeedPerSec,
			LeashRadius:     spawn.LeashRadius,
			RespawnDelayMs:  spawn.RespawnDelayMs,
			Classification:  spawn.Classification,
			Elite:           spawn.Elite,
			Alive:           true,
			Targetable:      true,
			AIState:         mobAIStateIdle,
		}
		s.mobs[spawn.ID] = mob
		s.emitWorldEventLocked(EventNPCSpawnPointLoaded, map[string]any{
			"spawnPointId": spawnPointID,
			"entityId":     spawn.ID,
			"archetypeId":  archetypeID,
			"zoneId":       zoneID,
		})
		s.emitWorldEventLocked(EventNPCSpawned, map[string]any{
			"entityId":     mob.ID,
			"archetypeId":  mob.ArchetypeID,
			"spawnPointId": mob.SpawnPointID,
			"zoneId":       mob.ZoneID,
		}, entitySpawnDelta(mob))
		s.emitWorldEventLocked(EventWorldEntitySpawned, map[string]any{
			"entityId": mob.ID,
			"kind":     mob.Kind,
			"zoneId":   mob.ZoneID,
		})
	}
}

func (s *worldServer) ensureGatheringNodesLocked() {
	if len(s.gatheringNodes) != 0 {
		return
	}

	allGatheringNodes := append([]gatheringNodeDefinition{}, stonewakeGatheringNodeDefinitions...)
	s.gatheringNodeOrder = make([]string, 0, len(allGatheringNodes))
	for _, node := range allGatheringNodes {
		if node.ZoneID == "" {
			node.ZoneID = defaultZoneID
		}
		if node.Radius <= 0 {
			node.Radius = starterInteractRadius
		}
		if node.RespawnDelayMs <= 0 {
			node.RespawnDelayMs = 1000
		}
		node.Z = clampSpawnGroundZ(node.Z)
		s.gatheringNodeOrder = append(s.gatheringNodeOrder, node.ID)
		s.gatheringNodes[node.ID] = &gatheringNodeState{Definition: node}
	}
}

func clampSpawnGroundZ(z float64) float64 {
	if z < playableGroundZ {
		return playableGroundZ
	}
	return z
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

func clampSeconds(delta float64) float64 {
	if delta < 0 {
		return 0
	}
	return math.Min(delta, worldTickMaxSeconds)
}
