# Targeting and Nameplate Contract

Targeting and nameplates are built-in AmandaCore UI features backed by authoritative world-session state.

## Data Source

- Target selection uses `currentTargetId` from the world session payload.
- Nameplates use `entities[]` entries with `id`, `displayName`, `kind`, `x`, `y`, `z`, `health`, `maxHealth`, `alive`, `targetable`, `isInCombat`, and `currentTargetEntityId`.
- "Targeting you" is derived only when an entity's `currentTargetEntityId` matches the local `characterId`.

## Behavior

- Selected target emphasis appears on player, hostile, and friendly nameplates when projection succeeds.
- Nameplates are distance culled to avoid severe clutter.
- Hostile and player nameplates may show compact health bars from real health fields.
- Nameplates are draw-only foreground elements and must not consume input.
- UI target-frame feedback is the primary safe targeting surface; broader world-space selection markers are future work.

## Unsupported States

- No copied external nameplate layout, fonts, textures, threat behavior, or addon API is used.
- Occlusion and advanced stacking are not implemented in this first pass.
- Numeric threat values are not displayed because the visible entity payload does not expose them.

No addon integration is allowed.
