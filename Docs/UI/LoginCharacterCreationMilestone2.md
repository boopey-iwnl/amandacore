# Login and Character Creation Milestone 2

## Current Flow Audit

Milestone 2 changes the launcher role. The launcher is now a patch/build status and client bootstrapper surface only. It checks the patch manifest, resolves the O3DE `GameLauncher`, and starts the game client without player credentials, account IDs, selected realms, selected characters, bearer tokens, refresh tokens, or world join tickets.

The game client owns the player flow:

- calls `POST /v1/auth/login`
- optionally restores a remembered session with `POST /v1/auth/refresh`
- calls `GET /v1/realms`
- calls `GET /v1/characters?realmId=<realm>`
- calls `POST /v1/characters` to create a character
- calls `POST /v1/world/join-ticket`
- enters the existing world connect/bootstrap path

The legacy direct-world `--join-ticket` and `--world-endpoint` client arguments remain available only for development and backward-compatible direct launch.

## Current Data Shape

Character summaries currently include `id`, `realmId`, `displayName`, `raceId`, `classId`, `archetypeId`, `level`, and `zoneId`. Character creation submits `realmId`, `displayName`, and `archetypeId`.

The current AmandaCore archetype used by this milestone is `wayfarer_warden`. Existing backend normalization preserves old clients by filling legacy race/class defaults when omitted.

## In-Client Shell

Milestone 2 adds a first-party pre-world screen stack:

- login
- remembered-login restore
- realm select
- character select
- character creation
- connecting
- in-world HUD

The in-client path uses the same backend routes that were previously launcher-critical while keeping backend authority over authentication, ownership, name validation, character creation, join tickets, and world sessions.

## Remembered Login

The in-client login screen offers `Keep me logged in`. This stores only OS-protected refresh-session material and account/service context. Raw passwords and reversible password material are never stored, passed in process arguments, or logged.

If restore succeeds, the client proceeds to realm select. If restore fails, the saved session is cleared and the login screen is shown.

## Limitations

Appearance customization is preview-only in this milestone. The procedural preview updates as options change, but appearance choices are not persisted and are not shown later as saved character data.

## Compatibility

Launcher login, launcher realm selection, launcher character creation, and launcher join-ticket handoff are no longer the normal player flow. The launcher starts the client without a join ticket. Existing direct-world ticket launch remains a development/backward-compatible path.

## Policy Notes

This milestone adds no addon API, Lua loading, AddOns folder, plugin runtime, user-installed modules, or arbitrary UI script execution.

The local texture source folder remains a read-only source asset pool. No textures are imported by default, and runtime/package references must stay repo-relative only.
