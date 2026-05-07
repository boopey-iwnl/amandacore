# Keybinds Contract

AmandaCore keybinds are first-party local client settings for built-in commands only. They do not expose addon APIs, script execution, user-installed UI modules, or addon-compatible binding formats.

## Categories

- Movement: fixed existing movement controls, not locally rebound in M8.
- Camera: fixed existing mouse/camera controls, not locally rebound in M8.
- Targeting: Target Hostile and Interact.
- Action Bars: 48 first-party action slot bindings.
- UI Panels: Spellbook, Bag, Character, Quest Log, Map, and Game Menu.
- Chat: Enter/Escape focus behavior is fixed to preserve text input reliability.
- Combat: combat commands route through action-bar slots and server-authoritative activation.
- Misc: reserved for future reviewed first-party commands.

## Behavior

- Clicking a binding starts capture mode.
- ESC cancels capture without changing the binding.
- Capture is blocked while chat or any text field is active.
- Binding a duplicate key clears the previous first-party binding and shows a warning.
- Clear removes one binding.
- Reset UI Panel Keybinds restores panel, target, and interact defaults.
- Reset Action Bar Keybinds restores slot defaults.
- Reset All Keybinds restores every first-party keybind default.

## Preservation Rules

Movement, camera, chat focus, action activation, and existing panel toggles must keep working when no custom settings exist. Missing settings fall back to defaults.
