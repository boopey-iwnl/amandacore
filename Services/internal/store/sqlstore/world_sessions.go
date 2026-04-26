package sqlstore

import (
	"database/sql"
	"errors"
	"fmt"

	filestore "amandacore/services/internal/store"
)

func (s *Store) CreateWorldSession(session filestore.WorldSessionRecord) (filestore.WorldSessionRecord, error) {
	now := s.now().Unix()
	if session.SessionToken == "" {
		session.SessionToken = randomID("worldsess")
	}
	if session.CreatedAt == 0 {
		session.CreatedAt = now
	}
	if session.UpdatedAt == 0 {
		session.UpdatedAt = session.CreatedAt
	}
	_, err := s.db.Exec(
		`INSERT INTO ac_world_sessions (
			world_session_token, account_id, character_id, realm_id, zone_id, connected,
			position_x, position_y, position_z, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.SessionToken,
		session.AccountID,
		session.CharacterID,
		session.RealmID,
		session.ZoneID,
		boolToInt(session.Connected),
		session.PositionX,
		session.PositionY,
		session.PositionZ,
		session.CreatedAt,
		session.UpdatedAt)
	return session, err
}

func (s *Store) GetWorldSession(sessionToken string) (*filestore.WorldSessionRecord, error) {
	row := s.db.QueryRow(
		`SELECT world_session_token, account_id, character_id, realm_id, zone_id, connected,
			position_x, position_y, position_z, created_at, updated_at
		FROM ac_world_sessions WHERE world_session_token = ?`,
		sessionToken)
	session, err := scanWorldSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("world session not found")
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Store) UpdateWorldSession(session filestore.WorldSessionRecord) (filestore.WorldSessionRecord, error) {
	if session.SessionToken == "" {
		return filestore.WorldSessionRecord{}, fmt.Errorf("world session token is required")
	}
	if session.UpdatedAt == 0 {
		session.UpdatedAt = s.now().Unix()
	}
	result, err := s.db.Exec(
		`UPDATE ac_world_sessions SET
			zone_id = ?, connected = ?, position_x = ?, position_y = ?, position_z = ?, updated_at = ?
		WHERE world_session_token = ?`,
		session.ZoneID,
		boolToInt(session.Connected),
		session.PositionX,
		session.PositionY,
		session.PositionZ,
		session.UpdatedAt,
		session.SessionToken)
	if err != nil {
		return filestore.WorldSessionRecord{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return filestore.WorldSessionRecord{}, err
	}
	if rowsAffected == 0 {
		return filestore.WorldSessionRecord{}, fmt.Errorf("world session not found")
	}
	loaded, err := s.GetWorldSession(session.SessionToken)
	if err != nil {
		return filestore.WorldSessionRecord{}, err
	}
	return *loaded, nil
}

func scanWorldSession(scanner rowScanner) (filestore.WorldSessionRecord, error) {
	var session filestore.WorldSessionRecord
	var connected int
	if err := scanner.Scan(
		&session.SessionToken,
		&session.AccountID,
		&session.CharacterID,
		&session.RealmID,
		&session.ZoneID,
		&connected,
		&session.PositionX,
		&session.PositionY,
		&session.PositionZ,
		&session.CreatedAt,
		&session.UpdatedAt); err != nil {
		return filestore.WorldSessionRecord{}, err
	}
	session.Connected = intToBool(connected)
	return session, nil
}
