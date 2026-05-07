# UI Milestone 7: Social and Economy Shells

## Scope

Milestone 7 exposes AmandaCore social and economy foundations through first-party built-in ImGui/O3DE UI. It does not add addon support, Lua loading, user-installed UI modules, plugin runtimes, or arbitrary UI script execution.

The implemented client surfaces are:

- polished chat filters and send-channel controls in `Gems/UiClient/Code/Source/UiClientSystemComponent.cpp`
- Social micro-menu access and Social panel tabs for Friends, Blocked, Party, Guild, and Trade
- party and guild membership/invite shells backed by the existing world social routes
- read-only mail shell from the existing auction/mail state
- auction browse/list/my-auctions shell polish through existing auction routes
- vendor buy/sell shell backed by the existing world vendor routes
- disabled trade placeholder because player trade has no runtime state or endpoints

## Current Runtime Support

Chat is live through `GET /v1/world/social/state` and `POST /v1/world/chat/send`. The client shows System, Say, Party, Guild, and Whisper filters. Party and Guild send controls are disabled until the social payload reports party or guild membership.

Friends are live through `POST /v1/world/friends/add` and `POST /v1/world/friends/remove`. Online state is rendered only from the server social payload.

Party is live through existing invite, accept, decline, leave, and disband routes. Membership changes are server-authoritative and are applied only after the returned social state is accepted by GameCore.

Guild is live through existing create, invite, accept, decline, leave, disband, promote, demote, remove, and message-of-the-day routes. Guild ranks and permissions are rendered only from the payload the server provides.

Auction browse, listing creation, buyout, and cancel remain live through the existing auction routes. The market panel also renders mail returned in the auction state, but mail claim/send actions are disabled because there is no live runtime claim/send route.

Vendor buy/sell is now wired through `IWorldHttpClient`, `NetClientSystemComponent`, `IGameCoreRequests`, and `GameCoreSystemComponent`. The client consumes `worldSession.vendor` from the existing world-session payload and calls `POST /v1/world/vendor/buy` or `POST /v1/world/vendor/sell`. Purchases and sales apply the returned world-session state and never move items or currency client-side.

## Disabled Or Read-Only Systems

Ignore/blocked lists are read-only disabled shells. Store-level foundations exist, but there is no active runtime HTTP/client wiring for ignore list or chat filtering in this milestone.

Mail is read-only. The client can display auction/mail state when present, but claim and compose buttons are disabled because runtime claim/send endpoints are not exposed.

Trade is disabled. There is no player-trade runtime state or endpoint set, so the client does not expose item or currency transfer controls.

## Persistence And Transactionality

All item, currency, chat, social, party, guild, vendor, and auction mutations are sent to GameCore and then to the world service. The UI does not mutate authoritative state locally. The client applies only server-returned `WorldSessionResponse`, `SocialStateResponse`, or `AuctionStateResponse` payloads.

## Texture Routing

No new textures were imported for this milestone. The UI uses existing repo-side icons, text, and procedural ImGui styling. There is no runtime, manifest, package, material, code, or docs-as-config reference to `Downloads/textures`.

## Data Gaps And Deferred Work

- runtime ignore/block routes and client DTOs
- runtime mail claim/send routes
- player trade state and endpoints
- richer vendor search/filtering
- duplicate-click request-in-flight UX for slow network responses
- separate mail access points outside the auction/mail state
