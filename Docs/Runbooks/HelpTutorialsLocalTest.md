# Help and Tutorials Local Test

Use this runbook for UI M9 manual validation.

## Setup

1. Check out `codex/ui-m9-help-tutorials-onboarding`.
2. Build and start the local stack with the repo-native validation scripts.
3. Launch AmandaCore through the launcher or current in-client flow.

## Smoke Checklist

- Login succeeds.
- Realm and character screens still work.
- Join world succeeds and visible world loads.
- Help opens from pre-world screens, Game Menu, micro menu, and `/help`.
- Help categories are visible and filterable.
- Tutorial hint appears on first world load or after `/tutorials reset`.
- Tutorial hints can be dismissed.
- Settings can disable tutorials, reset tutorials, hide keybind hints, hide tooltips, hide notifications, and change notification duration.
- Item, ability, quest/map, vendor/trainer, and disabled-control tooltips still work when tooltips are enabled.
- Notifications appear for safe events and do not block movement, camera, chat, inventory, action bars, or panel controls.
- `/tutorials`, `/tutorials reset`, and `/resetui` are safe and first-party.
- `/reloadui`, `/who`, and `/played` report unsupported state clearly.
- Chat focus, Escape ordering, keybind capture, UI edit mode, action bars, inventory, spellbook, character panel, quest log, map, combat HUD, and social/economy panels still work.
- No addon runtime, AddOns folder, Lua loader, plugin runtime, external help pack, or external tutorial module exists.
- No runtime reference to machine-local Downloads texture paths exists.

## Human Test Result

Record PASS/FAIL for each item and choose one:

- APPROVE MERGE TO DEVELOP
- NEEDS FIXES
- NEEDS MORE HUMAN TESTING
