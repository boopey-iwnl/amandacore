# Production Cutover Runbook

## Scope

This runbook covers AmandaCore's file-store-to-relational cutover preparation for hardened alpha release candidates. It is not a live production deployment script.

## Preflight

1. Confirm the work starts from `develop` and no one is working on `main`.
2. Run `git fetch --all --prune --tags`.
3. Confirm all required milestone branches are merged into `develop`.
4. Confirm the worktree is clean.
5. Back up any local `platform-state.json` that will be analyzed.

## Local Development Mode

Local/default gameplay uses:

```powershell
$env:AMANDACORE_ENV = "local"
$env:AMANDACORE_STORE_BACKEND = "file"
```

This path keeps launcher, login, realm list, character selection, join tickets, world connect, movement, quests, trainer, inventory, action bars, combat, loot, and social/economy tests on the current Alpha-compatible path.

## Staging/Production Guard

Staging and production reject file-backed storage unless explicitly allowed:

```powershell
$env:AMANDACORE_ENV = "staging"
$env:AMANDACORE_STORE_BACKEND = "file"
$env:AMANDACORE_ALLOW_FILE_STORE_IN_PRODUCTION = "false"
```

Expected result: service config validation fails.

Use the override only for a controlled emergency or a local reproduction:

```powershell
$env:AMANDACORE_ALLOW_FILE_STORE_IN_PRODUCTION = "true"
```

## Relational Migration Check

Use an explicit SQLite path. Do not commit `.sqlite`, `.db`, WAL, or SHM files.

```powershell
$db = Join-Path $env:TEMP "amandacore-cutover.sqlite"
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\run-db-migrations.ps1 -Backend sqlite -SQLitePath $db -Status
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\run-db-migrations.ps1 -Backend sqlite -SQLitePath $db
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Check-Migrations.ps1 -Backend sqlite -SQLitePath $db
```

Checksum mismatches or pending migrations block cutover.

## Legacy State Dry Run

Run the report-only analyzer:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Import-LegacyState.ps1 -Source .\Infra\dev\platform-state.json -Json
```

The report inventories accounts, realms, characters, sessions, join tickets, inventory rows, quest state, action bars, social rows, economy rows, audit/support rows, and housing rows. Expired or consumed runtime join tickets are excluded by default.

## Writable Import Status

Writable SQLite import is intentionally disabled in this milestone. The safe rollback model is backup plus redeploy previous build. Manual import must not proceed until a later milestone enables a tested writer and fixtures for every migrated domain.

## Startup Verification

Services log selected backend, environment, and migration state. If `AMANDACORE_STORE_BACKEND=sqlite` is selected, startup verifies migrations and then refuses runtime use until HTTP service adapters are enabled for SQLite.

## Rollback

For local/dev file storage, restore the backed-up `platform-state.json`.

For relational experiments, discard the test database and rerun migrations. For release candidates, redeploy the previous verified package and keep the previous SHA/release notes intact.

## Clean-Room Boundary

Cutover scripts and migration procedures are AmandaCore-owned. Do not copy external MMO schemas, table names, command names, operational scripts, or data IDs.
