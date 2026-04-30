# Login and Character Creation Milestone 2

## Current Flow Audit

AmandaCore still supports the external launcher as the primary alpha path. The launcher:

- calls `POST /v1/auth/login` with username and password
- calls `GET /v1/realms`
- calls `GET /v1/characters?realmId=<realm>`
- calls `POST /v1/characters` to create a character
- calls `POST /v1/world/join-ticket`
- launches the client with `--join-ticket` and `--world-endpoint`

The O3DE client consumes the join ticket through the existing GameCore world bootstrap path, then uses the existing world HTTP client for movement, combat, chat, inventory, quests, trainers, and social state.

## Current Data Shape

Character summaries currently include `id`, `realmId`, `displayName`, `raceId`, `classId`, `archetypeId`, `level`, and `zoneId`. Character creation remains compatible with the launcher contract and submits `realmId`, `displayName`, and `archetypeId`.

The current AmandaCore archetype used by this milestone is `wayfarer_warden`. Existing backend normalization preserves old clients by filling legacy race/class defaults when omitted.

## In-Client Shell

Milestone 2 adds a first-party pre-world screen stack:

- login
- realm select
- character select
- character creation
- connecting
- in-world HUD

The in-client path uses the same backend routes as the launcher and does not remove or alter the launcher path.

## Limitations

Appearance customization is preview-only in this milestone. The procedural preview updates as options change, but appearance choices are not persisted and are not shown later as saved character data.

## Compatibility

Launcher login, realm list, character list/create, join ticket, and client launch remain supported. The existing `--join-ticket` and `--world-endpoint` behavior is preserved.

## Policy Notes

This milestone adds no addon API, Lua loading, AddOns folder, plugin runtime, user-installed modules, or arbitrary UI script execution.

The local texture source folder remains a read-only source asset pool. No textures are imported by default, and runtime/package references must stay repo-relative only.
