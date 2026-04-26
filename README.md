# amandacore

`amandacore` is a clean-room O3DE MMO foundation targeting the structural feel of WoW `3.3.5a` without copying proprietary code, content, assets, names, maps, or protocols. It is organized as a monorepo containing the shared gameplay domain, O3DE Gem scaffolding, backend services, and a Windows launcher.

## Implemented foundation

- `Shared/AmandaCoreShared`: shared C++ gameplay and platform contracts for combat, movement, quests, loot, auth/session types, realm routing, character selection, and world join tickets.
- `Services`: Go-based account, auth, realm, character, world, and admin service binaries sharing a functional dev file-store backend.
- `Client/Launcher`: Windows-first C# launcher code for registration, login, realm listing, character flow, and world join handoff.
- `Client/Game/AmandaCore.WorldClient`: diagnostic .NET world client for movement, target selection, Basic Strike, server health/death/cooldown/aura feedback, and kill-credit display.
- `Gems`: expanded O3DE Gem source skeletons for the planned runtime split.
- `Content` and `Docs`: clean-room schemas, example authored content, architecture guidance, and reference capture notes.

## Current constraints

- The Go and .NET surfaces are verified locally through the dev scripts in `Infra/dev`.
- Shared C++ and O3DE Gem runtime proof are still blocked on a local CMake toolchain, C++ compiler, and O3DE SDK installation.
- The backend is implemented to be functional in local/dev/staging with a shared file-backed store and secret-driven admin bootstrap, while remaining aligned to a future Postgres/Redis deployment shape.
- The gameplay and content targets are `3.3.5a`-structured and greybox-equivalent, not exact Blizzard class kits, formulas, maps, or content data.

## Clean-room MMO architecture foundation

- `Docs/CleanRoomReferencePolicy.md` defines the formal guardrail: external emulator projects are read-only architectural references only.
- `Services/internal/simcore` defines original canonical server commands and domain events.
- `Services/internal/worlds` includes a lightweight fixed-step runtime, deterministic command queue, and neutral zone/instance ownership skeleton.
- `Services/internal/observability` exposes stable AmandaCore event names for ticks, command queues, entities, combat, admin actions, and persistence snapshots.
- `Docs/ContentPipeline.md` documents the AmandaCore-owned future content package path.

Intentionally not implemented yet: full O3DE combat HUD wiring, production encounter content, loot table expansion, full quest objective UI, and finalized Dawnwake Isles world-space transforms from authored maps.

## Key paths

- `Shared/AmandaCoreShared`
- `Client/Launcher/AmandaCore.Launcher`
- `Client/Game/AmandaCore.WorldClient`
- `Services`
- `Infra/dev`
- `Gems`
- `Content`
- `Docs/Milestone01-AccountToWorld.md`

## Dev admin bootstrap

- Local/dev/staging can seed the requested admin account through environment or a local ignored secrets file.
- Username defaults can be `amanda`; the password must come from local secret configuration and is hashed before storage.
- Copy `.secrets/amandacore.dev.env.example` to `.secrets/amandacore.dev.env` before running the local stack.

## Local runtime proof

1. `powershell -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1`
2. `powershell -ExecutionPolicy Bypass -File .\Infra\dev\start-local.ps1`
3. `powershell -ExecutionPolicy Bypass -File .\Infra\dev\verify-golden-path.ps1`
4. `powershell -ExecutionPolicy Bypass -File .\Infra\dev\verify-account-to-world-restart.ps1`
5. `powershell -ExecutionPolicy Bypass -File .\Infra\dev\stop-local.ps1`

The local stack waits for each service to report healthy before returning control. The verification script proves the current milestone path:

- register
- login
- list realms
- create character
- request join ticket
- launch the minimal world client
- connect to the real world service
- move
- disconnect
- reconnect
- retain position state

Milestone `0.1` hardening details, commands, and pass/fail behavior are documented in `Docs/Milestone01-AccountToWorld.md`.

## Local load simulation

Run the in-process content package harness from `Services`:

```powershell
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario content-package-basic --content ..\Content\Packs\dev_foundation\package.json
```

The default package can be overridden with `AMANDACORE_CONTENT_PACKAGE`.

Run the in-process multi-zone load harness from `Services`:

```powershell
go run ./cmd/loadsim --clients 5 --duration 10s --cmd-rate 2 --scenario multizone-pressure --content ../Content/Packs/dawnwake_isles/package.json --seed 42
```

Scale tiers, scenarios, reports, and sharding behavior are documented in `Docs/LoadTesting.md` and `Docs/MultiZoneSharding.md`.

Run the server-authoritative ability/effect/aura harness from `Services`:

```powershell
go run ./cmd/loadsim --clients 3 --duration 10s --cmd-rate 2 --scenario ability-aura-basic
```

The scenario exercises the original AmandaCore effect resolver, aura apply/tick/expire lifecycle, cast completion, and cooldown events without requiring O3DE.

The fallback world client can also drive a live combat diagnostic after a join ticket is issued:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --auto-combat-demo
```

The diagnostic client sends target and ability intents only. It renders target health, cooldowns, aura state, combat events, state diffs, NPC death, and kill credit from the authoritative world response.

The Dawnwake Isles multi-zone skeleton can be exercised without O3DE:

```powershell
Push-Location Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-traversal-basic --content ../Content/Packs/dawnwake_isles/package.json
Pop-Location
```

The scenario loads `dawnwake_isles`, activates its continent runtime, spawns simulated players at the default entry, transfers through the first zone gate, and reports transition and visibility counts.
