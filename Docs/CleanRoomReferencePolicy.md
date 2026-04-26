# Clean-Room Reference Policy

AmandaCore is an original clean-room MMO project. TrinityCore and AzerothCore may be used only as read-only architectural reference corpora for broad engineering lessons such as service separation, runtime ownership, migration discipline, testing practice, and operations posture.

This policy is engineering risk guidance, not legal advice.

## Non-Negotiable Boundary

Do not copy, translate, port, adapt, paste, vendor, import, or derive any artifact from TrinityCore, AzerothCore, MaNGOS, WoW private-server projects, or proprietary game data.

Forbidden artifacts include:

- source code, source comments, constants, algorithms as written, build scripts, and generated source
- SQL, schemas, migrations, table names, column names, seed data, and database layouts
- packet formats, opcodes, protocol fields, sniffed captures, parser outputs, and compatibility-tool outputs
- command names, GM/admin vocabularies, permission names, remote-access protocols, and console surfaces
- content IDs, quest IDs, item IDs, spell IDs, NPC IDs, encounter data, names, assets, scripts, maps, and authored gameplay data
- copied dashboards, alert rules, metric taxonomies, or logging categories

## Required AmandaCore Originality

AmandaCore must use original:

- service contracts
- package names
- schemas and migrations
- internal commands and domain events
- wire protocols and adapter contracts
- content manifests and content IDs
- admin APIs, permissions, audit events, and moderation workflows
- observability names and dashboards
- implementations and tests

Generic MMO/server architecture patterns are allowed when expressed as AmandaCore-owned designs and implementation. Examples include identity/control-plane separation, realm directory status, single-use join tickets, session gateway boundaries, fixed-step simulation, shard/zone ownership, structured observability, migration discipline, admin audit/RBAC, replay tests, and content packages.

## Required Reference Workflow

Any future work influenced by TrinityCore or AzerothCore must follow this workflow:

1. Treat upstream projects as read-only reference corpora.
2. Write neutral notes that describe abstract responsibilities or failure modes.
3. Convert those notes into an AmandaCore design note using AmandaCore names, concepts, and contracts.
4. Implement only from the AmandaCore design note.
5. Review the implementation for provenance and accidental artifact reuse.

Do not implement from upstream source side-by-side. Do not keep upstream source open while writing AmandaCore code.

## Codex Guardrails

Codex must not:

- open, clone, vendor, import, or paste TrinityCore or AzerothCore source into this repository
- create a fork plan from those projects
- translate upstream code, SQL, schemas, packet layouts, constants, comments, scripts, commands, IDs, or data structures into AmandaCore
- introduce upstream names or content identifiers as placeholders
- create compatibility artifacts unless a future explicitly approved task includes legal and provenance review

If a task appears to require upstream artifacts, stop and convert it into an AmandaCore-native design question first.
