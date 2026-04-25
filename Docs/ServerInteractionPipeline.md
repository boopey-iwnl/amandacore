# Server Interaction Pipeline

## Canonical Flow

The current server interaction path is:

1. The client sends an intent to an HTTP world endpoint.
2. The world server validates session attachment from the world session token.
3. The world advances authoritative simulation time.
4. The command handler validates player state, target state, range, cooldown, and world context.
5. The world mutates authoritative state under the world mutex.
6. The server emits domain events and state diffs.
7. The response returns the latest authoritative player state, visible entities, combat fields, kill credits, state diffs, and recent domain events.

## Combat Commands

Target selection uses the existing target endpoint and now emits `combat.target.selected` or `combat.target.rejected`.

Ability use goes through the existing ability endpoint and now supports `dev_basic_strike`, an original AmandaCore fixed-damage hostile-NPC strike. The server owns damage, cooldown, death, and kill credit.

## State Diffs

The world response includes protocol-neutral deltas for future clients:

- `EntitySpawnDelta`
- `EntityHealthDelta`
- `EntityCombatStateDelta`
- `TargetSelectionDelta`
- `AbilityResultDelta`
- `EntityDeathDelta`
- `ProgressionDelta`

These names are AmandaCore state contracts, not wire packet names. Transport adapters should map them into whatever client protocol is current without changing simulation authority.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, assets, formulas, or stat tables were copied or adapted.
