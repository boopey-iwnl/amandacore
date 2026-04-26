package sqlstore

import (
	"database/sql"
	"errors"
	"time"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func (s *Store) CreateSession(accountID string) (platform.Session, error) {
	now := s.now()
	session := platform.Session{
		ID:               randomID("sess"),
		AccountID:        accountID,
		AccessToken:      randomToken(),
		RefreshToken:     randomToken(),
		AccessExpiresAt:  now.Add(30 * time.Minute).Unix(),
		RefreshExpiresAt: now.Add(7 * 24 * time.Hour).Unix(),
		CreatedAt:        now.Unix(),
	}

	if err := s.WithTransaction("sqlstore.session_create", func(tx *Tx) error {
		return tx.CreateSession(session)
	}); err != nil {
		return platform.Session{}, err
	}
	return session, nil
}

func (tx *Tx) CreateSession(session platform.Session) error {
	_, err := tx.tx.Exec(
		`INSERT INTO ac_sessions (id, account_id, access_token, refresh_token, access_expires_at, refresh_expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		session.ID,
		session.AccountID,
		session.AccessToken,
		session.RefreshToken,
		session.AccessExpiresAt,
		session.RefreshExpiresAt,
		session.CreatedAt)
	if err != nil {
		return err
	}
	_, err = tx.tx.Exec(
		`UPDATE ac_accounts SET last_login_at = ?, last_session_id = ?, updated_at = ? WHERE id = ?`,
		session.CreatedAt,
		session.ID,
		session.CreatedAt,
		session.AccountID)
	return err
}

func (s *Store) ValidateAccessToken(token string) (*platform.Session, error) {
	session, err := s.sessionByToken("access_token", token)
	if err != nil {
		return nil, err
	}
	now := s.now().Unix()
	if session.AccessExpiresAt < now {
		return nil, filestore.ErrSessionExpired
	}
	account, err := s.GetAccountByID(session.AccountID)
	if err != nil {
		return nil, err
	}
	if account.Banned || account.SuspendedUntil > now {
		return nil, filestore.ErrAccountBanned
	}
	return session, nil
}

func (s *Store) RefreshSession(refreshToken string) (platform.Session, error) {
	session, err := s.sessionByToken("refresh_token", refreshToken)
	if err != nil {
		return platform.Session{}, err
	}
	now := s.now()
	if session.RefreshExpiresAt < now.Unix() {
		_ = s.RevokeSession(refreshToken)
		return platform.Session{}, filestore.ErrSessionExpired
	}

	session.AccessToken = randomToken()
	session.RefreshToken = randomToken()
	session.AccessExpiresAt = now.Add(30 * time.Minute).Unix()
	session.RefreshExpiresAt = now.Add(7 * 24 * time.Hour).Unix()
	_, err = s.db.Exec(
		`UPDATE ac_sessions SET access_token = ?, refresh_token = ?, access_expires_at = ?, refresh_expires_at = ? WHERE id = ?`,
		session.AccessToken,
		session.RefreshToken,
		session.AccessExpiresAt,
		session.RefreshExpiresAt,
		session.ID)
	return *session, err
}

func (s *Store) RevokeSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM ac_sessions WHERE access_token = ? OR refresh_token = ?`, token, token)
	return err
}

func (s *Store) sessionByToken(column string, token string) (*platform.Session, error) {
	row := s.db.QueryRow(
		`SELECT id, account_id, access_token, refresh_token, access_expires_at, refresh_expires_at, created_at
		FROM ac_sessions
		WHERE `+column+` = ? AND revoked_at = 0`,
		token)
	var session platform.Session
	if err := row.Scan(
		&session.ID,
		&session.AccountID,
		&session.AccessToken,
		&session.RefreshToken,
		&session.AccessExpiresAt,
		&session.RefreshExpiresAt,
		&session.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, filestore.ErrInvalidCredentials
		}
		return nil, err
	}
	return &session, nil
}
