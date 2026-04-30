# Action Bar Ability Contract

The action bar remains server-authoritative. Client drag/drop and keybind behavior request changes or activations through world endpoints and then re-render the returned session state.

## Assignment Rules

- Only valid learned active abilities can be assigned.
- Passive abilities are rejected server-side and do not create Spellbook drag payloads.
- Unlearned abilities remain visible in the Spellbook but cannot be assigned.
- Slot move, clear, and assign operations must preserve the server-returned action-bar state.

## Activation Rules

- Clicks and keybinds activate the slot only when the slot is learned, active, assignable, not on cooldown, and has required target/resource state.
- Cooldown, global cooldown, resource, range, and target-disabled feedback use real session payload fields only.
- Inventory and equipment drag payloads remain separate from ability drag payloads.

## Compatibility

The new fields `category`, `abilityType`, `passive`, `actionBarAssignable`, and `trainable` are additive. Older clients may ignore them; current clients must handle missing fields without crashing.
