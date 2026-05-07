# UI Layout Profiles Contract

UI layout profiles are local first-party AmandaCore data. They are not addon profiles, scriptable layouts, plugin manifests, or externally loaded UI modules.

## Profiles

- Default: built-in anchor and visibility defaults.
- Custom: user-local saved offsets and supported visibility toggles.

M8 persists only one local custom profile name and the current supported settings. Future multi-profile work must remain data-only and non-scriptable.

## Supported Controls

- Lock or unlock HUD frames.
- Enable or disable edit mode.
- Move supported HUD frames in edit mode.
- Reset supported HUD anchors.
- Show or hide chat frame, objective tracker, navigator/minimap, main action bar cluster, and supported secondary action bars.

## Safety Rules

- Normal gameplay mode keeps frames locked.
- Edit mode must not break action-slot drag/drop, inventory drag/drop, or world input.
- Invalid layout data must be ignored safely.
- Import/export stays disabled unless it is plain local data, non-executable, non-scriptable, and not addon-compatible.
