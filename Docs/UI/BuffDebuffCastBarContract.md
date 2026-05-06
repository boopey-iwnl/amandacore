# Buff, Debuff, and Cast Bar Contract

Buffs, debuffs, and cast bars are display-only views of server-owned combat state.

## Data Source

- Player auras come from `worldSession.auras[]`.
- Target auras come from the selected entity's `auras[]`.
- Player cast state comes from `castEndsAt` and `castingAbilityId`.

## Behavior

- Split aura rows by reported `kind` when available: buffs, debuffs, and uncategorized effects.
- Show duration/count only when `expiresAtMs` or `stackCount` is reported.
- Use procedural or repo-local fallback icon styling only.
- Show a cast shell only when the server reports active cast timing.
- Do not draw fake cast progress unless a future contract provides real start/duration data.

## Unsupported States

- Target cast/channel bars are unavailable until visible entities expose target cast timing.
- Interrupt, cancel, and channel states are unavailable unless reported by the server.
- No fake aura effects, fake buff/debuff lists, or client-authored gameplay effects are allowed.

No addon integration is allowed.
