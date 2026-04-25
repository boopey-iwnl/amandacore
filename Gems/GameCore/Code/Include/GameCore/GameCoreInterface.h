#pragma once

#include <AzCore/Interface/Interface.h>
#include <AzCore/Math/Transform.h>
#include <AzCore/std/string/string.h>
#include <NetClient/WorldHttpClient.h>

namespace GameCore
{
    struct ClientLaunchOptions
    {
        AZStd::string m_joinTicketId;
        AZStd::string m_worldEndpoint;

        bool IsValid() const
        {
            return !m_joinTicketId.empty() && !m_worldEndpoint.empty();
        }
    };

    struct ClientWorldState
    {
        bool m_launchOptionsPresent = false;
        bool m_connectAttempted = false;
        bool m_worldConnected = false;
        bool m_bootstrapReady = false;
        AZStd::string m_errorMessage;
        NetClient::WorldBootstrapResponse m_bootstrap;
        NetClient::WorldSessionResponse m_session;
        NetClient::SocialStateResponse m_social;
        NetClient::AuctionStateResponse m_auction;
        AZStd::string m_pendingInteractionEntityId;
        AZ::u64 m_pendingInteractionSequence = 0;
    };

    struct ClientCameraState
    {
        bool m_ready = false;
        AZ::Transform m_worldTransform = AZ::Transform::CreateIdentity();
        float m_verticalFovDegrees = 60.0f;
    };

    class IGameCoreRequests
    {
    public:
        AZ_RTTI(IGameCoreRequests, "{C657226D-7206-41D7-BD90-A0723B830C26}");

        virtual ~IGameCoreRequests() = default;

        static IGameCoreRequests* Get()
        {
            return AZ::Interface<IGameCoreRequests>::Get();
        }

        static void Register(IGameCoreRequests* instance)
        {
            AZ::Interface<IGameCoreRequests>::Register(instance);
        }

        static void Unregister(IGameCoreRequests* instance)
        {
            AZ::Interface<IGameCoreRequests>::Unregister(instance);
        }

        virtual const ClientLaunchOptions& GetLaunchOptions() const = 0;
        virtual const ClientWorldState& GetClientWorldState() const = 0;
        virtual const ClientCameraState& GetCameraState() const = 0;
        virtual bool SubmitMove(double deltaX, double deltaY) = 0;
        virtual bool SetTarget(const AZStd::string& targetId) = 0;
        virtual bool InteractWithEntity(const AZStd::string& entityId) = 0;
        virtual bool AcceptQuest(const AZStd::string& questId) = 0;
        virtual bool EnterDungeon(const AZStd::string& dungeonId) = 0;
        virtual bool ExitDungeon() = 0;
        virtual bool TrackQuest(const AZStd::string& questId, bool tracked) = 0;
        virtual bool SetAutoAttack(bool enabled) = 0;
        virtual bool ActivateAbility(const AZStd::string& abilityId) = 0;
        virtual bool RequestDuel(const AZStd::string& targetCharacterId, const AZStd::string& targetName) = 0;
        virtual bool AcceptDuel(const AZStd::string& duelId) = 0;
        virtual bool DeclineDuel(const AZStd::string& duelId) = 0;
        virtual bool CancelDuel(const AZStd::string& duelId) = 0;
        virtual bool SurrenderDuel(const AZStd::string& duelId) = 0;
        virtual bool LearnTrainerAbility(const AZStd::string& trainerId, const AZStd::string& abilityId) = 0;
        virtual bool SelectTalent(const AZStd::string& talentId) = 0;
        virtual bool AssignActionBarSlot(int slotIndex, const AZStd::string& abilityId) = 0;
        virtual bool MoveActionBarSlot(int fromSlotIndex, int toSlotIndex) = 0;
        virtual bool ClearActionBarSlot(int slotIndex) = 0;
        virtual bool MoveInventorySlot(int fromSlotIndex, int toSlotIndex) = 0;
        virtual bool BrowseAuctions(
            const AZStd::string& search,
            const AZStd::string& itemType,
            const AZStd::string& sort) = 0;
        virtual bool ListAuctionItem(int slotIndex, int stackCount, int buyoutCopper, AZ::s64 durationSeconds) = 0;
        virtual bool BuyoutAuction(const AZStd::string& auctionId) = 0;
        virtual bool CancelAuction(const AZStd::string& auctionId) = 0;
        virtual bool SubmitChatMessage(
            const AZStd::string& channel,
            const AZStd::string& targetName,
            const AZStd::string& messageText) = 0;
        virtual bool AddFriend(const AZStd::string& name) = 0;
        virtual bool RemoveFriend(const AZStd::string& name) = 0;
        virtual bool InviteParty(const AZStd::string& targetName, const AZStd::string& targetCharacterId) = 0;
        virtual bool AcceptPartyInvite(const AZStd::string& inviteId) = 0;
        virtual bool DeclinePartyInvite(const AZStd::string& inviteId) = 0;
        virtual bool LeaveParty() = 0;
        virtual bool DisbandParty() = 0;
        virtual bool CreateGuild(const AZStd::string& guildName) = 0;
        virtual bool InviteGuild(const AZStd::string& targetName) = 0;
        virtual bool AcceptGuildInvite(const AZStd::string& inviteId) = 0;
        virtual bool DeclineGuildInvite(const AZStd::string& inviteId) = 0;
        virtual bool LeaveGuild() = 0;
        virtual bool DisbandGuild() = 0;
        virtual bool PromoteGuildMember(const AZStd::string& targetName) = 0;
        virtual bool DemoteGuildMember(const AZStd::string& targetName) = 0;
        virtual bool RemoveGuildMember(const AZStd::string& targetName) = 0;
        virtual bool SetGuildMessageOfTheDay(const AZStd::string& messageOfTheDay) = 0;
        virtual bool DisconnectWorld() = 0;
        virtual bool ReconnectWorld() = 0;
        virtual void SetCameraState(const ClientCameraState& cameraState) = 0;
    };
} // namespace GameCore
