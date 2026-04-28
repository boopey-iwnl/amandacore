# Milestone 1 Contract Inventory

Status: Draft freeze for `codex/m1-foundation-contract-freeze`

Scope: Alpha 0.15-compatible launcher, service, diagnostic client, O3DE-facing client, shared C++ rule, content, persistence, and observability boundaries. This inventory documents current behavior and intended boundaries only; it does not authorize runtime rewrites.

## Contract Sources

| Surface | Current source | Milestone 1 treatment |
| --- | --- | --- |
| HTTP service routes | `Services/internal/*/http.go`, `Services/cmd/*/main.go` | Frozen in `Docs/Contracts/http-api-v1.json` and checked by Go tests. |
| Launcher service client | `Client/Launcher/AmandaCore.Launcher/Api/AmandaCoreApiClient.cs` | Launcher-critical routes are identified in the route manifest. |
| Launcher DTOs | `Client/Launcher/AmandaCore.Launcher/Models/ApiModels.cs` | Existing C# DTO names and fields remain compatibility expectations. |
| Diagnostic world client DTOs | `Client/Game/AmandaCore.WorldClient/WorldSessionResponses.cs` | World session response fields are current client convergence inputs. |
| Shared C++ rule contracts | `Shared/AmandaCoreShared/Include/AmandaCoreShared/*.h` | Kept as project-owned type names for C++ and O3DE integration. |
| Canonical server commands/events | `Services/internal/simcore/commands.go` | Used as AmandaCore command/event vocabulary for future adapters. |
| Persistence boundaries | `Services/internal/store/repositories.go`, `Docs/Persistence.md` | Repository and migration direction is frozen; storage is not replaced in Milestone 1. |
| Content package schemas | `Content/Schemas/*.json`, `Services/internal/content` | Content/runtime split is documented; compiler expansion is later milestone work. |
| Observability names | `Services/internal/observability`, `Docs/ServerInteractionPipeline.md` | Names remain AmandaCore-owned and may be expanded through explicit contracts. |

## Launcher-Critical Flow

The launcher path is the compatibility baseline for future persistence and world-authority work:

1. `POST /v1/accounts/register`
2. `POST /v1/auth/login`
3. `GET /v1/patch/manifest`
4. `GET /v1/realms`
5. `GET /v1/characters?realmId=<realmId>`
6. `POST /v1/characters`
7. `POST /v1/world/join-ticket`
8. Launch client with `--join-ticket <ticketId>` and `--world-endpoint <endpoint>`.

The diagnostic world client then consumes:

1. `POST /v1/world/connect`
2. `GET /v1/world/state?worldSessionToken=<token>`
3. `POST /v1/world/move`
4. `POST /v1/world/target`
5. `POST /v1/world/attack/ability`
6. `POST /v1/world/disconnect`
7. `POST /v1/world/reconnect`

## Current DTO Families

| Family | Direction | Representative fields |
| --- | --- | --- |
| Auth | launcher to auth service | `username`, `password`, `accessToken`, `refreshToken`, `accountId`, `roles` |
| Realm/build | launcher to realm service | `realms[]`, `id`, `displayName`, `endpoint`, `supportedBuild`, `protocolVersion`, `apiVersion` |
| Character | launcher to character service | `realmId`, `displayName`, `raceId`, `classId`, `archetypeId`, `zoneId`, `level` |
| World join | launcher to world service | `ticketId`, `sessionId`, `accountId`, `characterId`, `realmId`, `worldEndpoint`, `expiresAt` |
| World session | client to world service | `worldSessionToken`, `position`, `entities`, `actionBar`, `inventory`, `quests`, `domainEvents`, `stateDiffs`, `streaming`, additive replication metadata |
| Admin/support | operator clients to admin services | RBAC-gated account, character, quest, item, currency, moderation, support, and audit payloads |
| Social/economy | world client to world service | chat, friends, party, guild, auction, mail, inventory, currency, and audit state |

## Known Drift Points

- `Docs/RuntimeContract.md` documents the older `sunset_frontier` slice while current runtime defaults and README language use `stonewake_vale`.
- The launcher DTO `AuthResponse` consumes `accessToken`, `refreshToken`, and `accountId`; the service also returns `roles`.
- The launcher can send `raceId` and `classId` when creating a character, while older docs show only `archetypeId`.
- The world session response has grown beyond the original connect payload to include combat, progression, inventory, social, economy, and streaming fields.
- Admin functionality is split between `admin-service` routes and `world-service` admin routes.
- Current HTTP polling remains the compatibility transport; future push/delta replication must adapt to existing canonical command and state contracts.
- Milestone 6 adds optional world-session replication metadata (`cursor`, `snapshotVersion`, `deltaVersion`, `replication`) while preserving existing route names and full response payloads.

## Freeze Rules

- Route additions, removals, or renames must update `Docs/Contracts/http-api-v1.json`.
- Launcher-critical request/response fields must be additive or explicitly versioned.
- Contract validation failures block Milestone 1 completion until documentation or code is corrected.
- Runtime behavior changes are out of scope for this inventory phase.
- Clean-room provenance applies to contract examples, route names, DTO names, content IDs, and observability names.
