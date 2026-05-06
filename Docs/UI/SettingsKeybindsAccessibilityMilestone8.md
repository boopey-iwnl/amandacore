# Settings, Keybinds, and Accessibility Milestone 8

Milestone 8 turns the existing AmandaCore settings shell into a fuller first-party client configuration surface. It remains local-only, clean-room, and built into `UiClient`; it does not add addon support, Lua loading, runtime plugins, user-installed UI modules, arbitrary UI script execution, or an AddOns folder.

## Current State Audit

- `UiClient` already stores local UI settings in the user's `AmandaCore` local app data settings file.
- Existing settings keys for secondary action bars, HUD offsets, and keybinds remain supported.
- ESC already closed the topmost gameplay panel before opening the settings shell; M8 formalizes this into a Game Menu.
- Keybind capture already existed for panel toggles, interact, target hostile, and action-bar slots.
- Edit mode already supported moving chat, objectives, action bars, pack, and navigator anchors.
- Video, audio, and gameplay settings did not have stable engine-backed runtime controls, so they remain disabled placeholders until real paths exist.

## Implemented Settings

- Game Menu: Return to Game, Settings/Options, Keybinds, UI Layout/Edit Mode, disabled Logout/Character Select, and Exit Game.
- Interface: UI scale, readability scale, chat/objectives/navigator/action-bar visibility, tooltip comparison, and supported secondary action bars.
- Keybinds: grouped first-party binding rows, duplicate-key warning, clear binding, reset panel bindings, reset action-bar bindings, reset all bindings, and ESC capture cancel.
- UI Layout: Default and Custom local profiles, lock/unlock frames, edit mode, supported HUD visibility toggles, save custom profile, use default profile, and reset HUD layout.
- Accessibility: UI scale, readability scale, floating combat text visibility, nameplate visibility, and reduced UI motion/notification pulses.
- Chat: default visible filter and chat frame visibility.
- Combat: floating combat text and reduced notification motion.
- Nameplates: world nameplate visibility.
- Objectives / Map: objective tracker visibility, collapsed details, and navigator visibility.

## Disabled Placeholders

Audio, video/display, high contrast theme, colorblind marker mode, camera sensitivity, auto-loot, click-to-move, tracker category filters, hostile-only nameplates, and import/export are visible only as disabled options with explanatory status. They are not fake controls.

Logout/Character Select is disabled in this milestone because the current world handoff still returns through the launcher, and no safe in-client world-to-character-select transition is exposed.

## Persistence

Settings are local and user-specific. M8 writes `settings.version=2` while continuing to read old keys. Missing or invalid values fall back safely, numeric values are clamped, and the settings file must not store secrets, tokens, passwords, runtime tickets, account material, or session material.

## Texture Routing

No textures are imported for this milestone. The implementation uses existing repo-side icons and procedural ImGui/O3DE styling. The local texture source folder remains read-only source material only; runtime code, package scripts, manifests, materials, and docs-as-config must not reference machine-local source paths.

## Known Limitations

- Audio and display settings are shell-only until safe engine-backed controls exist.
- UI scale uses ImGui global font scaling and may need future per-panel polish.
- Import/export stays disabled to avoid scriptable or addon-compatible profile formats.
- Accessibility controls are first-pass readability toggles and do not claim full accessibility compliance.
