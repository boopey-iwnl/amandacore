# Milestone 18 Follow-Up Roadmap: 3.3.5a-Inspired Guild Foundation

This roadmap keeps Milestone 18 focused on guild foundation while pointing the next slices toward the social cadence, UI density, and low-friction group play patterns expected from a Wrath-era MMO.

## Immediate Next Steps

1. Add guild member notes and officer notes, with officer notes permission-gated.
2. Add a guild log for joins, leaves, promotions, demotions, removals, MOTD edits, and disbands.
3. Add roster sorting and filters by online status, rank, class, level, and zone.
4. Add right-click context actions from guild roster rows for invite, whisper, promote, demote, remove, and party invite.
5. Add `/o` officer chat only after officer notes and rank permissions are stable.
6. Add server-side invite expiry cleanup telemetry so social-state polling does not hide invite lifecycle bugs.

## UI Direction

The Guild tab should stay compact and operational: roster first, management actions close to the selected member, and status text kept short. Avoid a decorative guild landing page. The useful first-screen signal is guild name, MOTD, online count, rank, and roster.

Recommended Guild tab layout:

- Header: guild name, player rank, online count.
- MOTD row: visible to all, editable only with permission.
- Invite row: one name field and invite button, visible only with invite permission.
- Roster table: name, rank, level, class, zone, online state, last online.
- Selected member action row: whisper, party invite, promote, demote, remove.
- Footer actions: leave guild, disband only for leader.

## Systems To Defer

Keep these out until the foundation has survived multi-client testing:

- Guild bank
- Guild achievements
- Guild leveling and perks
- Calendar
- Recruitment browser
- Tabards and cosmetics
- Guild halls
- Raid tooling

## Design Notes

The current foundation should remain realm-local, character-based, and server-authoritative. Do not make guild membership a combat, dungeon, quest, or economy dependency yet. Guilds should improve communication and identity first, then provide later hooks for banks, raids, and long-term progression.
