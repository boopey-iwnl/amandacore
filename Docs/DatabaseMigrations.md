# AmandaCore Database Migrations

AmandaCore migrations are original, immutable store-version steps. They currently apply to the local durable store and define the discipline required before a Postgres adapter becomes the default service backend.

## Commands

From the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\run-db-migrations.ps1 -DryRun
powershell -ExecutionPolicy Bypass -File .\Infra\dev\run-db-migrations.ps1
powershell -ExecutionPolicy Bypass -File .\Infra\dev\run-db-migrations.ps1 -Status
```

From `Services`:

```powershell
go run ./cmd/dbmigrate --dry-run
go run ./cmd/dbmigrate
go run ./cmd/dbmigrate --status
```

Use `--store <path>` or `-Store <path>` to target a specific local state file. Use `--json` or `-Json` for machine-readable output.

## Migration Rules

- Migration IDs are stable and ordered.
- Each migration has a checksum derived from its ID, description, and schema contract text.
- A changed checksum for an already-applied migration fails validation.
- Dry-run mode validates migrations against a cloned in-memory state and does not write.
- Service startup applies pending migrations automatically before saving runtime build metadata.

## Current Migrations

| ID | Purpose |
| --- | --- |
| `202604260001_persistence_metadata` | Creates migration history metadata. |
| `202604260002_character_runtime_state` | Normalizes durable character runtime, inventory, action bar, quest, bind, travel, and mount state. |
| `202604260003_recovery_domains` | Prepares session recovery and account progression domains for reconnect-safe persistence. |

## Local Dev Store

The default store path is controlled by `AMANDACORE_STORE_PATH`. If unset, AmandaCore uses the platform user config directory. The local secret file remains `.secrets/amandacore.dev.env`; do not commit it.

Suggested local-only override:

```powershell
$env:AMANDACORE_STORE_PATH = "$env:TEMP\amandacore\platform-state.json"
```

## Future Postgres Adapter

The next implementation step is to map the repository and unit-of-work interfaces to a database adapter with real SQL transactions. The schema must remain AmandaCore-native and should be documented before implementation.

Recommended future tables:

- `amanda_schema_migrations`
- `accounts`
- `sessions`
- `realms`
- `characters`
- `character_runtime_state`
- `character_inventory_slots`
- `character_equipment_slots`
- `character_quest_progress`
- `character_action_bar_slots`
- `world_join_tickets`
- `account_progress`
- `audit_events`

These names are AmandaCore-owned and are not derived from emulator schemas.

## Clean-Room Reference Boundary

This implementation uses original AmandaCore code, interfaces, migration metadata, and schema planning. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, worldserver configs, database structures, command systems, remote admin systems, playerbot systems, performance scripts, or benchmark tools were copied or adapted.
