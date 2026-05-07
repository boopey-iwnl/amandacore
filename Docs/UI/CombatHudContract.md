# Combat HUD Contract

The Combat HUD is the first-party player and target combat surface rendered by `Gems/UiClient`.

## Data Source

- Player state comes from the world session payload: `health`, `maxHealth`, `resource`, `maxResource`, `resourceName`, `alive`, `autoAttackActive`, `globalCooldownEndsAt`, `castEndsAt`, `castingAbilityId`, and `auras`.
- Target state comes from `currentTargetId` and the matching `entities[]` entry.
- Combat feed state comes from `domainEvents`, `stateDiffs`, and `killCredits`.

## Behavior

- Show player health/resource clearly and update from server responses.
- Show target name, kind, health, alive state, distance, AI/combat state, and real aura rows when a target is selected.
- Clear the target frame when `currentTargetId` is empty or no matching visible entity is present.
- Show defeated/respawn state only when the visible entity reports real death or respawn fields.
- Show low-health styling based on real health ratios only.
- Do not calculate damage, mitigation, death, cooldowns, or aura ticks locally.

## Unsupported States

- Numeric threat meters are unavailable unless the server exposes numeric threat summaries.
- Target-of-target is limited to the reported visible entity target identifier and is not a full roster-resolved panel.
- Player respawn timing is not displayed unless the session payload later exposes it.

No addon integration is allowed.
