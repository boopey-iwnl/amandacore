# Settings Options Contract

AmandaCore settings are first-party client options implemented inside the game client. Settings are local-only unless a future milestone explicitly adds reviewed backend profile sync.

## Game Menu

ESC closes the topmost gameplay panel first. If no panel is open, ESC opens the Game Menu.

Enabled actions:

- Return to Game closes the Game Menu.
- Settings/Options opens the Options window on Interface.
- Keybinds opens the Options window on Keybinds.
- UI Layout/Edit Mode opens the Options window on UI Layout.
- Exit Game requests the O3DE main loop exit.

Logout/Character Select must stay disabled unless the client exposes a safe in-client transition that disconnects world state without corrupting session, login, realm, character, or join-ticket flow.

## Options Rules

- Every visible option must be functional or clearly disabled.
- Disabled options must explain why they are unavailable.
- No AddOns category, addon settings, addon manager, addon profile format, addon command surface, or plugin loader may be shown.
- Settings must not require an active world session unless the option is explicitly world-only.
- Settings must not store credentials, account/session material, runtime tickets, diagnostics, or local machine paths.

## Persistence

The settings file is local to the user profile. M8 writes `settings.version=2` and preserves compatibility with older bar, layout, and keybind keys.

Supported persisted values include:

- UI scale and readability scale
- HUD lock/edit state
- supported frame visibility
- secondary action-bar visibility
- combat text, nameplate, tooltip comparison, and reduced motion toggles
- chat default filter
- objective tracker collapsed state
- current local layout profile
- built-in keybinds

Invalid or missing values must be ignored, clamped, or replaced with safe defaults.
