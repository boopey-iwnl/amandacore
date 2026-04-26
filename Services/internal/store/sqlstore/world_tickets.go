package sqlstore

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func (s *Store) IssueWorldJoinTicket(accountID string, sessionID string, characterID string, realmID string) (platform.WorldJoinTicket, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return platform.WorldJoinTicket{}, err
	}
	if character.AccountID != accountID || character.RealmID != realmID {
		return platform.WorldJoinTicket{}, fmt.Errorf("character not available for realm")
	}

	realms, err := s.ListRealms()
	if err != nil {
		return platform.WorldJoinTicket{}, err
	}
	var realm platform.Realm
	found := false
	for _, candidate := range realms {
		if candidate.ID == realmID {
			realm = candidate
			found = true
			break
		}
	}
	if !found {
		return platform.WorldJoinTicket{}, fmt.Errorf("realm not found")
	}

	ticket := platform.WorldJoinTicket{
		TicketID:      randomID("ticket"),
		SessionID:     sessionID,
		AccountID:     accountID,
		CharacterID:   characterID,
		RealmID:       realmID,
		WorldEndpoint: realm.Endpoint,
		ExpiresAt:     s.now().Add(2 * time.Minute).Unix(),
	}
	_, err = s.db.Exec(
		`INSERT INTO ac_world_join_tickets (
			ticket_id, session_id, account_id, character_id, realm_id, world_endpoint, expires_at, consumed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ticket.TicketID,
		ticket.SessionID,
		ticket.AccountID,
		ticket.CharacterID,
		ticket.RealmID,
		ticket.WorldEndpoint,
		ticket.ExpiresAt,
		ticket.ConsumedAt)
	return ticket, err
}

func (s *Store) ValidateWorldJoinTicket(ticketID string) (*platform.WorldJoinTicket, error) {
	ticket, err := s.getWorldJoinTicket(ticketID)
	if err != nil {
		return nil, err
	}
	now := s.now().Unix()
	if ticket.ExpiresAt < now {
		return nil, fmt.Errorf("join ticket expired")
	}
	if ticket.ConsumedAt != 0 {
		return nil, filestore.ErrJoinTicketConsumed
	}
	return ticket, nil
}

func (s *Store) ConsumeWorldJoinTicket(ticketID string) (*platform.WorldJoinTicket, error) {
	ticket, err := s.ValidateWorldJoinTicket(ticketID)
	if err != nil {
		return nil, err
	}
	consumedAt := s.now().Unix()
	if _, err := s.db.Exec(`UPDATE ac_world_join_tickets SET consumed_at = ? WHERE ticket_id = ?`, consumedAt, ticketID); err != nil {
		return nil, err
	}
	ticket.ConsumedAt = consumedAt
	return ticket, nil
}

func (s *Store) RevokeCharacterJoinTickets(characterID string) error {
	_, err := s.db.Exec(`DELETE FROM ac_world_join_tickets WHERE character_id = ?`, characterID)
	return err
}

func (s *Store) getWorldJoinTicket(ticketID string) (*platform.WorldJoinTicket, error) {
	row := s.db.QueryRow(
		`SELECT ticket_id, session_id, account_id, character_id, realm_id, world_endpoint, expires_at, consumed_at
		FROM ac_world_join_tickets WHERE ticket_id = ?`,
		ticketID)
	var ticket platform.WorldJoinTicket
	if err := row.Scan(
		&ticket.TicketID,
		&ticket.SessionID,
		&ticket.AccountID,
		&ticket.CharacterID,
		&ticket.RealmID,
		&ticket.WorldEndpoint,
		&ticket.ExpiresAt,
		&ticket.ConsumedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("join ticket not found")
		}
		return nil, err
	}
	return &ticket, nil
}
