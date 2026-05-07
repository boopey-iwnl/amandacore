# Accessibility Options Contract

AmandaCore accessibility options in M8 are first-pass readability and distraction-reduction controls. They do not claim full accessibility compliance.

## Functional Options

- UI scale adjusts the ImGui client UI scale within a clamped safe range.
- Readability scale adjusts text readability within a clamped safe range.
- Floating combat text can be hidden.
- Nameplates can be hidden.
- Reduced UI motion suppresses floating notification pulse rendering.

## Disabled Options

- High contrast theme is disabled until a full client-wide color pass can be applied safely.
- Colorblind marker mode is disabled until first-party marker/icon variants are authored and packaged.
- Chat font family is disabled until approved first-party font assets exist.

## Rules

- Disabled accessibility options must explain their unavailable state.
- Defaults must remain readable.
- Accessibility settings are local only.
- No option may route to external assets, addon scripts, plugins, or user-installed modules.
