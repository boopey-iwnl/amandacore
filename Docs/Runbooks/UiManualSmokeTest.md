# UI Manual Smoke Test

Run this after automated validation and startup smoke. Record PASS, FAIL, or NOT TESTED for each item.

## Setup

- Branch `codex/ui-m10-integration-release-polish` is checked out.
- Worktree is clean before manual test.
- Local stack starts.
- Launcher opens patcher/play UI.
- In-client login works if enabled.
- Join world works.
- Visible world loads.
- No crash.

## Global UI

- HUD layout is readable.
- Panel visual style is consistent.
- ESC/topmost-panel behavior is reliable.
- UI clicks do not trigger world actions.
- No severe overlap appears at default resolution.
- Tooltips are readable and not stale.
- Notifications do not block input.

## Input And Drag

- Action-bar click/keybind works.
- Spellbook-to-action-bar drag works.
- Action-bar rearrange and clear work.
- Inventory rearrange works.
- Equipment drag works where implemented.
- UI edit-mode frame drag works.
- Chat focus/send/cancel works.
- Keybind capture works where implemented.
- Movement and camera recover after UI focus exits.

## Screens And Panels

- In-client login, realm, and character creation work where implemented.
- Character panel works.
- Spellbook, talents, professions, and trainer UI work.
- Quest log, objective tracker, and map work.
- NPC dialogue/gossip works.
- Combat HUD, nameplates, and combat feedback work.
- Social, economy, mail, auction, vendor, and trade shells work or clearly communicate unavailable state.
- Settings, keybinds, and accessibility work.
- Help, tutorials, and notifications work.

## Gameplay Regression

- Quest accept, progress, and turn-in work if tested.
- Combat kill loop works.
- Loot or reward works if tested.
- Trainer interaction works.
- Vendor interaction works if tested.
- Reconnect works if tested.

## Policy And Packaging

- No AddOns tab, folder, loader, API, or runtime is present.
- No Lua addon, plugin, or user-module system is present.
- No runtime reference to the local Downloads texture source folder is present.
- Repo-side UI assets resolve.
- Forbidden artifact scan passes.

## Overall Result

- APPROVE MERGE TO DEVELOP
- NEEDS FIXES
- NEEDS MORE HUMAN TESTING
