package worlds

import (
	"net/http"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/platform"
)

func (s *worldServer) handleDuelRequest(w http.ResponseWriter, r *http.Request) {
	var request duelRequest
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
	target, err := s.resolveDuelRequestTargetLocked(session, request)
	if err != nil {
		httpapi.Error(w, http.StatusBadRequest, "duel_target_invalid", err.Error())
		return
	}
	if _, err := s.requestDuelLocked(session, target); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "duel_request_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, s.buildResponse(session))
}

func (s *worldServer) handleDuelAccept(w http.ResponseWriter, r *http.Request) {
	var request duelActionRequest
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
	if _, err := s.acceptDuelLocked(session, request.DuelID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "duel_accept_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleDuelDecline(w http.ResponseWriter, r *http.Request) {
	var request duelActionRequest
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
	if err := s.declineDuelLocked(session, request.DuelID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "duel_decline_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleDuelCancel(w http.ResponseWriter, r *http.Request) {
	var request duelActionRequest
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
	if err := s.cancelDuelLocked(session, request.DuelID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "duel_cancel_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleDuelSurrender(w http.ResponseWriter, r *http.Request) {
	var request duelActionRequest
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
	if err := s.surrenderDuelLocked(session, request.DuelID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "duel_surrender_invalid", err.Error())
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func (s *worldServer) handleDuelState(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("worldSessionToken")
	if token == "" {
		httpapi.Error(w, http.StatusBadRequest, "missing_token", "worldSessionToken query parameter is required.")
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[token]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildPvPResponseLocked(session))
}

func (s *worldServer) handlePvPStats(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("worldSessionToken")
	if token == "" {
		httpapi.Error(w, http.StatusBadRequest, "missing_token", "worldSessionToken query parameter is required.")
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[token]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	stats := platform.NormalizeCharacterPvPStats(session.CharacterID, session.PvPStats)
	httpapi.WriteJSON(w, http.StatusOK, buildPvPStatsResponse(stats))
}
