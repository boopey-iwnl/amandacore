# Spellbook Contract

The Spellbook is a built-in AmandaCore panel backed by the world session payload. It does not load addons or user scripts.

## Tabs

- All: every server-exposed ability entry.
- Active: learned or unavailable abilities that are active actions.
- Passive: passive entries, displayed as non-draggable.
- Utility: content category for non-class utility actions.
- Warrior: player-facing class category for the current default class while preserving internal compatibility IDs.

## Ability Entry Fields

The UI consumes these optional additive fields when present:

- `category`
- `abilityType`
- `passive`
- `learned`
- `trainable`
- `actionBarAssignable`
- `tooltipText`
- `requirementText`
- `resourceName`
- `resourceCost`
- `resourceGeneration`
- `cooldownMs`
- `rangeMeters`
- `requiresTarget`

Missing metadata must degrade safely. Unsupported facts are omitted instead of showing fake numbers.

## Drag Rules

Only learned active abilities with action-bar assignability can be dragged. Passive, unlearned, unsupported, or invalid entries remain visible but do not create action-bar drag payloads.

## Tooltips

Tooltips show server-backed facts only: name, active/passive type, category, cost/generation, cooldown, range, description, source/requirement text, and disabled reason. Tooltip text must stay on screen and must not copy external MMO wording.
