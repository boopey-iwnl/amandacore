# AmandaCore Content Pipeline

AmandaCore content must be authored, validated, compiled, and loaded through original AmandaCore formats. Runtime servers must consume AmandaCore content packages, not TrinityCore or AzerothCore database tables, schemas, IDs, scripts, or data.

## Intended Flow

1. O3DE or editor-authored data exports neutral AmandaCore zone metadata.
2. Static source manifests live in AmandaCore-owned formats.
3. Validators check IDs, references, bounds, rewards, objectives, and package compatibility.
4. A future compiler produces versioned runtime content packages.
5. Runtime services load compiled packages through a controlled content registry.
6. Package checksums and provenance are recorded for test and release traceability.

The initial Go `contentpkg` package is a lightweight skeleton for package manifests only. It is not a content compiler yet.

## Initial Content Domains

- zones and world-region metadata
- spawn points and authored position markers
- NPC archetypes
- ability specs and effect references
- loot and reward rules
- quests and objective graphs
- dialogue and interaction references

## Migration Discipline

Content package versions and persistence schema migrations are separate tracks:

- Persistence migrations change durable service state.
- Content packages change authored world/gameplay data.
- Compatibility rules decide whether a runtime can load a package.
- Checksums make package and migration drift visible.
- Dry-run validation should run before local, staging, or release deployment.

Do not copy SQL schemas, table names, live-table layouts, content IDs, or content records from external projects. AmandaCore migrations must be derived from AmandaCore-owned aggregates and storage domains.

## Future Compiler Responsibilities

The future compiler should:

- validate manifest schema and cross-references
- reserve IDs in AmandaCore namespaces
- reject duplicate or missing references
- emit runtime packages with version, checksum, dependency, and compatibility metadata
- generate audit-friendly package summaries
- support deterministic test fixtures for replay and scenario tests

Hot reload should be added only where package compatibility can be proven without corrupting live simulation state.
