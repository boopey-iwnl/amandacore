# Auction House Validation

Use two characters on the same realm. Start both near the Highmere market board or auction clerk.

## Open UI

- Target the market board or auction clerk.
- Right-click/interact and confirm the Auction window opens.
- Confirm Browse, Sell, and My Auctions tabs are visible.
- Move out of range and confirm auction actions are blocked with a readable message.

## List Item

- On the seller, select a tradeable inventory item in the Sell tab.
- Confirm the deposit preview appears and changes with stack count.
- Create a buyout-only listing.
- Confirm the item leaves the seller inventory immediately.
- Confirm the listing appears in Browse and My Auctions.
- Try listing a quest/non-tradeable item and confirm it is blocked.

## Buy Item

- On a second character, open the auction UI.
- Browse or search for the seller listing.
- Buy out the listing and confirm the buyer copper decreases.
- Confirm the buyer receives the purchased item.
- Confirm the seller receives sale proceeds minus the auction cut, with deposit returned on sale.
- Confirm the listing state becomes sold and cannot be bought again.

## Cancel And Expire

- Create another seller listing.
- Cancel it from My Auctions and confirm the item returns once.
- Create a short-duration/dev listing if available.
- Wait for expiration or trigger a browse/cleanup pass after expiry.
- Confirm the expired item returns once.

## Persistence And Duplication

- Restart the services after creating an active listing.
- Confirm the active listing survives restart.
- Buy the listing, then restart again.
- Confirm the buyer item and seller proceeds are not duplicated.
- Re-run cancel/expire actions against completed listings and confirm they fail safely.

## Regression Smoke

- Verify mail records still display.
- Verify direct trading still rejects auction-custodied items because they are no longer in inventory.
- Verify vendors, inventory moves, equipment, chat/social, party, dungeon entry, movement, combat, and quests still function at a smoke-test level.
