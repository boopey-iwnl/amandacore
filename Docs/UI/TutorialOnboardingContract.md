# Tutorial and Onboarding Contract

Tutorials are local, first-party AmandaCore hints rendered by the in-client UI.

## Tutorial Hints

Supported hint families:

- first world load
- movement and camera
- NPC interaction
- quest guidance
- combat basics
- inventory and equipment
- spellbook and action bars
- action-bar editing
- settings and keybinds
- chat and safe slash commands

## State

- Tutorial completion is stored locally in `%LOCALAPPDATA%\AmandaCore\ui-settings.ini`.
- Missing or invalid tutorial state falls back to safe defaults.
- Hints can be dismissed individually.
- Settings can disable tutorials or reset dismissed hints.
- `/tutorials` enables hints and `/tutorials reset` clears dismissed state.

## Rules

- Tutorial hints are non-modal and should not trap gameplay input.
- The client may preview and explain UI behavior, but gameplay authority remains server-owned.
- No backend profile sync is added in this milestone.
- No external tutorial packs, addon tutorials, Lua scripts, or runtime-loaded modules are supported.
