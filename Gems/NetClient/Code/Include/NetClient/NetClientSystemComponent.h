#pragma once

#include <AzCore/Component/Component.h>
#include <NetClient/WorldHttpClient.h>

namespace NetClient
{
    class NetClientSystemComponent final
        : public AZ::Component
        , public IWorldHttpClient
    {
    public:
        AZ_COMPONENT(NetClientSystemComponent, "{148F8E93-02D1-4A32-BCCA-1EEA838D72C2}");

        NetClientSystemComponent() = default;
        ~NetClientSystemComponent() override = default;

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;

        bool Login(
            const AZStd::string& authEndpoint,
            const AZStd::string& username,
            const AZStd::string& password,
            AuthSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool RefreshSession(
            const AZStd::string& authEndpoint,
            const AZStd::string& refreshToken,
            AuthSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool ListRealms(
            const AZStd::string& realmEndpoint,
            AZStd::vector<RealmDescriptor>& outRealms,
            AZStd::string& outError) override;

        bool ListCharacters(
            const AZStd::string& characterEndpoint,
            const AZStd::string& accessToken,
            const AZStd::string& realmId,
            AZStd::vector<CharacterSummary>& outCharacters,
            AZStd::string& outError) override;

        bool CreateCharacter(
            const AZStd::string& characterEndpoint,
            const AZStd::string& accessToken,
            const AZStd::string& realmId,
            const AZStd::string& displayName,
            const AZStd::string& archetypeId,
            CharacterSummary& outCharacter,
            AZStd::string& outError) override;

        bool CreateJoinTicket(
            const AZStd::string& worldEndpoint,
            const AZStd::string& accessToken,
            const AZStd::string& realmId,
            const AZStd::string& characterId,
            WorldJoinTicketResponse& outTicket,
            AZStd::string& outError) override;

        bool Connect(
            const AZStd::string& worldEndpoint,
            const AZStd::string& ticketId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool Bootstrap(
            const AZStd::string& worldEndpoint,
            WorldBootstrapResponse& outResponse,
            AZStd::string& outError) override;

        bool Move(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            double deltaX,
            double deltaY,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool Disconnect(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            AZStd::string& outError) override;

        bool State(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& cursor,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool SocialState(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& afterMessageId,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool SendChat(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& channel,
            const AZStd::string& targetName,
            const AZStd::string& messageText,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool AddFriend(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& name,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool RemoveFriend(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& name,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool InviteParty(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& targetName,
            const AZStd::string& targetCharacterId,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool AcceptPartyInvite(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& inviteId,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool DeclinePartyInvite(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& inviteId,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool LeaveParty(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool DisbandParty(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool CreateGuild(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& guildName,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool InviteGuild(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& targetName,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool AcceptGuildInvite(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& inviteId,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool DeclineGuildInvite(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& inviteId,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool LeaveGuild(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool DisbandGuild(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool PromoteGuildMember(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& targetName,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool DemoteGuildMember(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& targetName,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool RemoveGuildMember(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& targetName,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool SetGuildMessageOfTheDay(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& messageOfTheDay,
            SocialStateResponse& outResponse,
            AZStd::string& outError) override;

        bool SetTarget(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& targetId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool AcceptQuest(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& questId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool CompleteQuest(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& questId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool EnterDungeon(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& dungeonId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool ExitDungeon(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool TrackQuest(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& questId,
            bool tracked,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool SetAutoAttack(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            bool enabled,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool ActivateAbility(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& abilityId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool RequestDuel(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& targetCharacterId,
            const AZStd::string& targetName,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool AcceptDuel(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& duelId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool DeclineDuel(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& duelId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool CancelDuel(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& duelId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool SurrenderDuel(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& duelId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool LearnTrainerAbility(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& trainerId,
            const AZStd::string& abilityId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool SelectTalent(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& talentId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool LearnProfession(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& trainerId,
            const AZStd::string& professionId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool AssignActionBarSlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int slotIndex,
            const AZStd::string& abilityId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool MoveActionBarSlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int fromSlotIndex,
            int toSlotIndex,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool ClearActionBarSlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int slotIndex,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool MoveInventorySlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int fromSlotIndex,
            int toSlotIndex,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool EquipInventorySlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int slotIndex,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool UnequipInventorySlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& equipmentSlot,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool BrowseAuctions(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& search,
            const AZStd::string& itemType,
            const AZStd::string& sort,
            AuctionStateResponse& outResponse,
            AZStd::string& outError) override;

        bool ListAuctionItem(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int slotIndex,
            int stackCount,
            int buyoutCopper,
            AZ::s64 durationSeconds,
            AuctionStateResponse& outResponse,
            AZStd::string& outError) override;

        bool BuyoutAuction(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& auctionId,
            AuctionStateResponse& outResponse,
            AZStd::string& outError) override;

        bool CancelAuction(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& auctionId,
            AuctionStateResponse& outResponse,
            AZStd::string& outError) override;

        bool Reconnect(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;
    };
}
