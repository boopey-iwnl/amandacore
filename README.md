# amandacore

`amandacore` is a clean-room O3DE MMO foundation targeting the structural feel of WoW `3.3.5a` without copying proprietary code, content, assets, names, maps, or protocols. It is organized as a monorepo containing the shared gameplay domain, O3DE Gem scaffolding, backend services, and a Windows launcher.

## Implemented foundation

- `Shared/AmandaCoreShared`: shared C++ gameplay and platform contracts for combat, movement, quests, loot, auth/session types, realm routing, character selection, and world join tickets.
- `Services`: Go-based account, auth, realm, character, world, and admin service binaries sharing a functional dev file-store backend.
- `Client/Launcher`: Windows-first C# launcher code for registration, login, realm listing, character flow, and world join handoff.
- `Gems`: expanded O3DE Gem source skeletons for the planned runtime split.
- `Content` and `Docs`: clean-room schemas, example authored content, architecture guidance, and reference capture notes.

## Current constraints

- The Go and .NET surfaces are verified locally through the dev scripts in `Infra/dev`.
- Shared C++ and O3DE Gem runtime proof are still blocked on a local CMake toolchain, C++ compiler, and O3DE SDK installation.
- The backend is implemented to be functional in local/dev/staging with a shared file-backed store and secret-driven admin bootstrap, while remaining aligned to a future Postgres/Redis deployment shape.
- The gameplay and content targets are `3.3.5a`-structured and greybox-equivalent, not exact Blizzard class kits, formulas, maps, or content data.

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

## Dawnwake traversal loadsim

The Dawnwake Isles multi-zone skeleton can be exercised without O3DE:

```powershell
Push-Location Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-traversal-basic --content ../Content/Packs/dawnwake_isles/package.json
Pop-Location
```

The scenario loads `dawnwake_isles`, activates its continent runtime, spawns simulated players at the default entry, transfers through the first zone gate, and reports transition and visibility counts.
