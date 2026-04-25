package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"
)

type MigrationState struct {
	Applied map[string]MigrationRecord `json:"applied"`
}

type MigrationRecord struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Checksum    string `json:"checksum"`
	AppliedAt   string `json:"appliedAt"`
}

type Migration struct {
	ID          string
	Description string
	Checksum    string
	Apply       func(context.Context) error
}

type MigrationRunner struct {
	migrations []Migration
}

func NewMigrationRunner(migrations []Migration) (*MigrationRunner, error) {
	normalized := append([]Migration{}, migrations...)
	sort.Slice(normalized, func(left, right int) bool {
		return normalized[left].ID < normalized[right].ID
	})

	seen := map[string]struct{}{}
	for index, migration := range normalized {
		if migration.ID == "" {
			return nil, fmt.Errorf("migration id is required")
		}
		if _, exists := seen[migration.ID]; exists {
			return nil, fmt.Errorf("migration %q is duplicated", migration.ID)
		}
		seen[migration.ID] = struct{}{}
		if migration.Checksum == "" {
			normalized[index].Checksum = MigrationChecksum(migration.ID, migration.Description)
		}
	}

	return &MigrationRunner{migrations: normalized}, nil
}

func (r *MigrationRunner) Plan(state MigrationState) ([]Migration, error) {
	applied := state.Applied
	if applied == nil {
		applied = map[string]MigrationRecord{}
	}

	plan := make([]Migration, 0)
	for _, migration := range r.migrations {
		record, exists := applied[migration.ID]
		if !exists {
			plan = append(plan, migration)
			continue
		}
		if record.Checksum != "" && record.Checksum != migration.Checksum {
			return nil, fmt.Errorf("migration %q checksum changed", migration.ID)
		}
	}
	return plan, nil
}

func (r *MigrationRunner) Apply(ctx context.Context, state *MigrationState) error {
	if state == nil {
		return fmt.Errorf("migration state is required")
	}
	if state.Applied == nil {
		state.Applied = map[string]MigrationRecord{}
	}

	plan, err := r.Plan(*state)
	if err != nil {
		return err
	}

	for _, migration := range plan {
		if migration.Apply != nil {
			if err := migration.Apply(ctx); err != nil {
				return fmt.Errorf("migration %q failed: %w", migration.ID, err)
			}
		}
		state.Applied[migration.ID] = MigrationRecord{
			ID:          migration.ID,
			Description: migration.Description,
			Checksum:    migration.Checksum,
			AppliedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		}
	}
	return nil
}

func MigrationChecksum(id string, description string) string {
	sum := sha256.Sum256([]byte(id + "\n" + description))
	return hex.EncodeToString(sum[:])
}
