# Character Creation Screen Contract

## Scope

Character creation is an AmandaCore-original first-party screen. It does not use external game races, classes, text, UI code, UI art, fonts, or addon behavior.

## Options

The first-pass option model contains:

- lineage
- archetype
- body
- skin
- face
- hair
- hair color
- marking
- name

The archetype submitted in this milestone is `wayfarer_warden`. The other controls drive the local procedural preview only.

## Preview

The preview is a procedural placeholder with rotate, zoom, reset, and randomize controls. It does not require a world session and does not enable gameplay movement.

## Name Validation

The client precheck requires a name, 3 to 16 letters, and letters only. The backend remains authoritative for create success, name availability, and account/realm ownership.

## Submit Behavior

Create submits only the fields supported by the current backend contract: `realmId`, `displayName`, and `archetypeId`. On success, the client returns to character select, refreshes the list, and selects the newly created character when present.

Appearance choices are not persisted in Milestone 2 and must not be displayed as saved character appearance data after creation.

## Non-Goals

This milestone does not add backend appearance persistence, character deletion, addon support, Lua, user-installed UI modules, plugin runtime, or texture import.
