# UI Release Candidate Checklist

Use this checklist before approving a UI release-candidate branch for merge.

## Automated Gates

- `git status --short`
- `git diff --check`
- `Push-Location Services; go test ./... -count=1 -timeout 15m; Pop-Location`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Validate-UiSmokeChecklist.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Scan-ForbiddenArtifacts.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-o3de-client.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\verify-o3de-client.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Validate-ReleaseCandidate.ps1 -SkipO3DE`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Smoke-Test.ps1 -SelfTest`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Run-ScaleSoak.ps1 -Mode runtime -Users 2 -Duration 1s`

`Smoke-Test.ps1 -SelfTest` validates script wiring only. It is not a real package smoke. `Validate-ReleaseCandidate.ps1 -SkipO3DE` does not prove O3DE readiness unless paired with explicit O3DE build and verifier runs.

## Manual Smoke

- Launcher opens patcher/play UI.
- In-client login works if enabled in the current build.
- Realm/character select and character creation work where implemented.
- Join world works and visible world loads.
- Movement and camera work before and after UI focus.
- HUD layout is readable at default resolution.
- ESC closes chat/keybind capture, modal/NPC surfaces, and topmost panels in order.
- Chat focus/send/cancel works.
- Keybind capture works and cancels safely.
- Action-bar click, keybind, drag, rearrange, and clear work.
- Spellbook-to-action-bar drag works; passive abilities cannot be assigned.
- Inventory drag/rearrange works.
- Equipment drag works where implemented.
- UI edit mode frame drag/reset/lock works.
- Character, spellbook, talents, professions, trainer, quest log, objectives, map, combat HUD, social/economy, settings, help, tutorials, notifications, and error states remain usable.
- Social/economy shells either work with live state or clearly communicate unavailable state.
- No crash occurs during login, world handoff, panel opening, drag/drop, settings reset, or disconnect.

## Policy Gates

- No addon API, Lua addon loader, AddOns folder, plugin runtime, user-installed UI module, arbitrary script execution, addon settings, addon manager, addon command system, addon profile format, addon package format, or compatibility layer.
- No runtime dependency on the local Downloads texture source folder.
- Repo-side UI assets resolve and missing icon fallback remains available.
- Release package excludes secrets, logs, screenshots, diagnostics, archives, caches, temp files, runtime tickets, local DBs, generated packages, raw texture dumps, and user-generated local settings.
