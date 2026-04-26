# Architecture Overview

## Goals

- Build original AmandaCore MMO behavior through shared, server-authoritative simulation rules.
- Keep engine integration thin so gameplay logic remains project-owned and portable.
- Support server-authoritative multiplayer from the beginning.
- Use original AmandaCore content and clean-room architecture notes instead of copied internals or compatibility artifacts.
- Support a real login flow with launcher, account services, character selection, realm routing, and world join tickets.
- Keep all gameplay protocols, schemas, command names, content IDs, formulas, and assets AmandaCore-original.

## High-level split

### Shared domain

`Shared/AmandaCoreShared` holds data contracts and rule evaluators that must stay consistent across client and server:

- Movement stepping and tunable locomotion constants.
- Combat resolution, mitigation, and outcome generation.
- Threat state evaluation for simple AI transitions.
- Quest objective progress rules.
- Loot table rolling and content validation helpers.
- Client/server message shapes, session/bootstrap types, realm descriptors, character summaries, and world join tickets.

### Launcher and services

The monorepo now includes a practical local/dev platform slice:

- `Client/Launcher/AmandaCore.Launcher`: Windows-first launcher for register/login, realm selection, character management, and join ticket handoff.
- `Services/cmd/auth-service`: register, login, refresh, logout, password change, and password recovery start.
- `Services/cmd/account-service`: authenticated account profile access.
- `Services/cmd/realm-service`: realm directory and patch manifest.
- `Services/cmd/character-service`: character listing and creation.
- `Services/cmd/world-service`: world join ticket issuance.
- `Services/cmd/admin-service`: account listing, bans, and role assignment.

These services currently share a file-backed state store for local/dev/staging execution. The long-term deployment target remains database/cache-backed multi-service infrastructure.

### Runtime Gems

Each Gem has a single, durable responsibility:

- `GameCore`: bootstrap, session orchestration, game state machine, and service registry.
- `MovementPhysics`: movement controllers, prediction, and server correction adapters.
- `CombatRules`: gameplay event application and combat presentation hooks.
- `StatsProgression`: stat curves, derived stats, and level progression.
- `InventoryLoot`: inventory runtime, item equipment, vendors, and drop flows.
- `QuestRuntime`: quest acceptance, tracking, completion, and UI feeds.
- `NpcAi`: behavior trees/state machines, leash handling, and encounter ownership.
- `ZoneStreaming`: outdoor cell streaming debug visualization, world partitioning hooks, and micro-instance handoff.
- `UiClient`: HUD, chat, targeting, hotbars, and player feedback.
- `NetClient`: transport, serialization, interpolation, and prediction feeds.
- `NetServer`: authority, replication, interest management, and session hosting.
- `Persistence`: save/load, profile state, and content version migration.
- `ContentTools`: authoring validators, packaging, and reference replay comparison.

### Content model

The data model is deliberately wider than the first slice so future work can scale without a rewrite. JSON Schema definitions live in `Content/Schemas/gameplay.schema.json`, and example authored content lives in `Content/GameData/ZoneSlice`.

The first server-side runtime content package loader is documented in `Docs/ContentPackageLoader.md`. It loads AmandaCore-owned JSON package manifests from `Content/Packs`, validates zones, map exports, and catalogs before activation, and activates validated content into the Go `worldServer` as additive runtime content while preserving existing hardcoded starter flows.

`Content/Packs/dawnwake_isles` adds the first original multi-zone package with server-side adjacency, transition metadata, map exports, streaming cells, quest providers, hostile spawns, and placeholder traversal content. Runtime ownership is single-zone for now: each character is owned by exactly one content zone runtime, and transfer gates move ownership between zone runtimes with state diffs and visibility deltas.

`Docs/DawnwakeIsles.md` documents the package, `Docs/WorldStreaming.md` documents the current map export and streaming hint boundary, and `Docs/O3DEMapExportWorkflow.md` documents the deterministic authoring-to-map-export workflow. The console world client reads the server `streaming` payload, maintains a client streaming preview frame, and emits either console preview events or O3DE-facing placeholder scene commands for zone, cell, bounds, and transition changes. The `ZoneStreaming` Gem consumes the mirrored C++ command contract and can tail the live JSONL debug bridge for debug-only AuxGeom volumes without moving authority into the client.

Current Dawnwake coordinates are placeholder server-side rectangles pending map tracing from the owner-supplied Dawnwake Isles and Kingsfall Harbor images. O3DE mapping remains a separate transform layer; the server package is the authoritative runtime input.

The Dawnwake load testing milestone adds deterministic population distribution, transition stress, zone command queues, queue backpressure reporting, tick duration percentiles, and a single-process shard assignment skeleton. Shard IDs currently bind active zones inside one process; they do not imply distributed runtime ownership yet.

### Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore and AzerothCore were used only as high-level architectural reference. Dawnwake Isles is AmandaCore-original world content. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.

## O3DE wiring

- Build `AmandaCoreShared` as a normal static library and link it into both client and server Gems.
- Keep Atom, Terrain, Prefabs, and networking as engine services underneath project-owned gameplay systems.
- Place O3DE serialization wrappers around the shared domain types instead of moving game rules into engine-specific components.
- Use Prefab and asset metadata for presentation. Use project-owned JSON or generated asset products for authoritative gameplay data.

The current O3DE client path keeps this boundary for combat. `NetClient` parses authoritative combat state from the world service, while `UiClient` renders target state, action bar cooldowns, active auras, kill credit, and recent combat state diffs/domain events. Targeting and ability input remain client intents; damage, cooldowns, aura lifecycle, death, respawn, and progression remain server decisions.

## Networking model

- Client sends intent: movement inputs, ability requests, target changes, and quest interactions.
- Server simulates the canonical world state using the same domain rules and publishes snapshots.
- Client performs prediction and interpolation but always accepts authoritative corrections.
- Domain message and snapshot shapes are defined in `AmandaCoreShared/Messages.h` so alternate transports or backends can adapt without rewriting gameplay code.

## Clean-room MMO architecture foundation

AmandaCore now keeps a formal clean-room reference policy in `Docs/CleanRoomReferencePolicy.md`. TrinityCore and AzerothCore are read-only architectural reference corpora only; AmandaCore implementation, schemas, protocols, content manifests, IDs, admin vocabulary, and observability names must remain original.

The backend foundation adds AmandaCore-native scaffolding for:

- canonical internal server commands and domain events in `Services/internal/simcore`
- a lightweight fixed-step `WorldRuntime` with deterministic command queue processing
- neutral `ZoneRuntime`, `InstanceRuntime`, `EntityRegistry`, `SpawnPoint`, and `RuntimeEntity` concepts
- structured observability event constants for ticks, command queue activity, zones, entities, combat, admin actions, and persistence snapshots
- admin actor/action/audit decision types for RBAC-oriented operations
- a lightweight migration runner convention for local file-store and future database state
- original content package manifest skeletons in `Services/internal/contentpkg`

Gameplay systems build on these boundaries through AmandaCore-owned entities, commands, events, state diffs, content packages, and persistence models.

## Server-authoritative NPC combat loop

The current server slice includes an original dev hostile NPC archetype, `dev_isle_stalker`, spawned as `Isle Stalker` for combat validation. Players can select the NPC, use the original `dev_basic_strike` ability, receive authoritative damage and cooldown results, kill the NPC, receive persisted kill credit, and observe server-scheduled respawn.

The world response exposes protocol-neutral state diffs for entity spawn, health, combat state, target selection, ability results, death, and progression. These are internal AmandaCore contracts, not copied packet or emulator surfaces.

## Ability, effect, and aura skeleton

The world server now owns an original typed ability resolver. Ability definitions expand into effect operations after session, target, range, resource, cooldown, and timing validation. The first supported primitives are direct damage, healing, aura application, aura periodic ticks, cast/channel timing placeholders, per-ability cooldowns, and shared cooldown categories.

Aura runtime state is authoritative on the server and exposed to clients as protocol-neutral state. The dev content package uses `dev_stalker_pressure` and `dev_pressure_mark` only as AmandaCore-owned validation data for aura apply/tick/expire behavior. Future ability work should add richer effect categories behind this same resolver rather than moving combat decisions into clients or transport adapters.

## Client-facing combat wiring

`Client/Game/AmandaCore.WorldClient` now consumes the server combat response fields directly. The diagnostic client can select the nearest hostile target, submit `dev_basic_strike`, poll authoritative state, and render player health, target health, cooldowns, cast state, aura state, combat domain events, state diffs, NPC death, and kill credit.

The O3DE HUD now consumes the same protocol-neutral response fields for player and target frames, cooldown overlays, aura lines, kill credit, and a compact combat feed. The important boundary remains unchanged: clients send target and ability intents only, and all damage, death, aura, cooldown, and progression results come back from the world service.

## Milestone one slice

- One outdoor zone with two streamable cells and one micro-instance hook.
- One player archetype, two hostile enemy archetypes, one vendor-type NPC, and a short quest chain.
- Traversal, combat, objective progression, loot, vendor, respawn, persistence seams, and chat support.
