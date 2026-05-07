# Character Panel Local Test

## Setup

1. Check out `codex/ui-m3-character-paperdoll-equipment`.
2. Confirm the worktree contains no secrets, logs, screenshots, zips, cache/build output, runtime tickets, local DBs, generated packages, raw texture dumps, or machine-local paths.
3. Start the local stack:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\start-local.ps1 -StartLauncher
```

## Smoke Checklist

- launcher opens patcher/play UI
- client login works
- realm select works
- character select works
- join world works
- visible world loads
- Character panel opens from the Char utility button and keybind
- Character tab shows paper doll slots and stylized preview
- Stats tab shows only live runtime values
- Currency tab shows live copper wallet data
- Reputation tab shows real data if available, otherwise `No faction standings available yet.`
- Details tab shows display name, archetype, level, zone, and unavailable lineage/origin
- inventory opens while Character panel is open
- item tooltip appears over inventory/equipment items
- comparison appears when an equip-compatible bag item has a matching equipped item
- bag-to-slot equip works when a compatible item is available
- slot-to-bag unequip works
- full-bag unequip rejection is visible if testable
- incompatible drop rejection is visible if testable
- no item duplication occurs
- action bars still work
- spellbook-to-action-bar drag still works
- chat focus/send/cancel still works
- movement and camera still work
- no crash

## Policy Checks

- no addon API
- no Lua addon loading
- no AddOns folder
- no plugin runtime
- no user-installed UI modules
- no arbitrary UI script execution
- no runtime/package dependency on the local Downloads texture source folder
- no copied WoW/Blizzard/private-server/addon assets or code

## Validation Commands

```powershell
git status --short
git diff --check

Push-Location Services
go test ./... -count=1 -timeout 15m
Pop-Location

powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Scan-ForbiddenArtifacts.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-o3de-client.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\verify-o3de-client.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Validate-ReleaseCandidate.ps1 -SkipO3DE
```
