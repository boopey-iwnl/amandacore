# UI Milestone 6 - Combat HUD, Targeting, and Feedback

Milestone 6 polishes AmandaCore's first-party combat HUD without adding addon support or changing server authority. The implementation consumes existing world-session combat fields and keeps ability, health, aura, death, target, cooldown, and threat decisions owned by the world service.

## Current Audit

- Player health, max health, resource, alive state, current target, auto-attack state, global cooldown, cast end time, active auras, kill credit, domain events, state diffs, and visible entities already arrive through the world session payload.
- Visible entities already include health, max health, alive, targetable, AI state, aura state, combat state, current target, last damaging entity, death tick, respawn tick, and respawn delay where the server has that data.
- Target selection is authoritative through `/v1/world/target`; rejected selections and target changes are exposed through domain events and state diffs.
- Ability activation remains authoritative through `/v1/world/attack/ability`; action-bar availability in the client is visual feedback and does not replace server validation.
- Aura and cast support exists as a server-owned skeleton. The client displays only reported aura rows and reported cast timing; it does not invent buffs, debuffs, channels, interrupts, or fake durations.
- Threat support exists as AmandaCore-owned mob target/threat state. The UI only exposes the real "targeting you" shell from visible entity target data and does not display invented threat values.
- The previous friendly NPC nameplate projection path is reusable for safe first-pass hostile and player nameplates.
- No texture import was needed for this milestone. Existing procedural ImGui styling and repo-local icon fallbacks remain sufficient.

## Implemented Behavior

- Player and target frames show readable health/resource state, alive/defeated state, target distance, real aura rows, and current combat/AI labels where available.
- Hostile target frames show an honest threat shell: "targeting you", "targeting another entity", or "no target reported" based only on parsed authoritative target data.
- Player cast state shows a shell only when the world response reports active cast timing. No fake progress bar is drawn without a real start/duration contract.
- Buff/debuff presentation is split by reported aura `kind` when available, with an "effects" row for uncategorized real auras.
- Hostile and player nameplates now use the existing projection path, selected-target emphasis, distance culling, and compact health bars for real health data.
- Floating combat feedback pulses are screen-space, non-interactive, rate-limited, and sourced only from real `domainEvents` and `stateDiffs`.
- Action bars preserve UI M4 passive, drag, rearrange, clear, click, and keybind behavior while showing cooldown, resource, target, and range overlays from real payload fields.

## Limitations And Follow-Up

- Target cast bars are not implemented because visible entities do not currently expose target cast timing.
- Cast progress is not drawn because the client receives cast end time but not a cast start time or duration.
- Threat meters are not implemented because the UI receives target state, not numeric threat values.
- World-space floating damage numbers remain future work. M6 adds a safe HUD-level combat pulse/feed foundation.
- Nameplate culling is intentionally conservative and does not attempt occlusion, stacking, or advanced clutter solving.
- Player death/respawn UI is limited to the current alive state; hostile respawn detail appears only when reported on the visible entity.

## Safety Notes

- No addon API, Lua loader, AddOns folder, plugin runtime, user-installed UI modules, or arbitrary UI script execution is added.
- No local texture source folder is referenced by runtime code, manifests, package scripts, material files, or docs-as-config.
- No combat formulas, threat formulas, external MMO spell data, aura data, copied UI code, copied fonts, or copied assets are introduced.
