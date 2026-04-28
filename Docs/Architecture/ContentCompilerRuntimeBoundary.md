# Content Compiler Runtime Boundary

## Purpose

Milestone 8 makes AmandaCore content a validated runtime input instead of an implicit set of hardcoded gameplay assumptions. The goal is an AmandaCore-owned content pipeline for quests, NPC archetypes, spawn groups, loot, abilities, vendors, trainers, dialogue, and declarative event hooks.

This is not emulator compatibility. Content schemas, IDs, hook names, compiler output, tests, and fixtures are original AmandaCore work.

## Current Content Sources

AmandaCore already has server-side JSON content packages under `Content/Packs/`:

- `Content/Packs/dev_foundation/package.json`
- `Content/Packs/dawnwake_isles/package.json`

The package manifest points to zone files, map exports, NPC catalogs, item catalogs, loot catalogs, quest catalogs, ability catalogs, and aura catalogs. Milestone 8 extends the same package model with vendor, trainer, dialogue, and hook catalogs.

## Current Content Loading Path

`Services/internal/content` owns the package loader and validation report model. `world-service` can load a package through `AMANDACORE_CONTENT_PACKAGE`, then activate validated zones, quest providers, spawn groups, items, loot tables, quests, abilities, and map hints.

Default local startup remains the existing Stonewake fallback unless a package is explicitly selected. This milestone does not replace current playable content.

## Current Hardcoded Gameplay/Content Behavior

Several systems still have hardcoded fallback definitions:

- Stonewake starter NPCs, quests, vendors, trainers, and ability catalogs
- vendor and trainer runtime UI responses
- some combat and progression affordances

The content package registry now exposes validated vendor and trainer catalogs, but public runtime cutover remains later work unless a specific path is already package-backed.

## Current Validation Gaps

Before Milestone 8, validation covered core package references but lacked:

- a deterministic compiler/check command
- explicit vendor/trainer/dialogue/hook catalogs
- hook-name validation against a safe registry
- read-only catalog lookup interfaces
- deterministic compiler reports for CI

## AmandaCore Content Package Model

`Content/Schemas/content-package-v1.schema.json` describes the manifest. Catalog schemas live in `Content/Schemas/` for:

- quests
- NPCs
- loot
- abilities
- vendors
- trainers
- zones
- dialogue
- hooks

The runtime validator remains the source of truth for cross-file references because JSON Schema cannot fully validate package-wide references.

Content IDs use stable lowercase AmandaCore IDs such as `dev_first_hunt` or `stonewake.quest.first_patrol`. External MMO IDs, schemas, scripts, and copied text are not allowed.

## Compiler And Validator Flow

The compiler command is:

```powershell
Push-Location Services
go run ./cmd/content-compiler --package ..\Content\Packs\dev_foundation\package.json --check
Pop-Location
```

The compiler:

- loads the package manifest
- loads referenced package files
- validates structure and cross-references
- validates hook names and declarative actions
- emits a deterministic compiled package report when requested
- fails before runtime activation when validation errors exist

Generated compiled output is not committed by default.

## Runtime Registry Design

`RuntimeContentRegistry` remains the loaded package registry. Milestone 8 adds behavior-oriented catalog interfaces for quests, NPCs, loot, abilities, vendors, trainers, and zones. Lookup methods return value copies and typed missing-content errors so callers cannot mutate registry state accidentally.

The registry is intended to be read-only after package load. Later milestones can pass registry interfaces into world-loop systems instead of reaching into package maps directly.

## Event Hook Boundary

Milestone 8 adds a declarative hook boundary. Supported hook names include:

- `on_npc_interact`
- `on_quest_accept`
- `on_quest_objective_progress`
- `on_quest_complete`
- `on_quest_reward_claim`
- `on_npc_defeated`
- `on_loot_generated`
- `on_loot_claimed`
- `on_vendor_buy`
- `on_vendor_sell`
- `on_trainer_learn`
- `on_zone_enter`
- `on_landmark_enter`

Hook bindings are data. They can reference allowed declarative actions such as `emit_event`, `progress_quest_objective`, `grant_item`, `grant_currency`, `show_dialogue`, and `unlock_trainer_ability`. Arbitrary filesystem, network, process, reflection, or code execution is not supported.

## Runtime Integration

This milestone integrates the new catalogs into the existing package loader and runtime registry. World activation reports the new catalog counts. Existing world-loop gameplay, vendor, trainer, and client payload behavior remains compatible.

Package-backed vendor/trainer runtime UI cutover is intentionally deferred until it can be done without destabilizing the current Alpha flow.

## Non-Goals

- no arbitrary scripting language
- no external MMO schema compatibility
- no TrinityCore/AzerothCore script, table, ID, or command compatibility
- no public runtime SQL/content cutover
- no client UI expansion
- no content editor
- no O3DE terrain or asset streaming format

## Clean-Room Notes

The compiler, schemas, hook names, catalog models, fixture content, and docs are AmandaCore-original. Public MMO-server architecture informed only broad principles: validate content before runtime, keep runtime registries read-only, and separate content hooks from core service code.

## Risks For Milestone 9

- The compiler should be added to CI once reliability gates are broadened.
- Runtime content-path configuration should be validated consistently across services.
- Future declarative actions will need tighter audit, rate-limit, and security review.
- Any future scripting language must be sandboxed and opt-in.
