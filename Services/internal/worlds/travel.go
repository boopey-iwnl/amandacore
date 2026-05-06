package worlds

import (
	"fmt"
	"net/http"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const (
	bindMasterServiceType  = "bind_master"
	routeMasterServiceType = "route_master"

	bindHearthwatchYardID  = platform.DefaultBindLocationID
	bindHighmereCrossingID = "bind_highmere_crossing"

	travelHearthwatchYardID  = platform.DefaultTravelPointID
	travelHighmereCrossingID = "travel_highmere_crossing"
	travelPinebarrowWatchID  = "travel_pinebarrow_watch"

	routeHearthwatchHighmereID = "route_hearthwatch_highmere"
	routeHighmerePinebarrowID  = "route_highmere_pinebarrow"

	mountStonewakeRoadstepperID = "mount_stonewake_roadstepper"

	recallCooldownSeconds = int64(10 * 60)
)

type bindLocationDefinition struct {
	ID              string
	DisplayName     string
	ZoneID          string
	X               float64
	Y               float64
	Z               float64
	ServiceEntityID string
}

type travelPointDefinition struct {
	ID                string
	DisplayName       string
	ZoneID            string
	X                 float64
	Y                 float64
	Z                 float64
	ServiceEntityID   string
	ConnectedRouteIDs []string
	RequiredLevel     int
}

type travelRouteDefinition struct {
	ID                string
	FromTravelPointID string
	ToTravelPointID   string
	CostCopper        int
	RequiredLevel     int
}

type mountDefinition struct {
	ID              string
	DisplayName     string
	RequiredLevel   int
	SpeedMultiplier float64
	Source          string
	AccountWide     bool
	Implemented     bool
}

var bindLocationDefinitions = map[string]bindLocationDefinition{
	bindHearthwatchYardID: {
		ID:              bindHearthwatchYardID,
		DisplayName:     "Hearthwatch Yard",
		ZoneID:          defaultZoneID,
		X:               232,
		Y:               130,
		Z:               0,
		ServiceEntityID: npcCommanderElianRookID,
	},
	bindHighmereCrossingID: {
		ID:              bindHighmereCrossingID,
		DisplayName:     "Highmere Crossing",
		ZoneID:          secondZoneID,
		X:               150,
		Y:               160,
		Z:               0,
		ServiceEntityID: npcBBCaptainMaraID,
	},
}

var travelPointDefinitions = map[string]travelPointDefinition{
	travelHearthwatchYardID: {
		ID:                travelHearthwatchYardID,
		DisplayName:       "Hearthwatch Yard",
		ZoneID:            defaultZoneID,
		X:                 232,
		Y:                 130,
		Z:                 0,
		ServiceEntityID:   npcCommanderElianRookID,
		ConnectedRouteIDs: []string{routeHearthwatchHighmereID},
	},
	travelHighmereCrossingID: {
		ID:                travelHighmereCrossingID,
		DisplayName:       "Highmere Crossing",
		ZoneID:            secondZoneID,
		X:                 150,
		Y:                 160,
		Z:                 0,
		ServiceEntityID:   npcBBCaptainMaraID,
		ConnectedRouteIDs: []string{routeHearthwatchHighmereID, routeHighmerePinebarrowID},
		RequiredLevel:     5,
	},
	travelPinebarrowWatchID: {
		ID:                travelPinebarrowWatchID,
		DisplayName:       "Pinebarrow Watch",
		ZoneID:            secondZoneID,
		X:                 485,
		Y:                 275,
		Z:                 0,
		ServiceEntityID:   npcBBWardenTalikID,
		ConnectedRouteIDs: []string{routeHighmerePinebarrowID},
		RequiredLevel:     8,
	},
}

var travelRouteDefinitions = map[string]travelRouteDefinition{
	routeHearthwatchHighmereID: {
		ID:                routeHearthwatchHighmereID,
		FromTravelPointID: travelHearthwatchYardID,
		ToTravelPointID:   travelHighmereCrossingID,
		CostCopper:        20,
		RequiredLevel:     5,
	},
	routeHighmerePinebarrowID: {
		ID:                routeHighmerePinebarrowID,
		FromTravelPointID: travelHighmereCrossingID,
		ToTravelPointID:   travelPinebarrowWatchID,
		CostCopper:        25,
		RequiredLevel:     8,
	},
}

var mountDefinitions = map[string]mountDefinition{
	mountStonewakeRoadstepperID: {
		ID:              mountStonewakeRoadstepperID,
		DisplayName:     "Stonewake Roadstepper",
		RequiredLevel:   6,
		SpeedMultiplier: 1.5,
		Source:          "debug_route_vendor",
		AccountWide:     false,
		Implemented:     true,
	},
}

func (s *worldServer) handleTravelState(w http.ResponseWriter, r *http.Request) {
	session, ok := s.sessionFromQuery(w, r)
	if !ok {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	httpapi.WriteJSON(w, http.StatusOK, s.buildTravelStateResponseLocked(session))
}

func (s *worldServer) handleSetBindPoint(w http.ResponseWriter, r *http.Request) {
	var request bindSetRequest
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

	bindLocation, err := s.validateBindAccessLocked(session, request.BindLocationID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "bind_set_failed", err.Error())
		return
	}

	session.BindPoint = platform.NormalizeCharacterBindPoint(session.CharacterID, platform.CharacterBindPoint{
		CharacterID:    session.CharacterID,
		ZoneID:         bindLocation.ZoneID,
		X:              bindLocation.X,
		Y:              bindLocation.Y,
		Z:              bindLocation.Z,
		BindLocationID: bindLocation.ID,
		DisplayName:    bindLocation.DisplayName,
		SetAt:          time.Now().Unix(),
	})
	if err := s.persistTravelSessionLocked(session); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "bind_save_failed", err.Error())
		return
	}
	observability.LogEvent("world-service", "world.bind_set", map[string]any{
		"characterId":    session.CharacterID,
		"bindLocationId": bindLocation.ID,
		"zoneId":         bindLocation.ZoneID,
		"x":              bindLocation.X,
		"y":              bindLocation.Y,
	})
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleUseRecall(w http.ResponseWriter, r *http.Request) {
	var request recallUseRequest
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
	if err := s.validateTransportAllowedLocked(session); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "recall_failed", err.Error())
		return
	}

	now := time.Now().Unix()
	session.TravelState = platform.NormalizeCharacterTravelState(session.TravelState)
	if session.TravelState.RecallReadyAt > now {
		httpapi.Error(w, http.StatusConflict, "recall_on_cooldown", "Return Signal is still recharging.")
		return
	}
	bindPoint := platform.NormalizeCharacterBindPoint(session.CharacterID, session.BindPoint)
	if err := s.validateDestinationPositionLocked(bindPoint.ZoneID, bindPoint.X, bindPoint.Y); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "recall_failed", err.Error())
		return
	}

	s.forceDismountLocked(session, "recall")
	s.resetSessionCombatStateLocked(session, "recall")
	session.ZoneID = bindPoint.ZoneID
	session.X = bindPoint.X
	session.Y = bindPoint.Y
	session.Z = bindPoint.Z
	session.TravelState.RecallReadyAt = now + recallCooldownSeconds
	session.LastSeenAt = now
	if err := s.persistTravelSessionLocked(session); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "recall_save_failed", err.Error())
		return
	}
	observability.LogEvent("world-service", "world.recall_used", map[string]any{
		"characterId":    session.CharacterID,
		"bindLocationId": bindPoint.BindLocationID,
		"zoneId":         session.ZoneID,
		"x":              session.X,
		"y":              session.Y,
		"recallReadyAt":  session.TravelState.RecallReadyAt,
	})
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleDiscoverTravelPoint(w http.ResponseWriter, r *http.Request) {
	var request travelDiscoverRequest
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
	point, err := s.validateTravelPointAccessLocked(session, request.TravelPointID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "travel_discover_failed", err.Error())
		return
	}

	session.TravelState = platform.NormalizeCharacterTravelState(session.TravelState)
	if !stringIDSetContains(session.TravelState.DiscoveredTravelPointIDs, point.ID) {
		session.TravelState.DiscoveredTravelPointIDs = append(session.TravelState.DiscoveredTravelPointIDs, point.ID)
	}
	session.TravelState.LastTravelPointID = point.ID
	if err := s.persistTravelSessionLocked(session); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "travel_discover_save_failed", err.Error())
		return
	}
	observability.LogEvent("world-service", "world.travel_point_discovered", map[string]any{
		"characterId":   session.CharacterID,
		"travelPointId": point.ID,
		"zoneId":        point.ZoneID,
	})
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleTravelDestinations(w http.ResponseWriter, r *http.Request) {
	session, ok := s.sessionFromQuery(w, r)
	if !ok {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	httpapi.WriteJSON(w, http.StatusOK, s.buildTravelDestinationsResponseLocked(session, r.URL.Query().Get("sourcePointId")))
}

func (s *worldServer) handleTravelRoute(w http.ResponseWriter, r *http.Request) {
	var request travelRouteRequest
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
	if err := s.validateTransportAllowedLocked(session); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "travel_failed", err.Error())
		return
	}

	source, err := s.validateTravelPointAccessLocked(session, request.SourcePointID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "travel_failed", err.Error())
		return
	}
	destination, found := travelPointDefinitions[request.DestinationPointID]
	if !found {
		httpapi.Error(w, http.StatusBadRequest, "travel_failed", "destination is not available")
		return
	}
	route, err := s.findTravelRoute(source.ID, destination.ID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "travel_failed", err.Error())
		return
	}
	session.TravelState = platform.NormalizeCharacterTravelState(session.TravelState)
	if !stringIDSetContains(session.TravelState.DiscoveredTravelPointIDs, source.ID) {
		httpapi.Error(w, http.StatusBadRequest, "travel_failed", "source travel point has not been discovered")
		return
	}
	if !stringIDSetContains(session.TravelState.DiscoveredTravelPointIDs, destination.ID) {
		httpapi.Error(w, http.StatusBadRequest, "travel_failed", "destination travel point has not been discovered")
		return
	}
	if route.RequiredLevel > 0 && session.Level < route.RequiredLevel {
		httpapi.Error(w, http.StatusBadRequest, "travel_failed", fmt.Sprintf("requires level %d", route.RequiredLevel))
		return
	}
	if route.CostCopper > 0 && session.CurrencyCopper < route.CostCopper {
		httpapi.Error(w, http.StatusBadRequest, "travel_failed", "not enough copper")
		return
	}
	if err := s.validateDestinationPositionLocked(destination.ZoneID, destination.X, destination.Y); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "travel_failed", err.Error())
		return
	}

	session.CurrentlyTraveling = true
	defer func() { session.CurrentlyTraveling = false }()
	s.forceDismountLocked(session, "travel")
	s.resetSessionCombatStateLocked(session, "travel")
	session.CurrencyCopper -= route.CostCopper
	session.ZoneID = destination.ZoneID
	session.X = destination.X
	session.Y = destination.Y
	session.Z = destination.Z
	session.TravelState.LastTravelPointID = destination.ID
	session.LastSeenAt = time.Now().Unix()
	if err := s.persistTravelSessionLocked(session); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "travel_save_failed", err.Error())
		return
	}
	observability.LogEvent("world-service", "world.travel_route_used", map[string]any{
		"characterId":   session.CharacterID,
		"routeId":       route.ID,
		"sourcePointId": source.ID,
		"destinationId": destination.ID,
		"costCopper":    route.CostCopper,
		"zoneId":        session.ZoneID,
		"x":             session.X,
		"y":             session.Y,
	})
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleMountCollection(w http.ResponseWriter, r *http.Request) {
	session, ok := s.sessionFromQuery(w, r)
	if !ok {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	httpapi.WriteJSON(w, http.StatusOK, s.buildMountCollectionResponseLocked(session))
}

func (s *worldServer) handleUnlockMount(w http.ResponseWriter, r *http.Request) {
	var request mountUnlockRequest
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
	mount, err := s.validateMountDefinitionLocked(session, request.MountID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "mount_unlock_failed", err.Error())
		return
	}
	session.MountState = platform.NormalizeCharacterMountState(session.MountState)
	if !stringIDSetContains(session.MountState.UnlockedMountIDs, mount.ID) {
		session.MountState.UnlockedMountIDs = append(session.MountState.UnlockedMountIDs, mount.ID)
	}
	if session.MountState.SelectedMountID == "" {
		session.MountState.SelectedMountID = mount.ID
	}
	if err := s.persistTravelSessionLocked(session); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "mount_unlock_save_failed", err.Error())
		return
	}
	observability.LogEvent("world-service", "world.mount_unlocked", map[string]any{
		"characterId": session.CharacterID,
		"mountId":     mount.ID,
		"source":      mount.Source,
	})
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleSelectMount(w http.ResponseWriter, r *http.Request) {
	var request mountSelectRequest
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
	if !stringIDSetContains(platform.NormalizeCharacterMountState(session.MountState).UnlockedMountIDs, request.MountID) {
		httpapi.Error(w, http.StatusBadRequest, "mount_select_failed", "mount has not been unlocked")
		return
	}
	if _, found := mountDefinitions[request.MountID]; !found {
		httpapi.Error(w, http.StatusBadRequest, "mount_select_failed", "mount is not available")
		return
	}
	session.MountState.SelectedMountID = request.MountID
	if err := s.persistTravelSessionLocked(session); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "mount_select_save_failed", err.Error())
		return
	}
	observability.LogEvent("world-service", "world.mount_selected", map[string]any{
		"characterId": session.CharacterID,
		"mountId":     request.MountID,
	})
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleSummonMount(w http.ResponseWriter, r *http.Request) {
	var request mountSummonRequest
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
	if err := s.validateMountAllowedLocked(session); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "mount_summon_failed", err.Error())
		return
	}
	session.MountState = platform.NormalizeCharacterMountState(session.MountState)
	mountID := request.MountID
	if mountID == "" {
		mountID = session.MountState.SelectedMountID
	}
	mount, found := mountDefinitions[mountID]
	if !found || !mount.Implemented {
		httpapi.Error(w, http.StatusBadRequest, "mount_summon_failed", "mount is not available")
		return
	}
	if !stringIDSetContains(session.MountState.UnlockedMountIDs, mount.ID) {
		httpapi.Error(w, http.StatusBadRequest, "mount_summon_failed", "mount has not been unlocked")
		return
	}
	session.MountState.SelectedMountID = mount.ID
	session.MountState.CurrentlyMounted = true
	session.MountState.MountedSince = time.Now().Unix()
	session.MountState.CurrentSpeedModifier = mount.SpeedMultiplier
	if err := s.persistTravelSessionLocked(session); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "mount_summon_save_failed", err.Error())
		return
	}
	observability.LogEvent("world-service", "world.mount_summoned", map[string]any{
		"characterId":     session.CharacterID,
		"mountId":         mount.ID,
		"speedMultiplier": mount.SpeedMultiplier,
	})
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleDismissMount(w http.ResponseWriter, r *http.Request) {
	var request mountDismissRequest
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
	if !session.MountState.CurrentlyMounted {
		httpapi.Error(w, http.StatusBadRequest, "mount_dismiss_failed", "you are not mounted")
		return
	}
	s.forceDismountLocked(session, "manual")
	if err := s.persistTravelSessionLocked(session); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "mount_dismiss_save_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) validateBindAccessLocked(session *worldSessionState, bindLocationID string) (bindLocationDefinition, error) {
	if session == nil {
		return bindLocationDefinition{}, fmt.Errorf("world session token was not found")
	}
	if !session.Alive {
		return bindLocationDefinition{}, fmt.Errorf("dead players cannot set a bind point")
	}
	for _, bindLocation := range bindLocationDefinitions {
		if bindLocationID != "" && bindLocation.ID != bindLocationID {
			continue
		}
		if session.CurrentTargetID != bindLocation.ServiceEntityID {
			continue
		}
		if !s.friendlyInRangeLocked(session, bindLocation.ServiceEntityID) {
			return bindLocationDefinition{}, fmt.Errorf("move closer to %s", bindLocation.DisplayName)
		}
		return bindLocation, nil
	}
	return bindLocationDefinition{}, fmt.Errorf("right-click a bind service first")
}

func (s *worldServer) validateTravelPointAccessLocked(session *worldSessionState, travelPointID string) (travelPointDefinition, error) {
	if session == nil {
		return travelPointDefinition{}, fmt.Errorf("world session token was not found")
	}
	for _, point := range travelPointDefinitions {
		if travelPointID != "" && point.ID != travelPointID {
			continue
		}
		if session.CurrentTargetID != point.ServiceEntityID {
			continue
		}
		if session.Level < point.RequiredLevel {
			return travelPointDefinition{}, fmt.Errorf("requires level %d", point.RequiredLevel)
		}
		if !s.friendlyInRangeLocked(session, point.ServiceEntityID) {
			return travelPointDefinition{}, fmt.Errorf("move closer to %s", point.DisplayName)
		}
		return point, nil
	}
	return travelPointDefinition{}, fmt.Errorf("right-click a route master first")
}

func (s *worldServer) validateTransportAllowedLocked(session *worldSessionState) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if !session.Alive {
		return fmt.Errorf("dead players cannot travel")
	}
	if session.CurrentlyTraveling {
		return fmt.Errorf("already traveling")
	}
	if session.InstanceID != "" {
		return fmt.Errorf("leave the dungeon before traveling")
	}
	if session.HousingSpaceID != "" {
		return fmt.Errorf("leave housing before traveling")
	}
	if s.sessionInCombatLocked(session) {
		return fmt.Errorf("cannot travel while in combat")
	}
	return nil
}

func (s *worldServer) validateMountAllowedLocked(session *worldSessionState) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if !session.Alive {
		return fmt.Errorf("dead players cannot mount")
	}
	if session.CurrentlyTraveling {
		return fmt.Errorf("cannot mount while traveling")
	}
	if session.InstanceID != "" {
		return fmt.Errorf("mounts are not allowed inside dungeons")
	}
	if session.HousingSpaceID != "" {
		return fmt.Errorf("mounts are not allowed inside housing")
	}
	if session.MountState.CurrentlyMounted {
		return fmt.Errorf("you are already mounted")
	}
	if s.sessionInCombatLocked(session) {
		return fmt.Errorf("cannot mount while in combat")
	}
	return nil
}

func (s *worldServer) validateDestinationPositionLocked(zoneID string, x float64, y float64) error {
	zone, found := s.zones[zoneID]
	if !found {
		return fmt.Errorf("destination zone is not available")
	}
	if x < zone.Bounds.MinX || x > zone.Bounds.MaxX || y < zone.Bounds.MinY || y > zone.Bounds.MaxY {
		return fmt.Errorf("destination is outside the playable area")
	}
	return nil
}

func (s *worldServer) validateMountDefinitionLocked(session *worldSessionState, mountID string) (mountDefinition, error) {
	mount, found := mountDefinitions[mountID]
	if !found || !mount.Implemented {
		return mountDefinition{}, fmt.Errorf("mount is not available")
	}
	if session.Level < mount.RequiredLevel {
		return mountDefinition{}, fmt.Errorf("requires level %d", mount.RequiredLevel)
	}
	return mount, nil
}

func (s *worldServer) sessionInCombatLocked(session *worldSessionState) bool {
	if session == nil {
		return false
	}
	if session.AutoAttackActive || session.CastingAbilityID != "" {
		return true
	}
	if duel := s.findDuelForCharacterLocked(session.CharacterID); duel != nil &&
		(duel.State == duelStateCountdown || duel.State == duelStateActive) {
		return true
	}
	for _, mob := range s.allHostileMobsLocked() {
		if mob != nil && mob.Alive && mob.CurrentTargetCharacter == session.CharacterID {
			return true
		}
	}
	return false
}

func (s *worldServer) findTravelRoute(sourcePointID string, destinationPointID string) (travelRouteDefinition, error) {
	source, found := travelPointDefinitions[sourcePointID]
	if !found {
		return travelRouteDefinition{}, fmt.Errorf("source travel point is not available")
	}
	for _, routeID := range source.ConnectedRouteIDs {
		route, found := travelRouteDefinitions[routeID]
		if !found {
			continue
		}
		if route.FromTravelPointID == sourcePointID && route.ToTravelPointID == destinationPointID {
			return route, nil
		}
		if route.ToTravelPointID == sourcePointID && route.FromTravelPointID == destinationPointID {
			reversed := route
			reversed.FromTravelPointID = sourcePointID
			reversed.ToTravelPointID = destinationPointID
			return reversed, nil
		}
	}
	return travelRouteDefinition{}, fmt.Errorf("destination is not connected to this route master")
}

func (s *worldServer) forceDismountLocked(session *worldSessionState, reason string) {
	if session == nil || !session.MountState.CurrentlyMounted {
		return
	}
	mountID := session.MountState.SelectedMountID
	session.MountState.CurrentlyMounted = false
	session.MountState.MountedSince = 0
	session.MountState.CurrentSpeedModifier = 1.0
	observability.LogEvent("world-service", "world.mount_force_dismissed", map[string]any{
		"characterId": session.CharacterID,
		"mountId":     mountID,
		"reason":      reason,
	})
}

func (s *worldServer) persistTravelSessionLocked(session *worldSessionState) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	wasMounted := session.MountState.CurrentlyMounted
	mountedSince := session.MountState.MountedSince
	speedModifier := session.MountState.CurrentSpeedModifier
	character, err := s.store.UpdateCharacterTravelState(
		session.CharacterID,
		session.BindPoint,
		session.TravelState,
		session.MountState,
		session.CurrencyCopper,
		session.ZoneID,
		session.X,
		session.Y,
		session.Z)
	if err != nil {
		return err
	}
	session.BindPoint = character.BindPoint
	session.TravelState = character.TravelState
	persistedMount := character.MountState
	if wasMounted {
		persistedMount.CurrentlyMounted = true
		persistedMount.MountedSince = mountedSince
		persistedMount.CurrentSpeedModifier = speedModifier
	}
	session.MountState = persistedMount
	session.CurrencyCopper = character.CurrencyCopper
	s.syncSessionZoneOwnershipLocked(session)
	return nil
}

func (s *worldServer) buildTravelStateResponseLocked(session *worldSessionState) map[string]any {
	if session == nil {
		return map[string]any{}
	}
	session.BindPoint = platform.NormalizeCharacterBindPoint(session.CharacterID, session.BindPoint)
	session.TravelState = platform.NormalizeCharacterTravelState(session.TravelState)
	return map[string]any{
		"bindPoint":          buildBindPointSummary(session.BindPoint),
		"recall":             s.buildRecallSummaryLocked(session),
		"travelPoints":       s.buildTravelPointSummariesLocked(session),
		"availableRoutes":    s.buildTravelDestinationsResponseLocked(session, ""),
		"lastTravelPointId":  session.TravelState.LastTravelPointID,
		"currentlyTraveling": session.CurrentlyTraveling,
	}
}

func (s *worldServer) buildRecallSummaryLocked(session *worldSessionState) map[string]any {
	now := time.Now().Unix()
	readyAt := session.TravelState.RecallReadyAt
	remaining := int64(0)
	if readyAt > now {
		remaining = readyAt - now
	}
	return map[string]any{
		"displayName":           "Return Signal",
		"cooldownSeconds":       recallCooldownSeconds,
		"readyAt":               readyAt,
		"remainingSeconds":      remaining,
		"canUse":                remaining == 0 && s.validateTransportAllowedLocked(session) == nil,
		"blockedReason":         s.travelBlockedReasonLocked(session),
		"interruptible":         false,
		"instant":               true,
		"persistsAcrossRestart": true,
	}
}

func (s *worldServer) buildTravelPointSummariesLocked(session *worldSessionState) []map[string]any {
	points := make([]map[string]any, 0, len(travelPointDefinitions))
	discovered := map[string]bool{}
	for _, pointID := range platform.NormalizeCharacterTravelState(session.TravelState).DiscoveredTravelPointIDs {
		discovered[pointID] = true
	}
	for _, point := range travelPointDefinitions {
		points = append(points, map[string]any{
			"travelPointId":     point.ID,
			"displayName":       point.DisplayName,
			"zoneId":            point.ZoneID,
			"x":                 point.X,
			"y":                 point.Y,
			"z":                 point.Z,
			"discovered":        discovered[point.ID],
			"connectedRouteIds": platform.NormalizeStringIDs(point.ConnectedRouteIDs),
			"requiredLevel":     point.RequiredLevel,
		})
	}
	return points
}

func (s *worldServer) buildTravelDestinationsResponseLocked(session *worldSessionState, sourcePointID string) map[string]any {
	source, err := s.travelSourceForSessionLocked(session, sourcePointID)
	if err != nil {
		return map[string]any{
			"sourcePointId": "",
			"destinations":  []map[string]any{},
			"error":         err.Error(),
		}
	}
	session.TravelState = platform.NormalizeCharacterTravelState(session.TravelState)
	destinations := make([]map[string]any, 0, len(source.ConnectedRouteIDs))
	for _, routeID := range source.ConnectedRouteIDs {
		route, found := travelRouteDefinitions[routeID]
		if !found {
			continue
		}
		destinationID := route.ToTravelPointID
		if destinationID == source.ID {
			destinationID = route.FromTravelPointID
		}
		destination := travelPointDefinitions[destinationID]
		unlocked := stringIDSetContains(session.TravelState.DiscoveredTravelPointIDs, destination.ID)
		reason := ""
		if !unlocked {
			reason = "Not discovered"
		} else if route.RequiredLevel > 0 && session.Level < route.RequiredLevel {
			unlocked = false
			reason = fmt.Sprintf("Requires level %d", route.RequiredLevel)
		} else if route.CostCopper > 0 && session.CurrencyCopper < route.CostCopper {
			unlocked = false
			reason = "Not enough copper"
		}
		destinations = append(destinations, map[string]any{
			"routeId":       route.ID,
			"travelPointId": destination.ID,
			"displayName":   destination.DisplayName,
			"zoneId":        destination.ZoneID,
			"x":             destination.X,
			"y":             destination.Y,
			"z":             destination.Z,
			"costCopper":    route.CostCopper,
			"requiredLevel": route.RequiredLevel,
			"available":     unlocked,
			"lockedReason":  reason,
			"discovered":    stringIDSetContains(session.TravelState.DiscoveredTravelPointIDs, destination.ID),
		})
	}
	return map[string]any{
		"sourcePointId": source.ID,
		"displayName":   source.DisplayName,
		"destinations":  destinations,
	}
}

func (s *worldServer) travelSourceForSessionLocked(session *worldSessionState, sourcePointID string) (travelPointDefinition, error) {
	if sourcePointID != "" {
		source, found := travelPointDefinitions[sourcePointID]
		if !found {
			return travelPointDefinition{}, fmt.Errorf("source travel point is not available")
		}
		return source, nil
	}
	if session == nil || session.CurrentTargetID == "" {
		return travelPointDefinition{}, fmt.Errorf("right-click a route master first")
	}
	for _, point := range travelPointDefinitions {
		if point.ServiceEntityID == session.CurrentTargetID {
			return point, nil
		}
	}
	return travelPointDefinition{}, fmt.Errorf("right-click a route master first")
}

func (s *worldServer) buildMountCollectionResponseLocked(session *worldSessionState) map[string]any {
	if session == nil {
		return map[string]any{}
	}
	session.MountState = normalizeSessionMountState(session.MountState)
	mounts := make([]map[string]any, 0, len(mountDefinitions))
	for _, mount := range mountDefinitions {
		unlocked := stringIDSetContains(session.MountState.UnlockedMountIDs, mount.ID)
		mounts = append(mounts, map[string]any{
			"mountId":          mount.ID,
			"displayName":      mount.DisplayName,
			"requiredLevel":    mount.RequiredLevel,
			"speedMultiplier":  mount.SpeedMultiplier,
			"source":           mount.Source,
			"accountWide":      mount.AccountWide,
			"implemented":      mount.Implemented,
			"unlocked":         unlocked,
			"selected":         session.MountState.SelectedMountID == mount.ID,
			"currentlyMounted": session.MountState.CurrentlyMounted && session.MountState.SelectedMountID == mount.ID,
		})
	}
	return map[string]any{
		"category":             "mounts",
		"selectedMountId":      session.MountState.SelectedMountID,
		"currentlyMounted":     session.MountState.CurrentlyMounted,
		"mountedSince":         session.MountState.MountedSince,
		"currentSpeedModifier": session.MountState.CurrentSpeedModifier,
		"mounts":               mounts,
	}
}

func buildBindPointSummary(bindPoint platform.CharacterBindPoint) map[string]any {
	return map[string]any{
		"characterId":    bindPoint.CharacterID,
		"zoneId":         bindPoint.ZoneID,
		"x":              bindPoint.X,
		"y":              bindPoint.Y,
		"z":              bindPoint.Z,
		"bindLocationId": bindPoint.BindLocationID,
		"displayName":    bindPoint.DisplayName,
		"setAt":          bindPoint.SetAt,
	}
}

func normalizeSessionMountState(mountState platform.CharacterMountState) platform.CharacterMountState {
	unlocked := platform.NormalizeStringIDs(mountState.UnlockedMountIDs)
	if mountState.SelectedMountID != "" && !stringIDSetContains(unlocked, mountState.SelectedMountID) {
		mountState.SelectedMountID = ""
	}
	mountState.UnlockedMountIDs = unlocked
	if !mountState.CurrentlyMounted {
		mountState.MountedSince = 0
		mountState.CurrentSpeedModifier = 1.0
	} else if mountState.CurrentSpeedModifier <= 0 {
		mountState.CurrentSpeedModifier = 1.0
	}
	return mountState
}

func (s *worldServer) travelBlockedReasonLocked(session *worldSessionState) string {
	if err := s.validateTransportAllowedLocked(session); err != nil {
		return err.Error()
	}
	return ""
}
