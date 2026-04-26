# Clean-Room Study: TrinityCore and AzerothCore

## Purpose

This document captures engineering lessons AmandaCore can learn from TrinityCore and AzerothCore while preserving a strict clean-room boundary.

TrinityCore and AzerothCore are useful read-only architectural reference corpora. They show how mature MMO server projects organize runtime processes, operational controls, persistence boundaries, update discipline, testing workflows, and extension practices. They are not AmandaCore starting points, fork bases, porting sources, schema sources, protocol sources, or content sources.

This document is engineering risk guidance, not legal advice. Legal questions about licenses, derivative works, distribution, or product risk must be reviewed by qualified counsel before any release decision depends on them.

## Executive Summary

Treating TrinityCore and AzerothCore as read-only study material is valuable. Treating either project as AmandaCore's base codebase or as a long-lived fork is not aligned with AmandaCore's goal of an original MMO server unless the project deliberately accepts the license, compatibility, and content-coupling consequences of that path.

The useful engineering lessons are:

- Keep identity, realm directory, and world simulation responsibilities separate.
- Keep network/session handling separate from authoritative simulation ownership.
- Use explicit world-loop, shard, map, or instance update controls.
- Treat persistence and migrations as first-class engineering systems.
- Build observability, remote operations, supervised startup, and validation workflows early.
- Use modularity and extension governance to protect the core.
- Require automated tests and manual scenario validation for gameplay-heavy changes.

The risky artifacts to avoid are:

- Source code, source structure, source comments, and translated logic.
- SQL schemas, migration files, table skeletons, table names, and content database organization.
- Packet layouts, protocol fields, opcodes, sniffed artifacts, generated parsers, and compatibility tooling outputs.
- Remote command vocabularies, admin verbs, permission names, and console surfaces.
- Metrics dashboards, generated dashboard JSON, and copied metric taxonomies.
- Names, assets, content identifiers, encounter data, item identifiers, quest identifiers, spell identifiers, and other content surfaces.

AmandaCore should extract principles, not artifacts. The implementation path should be an original Go service stack with an internal protocol-agnostic simulation core, original data models, original schemas, original wire protocols, original content IDs, and an AmandaCore-owned content pipeline.

## Clean-Room Boundary

AmandaCore uses TrinityCore and AzerothCore only as read-only reference corpora for architecture study. The allowed activity is observing broad engineering patterns, writing neutral notes, creating AmandaCore-specific RFCs, and then implementing AmandaCore systems from those original RFCs.

The clean-room workflow is:

1. Read upstream documentation and repository organization only for architectural study.
2. Capture analyst notes in neutral language.
3. Convert notes into AmandaCore-specific architecture RFCs and backlog items.
4. Implement AmandaCore systems from those RFCs without side-by-side porting.
5. Review changes for provenance, taint risk, and accidental artifact reuse.
6. Preserve provenance records with the related design documents and implementation reviews.

The bright-line rules are:

- Do not copy source code.
- Do not translate source code into another language.
- Do not copy SQL, migrations, schema skeletons, table names, or data layouts.
- Do not copy packet layouts, protocol identifiers, opcodes, parser outputs, or sniffed artifacts.
- Do not copy command names, permission names, dashboard JSON, metric names, or admin vocabularies.
- Do not copy names, content IDs, assets, quest structures, encounter data, item data, spell data, or other content.
- Do not use either upstream project as a fork plan.
- Do not preserve upstream surface identity by renaming fields or changing syntax.

Any subsystem inspired by upstream observations must be rewritten as an AmandaCore-native design first. Implementation should follow that AmandaCore design, not the upstream source.

## What We Can Learn

AmandaCore can learn high-level engineering patterns that are not tied to copied artifacts:

- How mature MMO servers separate login, realm discovery, session handling, and world simulation.
- How a world process can expose explicit update controls, queue boundaries, freeze detection, and runtime health metrics.
- How spatial ownership can be organized around zones, maps, instances, or equivalent authority domains.
- How gameplay data tends to separate immutable design-time descriptors from mutable runtime state.
- How content-heavy systems benefit from validation, migration discipline, and dedicated tooling.
- How modular extension boundaries reduce long-term fork pressure.
- How operations practice benefits from supervised startup, environment-driven configuration, logs, metrics, and runbooks.
- How community server projects use tests, manual tester approval, and review gates to reduce regression risk.

These are general architectural lessons. They must be expressed in AmandaCore's own service names, data contracts, protocols, schemas, tools, and content package formats.

## What We Must Not Copy

AmandaCore must not copy or derive implementation artifacts from TrinityCore or AzerothCore. The following are out of scope for reuse:

- GPL-covered source files, functions, algorithms as written, class/module organization, comments, build scripts, or generated source.
- SQL files, migrations, table names, column layouts, relationship skeletons, seed data, or content database organization.
- Protocol details, packet structures, opcodes, field names, parser outputs, captured traffic, compatibility-tool outputs, or wire-level assumptions.
- Remote admin protocols, command vocabularies, role names, permission names, console commands, or management surfaces.
- Dashboard JSON, alert definitions, exact metric names, logging categories, or copied operational assets.
- Names, assets, gameplay identifiers, quest identifiers, item identifiers, spell identifiers, encounter identifiers, NPC identifiers, and authored content.

AmandaCore should also avoid low-value similarity where originality is easy. New names, new domain concepts, new content IDs, and new operational vocabulary are cheaper than provenance ambiguity.

## TrinityCore Architectural Lessons

TrinityCore is most useful as a study source for runtime decomposition and compatibility-centric server concerns.

Relevant architectural lessons:

- Authentication and world simulation are separate process concerns.
- A world runtime benefits from explicit controls around update cadence, map or spatial update work, worker counts, freeze detection, and runtime metrics.
- Network ingress, session management, and authoritative simulation should be separated by clear ownership boundaries.
- Spatial simulation usually needs authority domains that can load, update, unload, reset, and report health independently.
- Tooling for extraction, parsing, compatibility, deployment, and metrics can grow around an MMO server quickly; AmandaCore should keep tools isolated from production authority.
- Observability should measure tick/update time, queue depth, player/session counts, load/unload behavior, persistence pressure, and uptime from early milestones.
- Compatibility-oriented projects accumulate client-specific and content-specific assumptions; AmandaCore must keep those assumptions outside its core.

AmandaCore should adapt the runtime spine concept, not any TrinityCore artifacts. The clean-room expression is a protocol-agnostic session gateway, a shard-owned simulation core, AmandaCore-owned content packages, and original observability vocabulary.

## AzerothCore Architectural Lessons

AzerothCore is most useful as a study source for maintainability, modularity, operations discipline, and contribution workflow.

Relevant architectural lessons:

- A module system can keep the core cleaner if extension boundaries are explicit and versioned.
- Modules should own their configuration and data lifecycle instead of requiring persistent source patches to the core.
- Schema and data updates need disciplined versioning, validation, cleanup, and squashing policies.
- Startup scripts, supervisors, environment overrides, and documented local workflows reduce operator error.
- Unit-test instructions, pull-request test guides, data-only validation flows, review requirements, and manual tester approval are part of the engineering system.
- Tooling and documentation can make content authoring safer, but only if generated assets and runtime truth are versioned and reproducible.

AmandaCore should adapt the governance model, not the module API. The clean-room expression is an AmandaCore plugin/content package SDK, original package metadata, original migration machinery, and project-owned validation tools.

## AmandaCore Target Architecture

AmandaCore should be an original MMO server stack with the following major layers:

- Control plane: identity, realm directory, admin API, audit, configuration, and feature flags.
- Edge/session plane: protocol adapters, session gateway, authentication of live sessions, backpressure, and rate limits.
- Simulation plane: shard coordinator, zone shards, instance shards, and gameplay services.
- Content plane: original source manifests, validators, compiler, compiled content packages, and package versioning.
- Persistence plane: identity state, player/account state, world/runtime state, journals, snapshots, and migrations.
- Observability plane: metrics, traces, structured logs, replay export, audit records, and alerts.

The central design rule is that the simulation core must consume canonical AmandaCore commands and emit canonical AmandaCore events. External client protocols are adapters, not the internal model.

This protects AmandaCore from compatibility-first design pressure and lets future protocol work evolve without contaminating the simulation core.

## Recommended Service Boundaries

Recommended AmandaCore service boundaries:

- Identity Service: owns users, credentials, account security state, MFA enrollment, bans, and auth risk policy.
- Realm Directory Service: owns available realms, realm status samples, supported build metadata, and signed realm-status publication.
- Session Gateway: accepts authenticated live sessions, applies rate limits, decodes protocol-adapter output into canonical commands, and routes sessions to shard owners.
- Protocol Adapters: translate external transports into AmandaCore canonical commands and translate outgoing diffs/events back to external presentation formats.
- Shard Coordinator: resolves spatial ownership, routes sessions, manages shard handoff, and exposes shard health.
- Zone Shards: own persistent outdoor-area simulation authority.
- Instance Shards: own temporary or resettable private simulation authority.
- Gameplay Services: own combat, effects, threat, quests, progression, AI, loot, inventory, social, and economy logic behind canonical domain contracts.
- Persistence Writer: persists events, snapshots, state transitions, and migration metadata outside the simulation hot path.
- Admin API: exposes typed operational actions with RBAC, audit, approval controls, and safe automation.

Service boundaries should be expressed in AmandaCore-native interfaces and contracts. They should not mirror upstream file, process, command, schema, or protocol surfaces.

## Recommended Simulation/Core Patterns

Recommended simulation patterns:

- Use a fixed-step simulation clock based on monotonic time.
- Use single-writer ownership per zone, instance, encounter, or equivalent authority domain.
- Put network work, persistence work, and simulation mutation behind explicit queues and backpressure.
- Validate every client command before it enters authoritative state mutation.
- Keep combat resolution deterministic within a tick.
- Order command resolution by stable AmandaCore-owned IDs and explicit priority rules.
- Represent abilities as typed effect graphs with explicit validation, target resolution, effect expansion, application, lifecycle, and event emission phases.
- Keep threat as a first-class encounter ledger rather than hidden script state.
- Represent quests and progression as event-fed objective graphs.
- Represent loot as auditable reward rules with deterministic seeding and explicit ownership/distribution policy.
- Keep AI local to the shard or encounter owner and drive it from original behavior trees, state machines, or planning DSLs.
- Export replays for simulation debugging and regression tests.

These patterns are AmandaCore design choices. They are not instructions to port any upstream implementation.

## Recommended Content Pipeline

AmandaCore content should be original, reproducible, and generated from AmandaCore-owned source manifests.

Recommended pipeline:

1. Author neutral content manifests in AmandaCore-owned formats.
2. Validate manifests with schema and cross-reference checks.
3. Compile manifests into versioned runtime content packages.
4. Store package metadata, checksums, dependency information, and compatibility rules.
5. Load packages through a controlled runtime content registry.
6. Support safe content reload where the runtime model can prove compatibility.
7. Keep O3DE or other authoring outputs as inputs, not as authoritative runtime truth.
8. Preserve provenance for generated packages and source manifests.

Recommended content domains:

- NPC archetypes, spawn points, interaction profiles, movement profiles, and reward references.
- Ability specs, cast rules, target resolvers, effect lists, aura/effect lifecycle rules, and modifier categories.
- Quest/story objective graphs, completion predicates, trigger references, reward bundles, and conversation references.
- Loot/reward tables, roll groups, conditions, weights, ownership rules, and distribution modes.
- Zone topology, path graphs, encounter markers, trigger volumes, and presentation metadata.

Do not import or reshape upstream content data. AmandaCore content IDs, names, manifests, schemas, and compiler outputs must be original.

## Recommended Persistence/Migration Practices

AmandaCore should treat persistence and migrations as product infrastructure.

Recommended persistence domains:

- Identity domain: users, credentials, MFA, bans, account risk, and auth audit.
- Player/account domain: characters, inventory, progression, social state, entitlements, and player-owned data.
- World/runtime domain: realm state, shard state, instance runs, market or guild state, moderation state, and operational records.
- Content domain: content package metadata, version compatibility, checksums, and deployment history.

Recommended migration practices:

- Use immutable migration IDs and checksums.
- Support dry-run validation.
- Enforce migration order and dirty-state detection in CI.
- Separate schema migrations from content package versioning.
- Keep rollback drills and restore procedures documented.
- Use environment-driven configuration without changing source.
- Preserve migration provenance and review records.
- Keep simulation hot paths away from synchronous database work.
- Prefer event journals and snapshots where replay, auditability, or recovery matter.

Do not copy upstream SQL or schema shapes. AmandaCore migrations should be generated from AmandaCore aggregates and data ownership decisions.

## Recommended Observability/Ops Practices

AmandaCore should make operations visible from the first playable milestones.

Recommended metrics and signals:

- Shard tick duration and missed tick budget.
- Command queue depth and queue wait time.
- Session counts, attach/detach rates, and disconnect reasons.
- Protocol adapter errors and malformed message counts.
- Persistence write latency, queue depth, and retry counts.
- Zone/instance load, unload, reset, and handoff counts.
- Entity counts by shard and authority domain.
- Login failure rates, MFA events, bans, and risk decisions.
- Admin actions, approval decisions, and audit records.
- Replay export counts and replay validation results.
- Content package load, validation, and compatibility failures.
- Anomaly scores and moderation review queue volume.

Recommended operations practices:

- Use structured logs with stable AmandaCore event names.
- Use distributed traces for account-to-world and session-to-shard flows.
- Build dashboards from AmandaCore-owned definitions.
- Use supervised startup and graceful shutdown.
- Keep configuration environment-driven and documented.
- Expose health checks for identity, realm directory, session gateway, shard coordinator, persistence, and admin services.
- Keep remote administration behind authenticated HTTPS or gRPC with RBAC and audit.

Do not copy upstream dashboard JSON, metric names, logging categories, remote admin protocols, or command surfaces.

## Recommended Testing/Governance Practices

AmandaCore should adopt a test and governance model that matches MMO server risk.

Recommended test suites:

- Unit tests for pure rules, validators, and service contracts.
- Deterministic simulation tests for movement, combat, effects, threat, loot, and progression.
- Migration tests for schema evolution, dirty-state handling, rollback expectations, and seeded local environments.
- Content validation tests for package compilation, references, compatibility, and runtime load checks.
- Replay regression tests for canonical scenarios and previously fixed bugs.
- Protocol fuzzing and malformed-message tests at adapter boundaries.
- Scenario tests for account-to-world, shard handoff, instance lifecycle, quest completion, combat loops, loot distribution, and admin moderation flows.

Recommended governance practices:

- Require design review for new service boundaries and content package contracts.
- Require migration review for persistence changes.
- Require provenance review for systems influenced by upstream study.
- Require automated green checks before merge.
- Require manual scenario signoff for gameplay-heavy changes.
- Keep implementation notes separate from read-only upstream study notes.
- Keep clean-room attestations with major subsystem PRs.

## Prioritized AmandaCore Roadmap

Immediate priorities:

1. Maintain the clean-room policy and provenance archive.
2. Build or harden the Identity Service and Realm Directory Service boundaries.
3. Build the canonical Session Gateway and protocol-adapter boundary.
4. Establish single-writer zone and instance shard ownership.
5. Define original persistence domains and migration runner behavior.
6. Add metrics, structured logs, and replay export for the first authoritative simulation loop.

Near-term priorities:

1. Build an Admin API with RBAC, audit, and approval controls.
2. Build the original content compiler and content package format.
3. Implement the combat/effect/threat pipeline.
4. Implement the quest/progression graph runtime.
5. Implement the loot/reward rule engine.

Mid-term priorities:

1. Implement the AI runtime and authored behavior DSL.
2. Define the plugin/module SDK.
3. Add safe content reload where package compatibility allows it.
4. Add replay regression and fuzzing harnesses.

Later priorities:

1. Add multi-region federation and transfer flows only after the single-region architecture is stable.
2. Revisit any legacy-client compatibility adapter only if it becomes an explicit product goal and receives a separate legal and clean-room review.

Always reject:

- Direct forks of TrinityCore or AzerothCore.
- Side-by-side source translation.
- Copied SQL, schemas, packets, commands, content IDs, assets, dashboards, or compatibility artifacts.

## Implementation Checklist

Clean-room process:

- Analyst notes are stored separately from implementation branches.
- AmandaCore RFCs use neutral language and AmandaCore-native names.
- Implementation PRs cite AmandaCore RFCs, not upstream source files.
- Review includes a provenance check for artifact reuse.

Identity and realm directory:

- Identity state is separate from world simulation state.
- Realm status publication is signed or otherwise trust-bounded.
- Auth and realm APIs use AmandaCore-owned schemas and tokens.

Session and protocol:

- Protocol adapters produce canonical AmandaCore commands.
- Session Gateway owns rate limits, backpressure, and session routing.
- Simulation code does not depend on external packet layouts.

Shard simulation:

- Each zone or instance has a single writer.
- Shard handoff is explicit and testable.
- Tick budget, queue depth, and replay export are observable.

Content:

- Source manifests are AmandaCore-owned.
- Content packages are compiled, versioned, and checksumed.
- O3DE or other authoring data is treated as an input to the compiler.
- Content IDs and names are original.

Persistence:

- Identity, player/account, world/runtime, and content metadata domains are separated.
- Migration IDs and checksums are immutable.
- CI validates migration ordering and dirty-state handling.

Admin and operations:

- Admin API uses RBAC and short-lived credentials.
- Destructive actions are audited and can require approval.
- Remote control surfaces are not telnet-style or command-shell-style public endpoints.

Testing:

- Unit, deterministic simulation, migration, content validation, replay, fuzz, and scenario tests are defined.
- Gameplay-heavy changes require manual scenario signoff.

## Legal/Provenance Notes

This document is engineering risk guidance, not legal advice. It does not decide license obligations, derivative-work questions, distribution obligations, or release risk. Counsel should review any legal conclusion before AmandaCore relies on it.

Working assumptions for engineering practice:

- GPL-covered code is not copied into AmandaCore.
- GPL-covered source is not translated into AmandaCore source.
- Upstream SQL, schemas, packet layouts, opcodes, command vocabularies, dashboards, assets, and content IDs are not copied.
- TrinityCore and AzerothCore remain read-only reference corpora.
- AmandaCore implementation is original and based on AmandaCore-owned RFCs, schemas, protocols, data models, content IDs, and tools.
- Provenance records are preserved for any subsystem influenced by upstream architectural study.

If a future product goal requires compatibility with a specific legacy client, that goal must be handled as a separate project with explicit legal review, clean-room controls, protocol provenance review, and documented acceptance of the associated product risk.
