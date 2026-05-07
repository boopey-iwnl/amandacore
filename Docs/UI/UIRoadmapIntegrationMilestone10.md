# UI Roadmap Integration Milestone 10

Milestone 10 is a hardening and release-readiness pass for the first-party AmandaCore UI stack delivered across UI M1-M9. It does not add a new gameplay feature family, addon surface, external module runtime, or broad backend contract.

## Integrated Surface Audit

- Default HUD: player/target frames, action bars, chat, objective tracker, navigator, utility menu, bags, and edit-mode frame movement.
- Account/front-end flow: launcher bootstrap, in-client login/realm/character shell, character creation draft, and world handoff.
- Character and inventory: character sheet tabs, equipment slots, currency, stats, inventory drag/rearrange, equipment drag, and item tooltips.
- Ability systems: spellbook, trainer, talents, professions, action-bar assignment, passive rejection, cooldown/resource/target/range states, and keybind hints.
- Quest and map systems: quest log, objective tracker, gossip/dialogue, map images, map markers, route hints, and quest reward previews.
- Combat and social: combat HUD, nameplates, floating combat feedback, party/guild/chat, mail/auction/vendor/trade shells, notifications, help, tutorials, settings, keybinds, and accessibility toggles.

## Current Integration Model

- Panel visibility remains represented by explicit first-party booleans in `UiClient`.
- Panel close ordering is now backed by an ordered gameplay panel stack instead of relying on one remembered top panel.
- ESC handling is deterministic: chat/keybind capture first, NPC/modal surfaces next, most recently opened gameplay panel next, then Game Menu.
- Pre-world login/realm/character shell ESC handling remains separate from in-world HUD panel handling.
- Pending action, inventory, and equipment interaction state is cleared when panels close, edit mode exits, settings reset, chat starts, or the client leaves the in-world state.

## Asset And Package Audit

- UI icons, map art, materials, and textures resolve through repo-side `Content/Art` paths.
- `Content/Art/Manifests` and `Content/GameData/Maps` are validated for repo-local asset paths.
- The local Downloads texture source folder remains read-only source material and is not a runtime dependency.
- No new textures were required for this milestone unless validation later proves a repo-side asset is missing.

## No-Addon Status

AmandaCore UI remains first-party only. This milestone must not add an addon API, Lua addon loader, AddOns folder, plugin runtime, user-installed UI modules, arbitrary script execution, addon settings, addon manager, addon command system, addon profiles, addon package format, or compatibility layer.

## Validation Coverage

- `Infra/qa/Validate-UiSmokeChecklist.ps1` validates required UI M10 docs, no AddOns directories, no tracked Lua scripts, no local Downloads texture runtime paths, and repo-side Content/Art manifest references.
- `Infra/qa/Smoke-Test.ps1 -SelfTest` includes the UI smoke checklist self-test as script wiring validation.
- Full release-candidate evidence still requires explicit O3DE build and verifier runs because `Validate-ReleaseCandidate.ps1 -SkipO3DE` intentionally skips O3DE validation.

## Known Limitations

- UI interaction automation is not complete enough to replace manual launcher and in-client smoke.
- Help/tutorial content remains compiled first-party UI content.
- Social/economy panels expose shell behavior where backend state is not available.
- Package validation proves repo-side assets and forbidden-path policy, but human review remains required for visual overlap and usability.
