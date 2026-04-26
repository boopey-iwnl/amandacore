# Closed-Alpha Tester Instructions

Use these steps for every assigned Alpha 0.1 release-candidate build. Stay on the assigned route and report only issues that block or degrade the existing playable slice.

## Start

1. Extract or open the tester package.
2. Open `Infra\dev\Launch-LocalOpsGui.cmd`.
3. Click `Start local stack`.
4. Wait until all services show healthy.
5. Click `Launch AmandaCore Launcher`.
6. Register or log in with the assigned test account.
7. Load realms, load characters, create or select a Human Warrior, then join world.
8. Confirm `Docs\QA\Alpha01FeatureFreeze.md`, `Docs\QA\KnownIssues.md`, and `Docs\QA\ReleaseNotes.md` are present.

## Assigned Route

Follow `Docs\QA\checklists\closed-alpha-route.md` from top to bottom. Mark every item as `PASS`, `FAIL`, `BLOCKED`, or `N/A`.

Primary areas:

- launcher, login, realm list, character list
- Human Warrior creation and world join
- Stonewake Vale quest progression
- combat, trainer, vendor, inventory, equipment
- professions foundation
- map, minimap, navigation markers
- chat, friends, party if available in the build
- persistence after exit and local stack restart
- second-zone handoff, professions, chat/friends/party, and dungeon entry only if assigned as optional rough coverage

Do not test guild management, mail, direct trade, auction house, housing/storage, achievements, travel, mounts, or PvP duels unless the build owner explicitly assigns that area. These systems are hidden, disabled, or deferred for Alpha 0.1.

## Bug Reports

For every `FAIL` or `BLOCKED` item:

1. Click `Collect diagnostics` in Local Ops.
2. Fill out `Docs\QA\bug-report-template.md`.
3. Attach the diagnostic zip from `%LOCALAPPDATA%\amandacore\diagnostics`.
4. Include the completed checklist.

Do not include real passwords, private keys, personal documents, or unrelated screenshots.

## Reset

Only reset local test state when instructed.

Preferred reset:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Reset-LocalTestState.ps1 -All -ConfirmReset
```

Selective account reset:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Reset-LocalTestState.ps1 -AccountUsername alpha_tester -ConfirmReset
```

The reset tool backs up state under `%LOCALAPPDATA%\amandacore\state-backups` and preserves logs.

## Recovery

- If login fails, confirm services are healthy and collect diagnostics.
- If the launcher cannot reach the realm list, restart the local stack once and collect diagnostics if it still fails.
- If the game client does not launch, collect diagnostics and include the launcher log text.
- If world state looks corrupted, stop services, collect diagnostics, then reset only when instructed.
- If a disabled or hidden system is reachable, report it as a release-scope issue instead of continuing deeper into that system.
