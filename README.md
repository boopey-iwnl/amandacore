# amandacore

`amandacore` is an original, first-party, clean-room O3DE MMO foundation. It targets the structural feel and dense RPG UI rhythm of WoW `3.3.5a`, but it does not copy proprietary code, content, assets, maps, protocols, UI code, addon systems, names, formulas, or data.

The repository is organized as a monorepo for shared gameplay contracts, Go backend services, O3DE Gems, launcher/bootstrap tooling, local operations tools, authored AmandaCore content, QA scripts, and architecture/runbook documentation.

## Current Release

Public release: [AmandaCore Alpha 0.2.0](https://github.com/boopey-iwnl/amandacore/releases/tag/alpha-0.2.0)

- Package: `AmandaCore-Alpha-0.2.0-Windows-x64.zip`
- SHA256: `ae41a4eb98fb3ac07ccf08e724570d990f419e32bead59cabbeaca57dd62af8e`
- Platform/state: Windows x64 alpha prerelease package.

Alpha 0.2.0 is a local/dev playable alpha. A downloader can launch the local stack, open the launcher/Play UI, start the game client, log in inside the client, select a realm, select or create a character, enter the world, and use the current first-party core UI systems.

## Implemented Foundation

- `Services`: Go services for auth, account, realm, character, world, and admin surfaces, with local/dev storage and validation hooks.
- `Shared/AmandaCoreShared`: shared C++ gameplay and platform contracts for combat, movement, quests, loot, auth/session types, realm routing, character selection, and world join tickets.
- `Client/Launcher`: Windows-first patcher/bootstrapper/Play UI that checks local build/status, resolves the preferred O3DE `GameLauncher`, starts the local stack when requested, and launches the client.
- `Gems/GameCore`, `Gems/NetClient`, and `Gems/UiClient`: O3DE client path for login, remembered-session restore, realm select, character select/create, join-ticket request, world entry, gameplay state, and UI rendering.
- First-party HUD shell with player/target frames, action bars, chat, objective tracker, minimap/navigation, inventory, panel stacking, edit mode, and local UI persistence.
- Character panel with equipment paper doll, stats, currency, reputation shell, and server-authoritative equipment interactions.
- Spellbook, trainer, talents/professions shells, and action-bar integration.
- Quest Log, Objective Tracker, Gossip/Dialogue, map UI, and authored Dawnwake/Stonewake map and traversal work.
- Combat HUD, target frames, nameplates, cast/buff/debuff presentation, and combat feedback.
- Social/economy/vendor shells including chat, party/guild/mail/auction/trade-facing UI surfaces where currently implemented.
- Settings, keybinds, accessibility options, UI layout/edit mode, help/tutorial surfaces, notifications, and error states.
- Integrated validation and hardening scripts for UI smoke, forbidden artifacts, release candidate checks, local builds, O3DE client build/verify, package hygiene, diagnostics, and local controls.
- `Client/Tools/AmandaCore.LocalControls`: Local Ops GUI for starting/stopping the local stack, building, opening the launcher, collecting diagnostics, and running supported QA helpers.

Detailed behavior is documented in `Docs/UI/DefaultScreenContract.md`, `Docs/Runbooks/InClientLoginCharacterCreation.md`, `Docs/Runbooks/ReleaseCandidateProcedure.md`, and `Docs/QA/UiReleaseCandidateChecklist.md`.

## Launcher And Client Flow

The launcher is no longer the normal player registration/login/realm/character/world-join UI. The current flow is:

1. Launcher checks patch/build/status information and opens the Play UI.
2. Launcher starts the O3DE game client.
3. Game client owns player login, remember-login/session restore, realm selection, character selection/creation, join-ticket request, and world entry.

The legacy direct `--join-ticket` / `--world-endpoint` launch path remains useful for development and backward-compatible diagnostics, but it is not the normal player Play flow.

## Local Run

Recommended local startup from the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\start-local.ps1 -StartLauncher
```

This starts the local services and opens the launcher/Play UI. Pressing Play starts the client; the client then handles login, realm selection, character selection or creation, join-ticket request, and world entry.

Stop local services when finished:

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\stop-local.ps1
```

## Build And Verify

Common local validation commands:

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1
powershell -ExecutionPolicy Bypass -File .\Infra\dev\build-o3de-client.ps1
powershell -ExecutionPolicy Bypass -File .\Infra\dev\verify-o3de-client.ps1
powershell -ExecutionPolicy Bypass -File .\Infra\qa\Validate-UiSmokeChecklist.ps1
powershell -ExecutionPolicy Bypass -File .\Infra\qa\Scan-ForbiddenArtifacts.ps1
```

Run Go service tests from the service module:

```powershell
Push-Location Services
go test ./... -count=1 -timeout 15m
Pop-Location
```

For release-candidate procedure, package gates, human test expectations, and wrapper caveats, use `Docs/Runbooks/ReleaseCandidateProcedure.md` and `Docs/Runbooks/ReleaseGateChecklist.md`.

## Local Ops GUI

The Local Ops GUI is a compiled Windows desktop app at `Client/Tools/AmandaCore.LocalControls`. It wraps supported `Infra/dev` and `Infra/qa` scripts for local stack control, builds, launcher startup, diagnostics, QA docs, and guarded state reset actions.

Build and run it with:

```powershell
dotnet build .\Client\Tools\AmandaCore.LocalControls\AmandaCore.LocalControls.csproj
dotnet run --project .\Client\Tools\AmandaCore.LocalControls\AmandaCore.LocalControls.csproj
```

The compatibility wrapper `Infra/dev/Launch-LocalOpsGui.cmd` opens the compiled Local Ops app.

## Clean-Room And No-Addon Policy

AmandaCore is first-party UI only:

- No addon API.
- No Lua addon loading.
- No `AddOns` folder.
- No plugin runtime.
- No user-installed UI modules.
- No arbitrary user script execution or addon compatibility layer.

Addon-like quality-of-life features should be implemented as built-in AmandaCore systems with normal code review, validation, and release gates.

`Docs/CleanRoomReferencePolicy.md` defines the formal reference policy. External projects and games may be used only as high-level architectural references. AmandaCore source, content, UI, assets, schemas, and runtime behavior must remain original.

## Asset And Path Hygiene

Repository and release packages must not depend on local machine texture folders or other developer-only source paths. Source art may be curated into repo-controlled paths such as `Content/Art` only after review, normalization, and package validation. Release/package scanners should remain clean of local absolute paths, secrets, logs, diagnostics, screenshots, archives, generated packages, runtime tickets, local databases, caches, and temporary files.

## Current Limitations

Alpha 0.2.0 is not production-ready.

- The package is focused on local/dev alpha testing.
- World and content scale are still evolving.
- Some systems are UI shells or first-pass implementations, especially where backend depth is intentionally deferred.
- Social, economy, mail, auction, and trade depth is not final.
- Authored map/world alignment is first-pass and will continue to mature.
- O3DE/Terrain dependency warnings may still appear during local builds when the build and verifier otherwise pass.
- Production hosting, cutover, telemetry, scaling, anti-cheat, and live operations remain future work.

## Branch And Release Workflow

- Permanent branches: `main`, `develop`, and `functional`.
- Feature branches should use `codex/<short-task-name>`.
- Releases are cut from `main` after explicit approval and validation.
- `develop` is the active integration branch.
- `functional` tracks validated playable state only when explicitly synchronized.
- Temporary feature or release branches should be cleaned up only after proof that useful work is merged and deletion is explicitly approved.

## Key Paths

- `Shared/AmandaCoreShared`
- `Services`
- `Client/Launcher/AmandaCore.Launcher`
- `Client/Tools/AmandaCore.LocalControls`
- `Client/Game/AmandaCore.WorldClient`
- `Gems/GameCore`
- `Gems/NetClient`
- `Gems/UiClient`
- `Content`
- `Infra/dev`
- `Infra/qa`
- `Docs`

## Additional Developer Scenarios

The diagnostic .NET world client remains available for direct-world development and test harnesses:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --auto-combat-demo
```

Load simulation, Dawnwake traversal, content export, world streaming, persistence migrations, release gates, and QA procedures are documented under `Docs/` and should be run from the repo-relative paths shown in those runbooks.
