package worlds

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"amandacore/services/internal/simcore"
)

type SessionLifecycleState string

const (
	SessionStateAttached            SessionLifecycleState = "attached"
	SessionStateActive              SessionLifecycleState = "active"
	SessionStateDisconnectRequested SessionLifecycleState = "disconnect_requested"
	SessionStateDisconnected        SessionLifecycleState = "disconnected"
	SessionStateReconnectPending    SessionLifecycleState = "reconnect_pending"
	SessionStateReconnected         SessionLifecycleState = "reconnected"
	SessionStateExpired             SessionLifecycleState = "expired"
)

type GatewaySession struct {
	SessionID             simcore.SessionID     `json:"sessionId"`
	AccountID             simcore.AccountID     `json:"accountId"`
	CharacterID           simcore.CharacterID   `json:"characterId"`
	RealmID               simcore.RealmID       `json:"realmId"`
	ZoneID                simcore.ZoneID        `json:"zoneId"`
	State                 SessionLifecycleState `json:"state"`
	AuthoritativePosition simcore.Vector3       `json:"authoritativePosition"`
	AttachedAt            time.Time             `json:"attachedAt"`
	UpdatedAt             time.Time             `json:"updatedAt"`
	LastDisconnectReason  string                `json:"lastDisconnectReason,omitempty"`
}

type AttachSessionRequest struct {
	SessionID             simcore.SessionID
	AccountID             simcore.AccountID
	CharacterID           simcore.CharacterID
	RealmID               simcore.RealmID
	ZoneID                simcore.ZoneID
	AuthoritativePosition simcore.Vector3
	Now                   time.Time
}

type AttachSessionResult struct {
	Session  GatewaySession
	Replaced *GatewaySession
}

type SessionGateway struct {
	mutex              sync.RWMutex
	nextSessionOrdinal uint64
	sessions           map[simcore.SessionID]GatewaySession
	activeByCharacter  map[simcore.CharacterID]simcore.SessionID
}

func NewSessionGateway() *SessionGateway {
	return &SessionGateway{
		sessions:          map[simcore.SessionID]GatewaySession{},
		activeByCharacter: map[simcore.CharacterID]simcore.SessionID{},
	}
}

func (g *SessionGateway) Attach(request AttachSessionRequest) (AttachSessionResult, error) {
	if request.AccountID == "" {
		return AttachSessionResult{}, errors.New("account id is required")
	}
	if request.CharacterID == "" {
		return AttachSessionResult{}, errors.New("character id is required")
	}
	if request.RealmID == "" {
		return AttachSessionResult{}, errors.New("realm id is required")
	}
	if request.ZoneID == "" {
		return AttachSessionResult{}, errors.New("zone id is required")
	}
	if request.Now.IsZero() {
		request.Now = time.Now().UTC()
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	if request.SessionID == "" {
		g.nextSessionOrdinal++
		request.SessionID = simcore.SessionID(fmt.Sprintf("world_session_%012d", g.nextSessionOrdinal))
	}

	var replaced *GatewaySession
	if existingSessionID, ok := g.activeByCharacter[request.CharacterID]; ok && existingSessionID != request.SessionID {
		existing := g.sessions[existingSessionID]
		existing.State = SessionStateDisconnected
		existing.UpdatedAt = request.Now
		existing.LastDisconnectReason = "replaced_by_new_session"
		g.sessions[existingSessionID] = existing
		replacedCopy := existing
		replaced = &replacedCopy
	}

	session := GatewaySession{
		SessionID:             request.SessionID,
		AccountID:             request.AccountID,
		CharacterID:           request.CharacterID,
		RealmID:               request.RealmID,
		ZoneID:                request.ZoneID,
		State:                 SessionStateActive,
		AuthoritativePosition: request.AuthoritativePosition,
		AttachedAt:            request.Now,
		UpdatedAt:             request.Now,
	}
	g.sessions[session.SessionID] = session
	g.activeByCharacter[session.CharacterID] = session.SessionID

	return AttachSessionResult{Session: session, Replaced: replaced}, nil
}

func (g *SessionGateway) ValidateCommand(envelope simcore.CommandEnvelope) (GatewaySession, simcore.CommandValidation) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	session, ok := g.sessions[envelope.SessionID]
	if !ok {
		return GatewaySession{}, simcore.CommandValidation{
			Accepted: false,
			Reason:   simcore.RejectionUnauthenticated,
			Message:  "world session is not attached",
		}
	}
	if session.CharacterID == "" {
		return session, simcore.CommandValidation{
			Accepted: false,
			Reason:   simcore.RejectionSessionUnbound,
			Message:  "world session is not bound to a character",
		}
	}
	if envelope.CharacterID != "" && envelope.CharacterID != session.CharacterID {
		return session, simcore.CommandValidation{
			Accepted: false,
			Reason:   simcore.RejectionSessionUnbound,
			Message:  "command character does not match world session",
		}
	}
	if session.State != SessionStateActive && session.State != SessionStateReconnected {
		return session, simcore.CommandValidation{
			Accepted: false,
			Reason:   simcore.RejectionSessionInactive,
			Message:  "world session is not active",
		}
	}
	return session, simcore.CommandValidation{Accepted: true}
}

func (g *SessionGateway) Session(sessionID simcore.SessionID) (GatewaySession, bool) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	session, ok := g.sessions[sessionID]
	return session, ok
}

func (g *SessionGateway) Disconnect(sessionID simcore.SessionID, reason string, now time.Time) (GatewaySession, error) {
	return g.setState(sessionID, SessionStateDisconnected, reason, now)
}

func (g *SessionGateway) RequestReconnect(sessionID simcore.SessionID, now time.Time) (GatewaySession, error) {
	return g.setState(sessionID, SessionStateReconnectPending, "", now)
}

func (g *SessionGateway) CompleteReconnect(sessionID simcore.SessionID, position simcore.Vector3, now time.Time) (GatewaySession, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	session, ok := g.sessions[sessionID]
	if !ok {
		return GatewaySession{}, errors.New("world session is not attached")
	}
	session.State = SessionStateReconnected
	session.AuthoritativePosition = position
	session.UpdatedAt = now
	session.LastDisconnectReason = ""
	g.sessions[sessionID] = session
	g.activeByCharacter[session.CharacterID] = session.SessionID
	return session, nil
}

func (g *SessionGateway) Expire(sessionID simcore.SessionID, now time.Time) (GatewaySession, error) {
	return g.setState(sessionID, SessionStateExpired, "expired", now)
}

func (g *SessionGateway) UpdatePosition(sessionID simcore.SessionID, zoneID simcore.ZoneID, position simcore.Vector3, now time.Time) (GatewaySession, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	session, ok := g.sessions[sessionID]
	if !ok {
		return GatewaySession{}, errors.New("world session is not attached")
	}
	session.ZoneID = zoneID
	session.AuthoritativePosition = position
	session.UpdatedAt = now
	g.sessions[sessionID] = session
	return session, nil
}

func (g *SessionGateway) ActiveSessionCount() int {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	count := 0
	for _, session := range g.sessions {
		if session.State == SessionStateActive || session.State == SessionStateReconnected {
			count++
		}
	}
	return count
}

func (g *SessionGateway) setState(sessionID simcore.SessionID, state SessionLifecycleState, reason string, now time.Time) (GatewaySession, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	session, ok := g.sessions[sessionID]
	if !ok {
		return GatewaySession{}, errors.New("world session is not attached")
	}
	session.State = state
	session.UpdatedAt = now
	session.LastDisconnectReason = reason
	g.sessions[sessionID] = session
	if state == SessionStateDisconnected || state == SessionStateExpired {
		if g.activeByCharacter[session.CharacterID] == session.SessionID {
			delete(g.activeByCharacter, session.CharacterID)
		}
	}
	return session, nil
}
