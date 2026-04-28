package store

import (
	"errors"

	"amandacore/services/internal/platform"
)

const (
	InviteStatePending  = "pending"
	InviteStateAccepted = "accepted"
	InviteStateDeclined = "declined"
	InviteStateExpired  = "expired"

	MailAttachmentKindItem     = "item"
	MailAttachmentKindCurrency = "currency"
)

var (
	ErrIgnoreExists          = errors.New("ignore entry already exists")
	ErrIgnoreMissing         = errors.New("ignore entry does not exist")
	ErrPartyInviteMissing    = errors.New("party invite does not exist")
	ErrPartyInviteConsumed   = errors.New("party invite is no longer pending")
	ErrPartyMemberExists     = errors.New("character is already in a party")
	ErrGuildInviteConsumed   = errors.New("guild invite is no longer pending")
	ErrAuctionMissing        = errors.New("auction listing does not exist")
	ErrAuctionInactive       = errors.New("auction listing is not active")
	ErrMailMissing           = errors.New("mail message does not exist")
	ErrMailAttachmentMissing = errors.New("mail attachment does not exist")
	ErrMailAttachmentClaimed = errors.New("mail attachment is already claimed")
	ErrInsufficientCurrency  = errors.New("insufficient currency")
	ErrDuplicateMutation     = errors.New("duplicate mutation")
)

type IgnoreRelationship struct {
	OwnerCharacterID   string
	IgnoredCharacterID string
	CreatedAt          int64
}

type PartyInvite struct {
	InviteID           string
	PartyID            string
	InviterCharacterID string
	TargetCharacterID  string
	State              string
	CreatedAt          int64
	ExpiresAt          int64
	RespondedAt        int64
}

type CurrencyLedgerEntry struct {
	EntryID          string
	CharacterID      string
	DeltaCopper      int
	BalanceAfter     int
	Reason           string
	Operation        string
	SourceKind       string
	SourceID         string
	ActorCharacterID string
	MutationKey      string
	CreatedAt        int64
}

type VendorPurchaseMutation struct {
	CharacterID      string
	ItemID           string
	DisplayName      string
	Quantity         int
	UnitPriceCopper  int
	MaxStack         int
	Stackable        bool
	MutationKey      string
	SourceID         string
	ActorCharacterID string
}

type VendorSaleMutation struct {
	CharacterID      string
	SlotIndex        int
	Quantity         int
	UnitPriceCopper  int
	MutationKey      string
	SourceID         string
	ActorCharacterID string
}

type AuctionCreateMutation struct {
	Listing         platform.AuctionListing
	SourceSlotIndex int
	StackCount      int
	MutationKey     string
}

type AuctionBuyoutMutation struct {
	AuctionID        string
	BuyerCharacterID string
	MutationKey      string
	Now              int64
}

type AuctionCancelMutation struct {
	AuctionID         string
	SellerCharacterID string
	MutationKey       string
	Now               int64
}

type MailAttachmentClaim struct {
	MailID       string
	AttachmentID string
	CharacterID  string
	MutationKey  string
	Now          int64
}

type SocialRepository interface {
	AddFriend(ownerCharacterID string, friendCharacterID string) (platform.FriendRelationship, error)
	RemoveFriend(ownerCharacterID string, friendCharacterID string) error
	ListFriends(ownerCharacterID string) ([]platform.FriendRelationship, error)
}

type IgnoreRepository interface {
	AddIgnore(ownerCharacterID string, ignoredCharacterID string) (IgnoreRelationship, error)
	RemoveIgnore(ownerCharacterID string, ignoredCharacterID string) error
	ListIgnores(ownerCharacterID string) ([]IgnoreRelationship, error)
}

type PartyRepository interface {
	CreateParty(leaderCharacterID string, memberCharacterIDs []string) (platform.Party, error)
	SaveParty(party platform.Party) (platform.Party, error)
	DeleteParty(partyID string) error
	GetPartyByID(partyID string) (*platform.Party, error)
	GetPartyForCharacter(characterID string) (*platform.Party, error)
	CreatePartyInvite(invite PartyInvite) (PartyInvite, error)
	AcceptPartyInvite(inviteID string, targetCharacterID string, options MutationOptions) (platform.Party, error)
	DeclinePartyInvite(inviteID string, targetCharacterID string, options MutationOptions) error
}

type GuildRepository interface {
	CreateGuild(guildName string, leaderCharacterID string) (platform.Guild, error)
	SaveGuild(guild platform.Guild) (platform.Guild, error)
	DeleteGuild(guildID string) error
	GetGuildByID(guildID string) (*platform.Guild, error)
	GetGuildForCharacter(characterID string) (*platform.Guild, error)
	CreateGuildInvite(guildID string, inviterCharacterID string, targetCharacterID string, expiresAt int64) (platform.GuildInvite, error)
	GetGuildInvite(inviteID string) (*platform.GuildInvite, error)
	ListGuildInvitesForCharacter(characterID string) ([]platform.GuildInvite, error)
	DeleteGuildInvite(inviteID string) error
	CleanupExpiredGuildInvites(nowUnix int64) error
	AcceptGuildInvite(inviteID string, targetCharacterID string, options MutationOptions) (platform.Guild, error)
	DeclineGuildInvite(inviteID string, targetCharacterID string, options MutationOptions) error
}

type ChatRepository interface {
	AppendChatMessage(message platform.ChatMessage) (platform.ChatMessage, error)
	ListRecentChatMessages(channel string, scopeID string, limit int) ([]platform.ChatMessage, error)
}

type CurrencyRepository interface {
	GetCurrencyBalance(characterID string) (int, error)
	AppendCurrencyMutation(entry CurrencyLedgerEntry) (CurrencyLedgerEntry, error)
}

type EconomyRepository interface {
	BuyVendorItem(mutation VendorPurchaseMutation) (platform.Character, CurrencyLedgerEntry, error)
	SellVendorItem(mutation VendorSaleMutation) (platform.Character, CurrencyLedgerEntry, error)
	CreateAuctionListing(mutation AuctionCreateMutation) (platform.AuctionListing, platform.Character, error)
	ListAuctionListings(realmID string, search string, itemType string, sortBy string, limit int, offset int) ([]platform.AuctionListing, error)
	ListAuctionsForSeller(sellerCharacterID string) ([]platform.AuctionListing, error)
	BuyoutAuction(mutation AuctionBuyoutMutation) (platform.AuctionListing, platform.Character, platform.Character, error)
	CancelAuction(mutation AuctionCancelMutation) (platform.AuctionListing, platform.Character, error)
	CreateMail(mail platform.MailEnvelope) (platform.MailEnvelope, error)
	ListMailForCharacter(characterID string) ([]platform.MailEnvelope, error)
	ClaimMailAttachment(claim MailAttachmentClaim) (platform.Character, platform.MailEnvelope, error)
}
