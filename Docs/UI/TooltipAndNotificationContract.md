# Tooltip and Notification Contract

Tooltips and notifications are first-party AmandaCore UI affordances. They explain current state without replacing server authority or persistent chat/system logs.

## Tooltips

- Item, equipment, ability, trainer, vendor, disabled-option, and action-bar tooltips must preserve existing behavior.
- Tooltips can be disabled locally through Settings.
- Tooltips should use consistent concise title/body/stat wording and stay readable at default UI scale.
- Disabled or unavailable controls should explain why the action is unavailable.

## Notifications

- Notifications are short-lived toasts for real client events.
- Current sources include quest updates, session notices, keybind changes, UI/tutorial resets, and user-facing error notices.
- Duplicate visible notifications are suppressed.
- Notifications respect the local show/hide setting, duration setting, and reduced-motion preference.
- Notifications are non-clickable in M9 so they do not block gameplay input.

## Prohibited

- Fake achievements without an achievement system.
- Addon notification hooks.
- External notification packs.
- Scripted or user-installed notification modules.
