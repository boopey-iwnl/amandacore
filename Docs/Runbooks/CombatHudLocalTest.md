# Combat HUD Local Test

Use this runbook to manually verify UI Milestone 6 after automated validation passes.

## Setup

1. Check out `codex/ui-m6-combat-hud-targeting-feedback`.
2. Confirm `git status --short` is clean or contains only expected M6 implementation changes before commit.
3. Start the local stack with `Infra\dev\start-local.ps1 -StartLauncher`.
4. Launch the O3DE client through the launcher flow.

## Smoke Steps

- Confirm the launcher opens the patcher/play UI.
- Log in through the in-client login flow.
- Select a realm and character, then join the world.
- Confirm the visible world loads without a crash.
- Target a hostile using click or the target-hostile keybind.
- Confirm the player frame updates health/resource/alive state from the server payload.
- Confirm the target frame updates selected target, health, distance, alive state, AI/combat state, and real aura rows.
- Start combat and confirm health changes are visible.
- Confirm action-bar click activation still works.
- Confirm action-bar keybind activation still works.
- Confirm cooldown, resource, target, and range overlays appear only where real payload data supports them.
- Confirm hostile/player nameplates are visible when nearby, not too cluttered, and emphasize the selected target.
- Confirm the targeting-you indicator appears if a hostile reports the local character as its target.
- Confirm combat feedback pulses appear only after real combat domain events or state diffs.
- Confirm target defeated and hostile respawn information appears only when testable and reported.
- Open Quest Log, Objectives, and Map and confirm M5 behavior still works.
- Open Spellbook and confirm M4 drag/passive behavior still works.
- Open Character panel and confirm M3 behavior still works.
- Open inventory, rearrange slots, and confirm bag behavior still works.
- Confirm chat focus, send, and cancel still work.
- Confirm movement and camera still work while combat pulses/nameplates are visible.

## Safety Checks

- No AddOns tab or addon runtime appears.
- No Lua addon loading, plugin UI runtime, user-installed module, or arbitrary UI scripting exists.
- No runtime/package reference points at the local Downloads texture source folder.
- No textures are imported for M6 unless a future focused import pass explicitly requires repo-local curated assets.
- Local texture source files remain unmodified.

## Result

Record one of:

- `APPROVE MERGE TO DEVELOP`
- `NEEDS FIXES`
- `NEEDS MORE HUMAN TESTING`
