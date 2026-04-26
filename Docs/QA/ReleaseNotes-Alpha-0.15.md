# AmandaCore Alpha 0.15 Release Notes

Release tag: `alpha-0.15`
Package asset: `AmandaCore-Alpha-0.15-Windows-x64.zip`
Package SHA256: `<filled after package validation>`
Package size: `<filled after package validation>`
Source commit: `57422cc71a77d104a9415be6812b77acdc496dea`
Package manifest: `release-package-manifest.json`

## Scope

Alpha 0.15 is an alpha prerelease for the current AmandaCore playable slice. It promotes the validated `develop` state to `main` and packages the local services, Local Ops tools, launcher, O3DE client runtime, Stonewake Vale playable content, high-resolution UI icons, and curated world texture/material assets.

## Playable Changes

- Stonewake Vale boots through the local service stack, launcher, character flow, and O3DE client.
- Login, realm, character selection, join-world, quest, trainer, inventory, item rearranging, action bar reassignment, NPC, and combat flows are included for the current slice.
- Ability and item icons use the high-resolution Alpha 0.15 assets.
- Stonewake world surfaces now include curated texture/material-backed terrain, road, prop, and building material coverage.
- Local Ops and launcher tooling are included for starting/stopping the local stack, launching the client, and collecting QA diagnostics.
- Backend content, persistence, load-test, sharding, and world-system work from the release consolidation is present in `main`.

## Known Limitations

- Stonewake Vale remains an early alpha playable slice, not a complete MMO zone.
- Some Stonewake layout and landmark work is still visibly blockout/proxy quality.
- World textures and material proxies are improved and visible, but not final art direction or final terrain technology.
- Dawnwake, Hearthmere, Tempest, mainland, and Kingsfall Harbor content should be treated as foundation or future content unless explicitly assigned for testing.
- This is a local Windows x64 prerelease package, not a public production service release.

## Install And Run

1. Download `AmandaCore-Alpha-0.15-Windows-x64.zip` from the GitHub draft prerelease after it is approved.
2. Extract it to a short writable path, for example `C:\AmandaCoreAlpha015`.
3. Open `Infra\dev\Launch-LocalOpsGui.cmd`.
4. Click `Start local stack`.
5. Wait for local services to show healthy.
6. Click `Launch AmandaCore Launcher`.
7. Register or log in, create or select a character, and join Stonewake Vale.

PowerShell fallback from the extracted package root:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -Command "& '.\Infra\dev\start-local.ps1' -BuildFirst:`$false -StartLauncher"
```

Stop local services after testing:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\stop-local.ps1
```

## Validation Performed

- `git diff --check`
- `Push-Location Services; go test ./... -count=1 -timeout 15m; Pop-Location`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-o3de-client.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\verify-o3de-client.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\start-local.ps1 -StartLauncher`
- Release package extraction, safety scan, package smoke, and downloaded draft asset retest are required before publication.

