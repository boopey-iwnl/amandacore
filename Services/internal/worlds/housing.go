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
	housingTemplatePersonalRoomID = "stonewake_personal_room"
	housingServicePersonalRoomID  = "personal_room"
	housingEntranceEntityID       = "obj_stonewake_room_door"
	housingExitEntityID           = "obj_personal_room_exit"
	housingStorageEntityID        = "obj_personal_storage_chest"
)

type housingTemplateDefinition struct {
	ID            string
	DisplayName   string
	ZoneID        string
	EntryPosition worldPosition
	ExitPosition  worldPosition
	Storage       worldPosition
	StorageRadius float64
	ExitRadius    float64
	PlacementMinX float64
	PlacementMinY float64
	PlacementMaxX float64
	PlacementMaxY float64
	PlacementZ    float64
	DefaultReturn worldPosition
}

type decorationDefinition struct {
	DecorationID string
	DisplayName  string
	Kind         string
}

var housingTemplates = map[string]housingTemplateDefinition{
	housingTemplatePersonalRoomID: {
		ID:            housingTemplatePersonalRoomID,
		DisplayName:   "Stonewake Personal Room",
		ZoneID:        housingZoneID,
		EntryPosition: worldPosition{ZoneID: housingZoneID, X: 6, Y: 6, Z: 0},
		ExitPosition:  worldPosition{ZoneID: housingZoneID, X: 4, Y: 4, Z: 0},
		Storage:       worldPosition{ZoneID: housingZoneID, X: 10, Y: 7, Z: 0},
		StorageRadius: 8,
		ExitRadius:    7,
		PlacementMinX: 8,
		PlacementMinY: 7,
		PlacementMaxX: 34,
		PlacementMaxY: 23,
		PlacementZ:    0,
		DefaultReturn: worldPosition{ZoneID: defaultZoneID, X: 18, Y: 14, Z: 0},
	},
}

var decorationCatalog = map[string]decorationDefinition{
	"simple_cot":    {DecorationID: "simple_cot", DisplayName: "Simple Cot", Kind: "rest"},
	"supply_crate":  {DecorationID: "supply_crate", DisplayName: "Supply Crate", Kind: "storage_prop"},
	"wall_torch":    {DecorationID: "wall_torch", DisplayName: "Wall Torch", Kind: "light"},
	"training_rack": {DecorationID: "training_rack", DisplayName: "Training Rack", Kind: "training_prop"},
	"small_table":   {DecorationID: "small_table", DisplayName: "Small Table", Kind: "table"},
}

var housingFriendlyNPCs = []friendlyNPCDefinition{
	{
		ID:          housingEntranceEntityID,
		ZoneID:      defaultZoneID,
		DisplayName: "Hearthwatch Room Door",
		Kind:        housingEntranceKind,
		X:           18.0,
		Y:           14.0,
		Z:           0.0,
		AIState:     "housing_entrance",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "housing", ServiceID: housingServicePersonalRoomID, Label: "Enter Personal Room"},
		},
	},
}

var housingZoneDefinitions = []zoneDefinition{
	{
		ID:          housingZoneID,
		DisplayName: "Personal Room",
		LevelBand:   "Personal",
		Bounds:      zoneBoundsDefinition{MinX: 0, MinY: 0, MaxX: 40, MaxY: 28},
		Roads: []zoneRoadDefinition{
			{
				ID:          "personal_room_floor",
				DisplayName: "Room Interior",
				Points: []zonePointDefinition{
					{ID: "room_entry", DisplayName: "Entry", Type: "entry", X: 6, Y: 6},
					{ID: "room_storage", DisplayName: "Storage Chest", Type: "storage", X: 10, Y: 7},
					{ID: "room_placement", DisplayName: "Placement Area", Type: "placement", X: 22, Y: 15},
				},
			},
		},
		Landmarks: []zonePointDefinition{
			{ID: "room_storage_chest", DisplayName: "Storage Chest", Type: "storage", X: 10, Y: 7},
			{ID: "room_placement_area", DisplayName: "Placement Area", Type: "placement", X: 22, Y: 15},
		},
		Transitions: []zonePointDefinition{
			{ID: "room_exit", DisplayName: "Exit", Type: "housing_exit", X: 4, Y: 4},
		},
	},
}

func (s *worldServer) handleHousingStatus(w http.ResponseWriter, r *http.Request) {
	session, ok := s.sessionFromQuery(w, r)
	if !ok {
		return
	}

	response, err := s.buildHousingStatusLocked(session)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "housing_status_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}

func (s *worldServer) handleHousingEnter(w http.ResponseWriter, r *http.Request) {
	var request housingEnterRequest
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
	if err := s.enterHousingLocked(session); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "housing_enter_failed", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleHousingLeave(w http.ResponseWriter, r *http.Request) {
	var request housingLeaveRequest
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
	template := housingTemplates[housingTemplatePersonalRoomID]
	if !s.sessionInsideHousingLocked(session) {
		httpapi.Error(w, http.StatusBadRequest, "housing_leave_failed", "You are not inside housing.")
		return
	}
	if distance2D(session.X, session.Y, template.ExitPosition.X, template.ExitPosition.Y) > template.ExitRadius {
		httpapi.Error(w, http.StatusBadRequest, "housing_leave_failed", "Move closer to the room exit.")
		return
	}
	if err := s.returnSessionFromHousingLocked(session, "manual_exit"); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "housing_leave_failed", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleHousingStorageList(w http.ResponseWriter, r *http.Request) {
	session, ok := s.sessionFromQuery(w, r)
	if !ok {
		return
	}
	if !s.sessionCanUseHousingStorageLocked(session) {
		httpapi.Error(w, http.StatusBadRequest, "housing_storage_unavailable", "Move closer to the personal storage chest.")
		return
	}
	storage, err := s.store.ListHousingStorage(session.HousingSpaceID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "housing_storage_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"storage": buildHousingStoragePayload(storage),
	})
}

func (s *worldServer) handleHousingStorageDeposit(w http.ResponseWriter, r *http.Request) {
	var request housingStorageDepositRequest
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
	if err := s.depositHousingStorageLocked(session, request.InventorySlotIndex, request.StorageSlotIndex, request.StackCount); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "housing_storage_deposit_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleHousingStorageWithdraw(w http.ResponseWriter, r *http.Request) {
	var request housingStorageWithdrawRequest
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
	if err := s.withdrawHousingStorageLocked(session, request.StorageSlotIndex, request.InventorySlotIndex, request.StackCount); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "housing_storage_withdraw_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleHousingStorageMove(w http.ResponseWriter, r *http.Request) {
	var request housingStorageMoveRequest
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
	if err := s.moveHousingStorageLocked(session, request.FromSlotIndex, request.ToSlotIndex); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "housing_storage_move_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleHousingDecorationsList(w http.ResponseWriter, r *http.Request) {
	session, ok := s.sessionFromQuery(w, r)
	if !ok {
		return
	}
	if !s.sessionInsideHousingLocked(session) {
		httpapi.Error(w, http.StatusBadRequest, "housing_decorations_unavailable", "You are not inside housing.")
		return
	}
	placements, err := s.store.ListHousingDecorations(session.HousingSpaceID)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "housing_decorations_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"decorations": buildDecorationsPayload(placements),
	})
}

func (s *worldServer) handleHousingDecorationPlace(w http.ResponseWriter, r *http.Request) {
	var request decorationPlaceRequest
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
	if err := s.placeDecorationLocked(session, request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "housing_decoration_place_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleHousingDecorationRemove(w http.ResponseWriter, r *http.Request) {
	var request decorationRemoveRequest
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
	if err := s.removeDecorationLocked(session, request.PlacementID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "housing_decoration_remove_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) sessionFromQuery(w http.ResponseWriter, r *http.Request) (*worldSessionState, bool) {
	token := r.URL.Query().Get("worldSessionToken")
	if token == "" {
		httpapi.Error(w, http.StatusBadRequest, "missing_token", "worldSessionToken query parameter is required.")
		return nil, false
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[token]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return nil, false
	}
	return session, true
}

func (s *worldServer) enterHousingLocked(session *worldSessionState) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if !session.Alive {
		return fmt.Errorf("dead players cannot enter housing")
	}
	if session.InstanceID != "" {
		return fmt.Errorf("leave the dungeon before entering housing")
	}
	if session.HousingSpaceID != "" {
		return fmt.Errorf("you are already inside housing")
	}

	entrance, ok := s.findFriendlyNPCDefinition(housingEntranceEntityID)
	if !ok || session.ZoneID != entrance.ZoneID || distance2D(session.X, session.Y, entrance.X, entrance.Y) > entrance.Radius {
		return fmt.Errorf("move closer to the %s", entrance.DisplayName)
	}

	_, space, err := s.store.GetOrCreateHousingForCharacter(session.CharacterID, housingTemplatePersonalRoomID)
	if err != nil {
		return err
	}
	template := housingTemplates[space.TemplateID]
	if template.ID == "" {
		template = housingTemplates[housingTemplatePersonalRoomID]
	}

	returnZoneID, returnX, returnY, returnZ := session.ZoneID, session.X, session.Y, session.Z
	persistStartedAt := time.Now()
	if _, err := s.store.UpdateCharacterState(session.CharacterID, returnZoneID, returnX, returnY, returnZ); err != nil {
		s.recordPersistenceDuration("character_state_housing_enter_return", persistStartedAt, err)
		return err
	}
	s.recordPersistenceDuration("character_state_housing_enter_return", persistStartedAt, nil)

	persistStartedAt = time.Now()
	if _, err := s.store.UpdateHousingVisit(space.HousingSpaceID, returnZoneID, returnX, returnY, returnZ); err != nil {
		s.recordPersistenceDuration("housing_visit", persistStartedAt, err)
		return err
	}
	s.recordPersistenceDuration("housing_visit", persistStartedAt, nil)

	s.housingInstanceCounter++
	session.HousingSpaceID = space.HousingSpaceID
	session.HousingInstanceID = fmt.Sprintf("housing_%06d", s.housingInstanceCounter)
	session.ReturnZoneID = returnZoneID
	session.ReturnX = returnX
	session.ReturnY = returnY
	session.ReturnZ = returnZ
	s.forceDismountLocked(session, "housing_enter")
	session.ZoneID = template.ZoneID
	session.X = template.EntryPosition.X
	session.Y = template.EntryPosition.Y
	session.Z = template.EntryPosition.Z
	s.resetSessionCombatStateLocked(session, "housing_enter")
	s.clearMobAggroForCharacterLocked(session.CharacterID)
	s.sendSystemMessageLocked("Entered personal room.", recipientSet(session.CharacterID))
	observability.LogEvent("world-service", "world.housing_entered", map[string]any{
		"characterId":    session.CharacterID,
		"housingSpaceId": space.HousingSpaceID,
		"templateId":     template.ID,
		"returnZoneId":   returnZoneID,
	})
	return nil
}

func (s *worldServer) returnSessionFromHousingLocked(session *worldSessionState, reason string) error {
	if session == nil || session.HousingSpaceID == "" {
		return fmt.Errorf("you are not inside housing")
	}

	returnPosition := s.housingReturnPositionLocked(session)
	persistStartedAt := time.Now()
	_, err := s.store.UpdateCharacterState(session.CharacterID, returnPosition.ZoneID, returnPosition.X, returnPosition.Y, returnPosition.Z)
	s.recordPersistenceDuration("character_state_housing_return", persistStartedAt, err)
	if err != nil {
		return err
	}

	housingSpaceID := session.HousingSpaceID
	session.HousingSpaceID = ""
	session.HousingInstanceID = ""
	session.ReturnZoneID = ""
	session.ReturnX = 0
	session.ReturnY = 0
	session.ReturnZ = 0
	session.ZoneID = returnPosition.ZoneID
	session.X = returnPosition.X
	session.Y = returnPosition.Y
	session.Z = returnPosition.Z
	s.resetSessionCombatStateLocked(session, reason)
	observability.LogEvent("world-service", "world.housing_exited", map[string]any{
		"characterId":    session.CharacterID,
		"housingSpaceId": housingSpaceID,
		"reason":         reason,
		"zoneId":         session.ZoneID,
		"x":              session.X,
		"y":              session.Y,
	})
	return nil
}

func (s *worldServer) housingReturnPositionLocked(session *worldSessionState) worldPosition {
	template := housingTemplates[housingTemplatePersonalRoomID]
	if session.ReturnZoneID != "" {
		return worldPosition{ZoneID: session.ReturnZoneID, X: session.ReturnX, Y: session.ReturnY, Z: session.ReturnZ}
	}
	return template.DefaultReturn
}

func (s *worldServer) sessionInsideHousingLocked(session *worldSessionState) bool {
	return session != nil && session.HousingSpaceID != "" && session.ZoneID == housingZoneID
}

func (s *worldServer) sessionCanUseHousingStorageLocked(session *worldSessionState) bool {
	if !s.sessionInsideHousingLocked(session) {
		return false
	}
	template := housingTemplates[housingTemplatePersonalRoomID]
	return distance2D(session.X, session.Y, template.Storage.X, template.Storage.Y) <= template.StorageRadius
}

func (s *worldServer) depositHousingStorageLocked(session *worldSessionState, inventorySlotIndex int, storageSlotIndex *int, stackCount int) error {
	if !s.sessionCanUseHousingStorageLocked(session) {
		return fmt.Errorf("move closer to the personal storage chest")
	}

	inventory := platform.NormalizeInventorySlots(session.Inventory)
	if inventorySlotIndex < 0 || inventorySlotIndex >= platform.InventorySlotCount {
		return fmt.Errorf("inventory slot is out of range")
	}
	source := inventory[inventorySlotIndex]
	if source.ItemID == "" || source.StackCount <= 0 {
		return fmt.Errorf("inventory slot is empty")
	}
	item, found := findItemDefinition(source.ItemID)
	if !found {
		return fmt.Errorf("item is not defined")
	}
	if item.Type == itemTypeQuest {
		return fmt.Errorf("quest items cannot be stored")
	}
	if stackCount <= 0 {
		stackCount = source.StackCount
	}
	if stackCount > source.StackCount {
		return fmt.Errorf("not enough items in slot")
	}

	storage, err := s.store.ListHousingStorage(session.HousingSpaceID)
	if err != nil {
		return err
	}
	nextInventory := platform.NormalizeInventorySlots(inventory)
	nextStorage := platform.NormalizeHousingStorageSlots(storage)
	_, removedCount, err := removeInventorySlotCount(&nextInventory, inventorySlotIndex, stackCount)
	if err != nil {
		return err
	}
	if err := addDefinedItemToHousingStorage(&nextStorage, item, storageSlotIndex, removedCount); err != nil {
		return err
	}

	character, _, err := s.store.UpdateCharacterInventoryAndHousingStorage(session.CharacterID, session.HousingSpaceID, nextInventory, nextStorage)
	if err != nil {
		return err
	}
	session.Inventory = platform.NormalizeInventorySlots(character.Inventory)
	observability.LogEvent("world-service", "world.housing_storage_deposit", map[string]any{
		"characterId":    session.CharacterID,
		"housingSpaceId": session.HousingSpaceID,
		"itemId":         item.ItemID,
		"stackCount":     removedCount,
	})
	return nil
}

func (s *worldServer) withdrawHousingStorageLocked(session *worldSessionState, storageSlotIndex int, inventorySlotIndex *int, stackCount int) error {
	if !s.sessionCanUseHousingStorageLocked(session) {
		return fmt.Errorf("move closer to the personal storage chest")
	}

	storage, err := s.store.ListHousingStorage(session.HousingSpaceID)
	if err != nil {
		return err
	}
	nextStorage := platform.NormalizeHousingStorageSlots(storage)
	source, removedCount, err := removeHousingStorageSlotCount(&nextStorage, storageSlotIndex, stackCount)
	if err != nil {
		return err
	}
	item, found := findItemDefinition(source.ItemID)
	if !found {
		return fmt.Errorf("item is not defined")
	}

	nextInventory := platform.NormalizeInventorySlots(session.Inventory)
	if err := addDefinedItemToInventoryAt(&nextInventory, item, inventorySlotIndex, removedCount); err != nil {
		return err
	}
	character, _, err := s.store.UpdateCharacterInventoryAndHousingStorage(session.CharacterID, session.HousingSpaceID, nextInventory, nextStorage)
	if err != nil {
		return err
	}
	session.Inventory = platform.NormalizeInventorySlots(character.Inventory)
	observability.LogEvent("world-service", "world.housing_storage_withdraw", map[string]any{
		"characterId":    session.CharacterID,
		"housingSpaceId": session.HousingSpaceID,
		"itemId":         item.ItemID,
		"stackCount":     removedCount,
	})
	return nil
}

func (s *worldServer) moveHousingStorageLocked(session *worldSessionState, fromSlotIndex int, toSlotIndex int) error {
	if !s.sessionCanUseHousingStorageLocked(session) {
		return fmt.Errorf("move closer to the personal storage chest")
	}
	if fromSlotIndex < 0 || fromSlotIndex >= platform.HousingStorageSlotCount ||
		toSlotIndex < 0 || toSlotIndex >= platform.HousingStorageSlotCount {
		return fmt.Errorf("storage slot is out of range")
	}
	if fromSlotIndex == toSlotIndex {
		return nil
	}

	storage, err := s.store.ListHousingStorage(session.HousingSpaceID)
	if err != nil {
		return err
	}
	slots := platform.NormalizeHousingStorageSlots(storage)
	fromSlot := slots[fromSlotIndex]
	toSlot := slots[toSlotIndex]
	if fromSlot.ItemID == "" || fromSlot.StackCount <= 0 {
		return fmt.Errorf("source storage slot is empty")
	}
	item, found := findItemDefinition(fromSlot.ItemID)
	if found && item.Stackable && toSlot.ItemID == fromSlot.ItemID {
		available := item.MaxStack - toSlot.StackCount
		if available > 0 {
			moved := minInt(fromSlot.StackCount, available)
			slots[toSlotIndex].StackCount += moved
			slots[fromSlotIndex].StackCount -= moved
			if slots[fromSlotIndex].StackCount <= 0 {
				slots[fromSlotIndex] = platform.HousingStorageSlot{SlotIndex: fromSlotIndex}
			}
			_, err := s.store.UpdateHousingStorage(session.CharacterID, session.HousingSpaceID, slots)
			return err
		}
	}

	slots[fromSlotIndex] = platform.HousingStorageSlot{SlotIndex: fromSlotIndex, ItemID: toSlot.ItemID, DisplayName: toSlot.DisplayName, StackCount: toSlot.StackCount}
	slots[toSlotIndex] = platform.HousingStorageSlot{SlotIndex: toSlotIndex, ItemID: fromSlot.ItemID, DisplayName: fromSlot.DisplayName, StackCount: fromSlot.StackCount}
	_, err = s.store.UpdateHousingStorage(session.CharacterID, session.HousingSpaceID, slots)
	return err
}

func (s *worldServer) placeDecorationLocked(session *worldSessionState, request decorationPlaceRequest) error {
	if !s.sessionInsideHousingLocked(session) {
		return fmt.Errorf("you are not inside housing")
	}
	definition, found := decorationCatalog[request.DecorationID]
	if !found {
		return fmt.Errorf("decoration is not available")
	}
	template := housingTemplates[housingTemplatePersonalRoomID]
	if request.X < template.PlacementMinX || request.X > template.PlacementMaxX ||
		request.Y < template.PlacementMinY || request.Y > template.PlacementMaxY {
		return fmt.Errorf("decoration placement is outside the allowed area")
	}

	placements, err := s.store.ListHousingDecorations(session.HousingSpaceID)
	if err != nil {
		return err
	}
	if len(placements) >= platform.HousingDecorationLimit {
		return fmt.Errorf("decoration placement limit reached")
	}

	placement := platform.DecorationPlacement{
		PlacementID:    "decor_" + randomWorldToken(),
		HousingSpaceID: session.HousingSpaceID,
		DecorationID:   definition.DecorationID,
		DisplayName:    definition.DisplayName,
		X:              request.X,
		Y:              request.Y,
		Z:              template.PlacementZ,
		RotationYaw:    request.RotationYaw,
		CreatedAt:      time.Now().Unix(),
	}
	placements = append(placements, placement)
	_, err = s.store.SaveHousingDecorations(session.CharacterID, session.HousingSpaceID, placements)
	if err != nil {
		return err
	}
	observability.LogEvent("world-service", "world.housing_decoration_placed", map[string]any{
		"characterId":    session.CharacterID,
		"housingSpaceId": session.HousingSpaceID,
		"placementId":    placement.PlacementID,
		"decorationId":   placement.DecorationID,
	})
	return nil
}

func (s *worldServer) removeDecorationLocked(session *worldSessionState, placementID string) error {
	if !s.sessionInsideHousingLocked(session) {
		return fmt.Errorf("you are not inside housing")
	}
	if placementID == "" {
		return fmt.Errorf("decoration placement is required")
	}

	placements, err := s.store.ListHousingDecorations(session.HousingSpaceID)
	if err != nil {
		return err
	}
	next := make([]platform.DecorationPlacement, 0, len(placements))
	removed := false
	for _, placement := range placements {
		if placement.PlacementID == placementID {
			removed = true
			continue
		}
		next = append(next, placement)
	}
	if !removed {
		return fmt.Errorf("decoration placement was not found")
	}
	_, err = s.store.SaveHousingDecorations(session.CharacterID, session.HousingSpaceID, next)
	if err != nil {
		return err
	}
	observability.LogEvent("world-service", "world.housing_decoration_removed", map[string]any{
		"characterId":    session.CharacterID,
		"housingSpaceId": session.HousingSpaceID,
		"placementId":    placementID,
	})
	return nil
}

func (s *worldServer) buildHousingResponseLocked(session *worldSessionState) map[string]any {
	response, err := s.buildHousingStatusLocked(session)
	if err != nil {
		return map[string]any{
			"available": false,
			"error":     err.Error(),
		}
	}
	return response
}

func (s *worldServer) buildHousingStatusLocked(session *worldSessionState) (map[string]any, error) {
	if session == nil {
		return nil, fmt.Errorf("world session token was not found")
	}
	entitlement, space, err := s.store.GetOrCreateHousingForCharacter(session.CharacterID, housingTemplatePersonalRoomID)
	if err != nil {
		return nil, err
	}

	inHousing := s.sessionInsideHousingLocked(session)
	response := map[string]any{
		"available":        true,
		"inHousing":        inHousing,
		"housingSpaceId":   space.HousingSpaceID,
		"templateId":       space.TemplateID,
		"unlocked":         entitlement.Unlocked,
		"storageSlotCount": platform.HousingStorageSlotCount,
		"maxDecorations":   platform.HousingDecorationLimit,
		"returnLocation": map[string]any{
			"zoneId": space.ReturnZoneID,
			"x":      space.ReturnX,
			"y":      space.ReturnY,
			"z":      space.ReturnZ,
		},
		"decorations": buildDecorationsPayload(nil),
	}
	if inHousing {
		if storage, err := s.store.ListHousingStorage(space.HousingSpaceID); err == nil {
			response["storage"] = buildHousingStoragePayload(storage)
		}
		if placements, err := s.store.ListHousingDecorations(space.HousingSpaceID); err == nil {
			response["decorations"] = buildDecorationsPayload(placements)
		}
	}
	return response, nil
}

func (s *worldServer) buildHousingEntitiesLocked(session *worldSessionState) []sessionEntity {
	if !s.sessionInsideHousingLocked(session) {
		return nil
	}
	template := housingTemplates[housingTemplatePersonalRoomID]
	entities := []sessionEntity{
		{
			ID:               housingExitEntityID,
			DisplayName:      "Room Exit",
			Kind:             housingExitKind,
			InteractionLabel: "Leave Personal Room",
			X:                template.ExitPosition.X,
			Y:                template.ExitPosition.Y,
			Z:                template.ExitPosition.Z,
			Health:           1,
			MaxHealth:        1,
			Alive:            true,
			Targetable:       true,
			AIState:          "housing_exit",
			Services: []npcService{
				{Type: "housing_exit", ServiceID: housingServicePersonalRoomID, Label: "Leave Personal Room"},
			},
		},
		{
			ID:               housingStorageEntityID,
			DisplayName:      "Personal Storage Chest",
			Kind:             housingStorageKind,
			InteractionLabel: "Open Personal Storage",
			X:                template.Storage.X,
			Y:                template.Storage.Y,
			Z:                template.Storage.Z,
			Health:           1,
			MaxHealth:        1,
			Alive:            true,
			Targetable:       true,
			AIState:          "housing_storage",
			Services: []npcService{
				{Type: "housing_storage", ServiceID: housingServicePersonalRoomID, Label: "Open Storage"},
			},
		},
	}

	placements, err := s.store.ListHousingDecorations(session.HousingSpaceID)
	if err != nil {
		return entities
	}
	for _, placement := range placements {
		entities = append(entities, sessionEntity{
			ID:               placement.PlacementID,
			DisplayName:      placement.DisplayName,
			Kind:             housingDecorationKind,
			InteractionLabel: "Decoration",
			X:                placement.X,
			Y:                placement.Y,
			Z:                placement.Z,
			Health:           1,
			MaxHealth:        1,
			Alive:            true,
			Targetable:       true,
			AIState:          placement.DecorationID,
		})
	}
	return entities
}

func (s *worldServer) findHousingEntityLocked(session *worldSessionState, entityID string) (sessionEntity, bool) {
	if entityID == "" || !s.sessionInsideHousingLocked(session) {
		return sessionEntity{}, false
	}
	for _, entity := range s.buildHousingEntitiesLocked(session) {
		if entity.ID == entityID {
			return entity, true
		}
	}
	return sessionEntity{}, false
}

func buildHousingStoragePayload(slots []platform.HousingStorageSlot) map[string]any {
	return map[string]any{
		"slotCount": platform.HousingStorageSlotCount,
		"slots":     platform.NormalizeHousingStorageSlots(slots),
	}
}

func buildDecorationsPayload(placements []platform.DecorationPlacement) map[string]any {
	catalog := make([]map[string]any, 0, len(decorationCatalog))
	for _, decoration := range decorationCatalog {
		catalog = append(catalog, map[string]any{
			"decorationId": decoration.DecorationID,
			"displayName":  decoration.DisplayName,
			"kind":         decoration.Kind,
		})
	}
	return map[string]any{
		"maxPlaced": platform.HousingDecorationLimit,
		"catalog":   catalog,
		"placed":    platform.NormalizeDecorationPlacements(placements),
	}
}

func addDefinedItemToHousingStorage(storage *[]platform.HousingStorageSlot, item itemDefinition, preferredSlotIndex *int, stackCount int) error {
	if item.ItemID == "" || stackCount <= 0 {
		return nil
	}

	slots := platform.NormalizeHousingStorageSlots(*storage)
	maxStack := item.MaxStack
	if maxStack <= 0 || !item.Stackable {
		maxStack = 1
	}

	if preferredSlotIndex != nil {
		if *preferredSlotIndex < 0 || *preferredSlotIndex >= platform.HousingStorageSlotCount {
			return fmt.Errorf("storage slot is out of range")
		}
		if !slotCanAcceptItem(slots[*preferredSlotIndex].ItemID, slots[*preferredSlotIndex].StackCount, item, stackCount) {
			return fmt.Errorf("storage slot cannot accept that item stack")
		}
		addItemToHousingSlot(&slots[*preferredSlotIndex], *preferredSlotIndex, item, stackCount)
		*storage = slots
		return nil
	}

	remaining := stackCount
	if item.Stackable {
		for index := range slots {
			if slots[index].ItemID != item.ItemID || slots[index].StackCount >= maxStack {
				continue
			}
			available := maxStack - slots[index].StackCount
			added := minInt(remaining, available)
			slots[index].StackCount += added
			remaining -= added
			if remaining <= 0 {
				*storage = slots
				return nil
			}
		}
	}

	for index := range slots {
		if slots[index].ItemID != "" && slots[index].StackCount > 0 {
			continue
		}
		added := 1
		if item.Stackable {
			added = minInt(remaining, maxStack)
		}
		slots[index] = platform.HousingStorageSlot{
			SlotIndex:   index,
			ItemID:      item.ItemID,
			DisplayName: item.DisplayName,
			StackCount:  added,
		}
		remaining -= added
		if remaining <= 0 {
			*storage = slots
			return nil
		}
	}

	return fmt.Errorf("personal storage is full")
}

func removeHousingStorageSlotCount(storage *[]platform.HousingStorageSlot, slotIndex int, stackCount int) (platform.HousingStorageSlot, int, error) {
	if slotIndex < 0 || slotIndex >= platform.HousingStorageSlotCount {
		return platform.HousingStorageSlot{}, 0, fmt.Errorf("storage slot is out of range")
	}

	slots := platform.NormalizeHousingStorageSlots(*storage)
	slot := slots[slotIndex]
	if slot.ItemID == "" || slot.StackCount <= 0 {
		return platform.HousingStorageSlot{}, 0, fmt.Errorf("storage slot is empty")
	}
	if stackCount <= 0 {
		stackCount = slot.StackCount
	}
	if stackCount > slot.StackCount {
		return platform.HousingStorageSlot{}, 0, fmt.Errorf("not enough items in storage slot")
	}

	slots[slotIndex].StackCount -= stackCount
	if slots[slotIndex].StackCount <= 0 {
		slots[slotIndex] = platform.HousingStorageSlot{SlotIndex: slotIndex}
	}
	*storage = slots
	return slot, stackCount, nil
}

func addDefinedItemToInventoryAt(inventory *[]platform.CharacterInventorySlot, item itemDefinition, preferredSlotIndex *int, stackCount int) error {
	if preferredSlotIndex == nil {
		return addDefinedItemToInventory(inventory, item, stackCount)
	}
	if *preferredSlotIndex < 0 || *preferredSlotIndex >= platform.InventorySlotCount {
		return fmt.Errorf("inventory slot is out of range")
	}

	slots := platform.NormalizeInventorySlots(*inventory)
	if !slotCanAcceptItem(slots[*preferredSlotIndex].ItemID, slots[*preferredSlotIndex].StackCount, item, stackCount) {
		return fmt.Errorf("inventory slot cannot accept that item stack")
	}
	addItemToInventorySlot(&slots[*preferredSlotIndex], *preferredSlotIndex, item, stackCount)
	*inventory = slots
	return nil
}

func slotCanAcceptItem(slotItemID string, slotStackCount int, item itemDefinition, addCount int) bool {
	maxStack := item.MaxStack
	if maxStack <= 0 || !item.Stackable {
		maxStack = 1
	}
	if slotItemID == "" || slotStackCount <= 0 {
		return addCount <= maxStack
	}
	return item.Stackable && slotItemID == item.ItemID && slotStackCount+addCount <= maxStack
}

func addItemToHousingSlot(slot *platform.HousingStorageSlot, slotIndex int, item itemDefinition, stackCount int) {
	if slot.ItemID == "" || slot.StackCount <= 0 {
		*slot = platform.HousingStorageSlot{
			SlotIndex:   slotIndex,
			ItemID:      item.ItemID,
			DisplayName: item.DisplayName,
			StackCount:  stackCount,
		}
		return
	}
	slot.StackCount += stackCount
}

func addItemToInventorySlot(slot *platform.CharacterInventorySlot, slotIndex int, item itemDefinition, stackCount int) {
	if slot.ItemID == "" || slot.StackCount <= 0 {
		*slot = platform.CharacterInventorySlot{
			SlotIndex:   slotIndex,
			ItemID:      item.ItemID,
			DisplayName: item.DisplayName,
			StackCount:  stackCount,
		}
		return
	}
	slot.StackCount += stackCount
}
