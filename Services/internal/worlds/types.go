package worlds

import (
	"math"
	"sync"
	"time"

	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

const (
	playerMaxHealth           = 100.0
	playerMaxResource         = 100.0
	playerResourceRegenPerSec = 12.0
	playerTargetRange         = 28.0
	playerAutoAttackRange     = 5.5
	playerAutoAttackDamage    = 10.0
	playerAutoAttackCadenceMs = int64(1800)
	playerGlobalCooldownMs    = int64(1500)

	hostileMobKind     = "hostile_mob"
	trainerNPCKind     = "trainer_npc"
	questGiverNPCKind  = "quest_giver_npc"
	worldObjectNPCKind = "quest_object"

	defaultZoneID       = "stonewake_vale"
	nextZoneID          = "west_approach"
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

type questAcceptRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	QuestID           string `json:"questId"`
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

type questDefinition struct {
	ID              string
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

type friendlyNPCDefinition struct {
	ID          string
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
	Experience         int
	CurrencyCopper     int
	Inventory          []platform.CharacterInventorySlot
	LearnedAbilityIDs  []string
	ActionBarSlots     []platform.CharacterActionBarSlot
	QuestProgress      map[string]platform.CharacterQuestProgress
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
	lastUpdatedAt      time.Time
}

func newWorldServer(fileStore *store.FileStore) *worldServer {
	server := &worldServer{
		store:              fileStore,
		sessionsByToken:    map[string]*worldSessionState{},
		sessionTokenByChar: map[string]string{},
		mobs:               map[string]*mobState{},
		quests:             map[string]questDefinition{},
		friendlyNPCs:       map[string]friendlyNPCDefinition{},
	}
	server.loadStarterContentLocked()
	server.ensureMobsLocked()
	return server
}

func (s *worldServer) loadStarterContentLocked() {
	s.questOrder = make([]string, 0, len(stonewakeQuestDefinitions))
	for _, quest := range stonewakeQuestDefinitions {
		if quest.TargetCount <= 0 {
			quest.TargetCount = 1
		}
		s.quests[quest.ID] = quest
		s.questOrder = append(s.questOrder, quest.ID)
	}
	if quest, ok := s.quests[starterQuestID]; ok {
		s.quest = quest
	}

	s.friendlyNPCOrder = make([]string, 0, len(stonewakeFriendlyNPCs))
	for _, npc := range stonewakeFriendlyNPCs {
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

	s.mobOrder = make([]string, 0, len(stonewakeMobSpawns))
	for _, spawn := range stonewakeMobSpawns {
		s.mobOrder = append(s.mobOrder, spawn.ID)
		s.mobs[spawn.ID] = &mobState{
			ID:              spawn.ID,
			MobTypeID:       spawn.MobTypeID,
			DisplayName:     spawn.DisplayName,
			Kind:            hostileMobKind,
			ZoneID:          defaultZoneID,
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
			AIState:         "idle",
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
