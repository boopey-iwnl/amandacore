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
- `ZoneStreaming`: outdoor cell streaming debug visualization, world partitioning hooks, and micro-instance handoff.
- `UiClient`: HUD, chat, targeting, hotbars, and player feedback.
- `NetClient`: transport, serialization, interpolation, and prediction feeds.
- `NetServer`: authority, replication, interest management, and session hosting.
- `Persistence`: save/load, profile state, and content version migration.
- `ContentTools`: authoring validators, packaging, and reference replay comparison.

### Content model

The data model is deliberately wider than the first slice so future work can scale without a rewrite. JSON Schema definitions live in `Content/Schemas/gameplay.schema.json`, and example authored content lives in `Content/GameData/ZoneSlice`.

The first server-side runtime content package loader is documented in `Docs/ContentPackageLoader.md`. It loads AmandaCore-owned JSON package manifests from `Content/Packs`, validates zones, map exports, and catalogs before activation, and activates validated content into the Go `worldServer` as additive runtime content while preserving existing hardcoded starter flows. `Docs/DawnwakeIsles.md` documents the first original multi-zone package with server-side adjacency and transition metadata for future streamed-world expansion. `Docs/WorldStreaming.md` documents the current map export and streaming hint boundary, and `Docs/O3DEMapExportWorkflow.md` documents the deterministic authoring-to-map-export workflow. The console world client now reads the server `streaming` payload, maintains a client streaming preview frame, and emits either console preview events or O3DE-facing placeholder scene commands for zone, cell, bounds, and transition changes. The `ZoneStreaming` Gem consumes the mirrored C++ command contract for debug-only AuxGeom volumes without moving authority into the client.

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
