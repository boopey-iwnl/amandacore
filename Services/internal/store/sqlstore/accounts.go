package sqlstore

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/argon2"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func (s *Store) RegisterAccount(username string, password string) (platform.Account, error) {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return platform.Account{}, err
	}

	now := s.now().Unix()
	account := platform.Account{
		ID:           randomID("acct"),
		Username:     strings.TrimSpace(username),
		PasswordHash: passwordHash,
		Roles:        []platform.Role{platform.RolePlayer},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.WithTransaction("sqlstore.account_register", func(tx *Tx) error {
		_, err := tx.CreateAccount(account)
		return err
	}); err != nil {
		return platform.Account{}, err
	}
	return account, nil
}

func (s *Store) Authenticate(username string, password string) (platform.Account, error) {
	account, err := s.GetAccountByUsername(username)
	if err != nil {
		return platform.Account{}, filestore.ErrInvalidCredentials
	}
	now := s.now().Unix()
	if account.Banned || account.SuspendedUntil > now {
		return platform.Account{}, filestore.ErrAccountBanned
	}
	if !verifyPassword(account.PasswordHash, password) {
		return platform.Account{}, filestore.ErrInvalidCredentials
	}
	return *account, nil
}

func (s *Store) GetAccountByID(accountID string) (*platform.Account, error) {
	return s.getAccount(`WHERE a.id = ?`, accountID)
}

func (s *Store) GetAccountByUsername(username string) (*platform.Account, error) {
	return s.getAccount(`WHERE a.normalized_username = ?`, normalize(username))
}

func (s *Store) ListAccounts() ([]platform.Account, error) {
	rows, err := s.db.Query(accountSelectSQL() + ` ORDER BY a.username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []platform.Account
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (tx *Tx) CreateAccount(account platform.Account) (platform.Account, error) {
	if account.ID == "" {
		account.ID = randomID("acct")
	}
	if account.CreatedAt == 0 {
		account.CreatedAt = tx.store.now().Unix()
	}
	if account.UpdatedAt == 0 {
		account.UpdatedAt = account.CreatedAt
	}
	if account.Roles == nil {
		account.Roles = []platform.Role{platform.RolePlayer}
	}
	rolesJSON, err := encodeJSON(account.Roles)
	if err != nil {
		return platform.Account{}, err
	}

	_, err = tx.tx.Exec(
		`INSERT INTO ac_accounts (
			id, username, normalized_username, roles_json, banned, suspended_until, suspension_reason,
			last_login_at, last_session_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		account.ID,
		account.Username,
		normalize(account.Username),
		rolesJSON,
		boolToInt(account.Banned),
		account.SuspendedUntil,
		account.SuspensionReason,
		account.LastLoginAt,
		account.LastSessionID,
		account.CreatedAt,
		account.UpdatedAt)
	if err != nil {
		if isConstraintError(err) {
			return platform.Account{}, filestore.ErrAccountExists
		}
		return platform.Account{}, err
	}

	_, err = tx.tx.Exec(
		`INSERT INTO ac_account_credentials (account_id, password_hash, updated_at) VALUES (?, ?, ?)`,
		account.ID,
		account.PasswordHash,
		account.UpdatedAt)
	if err != nil {
		return platform.Account{}, err
	}
	return account, nil
}

func (s *Store) getAccount(where string, args ...any) (*platform.Account, error) {
	row := s.db.QueryRow(accountSelectSQL()+" "+where, args...)
	account, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, filestore.ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func accountSelectSQL() string {
	return `SELECT
		a.id, a.username, c.password_hash, a.roles_json, a.banned, a.suspended_until,
		a.suspension_reason, a.last_login_at, a.last_session_id, a.created_at, a.updated_at
	FROM ac_accounts a
	JOIN ac_account_credentials c ON c.account_id = a.id`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAccount(scanner rowScanner) (platform.Account, error) {
	var account platform.Account
	var rolesJSON string
	var banned int
	if err := scanner.Scan(
		&account.ID,
		&account.Username,
		&account.PasswordHash,
		&rolesJSON,
		&banned,
		&account.SuspendedUntil,
		&account.SuspensionReason,
		&account.LastLoginAt,
		&account.LastSessionID,
		&account.CreatedAt,
		&account.UpdatedAt); err != nil {
		return platform.Account{}, err
	}
	if err := decodeJSON(rolesJSON, &account.Roles); err != nil {
		return platform.Account{}, err
	}
	account.Banned = intToBool(banned)
	return account, nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash), nil
}

func verifyPassword(encoded string, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	actual := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func isConstraintError(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "constraint") || strings.Contains(text, "unique")
}
