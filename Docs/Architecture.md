# Architecture Overview

## Goals

- Preserve mechanically faithful `3.3.5a`-structured MMO behavior through shared simulation rules.
- Keep engine integration thin so gameplay logic remains project-owned and portable.
- Support server-authoritative multiplayer from the beginning.
- Use replacement content and black-box reference captures instead of copied internals.
- Support a real login flow with launcher, account services, character selection, realm routing, and world join tickets.

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
- `ZoneStreaming`: outdoor cell streaming, world partitioning, and micro-instance handoff.
- `UiClient`: HUD, chat, targeting, hotbars, and player feedback.
- `NetClient`: transport, serialization, interpolation, and prediction feeds.
- `NetServer`: authority, replication, interest management, and session hosting.
- `Persistence`: save/load, profile state, and content version migration.
- `ContentTools`: authoring validators, packaging, and reference replay comparison.

### Content model

The data model is deliberately wider than the first slice so future work can scale without a rewrite. JSON Schema definitions live in `Content/Schemas/gameplay.schema.json`, and example authored content lives in `Content/GameData/ZoneSlice`.

`Content/Packs/dawnwake_isles` adds the first original multi-zone continent package. The Go content package loader validates the package manifest, continent topology, zone bounds, transition gates, spawn placement, and quest provider placement before activating a `ContinentRuntime`. Runtime ownership is single-zone for now: each character is owned by exactly one `ZoneRuntime`, and transfer gates move ownership between zone runtimes with state diffs and visibility deltas.

Current Dawnwake coordinates are placeholder server-side rectangles pending map tracing from the owner-supplied Dawnwake Isles and Kingsfall Harbor images. O3DE mapping remains a separate transform layer; the server package is the authoritative runtime input.

The Dawnwake load testing milestone adds deterministic population distribution, transition stress, zone command queues, queue backpressure reporting, tick duration percentiles, and a single-process shard assignment skeleton. Shard IDs currently bind active zones inside one process; they do not imply distributed runtime ownership yet.

### Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore and AzerothCore were used only as high-level architectural reference. Dawnwake Isles is AmandaCore-original world content. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.

## Recommended future O3DE wiring

- Build `AmandaCoreShared` as a normal static library and link it into both client and server Gems.
- Keep Atom, Terrain, Prefabs, and networking as engine services underneath project-owned gameplay systems.
- Place O3DE serialization wrappers around the shared domain types instead of moving game rules into engine-specific components.
- Use Prefab and asset metadata for presentation. Use project-owned JSON or generated asset products for authoritative gameplay data.

## Networking model

- Client sends intent: movement inputs, ability requests, target changes, and quest interactions.
- Server simulates the canonical world state using the same domain rules and publishes snapshots.
- Client performs prediction and interpolation but always accepts authoritative corrections.
- Domain message and snapshot shapes are defined in `AmandaCoreShared/Messages.h` so alternate transports or backends can adapt without rewriting gameplay code.

## Milestone one slice

- One outdoor zone with two streamable cells and one micro-instance hook.
- One player archetype, two hostile enemy archetypes, one vendor-type NPC, and a short quest chain.
- Traversal, combat, objective progression, loot, vendor, respawn, persistence seams, and chat support.
