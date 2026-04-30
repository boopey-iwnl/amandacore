#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/Component/TickBus.h>
#include <AzFramework/API/ApplicationAPI.h>
#include <GameCore/GameCoreInterface.h>

namespace GameCore
{
    class GameCoreSystemComponent final
        : public AZ::Component
        , public AZ::TickBus::Handler
        , public AzFramework::LevelSystemLifecycleNotificationBus::Handler
        , public IGameCoreRequests
    {
    public:
        AZ_COMPONENT(GameCoreSystemComponent, "{72D82A3D-0F0F-4F93-8E47-91D7446A218A}");

        GameCoreSystemComponent() = default;
        ~GameCoreSystemComponent() override = default;

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;

        void OnTick(float deltaTime, AZ::ScriptTimePoint time) override;
        void OnLoadingComplete(const char* levelName) override;

        const ClientLaunchOptions& GetLaunchOptions() const override;
        const ClientWorldState& GetClientWorldState() const override;
        const ClientFrontendState& GetClientFrontendState() const override;
        const ClientCameraState& GetCameraState() const override;
        bool IsPreWorldFrontendActive() const override;
        bool SubmitFrontendLogin(const AZStd::string& username, const AZStd::string& password) override;
        bool RefreshFrontendRealms() override;
        bool SelectFrontendRealm(int realmIndex) override;
        bool RefreshFrontendCharacters() override;
        bool SelectFrontendCharacter(int characterIndex) override;
        bool OpenFrontendCharacterCreation() override;
        bool CreateFrontendCharacter(const AZStd::string& displayName, const AZStd::string& archetypeId) override;
        bool EnterWorldWithSelectedCharacter() override;
        bool NavigateFrontendBack() override;
        void SetFrontendRememberLogin(bool rememberLogin) override;
        bool ForgetFrontendRememberedSession() override;
        void ClearFrontendError() override;
        bool SubmitMove(double deltaX, double deltaY) override;
        bool SetTarget(const AZStd::string& targetId) override;
        bool InteractWithEntity(const AZStd::string& entityId) override;
        bool AcceptQuest(const AZStd::string& questId) override;
        bool CompleteQuest(const AZStd::string& questId) override;
        bool EnterDungeon(const AZStd::string& dungeonId) override;
        bool ExitDungeon() override;
        bool TrackQuest(const AZStd::string& questId, bool tracked) override;
        bool SetAutoAttack(bool enabled) override;
        bool ActivateAbility(const AZStd::string& abilityId) override;
        bool RequestDuel(const AZStd::string& targetCharacterId, const AZStd::string& targetName) override;
        bool AcceptDuel(const AZStd::string& duelId) override;
        bool DeclineDuel(const AZStd::string& duelId) override;
        bool CancelDuel(const AZStd::string& duelId) override;
        bool SurrenderDuel(const AZStd::string& duelId) override;
        bool LearnTrainerAbility(const AZStd::string& trainerId, const AZStd::string& abilityId) override;
        bool SelectTalent(const AZStd::string& talentId) override;
        bool LearnProfession(const AZStd::string& trainerId, const AZStd::string& professionId) override;
        bool AssignActionBarSlot(int slotIndex, const AZStd::string& abilityId) override;
        bool MoveActionBarSlot(int fromSlotIndex, int toSlotIndex) override;
        bool ClearActionBarSlot(int slotIndex) override;
        bool MoveInventorySlot(int fromSlotIndex, int toSlotIndex) override;
        bool EquipInventorySlot(int slotIndex) override;
        bool UnequipInventorySlot(const AZStd::string& equipmentSlot) override;
        bool BrowseAuctions(
            const AZStd::string& search,
            const AZStd::string& itemType,
            const AZStd::string& sort) override;
        bool ListAuctionItem(int slotIndex, int stackCount, int buyoutCopper, AZ::s64 durationSeconds) override;
        bool BuyoutAuction(const AZStd::string& auctionId) override;
        bool CancelAuction(const AZStd::string& auctionId) override;
        bool SubmitChatMessage(
            const AZStd::string& channel,
            const AZStd::string& targetName,
            const AZStd::string& messageText) override;
        bool AddFriend(const AZStd::string& name) override;
        bool RemoveFriend(const AZStd::string& name) override;
        bool InviteParty(const AZStd::string& targetName, const AZStd::string& targetCharacterId) override;
        bool AcceptPartyInvite(const AZStd::string& inviteId) override;
        bool DeclinePartyInvite(const AZStd::string& inviteId) override;
        bool LeaveParty() override;
        bool DisbandParty() override;
        bool CreateGuild(const AZStd::string& guildName) override;
        bool InviteGuild(const AZStd::string& targetName) override;
        bool AcceptGuildInvite(const AZStd::string& inviteId) override;
        bool DeclineGuildInvite(const AZStd::string& inviteId) override;
        bool LeaveGuild() override;
        bool DisbandGuild() override;
        bool PromoteGuildMember(const AZStd::string& targetName) override;
        bool DemoteGuildMember(const AZStd::string& targetName) override;
        bool RemoveGuildMember(const AZStd::string& targetName) override;
        bool SetGuildMessageOfTheDay(const AZStd::string& messageOfTheDay) override;
        bool DisconnectWorld() override;
        bool ReconnectWorld() override;
        void SetCameraState(const ClientCameraState& cameraState) override;

    private:
        void ParseLaunchOptions();
        bool RequestStartupLevelLoad();
        void MarkLevelReady(const char* levelName);
        void AttemptInitialWorldConnect();
        void SetFrontendBusy(bool busy, const char* statusMessage);
        void SetFrontendError(const AZStd::string& errorMessage);
        bool TryRestoreRememberedFrontendSession();
        void ResetFrontendToLogin(const char* statusMessage);
        bool HasFrontendSession() const;
        bool HasSelectedRealm() const;
        bool HasSelectedCharacter() const;
        void SelectCreatedCharacterIfPresent();
        bool ApplyWorldSessionResponse(NetClient::WorldSessionResponse&& response, const char* source);
        bool ApplySocialStateResponse(NetClient::SocialStateResponse&& response, const char* source);
        bool ApplyAuctionStateResponse(NetClient::AuctionStateResponse&& response, const char* source);
        void EnsureAbilityPresentationDefaults(NetClient::WorldSessionResponse& session, const char* source);
        bool PollWorldState();
        bool PollSocialState();
        void LogCombatStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source);
        void LogAbilityStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source);
        void LogQuestStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source);
        void LogTrainerStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source);

        ClientLaunchOptions m_launchOptions;
        ClientWorldState m_worldState;
        ClientFrontendState m_frontendState;
        ClientCameraState m_cameraState;
        bool m_levelReady = false;
        bool m_startupLevelLoadPending = false;
        float m_startupLevelLoadRetryAccumulator = 0.0f;
        int m_startupLevelLoadAttempts = 0;
        bool m_worldConnectStartLogged = false;
        bool m_levelReadyLogged = false;
        float m_statePollAccumulator = 0.0f;
        float m_socialPollAccumulator = 0.0f;
        AZStd::string m_lastChatMessageId;
    };
}
