# Mail, Auction, Vendor, And Trade UI Contract

## Mail

Mail is read from `AuctionStateResponse.mail` when auction state is loaded. The Mail tab is read-only in this milestone.

Rendered data:

- sender display name
- subject
- body
- item attachments
- currency attachment amount

Disabled actions:

- claim
- compose
- send

Those actions remain disabled because no live runtime mail claim/send endpoint is exposed.

## Auction

Auction UI uses existing world economy routes:

- `GET /v1/world/auction/listings`
- `GET /v1/world/auction/mine`
- `POST /v1/world/auction/list`
- `POST /v1/world/auction/buyout`
- `POST /v1/world/auction/cancel`

The UI renders browse, sell, and my-auctions tabs. The client does not grant items or currency locally; it applies only the returned auction state.

## Vendor

Vendor UI reads `worldSession.vendor` from the world-session response and opens only for NPCs whose service list includes `vendor`.

Rendered data:

- vendor ID
- vendor NPC ID
- display name
- in-range state
- item offers
- buy price and sell value
- current currency
- current inventory sell slots

Live actions:

- buy item through `POST /v1/world/vendor/buy`
- sell inventory slot through `POST /v1/world/vendor/sell`

The UI disables buy when the player is out of range, the offer has no buy price, or the current currency payload is insufficient. The UI disables sell when no sellable inventory item is selected or the player is out of range.

## Trade

Player trade is disabled in this milestone. No request, offer, lock, accept, cancel, item transfer, or currency transfer control should be shown until a server-authoritative trade state and route set exists.

## Texture And Addon Policy

This milestone imports no new textures. Runtime code, manifests, package scripts, materials, and docs-as-config must not reference `Downloads/textures`.

No addon runtime, Lua loading, AddOns folder, plugin runtime, user-installed UI modules, or arbitrary UI script execution is allowed.
