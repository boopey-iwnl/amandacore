# amandacore Services

This Go module contains the backend service entrypoints for local/dev/staging MMO platform testing.

## Service binaries

- `cmd/auth-service`: registration, login, refresh, logout, password change, password recovery start.
- `cmd/account-service`: authenticated account profile and session visibility.
- `cmd/realm-service`: realm directory and patch manifest.
- `cmd/character-service`: character roster, create, and select-ready retrieval.
- `cmd/world-service`: world join ticket issuance and basic world bootstrap visibility.
- `cmd/admin-service`: account listing, bans, and role assignment.

## Storage mode

The current implementation uses a shared JSON file store for local/dev/staging so the services can operate together without a full Postgres/Redis runtime in this workspace. The API and type surfaces are intended to remain compatible with a future database-backed implementation.

By default, services read:

- store file: `%LocalAppData%\amandacore\platform-state.json`
- local secret seed file: `.secrets/amandacore.dev.env`

## Seeded admin

The requested admin seed is loaded through environment variables or the ignored local secret file:

- `AMANDACORE_ADMIN_SEED_USERNAME`
- `AMANDACORE_ADMIN_SEED_PASSWORD`

The password is hashed with Argon2id before storage.
