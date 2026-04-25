# Black-box Reference Workflow

This project uses observed outcomes as the fidelity oracle for WoW `3.3.5a`-style structure and pacing. It does not rely on copied source code, proprietary data extraction, or one-to-one protocol cloning.

## Capture workflow

1. Record player-visible behavior from a lawful reference environment.
2. Log measurable outcomes only:
   - movement speeds, acceleration, turn rates, and jump arcs
   - swing cadence, damage bands, crit frequency, and armor mitigation
   - aggro radius, leash distance, assist behavior, and evade timing
   - quest progression order, item drop frequency, respawn windows, and travel pacing
3. Distill captures into compact test vectors and replay fixtures.
4. Update the shared domain tests until the resulting outputs stay within agreed tolerances.

## Behavioral tolerances

- Prefer bounded acceptance ranges over exact byte-for-byte or frame-for-frame parity.
- Split tests into "must match" rules and "feel tuning" rules.
- Track every future tuning change with a fixture update and a note about why the change was accepted.

## Content boundaries

- Replacement quest text, names, item labels, zones, landmarks, and creature identities only.
- Replacement geometry and layout authored from scratch.
- No direct import or redistribution of third-party assets or game data in this repository.
