# Help, Tutorials, and Onboarding Milestone 9

Milestone 9 adds first-party in-client guidance for new AmandaCore players. The implementation is client-local in `UiClient`, uses existing ImGui/O3DE styling, and does not add backend routes, addon support, Lua, plugin loading, external help packs, runtime-loaded tutorial modules, or arbitrary UI script execution.

## Current State Audit

- Tooltips already existed for abilities, spellbook entries, trainer offers, inventory items, equipment, vendor offers, disabled options, and some action-bar states.
- System and combat messages already used the lower-left event log, combat pulses, and quest update toast.
- Loading and connection states already existed in the pre-world login/realm/character flow and the world-link panel, but raw network/service errors could surface directly.
- Keybinds and UI settings already persisted locally through `%LOCALAPPDATA%\AmandaCore\ui-settings.ini`.
- Chat already had a safe slash parser for first-party social commands.
- New-player guidance was mostly implicit: players had controls text, panel labels, and disabled-option explanations, but no Help panel, no persistent tutorial state, and no reset path.

## Implemented Scope

- Added a built-in Help / Guide panel with AmandaCore-original topics grouped by gameplay area.
- Added local tutorial state with dismissible hints, settings reset, `/tutorials`, and `/tutorials reset`.
- Added keybind hint visibility, tooltip visibility, notification visibility, notification duration, and tutorial toggles to Settings.
- Added a small non-blocking notification queue for real client events such as quest updates, session notices, keybind changes, UI reset, and user-facing error notices.
- Polished client-facing error messages so raw transport or session wording is mapped to actionable player text where practical.
- Added safe slash commands: `/help`, `/tutorials`, `/tutorials reset`, and `/resetui`.

## No-Addon Policy

All M9 guidance is built into AmandaCore. The milestone does not add AddOns folders, addon APIs, Lua loading, user-installed modules, plugin runtimes, macro support, external command modules, external tutorial packs, or runtime help-pack loading.

## Texture Routing

No textures were imported. The local Downloads texture source folder remains read-only source material only. Runtime code, docs-as-config, manifests, materials, package scripts, and release packages must not reference machine-local source paths.

## Backend Changes

None. Help, tutorials, notifications, tooltip visibility, and tutorial completion are local client UI behavior only. No HTTP contract, service route, database schema, or server persistence changed in this milestone.

## Known Limitations

- Help content is compiled into the first-party UI for M9; a future reviewed content pipeline can move it to repo-controlled JSON and schema validation.
- Tutorial completion is local-only and does not sync across accounts or machines.
- Notification toasts are non-clickable so they do not block gameplay input.
- `/reloadui`, `/who`, and `/played` intentionally return clear unsupported messages instead of exposing fake or unsafe behavior.
