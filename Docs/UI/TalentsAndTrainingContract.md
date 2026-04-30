# Talents and Training Contract

Talents, ability training, and professions are first-party AmandaCore UI systems. They are server-authoritative and must not emulate addon behavior.

## Ability Trainer

Trainer offers are read from the world session `trainer.offers` payload. The UI shows learned, available, unavailable, insufficient-cost, and requirement states from server data. The Learn button is enabled only when `canLearn` is true and calls `/v1/world/trainer/learn`.

Player-facing class labels use `Warrior` for the current default class. Internal IDs such as `wayfarer_warden` remain compatibility identifiers only.

## Talents

The Talents panel renders current server-backed Disciplines/Paths-style data from `talents`. Selection mutates state only through `/v1/world/talent/select`. If the server reports no available points or the entry cannot be selected, the UI shows disabled state and does not fake point spending.

## Professions

The Professions panel renders:

- learned professions
- profession catalog
- profession trainer offers
- read-only recipe summaries

Learning is allowed only through `/v1/world/profession/learn` when the server marks the offer `canLearn`. M4 does not add crafting buttons, crafting mutations, or client-authoritative item/currency changes.

## Unsupported Systems

If a backend payload is absent, the panel must show a clear unavailable state. The UI must not fabricate learned abilities, talent points, professions, recipes, costs, or successful training.
