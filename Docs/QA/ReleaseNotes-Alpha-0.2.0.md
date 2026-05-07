# AmandaCore Alpha 0.2.0 Release Notes

Release tag: `alpha-0.2.0`
Package asset: `AmandaCore-Alpha-0.2.0-Windows-x64.zip`
Package SHA256: `<filled after package validation>`
Package size: `<filled after package validation>`
Package manifest: `release-package-manifest.json`

## Scope

Alpha 0.2.0 promotes the consolidated UI overhaul to the prerelease line. It keeps the .NET launcher as the patcher, updater, local-ops, and bootstrap shell while moving the playable flow toward an O3DE-rendered in-client login, realm, character, and world-entry experience.

This release is clean-room AmandaCore work. It does not add addon APIs, Lua addon loading, an AddOns folder, plugin runtimes, user-installed UI modules, arbitrary UI script execution, or dependencies on local texture download folders.

## UI M1-M10 Summary

- M1: Default HUD shell polish for the first-party in-world UI.
- M2: In-client login, realm, character creation, selection, join-ticket, and world-entry shell.
- M3: Character panel, paper-doll equipment, stats, currency, inventory equip flow, and character-facing UI contracts.
- M4: Spellbook, trainer, talents, and professions shell with server-owned ability/training data.
- M5: Quest log, objective tracker, world map improvements, and authored Dawnwake/Stonewake map readability work.
- M6: Combat HUD, targeting, nameplates, cast/buff/debuff surfaces, respawn timing, and combat feedback.
- M7: Social, chat, party, guild, vendor, mail, auction, trade, and economy shell surfaces.
- M8: Settings, keybinds, accessibility options, HUD edit mode, and UI state persistence hardening.
- M9: Help, tutorials, onboarding prompts, notifications, and built-in guidance surfaces.
- M10: Integrated UI hardening for panel stacking, input focus, drag/drop, UI checklist validation, package hygiene, and release readiness.

## Playable Changes

- Launcher remains the patcher/bootstrapper and local stack entry point.
- O3DE client flow now covers in-client login, realm-adjacent character flow, join-ticket issuance, world connection, and player spawn verification.
- Character info covers equipment, stats, paper-doll presentation, inventory interaction, and currency presentation.
- Spellbook, trainer, talent, and professions surfaces are available as built-in first-party UI shells.
- Quest log, objective tracker, and world map presentation are more consistent with the default HUD layout.
- Authored Dawnwake/Stonewake map art, map calibration, and route readability work are included.
- Combat HUD, target feedback, nameplates, floating combat text, and combat status presentation are improved.
- Social/economy/vendor shells now expose chat, party, guild, mail, auction, trade, and vendor-adjacent flows as first-party panels.
- Settings, keybinds, accessibility, help, tutorial, and notification surfaces are integrated into the UI shell.

## Known Limitations

- Alpha 0.2.0 is still a local Windows x64 prerelease package, not a production public service release.
- Character creation/customization remains an in-client shell milestone, not a final 3D appearance-authoring system.
- Professions, auction, mail, guild, trade, and help/tutorial surfaces are functional shells and need deeper production content and transaction polish.
- Some map, marker, layout, and window-position behaviors remain alpha-quality and should be tested against the UI smoke checklist.
- `Smoke-Test.ps1 -SelfTest` validates script wiring and presence only; it is not a real package smoke.
- `Validate-ReleaseCandidate.ps1 -SkipO3DE` skips O3DE, package smoke, and soak rows, so explicit O3DE build/verify, package smoke, and soak results must be checked separately.

## Validation Performed

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
- Release package assertion, package smoke, downloaded asset assertion, and downloaded package smoke are required before publication.

## Clean-Room And Packaging Notes

- No Blizzard, WoW, private-server, addon, or third-party UI assets are intentionally included.
- No runtime/package reference to local texture download folders is allowed.
- No AddOns folder, addon API, Lua addon loading, plugin runtime, user-installed UI module, or arbitrary UI script execution is included.
- Package validation must confirm the archive excludes secrets, local DBs, logs, diagnostics, screenshots, nested zips, temp/cache/build junk, runtime tickets, and machine-local paths.
