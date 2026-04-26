package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const (
	EventPersistenceMigrationStarted   = "persistence.migration_started"
	EventPersistenceMigrationCompleted = "persistence.migration_completed"
	EventPersistenceMigrationFailed    = "persistence.migration_failed"
)

var ErrMigrationChecksumMismatch = errors.New("migration checksum mismatch")

type MigrationRecord struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Checksum    string `json:"checksum"`
	AppliedAt   int64  `json:"appliedAt"`
	DurationMs  int64  `json:"durationMs"`
}

type MigrationOptions struct {
	DryRun bool
}

type MigrationResult struct {
	DryRun         bool              `json:"dryRun"`
	CurrentID      string            `json:"currentId"`
	Applied        []MigrationRecord `json:"applied"`
	AlreadyApplied []MigrationRecord `json:"alreadyApplied"`
}

type stateMigration struct {
	ID          string
	Description string
	Schema      string
	Apply       func(*state) error
}

func (m stateMigration) checksum() string {
	sum := sha256.Sum256([]byte(m.ID + "\n" + m.Description + "\n" + m.Schema))
	return hex.EncodeToString(sum[:])
}

func CurrentMigrationIDs() []string {
	migrations := currentStateMigrations()
	ids := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		ids = append(ids, migration.ID)
	}
	return ids
}

func (s *FileStore) ApplyMigrations(options MigrationOptions) (MigrationResult, error) {
	if err := s.lockState(true); err != nil {
		return MigrationResult{}, err
	}
	defer s.unlockState()

	result, err := s.applyMigrationsLocked(options)
	if err != nil {
		return result, err
	}
	if !options.DryRun && len(result.Applied) > 0 {
		if err := s.saveLocked(); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *FileStore) MigrationHistory() ([]MigrationRecord, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	records := make([]MigrationRecord, 0, len(s.state.MigrationHistory))
	for _, record := range s.state.MigrationHistory {
		records = append(records, record)
	}
	sort.Slice(records, func(left int, right int) bool {
		return records[left].ID < records[right].ID
	})
	return records, nil
}

func (s *FileStore) applyMigrationsLocked(options MigrationOptions) (MigrationResult, error) {
	target := &s.state
	if options.DryRun {
		cloned, err := cloneStateForMigration(s.state)
		if err != nil {
			return MigrationResult{DryRun: true}, err
		}
		target = &cloned
	}
	ensureMigrationState(target)

	result := MigrationResult{DryRun: options.DryRun}
	for _, migration := range currentStateMigrations() {
		checksum := migration.checksum()
		if record, ok := target.MigrationHistory[migration.ID]; ok {
			if record.Checksum != checksum {
				err := fmt.Errorf("%w: %s", ErrMigrationChecksumMismatch, migration.ID)
				observability.LogEvent("store", EventPersistenceMigrationFailed, map[string]any{
					"migrationId": migration.ID,
					"reason":      "checksum_mismatch",
					"dryRun":      options.DryRun,
				})
				return result, err
			}
			result.AlreadyApplied = append(result.AlreadyApplied, record)
			result.CurrentID = record.ID
			continue
		}

		observability.LogEvent("store", EventPersistenceMigrationStarted, map[string]any{
			"migrationId": migration.ID,
			"dryRun":      options.DryRun,
		})
		started := time.Now()
		if migration.Apply != nil {
			if err := migration.Apply(target); err != nil {
				observability.LogEvent("store", EventPersistenceMigrationFailed, map[string]any{
					"migrationId": migration.ID,
					"reason":      err.Error(),
					"dryRun":      options.DryRun,
				})
				return result, err
			}
		}
		record := MigrationRecord{
			ID:          migration.ID,
			Description: migration.Description,
			Checksum:    checksum,
			AppliedAt:   time.Now().Unix(),
			DurationMs:  time.Since(started).Milliseconds(),
		}
		target.MigrationHistory[migration.ID] = record
		result.Applied = append(result.Applied, record)
		result.CurrentID = record.ID
		observability.LogEvent("store", EventPersistenceMigrationCompleted, map[string]any{
			"migrationId": migration.ID,
			"durationMs":  record.DurationMs,
			"dryRun":      options.DryRun,
		})
	}

	if !options.DryRun {
		s.state = *target
	}
	return result, nil
}

func currentStateMigrations() []stateMigration {
	return []stateMigration{
		{
			ID:          "202604260001_persistence_metadata",
			Description: "Create AmandaCore migration history metadata for durable store state.",
			Schema:      "migration_history(id,description,checksum,applied_at,duration_ms)",
			Apply: func(st *state) error {
				ensureMigrationState(st)
				return nil
			},
		},
		{
			ID:          "202604260002_character_runtime_state",
			Description: "Normalize character runtime, inventory, action bar, quest, bind, travel, and mount state.",
			Schema:      "characters(runtime_position,inventory,equipment,action_bar,quests,travel,mount)",
			Apply: func(st *state) error {
				ensureMigrationState(st)
				for characterID, character := range st.Characters {
					st.Characters[characterID] = platform.NormalizeCharacter(character)
				}
				return nil
			},
		},
		{
			ID:          "202604260003_recovery_domains",
			Description: "Prepare session recovery and account progression domains for reconnect-safe persistence.",
			Schema:      "sessions,world_join_tickets,account_progress",
			Apply: func(st *state) error {
				ensureMigrationState(st)
				for accountID, progress := range st.AccountProgress {
					st.AccountProgress[accountID] = platform.NormalizeAccountProgress(accountID, progress)
				}
				return nil
			},
		},
	}
}

func ensureMigrationState(st *state) {
	if st.Accounts == nil {
		st.Accounts = map[string]platform.Account{}
	}
	if st.Realms == nil {
		st.Realms = map[string]platform.Realm{}
	}
	if st.Characters == nil {
		st.Characters = map[string]platform.Character{}
	}
	if st.Sessions == nil {
		st.Sessions = map[string]platform.Session{}
	}
	if st.WorldJoinTickets == nil {
		st.WorldJoinTickets = map[string]platform.WorldJoinTicket{}
	}
	if st.PasswordReset == nil {
		st.PasswordReset = map[string]platform.PasswordResetTicket{}
	}
	if st.Friends == nil {
		st.Friends = map[string]platform.FriendRelationship{}
	}
	if st.Parties == nil {
		st.Parties = map[string]platform.Party{}
	}
	if st.Guilds == nil {
		st.Guilds = map[string]platform.Guild{}
	}
	if st.GuildInvites == nil {
		st.GuildInvites = map[string]platform.GuildInvite{}
	}
	if st.Auctions == nil {
		st.Auctions = map[string]platform.AuctionListing{}
	}
	if st.Mail == nil {
		st.Mail = map[string]platform.MailEnvelope{}
	}
	if st.AuditEvents == nil {
		st.AuditEvents = map[string]platform.AuditEvent{}
	}
	if st.SupportTickets == nil {
		st.SupportTickets = map[string]platform.SupportTicket{}
	}
	if st.Mutes == nil {
		st.Mutes = map[string]platform.MuteRecord{}
	}
	if st.HousingEntitlements == nil {
		st.HousingEntitlements = map[string]platform.HousingEntitlement{}
	}
	if st.HousingSpaces == nil {
		st.HousingSpaces = map[string]platform.HousingSpace{}
	}
	if st.HousingStorage == nil {
		st.HousingStorage = map[string][]platform.HousingStorageSlot{}
	}
	if st.HousingDecorations == nil {
		st.HousingDecorations = map[string][]platform.DecorationPlacement{}
	}
	if st.AccountProgress == nil {
		st.AccountProgress = map[string]platform.AccountProgressState{}
	}
	if st.MigrationHistory == nil {
		st.MigrationHistory = map[string]MigrationRecord{}
	}
}

func cloneStateForMigration(source state) (state, error) {
	payload, err := json.Marshal(source)
	if err != nil {
		return state{}, err
	}
	var cloned state
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return state{}, err
	}
	ensureMigrationState(&cloned)
	return cloned, nil
}
