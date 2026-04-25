package worlds

import (
	"math"
	"sync"
	"time"

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
	trainerNPCKind           = "trainer_npc"
	professionTrainerNPCKind = "profession_trainer_npc"
	questGiverNPCKind        = "quest_giver_npc"
	worldObjectNPCKind       = "quest_object"

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
	nextZoneID          = secondZoneID
	worldTickMaxSeconds = 0.25

	questStateNotStarted    = "not_started"
	questStateActive        = "active"
	questStateCompleted     = "completed"
	questStateRewardGranted = "reward_granted"

	objectiveTalk    = "talk"
	objectiveKill    = "kill_hostile_mob"
	objectiveCollect = "collect"
	objectiveTrainer = "trainer"
	objectiveExplore = "explore"
	objectiveUse     = "use_location"

	starterQuestID        = "sv_first_muster"
	legacyEmberQuestID    = "defeat_ember_hounds_01"
	starterSpawnX         = 10.0
	starterSpawnY         = 10.0
	starterSpawnZ         = 0.0
	starterInteractRadius = 5.0
	starterZoneMaxX       = 480.0
	starterZoneMaxY       = 300.0
	secondZoneEntryX      = 34.0
	secondZoneEntryY      = 148.0
	secondZoneMaxX        = 720.0
	secondZoneMaxY        = 420.0
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

type moveRequest struct {
	WorldSessionToken string  `json:"worldSessionToken"`
	DeltaX            float64 `json:"deltaX"`
	DeltaY            float64 `json:"deltaY"`
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

type questTrackRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	QuestID           string `json:"questId"`
	Tracked           bool   `json:"tracked"`
}

type npcService struct {
	Type      string `json:"type"`
	ServiceID string `json:"serviceId"`
	Label     string `json:"label"`
}

type sessionEntity struct {
	ID          string       `json:"id"`
	DisplayName string       `json:"displayName"`
	Kind        string       `json:"kind"`
	MobTypeID   string       `json:"mobTypeId,omitempty"`
	X           float64      `json:"x"`
	Y           float64      `json:"y"`
	Z           float64      `json:"z"`
	Health      float64      `json:"health"`
	MaxHealth   float64      `json:"maxHealth"`
	Alive       bool         `json:"alive"`
	Targetable  bool         `json:"targetable"`
	AIState     string       `json:"aiState,omitempty"`
	Services    []npcService `json:"npcServices,omitempty"`
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

type questDefinition struct {
	ID              string
	ZoneID          string
	Title           string
	ObjectiveType   string
	ObjectiveText   string
	GiverNPCID      string
	TurnInNPCID     string
	TargetEntityID  string
	TargetMobType   string
	TargetItemID    string
	TargetItemName  string
	TargetCount     int
	RewardXP        int
	RewardCopper    int
	RewardItems     []itemRewardDefinition
	PrerequisiteIDs []string
	LevelBand       string
	MarkerX         float64
	MarkerY         float64
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

type currencyBreakdown struct {
	Gold   int `json:"gold"`
	Silver int `json:"silver"`
	Copper int `json:"copper"`
}

type inventoryResponse struct {
	SlotCount int                               `json:"slotCount"`
	Slots     []platform.CharacterInventorySlot `json:"slots"`
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

type worldSessionState struct {
	Token              string
	AccountID          string
	CharacterID        string
	DisplayName        string
	ClassID            string
	Level              int
	RealmID            string
	ZoneID             string
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
}

type mobState struct {
	ID                     string
	MobTypeID              string
	DisplayName            string
	Kind                   string
	ZoneID                 string
	Level                  int
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
	Alive                  bool
	Targetable             bool
	AIState                string
	CurrentTargetCharacter string
	LastAttackAtMs         int64
	RespawnAtMs            int64
}

type mobSpawnDefinition struct {
	ID              string
	ZoneID          string
	MobTypeID       string
	DisplayName     string
	Level           int
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
}

type worldServer struct {
	store              *store.FileStore
	metrics            *worldMetrics
	mutex              sync.Mutex
	sessionsByToken    map[string]*worldSessionState
	sessionTokenByChar map[string]string
	mobs               map[string]*mobState
	mobOrder           []string
	quests             map[string]questDefinition
	questOrder         []string
	quest              questDefinition
	friendlyNPCs       map[string]friendlyNPCDefinition
	friendlyNPCOrder   []string
	zones              map[string]zoneDefinition
	chatMessages       []chatEnvelope
	chatSequence       int64
	partyInvites       map[string]partyInviteState
	partyInviteCounter int64
	lastUpdatedAt      time.Time
}

func newWorldServer(fileStore *store.FileStore) *worldServer {
	server := &worldServer{
		store:              fileStore,
		metrics:            newWorldMetrics(),
		sessionsByToken:    map[string]*worldSessionState{},
		sessionTokenByChar: map[string]string{},
		mobs:               map[string]*mobState{},
		quests:             map[string]questDefinition{},
		friendlyNPCs:       map[string]friendlyNPCDefinition{},
		zones:              map[string]zoneDefinition{},
		partyInvites:       map[string]partyInviteState{},
	}
	server.loadStarterContentLocked()
	server.ensureMobsLocked()
	return server
}

func (s *worldServer) loadStarterContentLocked() {
	for _, zone := range zoneDefinitions {
		s.zones[zone.ID] = zone
	}

	allQuests := append([]questDefinition{}, stonewakeQuestDefinitions...)
	allQuests = append(allQuests, brindlebrookQuestDefinitions...)
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

	allFriendlyNPCs := append([]friendlyNPCDefinition{}, stonewakeFriendlyNPCs...)
	allFriendlyNPCs = append(allFriendlyNPCs, brindlebrookFriendlyNPCs...)
	s.friendlyNPCOrder = make([]string, 0, len(allFriendlyNPCs))
	for _, npc := range allFriendlyNPCs {
		if npc.ZoneID == "" {
			npc.ZoneID = defaultZoneID
		}
		if npc.Radius <= 0 {
			npc.Radius = starterInteractRadius
		}
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
	s.mobOrder = make([]string, 0, len(allMobSpawns))
	for _, spawn := range allMobSpawns {
		zoneID := spawn.ZoneID
		if zoneID == "" {
			zoneID = defaultZoneID
		}
		s.mobOrder = append(s.mobOrder, spawn.ID)
		s.mobs[spawn.ID] = &mobState{
			ID:              spawn.ID,
			MobTypeID:       spawn.MobTypeID,
			DisplayName:     spawn.DisplayName,
			Kind:            hostileMobKind,
			ZoneID:          zoneID,
			Level:           spawn.Level,
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
			Alive:           true,
			Targetable:      true,
			AIState:         mobAIStateIdle,
		}
	}
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
