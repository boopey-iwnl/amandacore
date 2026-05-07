# UI Release Candidate QA Checklist

This QA checklist is the release-facing companion to the UI contracts and manual smoke runbook.

## Required Evidence

- Dependency gate proves UI M1-M9 are merged into `develop`.
- Branch is based on current `develop`.
- Worktree is clean before validation and before staging.
- Automated UI smoke checklist passes.
- Forbidden artifact scan passes with no errors.
- Services tests pass.
- Local build passes.
- O3DE build and verifier pass.
- Release-candidate wrapper passes with `-SkipO3DE`, with the skip called out.
- Smoke self-test passes and is reported as script wiring only.
- Scale soak passes.
- Startup smoke starts the local stack and launcher.

## Human QA Checklist

- Launcher opens patcher/play UI.
- In-client login works if enabled.
- Join world works.
- Visible world loads.
- HUD layout is readable.
- ESC closes UI in the documented order.
- Chat focus/send/cancel works.
- Keybind capture works.
- Action-bar click/keybind/drag/clear works.
- Spellbook-to-action-bar drag works.
- Inventory drag/rearrange works.
- Equipment drag works where implemented.
- UI edit mode frame drag/reset/lock works.
- Character panel works.
- Spellbook, quest log, objective tracker, map, combat HUD, social/economy shells, settings, help, tutorials, and notifications work.
- Movement and camera recover after UI focus exits.
- No crash occurs.

## Release Policy

- No addon system is present.
- No AddOns tab, folder, loader, API, settings, manager, command system, profile format, package format, or compatibility layer is present.
- No Lua addon loading, plugin runtime, user-installed UI module, or arbitrary UI script execution is present.
- No runtime dependency points at the local Downloads texture source folder.
- All UI assets resolve through repo-side paths and package validation includes repo-side `Content/Art` assets.

## Disposition

- APPROVE MERGE TO DEVELOP: all required automated gates pass and human QA finds no blocking regressions.
- NEEDS FIXES: automated gates fail or human QA finds a blocking regression.
- NEEDS HUMAN REVIEW: automated gates pass, but manual UI coverage was incomplete or inconclusive.
