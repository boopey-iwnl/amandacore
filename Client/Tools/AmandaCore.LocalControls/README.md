# AmandaCore Local Controls

`AmandaCore.LocalControls` is the compiled Windows desktop app for the Local Playable Slice Controls surface. It wraps the existing `Infra/dev` PowerShell scripts and shows command output, errors, and exit codes in the app window.

## Build

```powershell
dotnet build .\Client\Tools\AmandaCore.LocalControls\AmandaCore.LocalControls.csproj
```

`Infra/dev/build-playable-client.ps1` also builds this project as part of the normal local playable client build.

## Run

```powershell
dotnet run --project .\Client\Tools\AmandaCore.LocalControls\AmandaCore.LocalControls.csproj
```

The existing `Infra/dev/Launch-LocalOpsGui.cmd` wrapper now starts this compiled app, so existing desktop shortcuts can keep targeting that wrapper while the UI itself is no longer a loose PowerShell script.

## Controls

- Start local stack: runs `Infra/dev/start-local.ps1 -BuildFirst`.
- Stop local stack: runs `Infra/dev/stop-local.ps1`.
- Build local: runs `Infra/dev/build-local.ps1`.
- Build O3DE client: runs `Infra/dev/build-o3de-client.ps1`.
- Verify O3DE client: runs `Infra/dev/verify-o3de-client.ps1`.
- Launch AmandaCore Launcher: runs `Infra/dev/build-playable-client.ps1`, then starts the built launcher executable.
- Open logs/output folder: opens the first available local logs or output folder.
- Collect diagnostics: runs `Infra/qa/Collect-Diagnostics.ps1`.
- Open QA docs: opens `Docs/QA`.
- Reset test state: confirms first, then runs `Infra/qa/Reset-LocalTestState.ps1 -All -ConfirmReset`.
- Open admin portal: opens `Client/Portal/admin-portal.html`.
