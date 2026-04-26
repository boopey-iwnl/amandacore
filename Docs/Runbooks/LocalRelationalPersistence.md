# Local Relational Persistence Runbook

## Status

The relational store is available for Milestone 2 tests and local experiments only. AmandaCore services still default to the existing file-backed `platform-state.json` path.

## Run Migration And Repository Tests

From the repository root:

```powershell
Push-Location Services
go test ./internal/store/sqlstore -count=1
Pop-Location
```

For full service validation:

```powershell
Push-Location Services
go test ./... -count=1 -timeout 15m
Pop-Location
```

## Open A Throwaway SQLite Store

The SQL store applies embedded migrations when opened through `sqlstore.Open(path)`. Use a temp path for manual experiments:

```powershell
$db = Join-Path $env:TEMP "amandacore-sqlstore-dev.sqlite"
Remove-Item $db -ErrorAction SilentlyContinue
Push-Location Services
go test ./internal/store/sqlstore -run TestAccountSessionRealmCharacterAndTicketRoundTrip -count=1
Pop-Location
```

Do not commit `.sqlite`, `.db`, WAL, SHM, or other local database files.

## Migration Behavior

Migration metadata is stored in `ac_schema_migrations` with:

- migration ID
- migration name
- checksum
- applied timestamp
- duration in milliseconds

Re-running migrations is a no-op. Editing a migration after it has been applied causes a checksum mismatch error. Add a new migration instead.

## Store Backend Configuration

Milestone 2 does not add `AMANDACORE_STORE_BACKEND` or `AMANDACORE_SQLITE_PATH` to service startup. That selector is intentionally deferred so Alpha 0.15 gameplay and local service scripts keep the existing file-backed behavior.

Until Milestone 3 approves the cutover path, SQLite is test-only through the `sqlstore` package.

## Seed Fixtures

`Services/internal/store/sqlstore/seeds.go` provides deterministic test helpers:

- `DevRealm`
- `SeedDevRealm`
- `SeedTestAccount`

These helpers use fake local/test values only. They do not read secrets and do not create real credentials.

## Rollback

Milestone 2 does not migrate runtime data. If a local SQLite experiment fails, discard the test database file and continue using `platform-state.json`.

If later milestones enable SQL service startup, capture the migration status and back up the database before any import or cutover. Do not mix file-store and SQL writers for the same runtime environment.

## Clean-Room Boundary

The relational schema and migrations are AmandaCore-owned. Do not add external MMO emulator schema names, table layouts, IDs, command names, packet structures, comments, or scripts to this directory.
