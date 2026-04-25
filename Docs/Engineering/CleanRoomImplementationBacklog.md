# Clean-Room Implementation Backlog

This backlog converts the clean-room architecture study into AmandaCore implementation epics. It is a planning document only. It does not create GitHub issues, milestones, or implementation commitments by itself.

Every epic must preserve the clean-room boundary: TrinityCore and AzerothCore are read-only reference corpora, and AmandaCore must use original implementation, original schemas, original protocols, original content IDs, original data models, and original operational vocabulary.

## Epic 1 - Identity and Realm Directory

Goal:

- Define and harden AmandaCore's account identity, authentication, ban/risk state, MFA-ready account security, and realm-directory publication boundaries.

Why it matters:

- Identity and realm discovery are foundational control-plane concerns. Keeping them outside world simulation reduces coupling and lets realm status evolve independently from shard runtime.

Clean-room constraint:

- Use AmandaCore-owned account schemas, credential flows, token formats, realm descriptors, and status payloads. Do not copy upstream database shapes, auth flows, ticket formats, or realm list structures.

Likely files/systems affected:

- `Services/cmd/auth-service`
- `Services/cmd/account-service`
- `Services/cmd/realm-service`
- `Services/internal/platform`
- `Docs/RuntimeContract.md`
- configuration examples under `Docs/Config`

Proposed acceptance tests:

- Register, login, refresh, logout, and password-change flows pass in local/staging mode.
- Realm directory publishes signed or trust-bounded status for configured realms.
- Bans and disabled accounts prevent login and are audited.
- Realm service failure does not corrupt identity state.
- Contract tests confirm payloads are AmandaCore-native.

Suggested milestone/release window:

- Immediate, before expanding authoritative world scale.

Risks:

- Token/session ambiguity between account services and world join.
- File-backed local state diverging from the future database-backed model.
- Under-specified ban, MFA, and account recovery workflows.

## Epic 2 - Session Gateway and Protocol Adapter Boundary

Goal:

- Build a clear boundary where external transports and protocol adapters produce canonical AmandaCore commands for the simulation core.

Why it matters:

- This prevents packet or client compatibility assumptions from entering authoritative simulation code and keeps future client protocols replaceable.

Clean-room constraint:

- Define original AmandaCore command/event contracts. Do not copy packet layouts, field names, opcodes, parser outputs, sniffed artifacts, or compatibility-tool behavior.

Likely files/systems affected:

- `Shared/AmandaCoreShared`
- `Services/cmd/world-service`
- `Services/internal/platform`
- network/runtime integration code under `Client` and `Services`
- `Docs/RuntimeContract.md`

Proposed acceptance tests:

- Protocol adapter tests translate sample client intents into canonical AmandaCore commands.
- Malformed messages are rejected before reaching simulation.
- Backpressure and rate-limit behavior is observable.
- Session attach, detach, and reconnect scenarios are covered.
- Simulation tests can run without any external protocol dependency.

Suggested milestone/release window:

- Immediate, before adding additional protocol features.

Risks:

- Leaking presentation or transport details into shared simulation types.
- Underestimating reconnect, duplicate command, and handoff behavior.
- Creating a gateway that is too thin to enforce abuse controls.

## Epic 3 - Single-Writer Zone/Instance Shard Simulation

Goal:

- Establish single-writer ownership for zone and instance simulation, including tick scheduling, command queues, handoff, and replay export hooks.

Why it matters:

- MMO simulation needs deterministic authority. Single-writer shard ownership reduces race conditions and makes combat, AI, quests, and persistence easier to reason about.

Clean-room constraint:

- Use AmandaCore-owned shard, zone, instance, tick, and handoff models. Do not mirror upstream map ownership APIs, update-loop implementations, or runtime class structures.

Likely files/systems affected:

- `Services/cmd/world-service`
- `Services/internal/worlds`
- `Shared/AmandaCoreShared`
- O3DE runtime Gems related to zone streaming and server authority
- `Docs/Architecture.md`

Proposed acceptance tests:

- A zone shard processes commands in deterministic tick order.
- An instance shard can start, reset, and stop without corrupting player state.
- Handoff tests move a session between authority domains.
- Queue depth and tick duration metrics are emitted.
- Replay output can reproduce a deterministic scenario.

Suggested milestone/release window:

- Immediate to near-term, before complex combat or AI.

Risks:

- Cross-shard interactions becoming implicit shared state.
- Tick budget pressure from persistence or network work.
- Handoff semantics being delayed until too late.

## Epic 4 - Original Persistence Domains and Migration Runner

Goal:

- Define AmandaCore's persistence domains and migration runner for identity, player/account state, world/runtime state, and content metadata.

Why it matters:

- Persistence decisions determine recovery, replay, deployment safety, and long-term content evolution. Migration discipline prevents dirty environments and non-reproducible installs.

Clean-room constraint:

- Create AmandaCore-native schemas from AmandaCore aggregates. Do not copy SQL, table names, relationship skeletons, seed data, migration files, or data organization from upstream projects.

Likely files/systems affected:

- `Services/internal/platform`
- service storage layers under `Services`
- `Docs/Config`
- deployment and local environment scripts
- future database migration directories

Proposed acceptance tests:

- Migration runner applies immutable ordered migrations with checksums.
- Dirty migration state fails safely.
- Dry-run migration validation runs in CI.
- Identity, player/account, world/runtime, and content metadata can be reset independently in local dev.
- Snapshot and journal behavior is covered for at least one simulation aggregate.

Suggested milestone/release window:

- Immediate, before data model growth accelerates.

Risks:

- Local file storage shaping the future database model accidentally.
- Migrations coupled to content package changes without explicit versioning.
- Synchronous persistence calls entering shard hot paths.

## Epic 5 - Observability, Metrics, Structured Logs, Replay Export

Goal:

- Add first-class observability for services, sessions, shards, persistence, admin actions, content packages, and simulation replay.

Why it matters:

- MMO failures often appear as latency, queue pressure, inconsistent state, or hard-to-reproduce gameplay regressions. Observability must exist before scale hides root causes.

Clean-room constraint:

- Use AmandaCore-owned metric names, log event names, dashboards, alert definitions, and replay formats. Do not copy upstream dashboard JSON, metric taxonomies, or logging categories.

Likely files/systems affected:

- `Services/internal/platform`
- `Services/cmd/*`
- `Shared/AmandaCoreShared`
- deployment scripts/configuration
- `Docs/QA`

Proposed acceptance tests:

- Health checks exist for identity, realm, session/world, persistence, and admin services.
- Shard tick time, command queue depth, session counts, and persistence latency are emitted.
- Structured logs include correlation IDs across account-to-world flows.
- Replay export captures a deterministic scenario and can be validated offline.
- Admin actions produce immutable audit records.

Suggested milestone/release window:

- Immediate for baseline metrics and logs; near-term for replay export.

Risks:

- Adding observability after code paths are already hard to instrument.
- Metric cardinality problems from player/session identifiers.
- Replay export exposing private data without filtering.

## Epic 6 - Admin API with RBAC and Audit

Goal:

- Build a secure AmandaCore admin API for moderation, account operations, realm operations, shard inspection, and audited operational actions.

Why it matters:

- Operational control is necessary, but unsafe remote control surfaces become security liabilities. Admin actions need RBAC, short-lived credentials, audit, and approval gates for destructive operations.

Clean-room constraint:

- Use original API routes, permission names, operation names, audit events, and role models. Do not copy upstream remote-admin protocols, command vocabularies, console commands, or permission structures.

Likely files/systems affected:

- `Services/cmd/admin-service`
- `Services/internal/platform`
- account and realm service integrations
- `Docs/RuntimeContract.md`
- deployment configuration

Proposed acceptance tests:

- Admin login or service authentication requires privileged credentials.
- RBAC denies unauthorized account, realm, and shard operations.
- Destructive operations require explicit approval or elevated permission.
- Every admin action writes an audit record.
- Admin API remains unavailable over unauthenticated local or public command channels.

Suggested milestone/release window:

- Near-term, after identity and realm boundaries are stable.

Risks:

- Overly broad roles becoming permanent shortcuts.
- Missing audit details for moderation disputes.
- Admin operations racing against live simulation state.

## Epic 7 - Original Content Compiler and Content Package Format

Goal:

- Define AmandaCore-owned source manifests, validators, compiler outputs, package metadata, checksums, and runtime package loading.

Why it matters:

- Content is core product surface. A compiler and package format keep runtime data reproducible, validateable, and separate from external authoring tools.

Clean-room constraint:

- Use original content schemas, source manifest names, package metadata, content IDs, and compiled formats. Do not import upstream content data, schema layouts, table names, identifiers, names, or assets.

Likely files/systems affected:

- `Content/Schemas`
- `Content/GameData`
- `Shared/AmandaCoreShared`
- `ContentTools` or equivalent tooling
- `Docs/Architecture.md`

Proposed acceptance tests:

- Source manifests validate against AmandaCore schemas.
- Compiler emits a versioned package with checksums and dependency metadata.
- Runtime loads a compiled package and rejects incompatible versions.
- Missing references fail validation before runtime.
- Package provenance is recorded.

Suggested milestone/release window:

- Near-term, before large authored content growth.

Risks:

- Runtime logic depending directly on authoring formats.
- Content hot reload being added without compatibility checks.
- O3DE export metadata becoming authoritative by accident.

## Epic 8 - Combat / Effect / Threat Pipeline

Goal:

- Implement deterministic combat command validation, resource/cooldown checks, target resolution, effect expansion, state mutation, event emission, and threat accounting.

Why it matters:

- Combat is a hot path and a major regression source. A typed, deterministic pipeline makes abilities testable and keeps encounter behavior explainable.

Clean-room constraint:

- Use AmandaCore-owned ability specs, effect categories, target resolvers, threat rules, and combat event names. Do not copy upstream spell IDs, effect tables, combat formulas, packet assumptions, or special-case logic.

Likely files/systems affected:

- `Shared/AmandaCoreShared`
- `Services/internal/worlds`
- `CombatRules` runtime Gem
- `StatsProgression` runtime Gem
- combat content manifests and validators

Proposed acceptance tests:

- Ability command validation rejects invalid resource, cooldown, range, and target states.
- Effects apply deterministically in stable order.
- Aura/effect lifecycle tests cover apply, tick, refresh, expire, and remove behavior.
- Threat ledger updates from combat events and supports taunt/visibility rules.
- Replay tests reproduce combat outcomes exactly.

Suggested milestone/release window:

- Near-term, after shard ownership and content package basics.

Risks:

- Special cases bypassing typed effect modeling.
- Non-deterministic ordering across entities or shards.
- Combat formulas becoming embedded in presentation code.

## Epic 9 - Quest / Progression Graph Runtime

Goal:

- Build an event-fed quest and progression runtime based on AmandaCore-owned objective graphs, completion predicates, rewards, and trigger references.

Why it matters:

- Quest and progression state touches NPCs, items, combat, locations, conversations, rewards, and persistence. It needs a clear event model before content grows.

Clean-room constraint:

- Use original quest graph schemas, objective names, progression events, rewards, and content IDs. Do not copy upstream quest structures, flags, text, links, identifiers, or table organization.

Likely files/systems affected:

- `QuestRuntime` runtime Gem
- `StatsProgression` runtime Gem
- `Shared/AmandaCoreShared`
- `Services/internal/worlds`
- content schemas and packages

Proposed acceptance tests:

- Player events update objective state deterministically.
- Quest acceptance, progress, completion, abandonment, and reward flows are covered.
- Orphaned objective, trigger, and reward references fail package validation.
- Progression state persists and restores across service restart.
- Scenario test covers a full short quest chain.

Suggested milestone/release window:

- Near-term, after event model and content package format stabilize.

Risks:

- Objective semantics changing after content has been authored.
- Race conditions between combat, loot, and progression events.
- Quest text or identifiers being treated as stable before localization/content policy is ready.

## Epic 10 - Loot / Reward Rule Engine

Goal:

- Implement an auditable AmandaCore reward engine for roll groups, conditions, weights, ownership rules, and distribution modes.

Why it matters:

- Loot is player-visible and trust-sensitive. Deterministic seeding and auditable rules reduce disputes and make replay validation possible.

Clean-room constraint:

- Use AmandaCore-owned reward schemas, roll rules, item IDs, distribution modes, and audit events. Do not copy upstream loot data, template structures, item identifiers, or drop rules.

Likely files/systems affected:

- `InventoryLoot` runtime Gem
- `Shared/AmandaCoreShared`
- `Services/internal/worlds`
- content schemas and packages
- replay/audit tooling

Proposed acceptance tests:

- Reward rules validate before package load.
- Roll output is deterministic for a fixed seed and scenario.
- Ownership and distribution policies are enforced.
- Duplicate, invalid, or unreachable rewards fail validation.
- Replay can explain why a reward did or did not drop.

Suggested milestone/release window:

- Near-term, after combat events and content packages.

Risks:

- Unclear player ownership rules.
- Randomness not captured in replay.
- Economy impact from poorly validated reward weights.

## Epic 11 - AI Runtime and Behavior DSL

Goal:

- Build an AmandaCore AI runtime using original behavior trees, state machines, or planning DSLs compiled from authored content.

Why it matters:

- AI must respond to perception, combat, threat, pathing, objectives, and scripted encounter events without turning into untestable special-case code.

Clean-room constraint:

- Use original behavior schemas, action verbs, event names, waypoint formats, and authoring tools. Do not copy upstream event/action vocabularies, scripted-behavior structures, waypoint data layouts, or behavior data.

Likely files/systems affected:

- `NpcAi` runtime Gem
- `Services/internal/worlds`
- `Shared/AmandaCoreShared`
- content schemas and compiler
- replay/scenario test tooling

Proposed acceptance tests:

- Basic behavior graph validates and compiles.
- NPC perception and threat events drive state transitions.
- Waypoint or path behavior runs inside shard ownership.
- Invalid action references fail content validation.
- Replay tests reproduce AI decisions for a fixed scenario.

Suggested milestone/release window:

- Mid-term, after combat, threat, and content packages are stable.

Risks:

- DSL becoming too broad before runtime needs are proven.
- AI mutating state outside shard ownership.
- Authored behaviors becoming hard to debug without visual tooling.

## Epic 12 - Plugin / Module SDK

Goal:

- Define a versioned AmandaCore plugin or module SDK for extension points, content packages, configuration, lifecycle hooks, and compatibility checks.

Why it matters:

- Extension governance protects the core from long-lived patches and makes optional features easier to test, ship, and remove.

Clean-room constraint:

- Use original module metadata, hook names, lifecycle names, configuration conventions, and compatibility rules. Do not copy upstream module APIs, hook vocabularies, directory conventions, or SQL lifecycle patterns.

Likely files/systems affected:

- service/plugin loading code
- content package loader
- configuration system
- `Docs/Architecture.md`
- future SDK examples

Proposed acceptance tests:

- A sample AmandaCore module declares metadata, dependencies, config, and content package requirements.
- Incompatible module versions fail before runtime mutation.
- Module hooks are deterministic and isolated.
- Module-owned migrations or content packages are validated separately.
- Disabling a module leaves core startup healthy.

Suggested milestone/release window:

- Mid-term, after core service contracts and content packages are stable.

Risks:

- Freezing extension contracts too early.
- Plugins bypassing clean-room and provenance review.
- Module hooks becoming hidden global state.

## Epic 13 - Replay Regression and Fuzzing Harness

Goal:

- Build replay regression and fuzzing tools for protocol adapters, canonical commands, shard simulation, combat, quests, loot, AI, and persistence boundaries.

Why it matters:

- Replays make gameplay bugs reproducible. Fuzzing protects protocol and command boundaries from malformed inputs and abuse cases.

Clean-room constraint:

- Use AmandaCore-owned replay formats, scenario definitions, fuzz corpora, and command generators. Do not use upstream packet captures, parser outputs, sniffed artifacts, or compatibility test data.

Likely files/systems affected:

- `Services/internal/worlds`
- `Shared/AmandaCoreShared`
- protocol adapter tests
- `Docs/QA`
- test harness and CI scripts

Proposed acceptance tests:

- Canonical replay files can drive deterministic simulation tests.
- Replays capture command inputs, timing, random seeds, and domain events.
- Fuzz tests reject malformed adapter input without panics or state corruption.
- Previously fixed gameplay bugs are added as replay regressions.
- CI runs a bounded replay and fuzz subset on every relevant change.

Suggested milestone/release window:

- Mid-term, with baseline replay export earlier in Epic 5.

Risks:

- Replay files containing private account or chat data.
- Fuzz tests being flaky or too slow for CI.
- Insufficient determinism in simulation making replay comparisons noisy.
