# Default HUD Shell Milestone 1

## Current UI Audit

AmandaCore's in-world UI is currently a first-party ImGui HUD rendered by the `UiClient` gem. It already includes the core gameplay panels needed for the playable slice:

- player and target frames with health/resource meters
- combat feed and event log
- chat shell with Say channel input
- objective tracker
- minimap/navigator shell
- main action bar plus optional upper and right-side bars
- spellbook with drag-to-action-slot behavior
- inventory pack with item drag/rearrange behavior
- character, quest log, zone map, trainer, quest, market, social, party, and prompt panels
- local keybinding settings stored in the existing user UI settings file

## Current Gaps

Milestone 1 focuses on reliability and presentation gaps rather than adding new gameplay systems:

- panel close behavior was coarse: Escape closed all gameplay panels instead of the topmost panel first
- the bottom-right utility buttons were compact and did not maintain top-panel ordering
- action bar edit behavior relied on holding Shift only
- HUD frame positions were fixed and not exposed through a built-in edit mode
- settings grouped interface and keybind behavior together without a dedicated layout/edit surface
- panel close affordances were inconsistent
- the default layout needed clearer first-party rules for future UI work

## Milestone 1 Behavior

This milestone keeps the existing ImGui HUD architecture and hardens it as the default built-in MMORPG shell:

- normal gameplay mode keeps HUD frames locked
- built-in UI edit mode unlocks supported HUD anchors and shows edit outlines
- supported M1 editable anchors are chat, objectives, action bar cluster, pack, and navigator
- Reset HUD Layout restores those anchors to the default layout
- Escape closes one open gameplay panel at a time, preferring the most recently opened panel
- the utility bar reliably toggles Character, Spells, Quests, Map, Bag, and Settings
- action bars keep click, keybind, drag, rearrange, and clear behavior
- inventory keeps open, drag, click-to-move fallback, and rearrange behavior
- chat keeps Enter focus, Escape cancel, send, and gameplay input release behavior

## Regression Risks

The most important regressions to avoid are input capture, panel overlap, and drag/drop interference. Validation must cover movement after chat exits, action activation after UI edits, spellbook-to-bar drag, inventory rearrange, and service/launcher startup.

## No-Addon Policy

AmandaCore UI features are first-party systems. This milestone does not add an addon runtime, addon API, user script execution, user-installed UI modules, or plugin loading. Addon-quality features such as edit mode, keybind mode, frame movement, and bag polish must be implemented directly in AmandaCore, reviewed, tested, and packaged as first-party code.
