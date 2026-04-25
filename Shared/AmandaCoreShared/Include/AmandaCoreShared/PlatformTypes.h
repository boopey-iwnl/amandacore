#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace amandacore
{
    using AccountId = std::string;
    using SessionId = std::string;
    using RealmId = std::string;
    using CharacterId = std::string;
    using InstanceId = std::string;
    using BuildId = std::string;

    enum class Role
    {
        Player,
        Moderator,
        GameMaster,
        Administrator
    };

    enum class BuildChannel
    {
        Local,
        Development,
        Staging,
        Production
    };

    struct RealmDescriptor
    {
        RealmId id;
        std::string displayName;
        std::string region;
        std::string endpoint;
        std::uint32_t onlinePlayers = 0;
        bool online = true;
        BuildId supportedBuild;
    };

    struct CharacterSummary
    {
        CharacterId id;
        RealmId realmId;
        std::string displayName;
        std::string archetypeId;
        std::uint32_t level = 1;
        std::string zoneId;
        bool online = false;
    };

    struct GuildSummary
    {
        std::string id;
        std::string displayName;
        std::uint32_t memberCount = 0;
    };

    struct AuctionListing
    {
        std::string id;
        std::string itemId;
        std::uint32_t stackCount = 1;
        std::uint64_t startingBid = 0;
        std::uint64_t buyout = 0;
        std::string sellerCharacterId;
    };

    struct MailEnvelope
    {
        std::string id;
        std::string senderCharacterId;
        std::string recipientCharacterId;
        std::string subject;
        bool hasAttachments = false;
    };

    struct BuildDescriptor
    {
        BuildId id;
        BuildChannel channel = BuildChannel::Local;
        std::string displayVersion;
        std::string launcherManifestUrl;
        bool allowedForLogin = true;
    };

    struct WorldJoinTicket
    {
        std::string ticketId;
        SessionId sessionId;
        AccountId accountId;
        CharacterId characterId;
        RealmId realmId;
        std::string worldEndpoint;
        std::uint64_t expiresAtUnixSeconds = 0;
    };
}
