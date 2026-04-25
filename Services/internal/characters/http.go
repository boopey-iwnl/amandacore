package characters

import (
	"net/http"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

type createCharacterRequest struct {
	RealmID     string `json:"realmId"`
	DisplayName string `json:"displayName"`
	RaceID      string `json:"raceId"`
	ClassID     string `json:"classId"`
	ArchetypeID string `json:"archetypeId"`
}

func RegisterRoutes(mux *http.ServeMux, fileStore *store.FileStore) {
	mux.Handle("GET /v1/characters", httpapi.RequireSession(fileStore, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		realmID := r.URL.Query().Get("realmId")
		characters, err := fileStore.ListCharacters(session.AccountID, realmID)
		if err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "character_list_failed", err.Error())
			return
		}

		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"characters": characters})
	}))

	mux.Handle("POST /v1/characters", httpapi.RequireSession(fileStore, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		var request createCharacterRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		request.ArchetypeID, request.RaceID, request.ClassID = platform.NormalizeCharacterIdentity(
			request.ArchetypeID,
			request.RaceID,
			request.ClassID)

		character, err := fileStore.CreateCharacter(
			session.AccountID,
			request.RealmID,
			request.DisplayName,
			request.RaceID,
			request.ClassID,
			request.ArchetypeID)
		if err != nil {
			httpapi.Error(w, http.StatusConflict, "character_create_failed", err.Error())
			return
		}

		observability.LogEvent("character-service", "character.created", map[string]any{
			"accountId":   session.AccountID,
			"characterId": character.ID,
			"realmId":     character.RealmID,
			"displayName": character.DisplayName,
			"raceId":      character.RaceID,
			"classId":     character.ClassID,
			"archetypeId": character.ArchetypeID,
			"spawnZoneId": character.ZoneID,
			"spawnX":      character.PositionX,
			"spawnY":      character.PositionY,
			"spawnZ":      character.PositionZ,
		})

		httpapi.WriteJSON(w, http.StatusCreated, character)
	}))
}
