package worlds

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const (
	dungeonTallowdeepSluiceID = "dun_tallowdeep_sluice"
	dungeonQuestTallowdeepID  = "bb_tallowdeep_first_descent"

	npcTallowdeepEntranceID = "obj_tallowdeep_sluice_gate"
	npcTallowdeepExitID     = "obj_tallowdeep_exit_winch"

	mobTDSSluiceGuardTypeID  = "tds_sluice_guard"
	mobTDSRunoffCutterTypeID = "tds_runoff_cutter"
	mobTDSPressureHandTypeID = "tds_pressure_hand"
	mobTDSVellOrdrinTypeID   = "tds_vell_ordrin"

	itemTDSSluiceguardHandwrapsID = "tds_sluiceguard_handwraps"

	dungeonStateActive        = "active"
	dungeonStateCompleted     = "completed"
	dungeonStateEmptyExpiring = "empty_expiring"
	dungeonStateExpired       = "expired"
)

var dungeonDefinitions = map[string]dungeonDefinition{
	dungeonTallowdeepSluiceID: {
		ID:               dungeonTallowdeepSluiceID,
		DisplayName:      "Tallowdeep Sluice",
		LevelBand:        "8-12",
		InstanceZoneID:   dungeonZoneID,
		EntranceZoneID:   secondZoneID,
		EntranceEntityID: npcTallowdeepEntranceID,
		ExitEntityID:     npcTallowdeepExitID,
		StartPositions: []worldPosition{
			{ZoneID: dungeonZoneID, X: 12, Y: 12, Z: 0},
			{ZoneID: dungeonZoneID, X: 14, Y: 12, Z: 0},
			{ZoneID: dungeonZoneID, X: 10, Y: 12, Z: 0},
			{ZoneID: dungeonZoneID, X: 12, Y: 14, Z: 0},
			{ZoneID: dungeonZoneID, X: 12, Y: 10, Z: 0},
		},
		ExitPosition:   worldPosition{ZoneID: dungeonZoneID, X: 166, Y: 34, Z: 0},
		ReturnPosition: worldPosition{ZoneID: secondZoneID, X: 590, Y: 342, Z: 0},
		BossMobTypeID:  mobTDSVellOrdrinTypeID,
		QuestID:        dungeonQuestTallowdeepID,
		EmptyExpiryMs:  int64((5 * time.Minute).Milliseconds()),
		HardExpiryMs:   int64((60 * time.Minute).Milliseconds()),
		MobSpawns: []mobSpawnDefinition{
			{ID: "mob_tds_sluice_guard_01", ZoneID: dungeonZoneID, MobTypeID: mobTDSSluiceGuardTypeID, DisplayName: "Sluice Guard", Level: 8, X: 40, Y: 15, MaxHealth: 175, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 13, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.0, LeashRadius: 18, RespawnDelayMs: 0},
			{ID: "mob_tds_sluice_guard_02", ZoneID: dungeonZoneID, MobTypeID: mobTDSSluiceGuardTypeID, DisplayName: "Sluice Guard", Level: 8, X: 44, Y: 18, MaxHealth: 175, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 13, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.0, LeashRadius: 18, RespawnDelayMs: 0},
			{ID: "mob_tds_sluice_guard_03", ZoneID: dungeonZoneID, MobTypeID: mobTDSSluiceGuardTypeID, DisplayName: "Sluice Guard", Level: 9, X: 76, Y: 16, MaxHealth: 185, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 14, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.0, LeashRadius: 20, RespawnDelayMs: 0},
			{ID: "mob_tds_runoff_cutter_01", ZoneID: dungeonZoneID, MobTypeID: mobTDSRunoffCutterTypeID, DisplayName: "Runoff Cutter", Level: 9, X: 82, Y: 20, MaxHealth: 160, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 15, AttackCadenceMs: 1900, MoveSpeedPerSec: 4.3, LeashRadius: 20, RespawnDelayMs: 0},
			{ID: "mob_tds_runoff_cutter_02", ZoneID: dungeonZoneID, MobTypeID: mobTDSRunoffCutterTypeID, DisplayName: "Runoff Cutter", Level: 9, X: 109, Y: 27, MaxHealth: 165, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 15, AttackCadenceMs: 1900, MoveSpeedPerSec: 4.3, LeashRadius: 20, RespawnDelayMs: 0},
			{ID: "mob_tds_pressure_hand_01", ZoneID: dungeonZoneID, MobTypeID: mobTDSPressureHandTypeID, DisplayName: "Pressure Hand", Level: 9, X: 116, Y: 33, MaxHealth: 190, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 14, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.8, LeashRadius: 20, RespawnDelayMs: 0},
			{ID: "mob_tds_vell_ordrin_01", ZoneID: dungeonZoneID, MobTypeID: mobTDSVellOrdrinTypeID, DisplayName: "Vell Ordrin, Sluice Warden", Level: 10, X: 148, Y: 34, MaxHealth: 520, AggroRadius: 7, AttackRange: 2.9, AttackDamage: 20, AttackCadenceMs: 1900, MoveSpeedPerSec: 4.1, LeashRadius: 30, RespawnDelayMs: 0},
		},
	},
}

var (
	dungeonQuestDefinitions []questDefinition
	dungeonFriendlyNPCs     []friendlyNPCDefinition
)

func init() {
	dungeonFriendlyNPCs = []friendlyNPCDefinition{
		{
			ID:          npcTallowdeepEntranceID,
			ZoneID:      secondZoneID,
			DisplayName: "Tallowdeep Sluice Gate",
			Kind:        dungeonEntranceKind,
			X:           590.0,
			Y:           342.0,
			Z:           0.0,
			AIState:     "dungeon_entrance",
			Radius:      starterInteractRadius,
			Services: []npcService{
				{Type: "dungeon_entrance", ServiceID: dungeonTallowdeepSluiceID, Label: "Enter Tallowdeep Sluice"},
			},
		},
		{
			ID:          npcTallowdeepExitID,
			ZoneID:      dungeonZoneID,
			DisplayName: "Exit Winch",
			Kind:        dungeonExitKind,
			X:           166.0,
			Y:           34.0,
			Z:           0.0,
			AIState:     "dungeon_exit",
			Radius:      starterInteractRadius,
			Services: []npcService{
				{Type: "dungeon_exit", ServiceID: dungeonTallowdeepSluiceID, Label: "Leave Dungeon"},
			},
		},
	}

	dungeonQuestDefinitions = []questDefinition{
		{
			ID:            dungeonQuestTallowdeepID,
			ZoneID:        secondZoneID,
			Title:         "First Descent into Tallowdeep",
			ObjectiveType: objectiveKill,
			ObjectiveText: "Enter Tallowdeep Sluice and defeat Vell Ordrin, Sluice Warden. Group recommended.",
			GiverNPCID:    npcBBStonesetterBarnID,
			TurnInNPCID:   npcBBStonesetterBarnID,
			TargetMobType: mobTDSVellOrdrinTypeID,
			TargetCount:   1,
			RewardXP:      650,
			RewardCopper:  160,
			RewardItems: []itemRewardDefinition{
				{ItemID: itemTDSSluiceguardHandwrapsID, DisplayName: "Sluiceguard Handwraps", StackCount: 1},
			},
			PrerequisiteIDs:    []string{"bb_teeth_in_shallows"},
			LevelBand:          "8-12 Dungeon",
			MarkerX:            590.0,
			MarkerY:            342.0,
			PartyShareable:     true,
			GroupRecommended:   true,
			RecommendedPlayers: 3,
			PartyCreditRadius:  80.0,
		},
	}
}

var tallowdeepZoneMap = zoneMapDefinition{
	ZoneID:      dungeonZoneID,
	DisplayName: "Tallowdeep Sluice",
	MinX:        0,
	MinY:        0,
	MaxX:        180,
	MaxY:        70,
	Roads: []mapRoadDefinition{
		{
			ID:          "tds_main_sluice",
			DisplayName: "Sluice Route",
			Points: []mapPointDefinition{
				{X: 12, Y: 12},
				{X: 42, Y: 16},
				{X: 78, Y: 18},
				{X: 112, Y: 30},
				{X: 148, Y: 34},
				{X: 166, Y: 34},
			},
		},
	},
	Landmarks: []mapLandmarkDefinition{
		{ID: "tds_entry_shelf", DisplayName: "Entry Shelf", Kind: "start", X: 12, Y: 12},
		{ID: "tds_first_pack", DisplayName: "Guard Shelf", Kind: "trash", X: 42, Y: 16},
		{ID: "tds_sluice_tunnel", DisplayName: "Sluice Tunnel", Kind: "trash", X: 78, Y: 18},
		{ID: "tds_pressure_room", DisplayName: "Pressure Room", Kind: "trash", X: 112, Y: 30},
		{ID: "tds_boss_platform", DisplayName: "Sluice Warden Platform", Kind: "boss", X: 148, Y: 34},
		{ID: "tds_exit_winch", DisplayName: "Exit Winch", Kind: "exit", X: 166, Y: 34},
	},
}

var dungeonZoneDefinitions = []zoneDefinition{
	{
		ID:          dungeonZoneID,
		DisplayName: "Tallowdeep Sluice",
		LevelBand:   "8-12",
		Bounds:      zoneBoundsDefinition{MinX: 0, MinY: 0, MaxX: 180, MaxY: 70},
		Roads: []zoneRoadDefinition{
			{
				ID:          "tds_main_sluice",
				DisplayName: "Sluice Route",
				Points: []zonePointDefinition{
					{ID: "tds_entry", DisplayName: "Entry Shelf", Type: "start", X: 12, Y: 12},
					{ID: "tds_guard_shelf", DisplayName: "Guard Shelf", Type: "trash", X: 42, Y: 16},
					{ID: "tds_pressure_room", DisplayName: "Pressure Room", Type: "trash", X: 112, Y: 30},
					{ID: "tds_boss_platform", DisplayName: "Sluice Warden Platform", Type: "boss", X: 148, Y: 34},
					{ID: "tds_exit_winch", DisplayName: "Exit Winch", Type: "exit", X: 166, Y: 34},
				},
			},
		},
		Landmarks: []zonePointDefinition{
			{ID: "tds_entry_shelf", DisplayName: "Entry Shelf", Type: "start", X: 12, Y: 12},
			{ID: "tds_pressure_room", DisplayName: "Pressure Room", Type: "trash", X: 112, Y: 30},
			{ID: "tds_boss_platform", DisplayName: "Sluice Warden Platform", Type: "boss", X: 148, Y: 34},
		},
		Transitions: []zonePointDefinition{
			{ID: "tds_exit_winch", DisplayName: "Exit Winch", Type: "dungeon_exit", X: 166, Y: 34},
		},
	},
}

var tallowdeepNavigationAreas = []navigationAreaDefinition{
	{ID: "tds_entry_shelf", DisplayName: "Entry Shelf", Kind: "start", CenterX: 12, CenterY: 12, Radius: 10, RouteHintText: "Regroup on the entry shelf before pulling the first guard pair.", QuestIDs: []string{dungeonQuestTallowdeepID}},
	{ID: "tds_guard_shelf", DisplayName: "Guard Shelf", Kind: "trash", CenterX: 42, CenterY: 16, Radius: 12, RouteHintText: "Clear the first guard pair and continue along the sluice route.", QuestIDs: []string{dungeonQuestTallowdeepID}, TargetMobType: mobTDSSluiceGuardTypeID},
	{ID: "tds_pressure_room", DisplayName: "Pressure Room", Kind: "trash", CenterX: 112, CenterY: 30, Radius: 16, RouteHintText: "Clear the pressure room before engaging the warden.", QuestIDs: []string{dungeonQuestTallowdeepID}, TargetMobType: mobTDSPressureHandTypeID},
	{ID: "tds_boss_platform", DisplayName: "Sluice Warden Platform", Kind: "boss", CenterX: 148, CenterY: 34, Radius: 18, RouteHintText: "Defeat Vell Ordrin, Sluice Warden.", QuestIDs: []string{dungeonQuestTallowdeepID}, TargetMobType: mobTDSVellOrdrinTypeID},
	{ID: "tds_exit_winch", DisplayName: "Exit Winch", Kind: "exit", CenterX: 166, CenterY: 34, Radius: 8, RouteHintText: "Use the exit winch to return to the quarry.", TargetEntityID: npcTallowdeepExitID},
}

func (s *worldServer) handleDungeonEnter(w http.ResponseWriter, r *http.Request) {
	var request dungeonEnterRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}

	if err := s.enterDungeonLocked(session, request.DungeonID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "dungeon_enter_failed", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleDungeonExit(w http.ResponseWriter, r *http.Request) {
	var request dungeonExitRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if err := s.exitDungeonLocked(session, "manual_exit"); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "dungeon_exit_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleDungeonReset(w http.ResponseWriter, r *http.Request) {
	var request dungeonResetRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if err := s.resetDungeonForSessionLocked(session, request.DungeonID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "dungeon_reset_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) enterDungeonLocked(session *worldSessionState, dungeonID string) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if !session.Alive {
		return fmt.Errorf("dead players cannot enter a dungeon")
	}
	if session.InstanceID != "" {
		return fmt.Errorf("you are already inside this dungeon")
	}

	definition, found := dungeonDefinitions[dungeonID]
	if !found {
		return fmt.Errorf("dungeon is not available")
	}
	entrance, ok := s.findFriendlyNPCDefinition(definition.EntranceEntityID)
	if !ok || session.ZoneID != definition.EntranceZoneID ||
		distance2D(session.X, session.Y, entrance.X, entrance.Y) > entrance.Radius {
		return fmt.Errorf("move closer to the %s", entrance.DisplayName)
	}

	party, err := s.store.GetPartyForCharacter(session.CharacterID)
	if err != nil {
		if !soloDungeonEntryAllowed() {
			return fmt.Errorf("join a party before entering this dungeon")
		}
		partyID := "solo_" + session.CharacterID
		party = &platform.Party{ID: partyID, LeaderCharacterID: session.CharacterID, MemberCharacterIDs: []string{session.CharacterID}}
	} else if party == nil {
		return fmt.Errorf("join a party before entering this dungeon")
	}

	instance := s.activeDungeonInstanceForPartyLocked(party.ID, definition.ID)
	if instance == nil {
		instance = s.createDungeonInstanceLocked(definition, party.ID, party.MemberCharacterIDs)
	}
	if !containsString(instance.MemberCharacterIDs, session.CharacterID) {
		return fmt.Errorf("only members of this party can enter this instance")
	}

	placed := 0
	for _, memberID := range instance.MemberCharacterIDs {
		memberSession := s.findConnectedSessionByCharacterLocked(memberID)
		if memberSession == nil || memberSession.InstanceID != "" || memberSession.ZoneID != definition.EntranceZoneID {
			continue
		}
		if distance2D(memberSession.X, memberSession.Y, entrance.X, entrance.Y) > entrance.Radius+12.0 && memberSession.CharacterID != session.CharacterID {
			continue
		}
		s.placeSessionInDungeonLocked(memberSession, instance, placed)
		placed++
	}
	if session.InstanceID != instance.InstanceID {
		s.placeSessionInDungeonLocked(session, instance, placed)
	}

	s.sendSystemMessageLocked(
		fmt.Sprintf("Entered %s.", definition.DisplayName),
		recipientSet(instance.MemberCharacterIDs...))
	observability.LogEvent("world-service", "world.dungeon_entered", map[string]any{
		"instanceId":  instance.InstanceID,
		"dungeonId":   definition.ID,
		"partyId":     instance.PartyID,
		"characterId": session.CharacterID,
	})
	return nil
}

func (s *worldServer) createDungeonInstanceLocked(definition dungeonDefinition, partyID string, memberIDs []string) *dungeonInstanceState {
	nowMs := nowMillis()
	s.instanceCounter++
	instanceID := fmt.Sprintf("inst_%06d", s.instanceCounter)
	instance := &dungeonInstanceState{
		InstanceID:         instanceID,
		DungeonID:          definition.ID,
		PartyID:            partyID,
		ZoneID:             definition.InstanceZoneID,
		CreatedAtMs:        nowMs,
		ExpiresAtMs:        nowMs + definition.HardExpiryMs,
		State:              dungeonStateActive,
		MemberCharacterIDs: platform.NormalizeStringIDs(memberIDs),
		PlayersInside:      map[string]bool{},
		ReturnPositions:    map[string]worldPosition{},
		Mobs:               map[string]*mobState{},
		BossRewardGranted:  map[string]bool{},
	}
	for _, spawn := range definition.MobSpawns {
		mobID := instanceID + "_" + spawn.ID
		instance.MobOrder = append(instance.MobOrder, mobID)
		instance.Mobs[mobID] = &mobState{
			ID:              mobID,
			InstanceID:      instanceID,
			MobTypeID:       spawn.MobTypeID,
			DisplayName:     spawn.DisplayName,
			Kind:            hostileMobKind,
			ZoneID:          definition.InstanceZoneID,
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
	s.dungeonInstances[instanceID] = instance
	s.instanceByParty[partyID] = instanceID
	observability.LogEvent("world-service", "world.dungeon_instance_created", map[string]any{
		"instanceId": instanceID,
		"dungeonId":  definition.ID,
		"partyId":    partyID,
		"members":    instance.MemberCharacterIDs,
		"expiresAt":  instance.ExpiresAtMs,
	})
	return instance
}

func (s *worldServer) placeSessionInDungeonLocked(session *worldSessionState, instance *dungeonInstanceState, index int) {
	definition := dungeonDefinitions[instance.DungeonID]
	position := definition.StartPositions[index%len(definition.StartPositions)]
	returnPosition := worldPosition{ZoneID: session.ZoneID, X: session.X, Y: session.Y, Z: session.Z}
	if returnPosition.ZoneID == "" || returnPosition.ZoneID == definition.InstanceZoneID {
		returnPosition = definition.ReturnPosition
	}
	instance.ReturnPositions[session.CharacterID] = returnPosition
	instance.PlayersInside[session.CharacterID] = true
	instance.State = dungeonStateActive
	instance.LastPlayerLeftAtMs = 0

	session.InstanceID = instance.InstanceID
	session.ReturnZoneID = returnPosition.ZoneID
	session.ReturnX = returnPosition.X
	session.ReturnY = returnPosition.Y
	session.ReturnZ = returnPosition.Z
	session.ZoneID = definition.InstanceZoneID
	session.X = position.X
	session.Y = position.Y
	session.Z = position.Z
	s.resetSessionCombatStateLocked(session, "dungeon_enter")
}

func (s *worldServer) exitDungeonLocked(session *worldSessionState, reason string) error {
	if session == nil || session.InstanceID == "" {
		return fmt.Errorf("you are not inside a dungeon")
	}
	instance := s.dungeonInstances[session.InstanceID]
	returnPosition := worldPosition{ZoneID: session.ReturnZoneID, X: session.ReturnX, Y: session.ReturnY, Z: session.ReturnZ}
	if stored, ok := instance.ReturnPositions[session.CharacterID]; ok {
		returnPosition = stored
	}
	if returnPosition.ZoneID == "" {
		if definition, ok := dungeonDefinitions[instance.DungeonID]; ok {
			returnPosition = definition.ReturnPosition
		} else {
			returnPosition = worldPosition{ZoneID: secondZoneID, X: secondZoneEntryX, Y: secondZoneEntryY, Z: 0}
		}
	}

	s.resetSessionCombatStateLocked(session, reason)
	if instance != nil {
		delete(instance.PlayersInside, session.CharacterID)
		s.markDungeonEmptyIfNeededLocked(instance)
	}
	session.InstanceID = ""
	session.ReturnZoneID = ""
	session.ReturnX = 0
	session.ReturnY = 0
	session.ReturnZ = 0
	session.ZoneID = returnPosition.ZoneID
	session.X = returnPosition.X
	session.Y = returnPosition.Y
	session.Z = returnPosition.Z

	persistStartedAt := time.Now()
	_, err := s.store.UpdateCharacterState(session.CharacterID, session.ZoneID, session.X, session.Y, session.Z)
	s.recordPersistenceDuration("character_state_dungeon_exit", persistStartedAt, err)
	if err != nil {
		return err
	}
	observability.LogEvent("world-service", "world.dungeon_exited", map[string]any{
		"characterId": session.CharacterID,
		"reason":      reason,
		"zoneId":      session.ZoneID,
		"x":           session.X,
		"y":           session.Y,
	})
	return nil
}

func (s *worldServer) resetDungeonForSessionLocked(session *worldSessionState, dungeonID string) error {
	party, err := s.store.GetPartyForCharacter(session.CharacterID)
	if err != nil && !soloDungeonEntryAllowed() {
		return fmt.Errorf("you are not in a party")
	}
	if party != nil && party.LeaderCharacterID != session.CharacterID {
		return fmt.Errorf("only the party leader can reset this dungeon")
	}
	partyID := "solo_" + session.CharacterID
	if party != nil {
		partyID = party.ID
	}
	instanceID := s.instanceByParty[partyID]
	instance := s.dungeonInstances[instanceID]
	if instance == nil || instance.DungeonID != dungeonID {
		return nil
	}
	for _, memberID := range instance.MemberCharacterIDs {
		memberSession := s.findConnectedSessionByCharacterLocked(memberID)
		if memberSession != nil && memberSession.InstanceID == instance.InstanceID {
			return fmt.Errorf("all players must leave before reset")
		}
	}
	delete(s.dungeonInstances, instance.InstanceID)
	delete(s.instanceByParty, partyID)
	s.sendSystemMessageLocked("Dungeon instance reset.", recipientSet(instance.MemberCharacterIDs...))
	return nil
}

func (s *worldServer) activeDungeonInstanceForPartyLocked(partyID string, dungeonID string) *dungeonInstanceState {
	instanceID := s.instanceByParty[partyID]
	instance := s.dungeonInstances[instanceID]
	if instance == nil || instance.DungeonID != dungeonID || instance.State == dungeonStateExpired {
		return nil
	}
	return instance
}

func (s *worldServer) dungeonInstanceActiveForSessionLocked(session *worldSessionState) bool {
	if session == nil || session.InstanceID == "" {
		return false
	}
	instance := s.dungeonInstances[session.InstanceID]
	return instance != nil && instance.State != dungeonStateExpired && nowMillis() < instance.ExpiresAtMs
}

func (s *worldServer) markDungeonEmptyIfNeededLocked(instance *dungeonInstanceState) {
	if instance == nil || len(instance.PlayersInside) != 0 {
		return
	}
	if definition, ok := dungeonDefinitions[instance.DungeonID]; ok {
		instance.State = dungeonStateEmptyExpiring
		instance.LastPlayerLeftAtMs = nowMillis()
		instance.ExpiresAtMs = minInt64(instance.ExpiresAtMs, instance.LastPlayerLeftAtMs+definition.EmptyExpiryMs)
	}
}

func (s *worldServer) cleanupDungeonInstancesLocked(now time.Time) {
	nowMs := now.UnixMilli()
	for _, instance := range s.dungeonInstances {
		if instance == nil {
			continue
		}
		if len(instance.PlayersInside) == 0 && instance.State == dungeonStateActive {
			s.markDungeonEmptyIfNeededLocked(instance)
		}
		if instance.ExpiresAtMs > nowMs {
			continue
		}
		for _, memberID := range instance.MemberCharacterIDs {
			if memberSession := s.findConnectedSessionByCharacterLocked(memberID); memberSession != nil && memberSession.InstanceID == instance.InstanceID {
				_ = s.exitDungeonLocked(memberSession, "instance_expired")
			}
		}
		instance.State = dungeonStateExpired
		delete(s.dungeonInstances, instance.InstanceID)
		delete(s.instanceByParty, instance.PartyID)
		observability.LogEvent("world-service", "world.dungeon_instance_expired", map[string]any{
			"instanceId": instance.InstanceID,
			"dungeonId":  instance.DungeonID,
			"partyId":    instance.PartyID,
		})
	}
}

func (s *worldServer) applyDungeonKillCreditLocked(session *worldSessionState, mob *mobState) error {
	if session == nil || mob == nil || mob.InstanceID == "" {
		return s.applyQuestKillCreditLocked(session, mob)
	}
	instance := s.dungeonInstances[mob.InstanceID]
	if instance == nil {
		return s.applyQuestKillCreditLocked(session, mob)
	}
	definition := dungeonDefinitions[instance.DungeonID]
	if mob.MobTypeID != definition.BossMobTypeID {
		return nil
	}

	instance.Objective.BossDefeated = true
	instance.Objective.UpdatedAtMs = nowMillis()
	instance.State = dungeonStateCompleted
	var firstErr error
	for _, memberID := range instance.MemberCharacterIDs {
		if !instance.PlayersInside[memberID] {
			continue
		}
		memberSession := s.findConnectedSessionByCharacterLocked(memberID)
		if memberSession == nil || memberSession.InstanceID != instance.InstanceID {
			continue
		}
		if err := s.applyQuestKillCreditLocked(memberSession, mob); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.sendSystemMessageLocked("Dungeon objective complete: Vell Ordrin defeated.", recipientSet(instance.MemberCharacterIDs...))
	return firstErr
}

func (s *worldServer) buildDungeonInstanceResponse(session *worldSessionState) map[string]any {
	if session == nil || session.InstanceID == "" {
		return map[string]any{}
	}
	instance := s.dungeonInstances[session.InstanceID]
	if instance == nil {
		return map[string]any{}
	}
	definition := dungeonDefinitions[instance.DungeonID]
	return map[string]any{
		"instanceId":     instance.InstanceID,
		"dungeonId":      instance.DungeonID,
		"displayName":    definition.DisplayName,
		"levelBand":      definition.LevelBand,
		"state":          instance.State,
		"createdAt":      instance.CreatedAtMs,
		"expiresAt":      instance.ExpiresAtMs,
		"objectiveState": instance.Objective,
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (s *worldServer) hostileMobsForSessionLocked(session *worldSessionState) []*mobState {
	if session == nil || session.InstanceID == "" {
		return s.hostileMobsLocked()
	}
	instance := s.dungeonInstances[session.InstanceID]
	if instance == nil {
		return nil
	}
	mobs := make([]*mobState, 0, len(instance.MobOrder))
	for _, mobID := range instance.MobOrder {
		if mob := instance.Mobs[mobID]; mob != nil {
			mobs = append(mobs, mob)
		}
	}
	return mobs
}

func (s *worldServer) recoverExpiredDungeonSessionLocked(session *worldSessionState) {
	if session == nil || session.InstanceID == "" || s.dungeonInstanceActiveForSessionLocked(session) {
		return
	}
	returnPosition := worldPosition{ZoneID: session.ReturnZoneID, X: session.ReturnX, Y: session.ReturnY, Z: session.ReturnZ}
	if returnPosition.ZoneID == "" {
		returnPosition = worldPosition{ZoneID: secondZoneID, X: 590, Y: 342, Z: 0}
	}
	session.InstanceID = ""
	session.ReturnZoneID = ""
	session.ReturnX = 0
	session.ReturnY = 0
	session.ReturnZ = 0
	session.ZoneID = returnPosition.ZoneID
	session.X = returnPosition.X
	session.Y = returnPosition.Y
	session.Z = returnPosition.Z
}

func soloDungeonEntryAllowed() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("AMANDACORE_ALLOW_SOLO_DUNGEON")))
	return value == "1" || value == "true" || value == "yes"
}

func minInt64(left int64, right int64) int64 {
	if left < right {
		return left
	}
	return right
}
