# Quest UI Local Test

Use this runbook to manually verify UI Milestone 5 after automated validation passes.

## Setup

1. Check out `codex/ui-m5-quest-log-objectives-map`.
2. Confirm `git status --short` is clean or contains only the expected M5 working changes during implementation.
3. Start the local stack with `Infra\dev\start-local.ps1 -StartLauncher`.
4. Launch the O3DE client through the launcher flow.

## Smoke Steps

- Log in through the in-client login flow.
- Select a realm and character, then join the world.
- Confirm the visible world loads without a crash.
- Open and close Quest Log from the micro menu and keybind.
- Select quests in the Quest Log and verify the detail pane updates.
- Track and untrack accepted or ready-to-turn-in quests.
- Interact with a quest NPC and accept an available quest.
- Progress an objective where current content supports it.
- Return to a turn-in NPC and complete the quest where current content supports it.
- Confirm reward preview shows only real reward data.
- Confirm Objective Tracker updates and ready-to-turn-in state is visually distinct.
- Open and close World Map.
- Confirm the World tab shows the repo-local Dawnwake world map art.
- Confirm the Zone tab shows the repo-local Stonewake Vale map art.
- Confirm the World and Zone maps are readable at normal gameplay resolution, then use Fit, zoom, and pan to inspect labels without marker drift.
- Click quest/map markers and confirm supported selection or tracking behavior on calibrated views.
- Open Reference Maps and confirm non-Stonewake zone art is display-only without marker precision claims.
- Visit Armsmaster Corin Vale, Scout Rowan Bell, Roadwarden Ilya Brant, the Watch Lantern, and Quartermaster Lyra. Confirm each is visible, reachable, and right-clickable without being hidden by building, ramp, floor, prop, or terrain geometry.
- Verify action bars, inventory, character panel, spellbook, chat, movement, camera, trainer, and vendor flows still work.

## Safety Checks

- No AddOns tab or addon runtime appears.
- No Lua addon loading, plugin UI runtime, user-installed modules, or arbitrary UI scripting exists.
- No runtime/package reference points at the local texture source folder.
- Copied map PNGs are under `Content/Art/UI/Maps/**`.
- Local texture source files remain unmodified.

## Result

Record one of:

- `APPROVE MERGE TO DEVELOP`
- `NEEDS FIXES`
- `NEEDS MORE HUMAN TESTING`
