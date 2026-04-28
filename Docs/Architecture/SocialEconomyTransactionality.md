# Social Economy Transactionality

## Purpose

Milestone 7 adds transactional foundations for AmandaCore social and economy state while preserving the current file-backed runtime path. The goal is durable, retry-safe, auditable SQL behavior for friends, ignores, party and guild membership, chat, currency, vendor transactions, auctions, and mail attachment claims.

This milestone does not make SQL the production runtime default. It adds repository contracts, AmandaCore-owned schema, SQL implementations, and tests so later service cutovers can happen behind stable boundaries.

## Current Social API And Path Inventory

The active world service exposes existing HTTP social routes for chat, friends, party, guild, and social state. These routes currently use the world server lock plus file-store helpers:

- friends are stored in the file-backed `friends` map
- party membership is stored in the file-backed `parties` map, while short-lived party invites are world-memory state
- guilds and guild invites are stored in file-backed `guilds` and `guildInvites` maps
- chat is session-backed world memory and visible through current world state responses

Milestone 7 does not rename or remove these routes.

## Current Economy API And Path Inventory

Existing economy routes include vendor buy/sell, auction list/buy/cancel, auction listing queries, mail listing, and currency-changing admin routes. Current runtime behavior remains file-backed:

- character currency is on the character aggregate
- vendor operations mutate character currency and inventory through the file store
- auction state is stored in the file-backed `auctions` map
- mail is stored in the file-backed `mail` map
- audit events are appended to file-backed audit state

Milestone 7 adds SQL equivalents for the transactional foundation. Runtime route cutover is deferred.

## Current Chat Behavior

Chat remains compatible with the current world response model. The SQL foundation adds `ac_chat_messages` with channel, scope, sender, message text, sequence, and timestamp fields so a future service path can persist and replay recent chat by scope.

## Current Auction Vendor Mail Currency Behavior

The active file-backed implementation already prevents many simple local duplicate cases through process locking. Milestone 7 adds relational constraints and idempotency records for the SQL path:

- currency ledger entries are append-only and keyed by character, operation, and mutation key when provided
- vendor buy/sell commits inventory and currency together
- auction list removes the listed item and deposit in one transaction
- auction buyout marks the listing sold, debits buyer currency, credits seller proceeds, grants the item, and records audit data in one transaction
- auction cancel returns the item and marks the listing canceled in one transaction
- mail attachment claim grants item or currency and marks the attachment claimed in one transaction

## Existing Persistence Coverage

Milestones 2 and 3 introduced SQL identity, character, gameplay, world-session, audit, and transactional character-state tables. Milestone 7 extends that SQL path with:

- `000008_social_state.sql`
- `000009_economy_state.sql`
- `000010_mail_and_audit_state.sql`

The file-backed dev path is still intact.

## Transaction Boundaries

SQL social and economy mutations use explicit transactions:

- party and guild invite accept transitions the invite and membership together
- currency mutations update character balance and ledger together
- vendor buy/sell updates inventory, currency, ledger, and transaction records together
- auction list/buy/cancel updates listing, character aggregates, ledger rows, transaction rows, and transfer audit rows together
- mail claim updates character state, attachment claim state, ledger rows when currency is attached, and transfer audit rows together

If validation or persistence fails, the transaction rolls back and no partial item, currency, membership, listing, or attachment state is committed.

## Idempotency Strategy

Retry-sensitive SQL methods accept `MutationOptions.MutationKey` or a mutation-specific key field. The store records replayable responses for social and economy mutations where useful and uses unique indexes for ledger/vendor/auction/mail retry surfaces.

Duplicate retries either replay the previous response or fail with a clear duplicate/consumed/claimed error without applying another item, currency, listing, or membership mutation.

## Audit And Event Strategy

Milestone 7 adds relational audit tables for transfer-sensitive operations:

- `ac_social_mutations`
- `ac_economy_mutations`
- `ac_transfer_audit_events`

These rows record actor character, target identifiers, mutation keys, result status, item/currency details, and metadata. They intentionally avoid session tokens, passwords, or local secrets.

## Client Compatibility Strategy

No client-facing route names, DTO shapes, O3DE payloads, or fallback-client parsing paths changed. Existing clients continue to use the current file-backed service path.

The SQL repository behavior is test-backed and ready for later low-risk service cutover, but this milestone does not expose dead UI paths or require client changes.

## Known Limitations

- SQL social/economy behavior is not the runtime default.
- Party and guild membership is relationally durable in SQL tests, but cross-zone party synchronization is still future work.
- Auction settlement uses direct inventory/currency mutation in the SQL foundation; richer mailbox delivery UX remains future work.
- Chat persistence has local/test retention behavior only; pruning and moderation workflows remain future work.
- Ignore/block is SQL-backed but not wired into current world chat filtering.

## Non-Goals

- No full global social service.
- No cross-zone party authority model.
- No advanced auction-house UX.
- No mail UI expansion.
- No runtime SQL cutover.
- No file-store removal.
- No copied external MMO schemas, table names, packet layouts, command names, IDs, scripts, or module structures.

## Clean-Room Notes

The M7 social and economy schema, repository names, mutation keys, transaction boundaries, and tests are original AmandaCore work. Public MMO architecture is used only as behavioral context for durable membership, ledger, auction, mail, and audit guarantees.

## Risks For Milestone 8 Content Script Runtime

- Content-driven vendors, loot, and rewards will need a validated script/content boundary before being routed broadly through social/economy repositories.
- Mail and auction item grants must use only validated AmandaCore content IDs.
- Future plugin/script hooks must not bypass idempotency keys, transfer audit rows, or transaction boundaries.
- Runtime cutover still needs backend selection, import, rollback, and corruption recovery runbooks.
