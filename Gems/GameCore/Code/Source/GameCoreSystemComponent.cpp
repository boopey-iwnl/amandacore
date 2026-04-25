#include <GameCore/GameCoreSystemComponent.h>

#include <AzCore/Console/IConsole.h>
#include <AzCore/Debug/Trace.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <AzCore/std/algorithm.h>
#include <AzFramework/API/ApplicationAPI.h>
#include <AzFramework/CommandLine/CommandLine.h>
#include <NetClient/WorldHttpClient.h>

namespace GameCore
{
    namespace
    {
        constexpr float WorldStatePollIntervalSeconds = 0.10f;
        constexpr float SocialStatePollIntervalSeconds = 0.50f;

        struct AbilityPresentationDefinition
        {
            const char* m_id;
            const char* m_displayName;
            const char* m_description;
            const char* m_requirementText;
            const char* m_iconKind;
            int m_requiredLevel;
            int m_actionBarSlot;
            const char* m_actionBarHotkey;
            const char* m_actionBarLabel;
            bool m_requiresTarget;
        };

        constexpr AbilityPresentationDefinition WarriorAbilityCatalog[] = {
            {
                "auto_attack",
                "Auto Attack",
                "Maintain pressure with your weapon while a target stays in melee range.",
                "Known by all Warriors.",
                "weapon",
                1,
                0,
                "F",
                "Atk",
                true,
            },
            {
                "steady_strike",
                "Steady Strike",
                "A measured weapon strike that builds Grit through steady contact.",
                "Known by default.",
                "strike",
                1,
                1,
                "1",
                "Strike",
                true,
            },
            {
                "brace",
                "Brace",
                "Set your stance and recover a small amount of health without needing a target.",
                "Known by default.",
                "defense",
                1,
                2,
                "2",
                "Brace",
                false,
            },
            {
                "driving_blow",
                "Driving Blow",
                "A harder follow-through strike trained early in the starter journey.",
                "Requires level 2 and a Warrior trainer.",
                "strike",
                2,
                -1,
                "",
                "",
                true,
            },
            {
                "rallying_call",
                "Rallying Call",
                "A short shout that restores Grit before the next exchange.",
                "Requires level 4 and a Warrior trainer.",
                "utility",
                4,
                -1,
                "",
                "",
                false,
            },
            {
                "hampering_strike",
                "Hampering Strike",
                "A controlling strike previewed for the next band of Warrior progression.",
                "Requires level 6 and a Warrior trainer.",
                "strike",
                6,
                -1,
                "",
                "",
                true,
            },
            {
                "guarded_form",
                "Guarded Form",
                "Set your feet and recover while under pressure.",
                "Requires level 8 and a Warrior trainer.",
                "defense",
                8,
                -1,
                "",
                "",
                false,
            },
            {
                "overhand_cut",
                "Overhand Cut",
                "Spend stored Grit on a heavy weapon attack.",
                "Requires level 10 and a Warrior trainer.",
                "strike",
                10,
                -1,
                "",
                "",
                true,
            },
        };

        AZStd::string NormalizeAbilityId(const AZStd::string& abilityId)
        {
            if (abilityId == "ember_bolt")
            {
                return "steady_strike";
            }
            if (abilityId == "steady_blast")
            {
                return "brace";
            }
            if (abilityId == "war_cry")
            {
                return "rallying_call";
            }
            return abilityId;
        }

        bool SpellbookPayloadLooksEmpty(const NetClient::WorldSessionResponse& session)
        {
            if (session.m_spellbookEntries.empty())
            {
                return true;
            }

            for (const auto& entry : session.m_spellbookEntries)
            {
                if (!entry.m_id.empty() || !entry.m_displayName.empty())
                {
                    return false;
                }
            }

            return true;
        }

        bool ActionBarPayloadLooksEmpty(const NetClient::WorldSessionResponse& session)
        {
            if (session.m_actionBarSlots.empty())
            {
                return true;
            }
            if (session.m_actionBarSlots.size() >= 48)
            {
                return false;
            }

            for (const auto& slot : session.m_actionBarSlots)
            {
                if (!slot.m_abilityId.empty() || !slot.m_buttonLabel.empty() || !slot.m_displayName.empty())
                {
                    return false;
                }
            }

            return true;
        }
    }

    void GameCoreSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<GameCoreSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void GameCoreSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("GameCoreService"));
    }

    void GameCoreSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("GameCoreService"));
    }

    void GameCoreSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("NetClientService"));
    }

    void GameCoreSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void GameCoreSystemComponent::Activate()
    {
        IGameCoreRequests::Register(this);
        ParseLaunchOptions();
        AzFramework::LevelSystemLifecycleNotificationBus::Handler::BusConnect();
        AZ::TickBus::Handler::BusConnect();

        if (const auto* levelLifecycle = AzFramework::LevelSystemLifecycleInterface::Get();
            levelLifecycle && levelLifecycle->IsLevelLoaded())
        {
            MarkLevelReady(levelLifecycle->GetCurrentLevelName());
        }
        else if (auto* console = AZ::Interface<AZ::IConsole>::Get())
        {
            const auto result = console->PerformCommand("LoadLevel testzone01");
            if (!result.IsSuccess())
            {
                AZ_Warning("amandacore", false, "Unable to request startup level load: %s", result.GetError().c_str());
            }
        }
    }

    void GameCoreSystemComponent::Deactivate()
    {
        AZ::TickBus::Handler::BusDisconnect();
        AzFramework::LevelSystemLifecycleNotificationBus::Handler::BusDisconnect();
        if (m_worldState.m_worldConnected)
        {
            DisconnectWorld();
        }

        if (IGameCoreRequests::Get() == this)
        {
            IGameCoreRequests::Unregister(this);
        }
    }

    void GameCoreSystemComponent::OnTick(float deltaTime, AZ::ScriptTimePoint)
    {
        if (!m_worldState.m_worldConnected)
        {
            return;
        }

        m_statePollAccumulator += deltaTime;
        if (m_statePollAccumulator < WorldStatePollIntervalSeconds)
        {
            m_socialPollAccumulator += deltaTime;
            if (m_socialPollAccumulator >= SocialStatePollIntervalSeconds)
            {
                m_socialPollAccumulator = 0.0f;
                PollSocialState();
            }
            return;
        }

        m_statePollAccumulator = 0.0f;
        PollWorldState();

        m_socialPollAccumulator += deltaTime;
        if (m_socialPollAccumulator >= SocialStatePollIntervalSeconds)
        {
            m_socialPollAccumulator = 0.0f;
            PollSocialState();
        }
    }

    void GameCoreSystemComponent::OnLoadingComplete(const char* levelName)
    {
        MarkLevelReady(levelName);
    }

    const ClientLaunchOptions& GameCoreSystemComponent::GetLaunchOptions() const
    {
        return m_launchOptions;
    }

    const ClientWorldState& GameCoreSystemComponent::GetClientWorldState() const
    {
        return m_worldState;
    }

    const ClientCameraState& GameCoreSystemComponent::GetCameraState() const
    {
        return m_cameraState;
    }

    void GameCoreSystemComponent::SetCameraState(const ClientCameraState& cameraState)
    {
        m_cameraState = cameraState;
    }

    bool GameCoreSystemComponent::SubmitMove(double deltaX, double deltaY)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        AZ_Printf(
            "amandacore",
            "client.move_submitted token=%s delta=(%.3f, %.3f)",
            m_worldState.m_session.m_worldSessionToken.c_str(),
            deltaX,
            deltaY);

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->Move(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                deltaX,
                deltaY,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "Move failed: %s", error.c_str());
            return false;
        }

        ApplyWorldSessionResponse(AZStd::move(response), "move");
        AZ_Printf(
            "amandacore",
            "client.authoritative_position_applied token=%s position=(%.3f, %.3f, %.3f)",
            m_worldState.m_session.m_worldSessionToken.c_str(),
            m_worldState.m_session.m_position.m_x,
            m_worldState.m_session.m_position.m_y,
            m_worldState.m_session.m_position.m_z);
        return true;
    }

    bool GameCoreSystemComponent::SetTarget(const AZStd::string& targetId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->SetTarget(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                targetId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "SetTarget failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "target");
        return true;
    }

    bool GameCoreSystemComponent::InteractWithEntity(const AZStd::string& entityId)
    {
        if (entityId.empty() || !SetTarget(entityId))
        {
            return false;
        }

        m_worldState.m_pendingInteractionEntityId = entityId;
        ++m_worldState.m_pendingInteractionSequence;
        AZ_Printf(
            "amandacore",
            "client.npc_interaction_requested targetId=%s sequence=%llu",
            entityId.c_str(),
            static_cast<unsigned long long>(m_worldState.m_pendingInteractionSequence));
        return true;
    }

    bool GameCoreSystemComponent::AcceptQuest(const AZStd::string& questId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->AcceptQuest(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                questId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "AcceptQuest failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "quest_accept");
        return true;
    }

    bool GameCoreSystemComponent::EnterDungeon(const AZStd::string& dungeonId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->EnterDungeon(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                dungeonId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "EnterDungeon failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "dungeon_enter");
        return true;
    }

    bool GameCoreSystemComponent::ExitDungeon()
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->ExitDungeon(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "ExitDungeon failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "dungeon_exit");
        return true;
    }

    bool GameCoreSystemComponent::TrackQuest(const AZStd::string& questId, bool tracked)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->TrackQuest(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                questId,
                tracked,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "TrackQuest failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), tracked ? "quest_track" : "quest_untrack");
        return true;
    }

    bool GameCoreSystemComponent::SetAutoAttack(bool enabled)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->SetAutoAttack(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                enabled,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "SetAutoAttack failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "auto_attack");
        return true;
    }

    bool GameCoreSystemComponent::ActivateAbility(const AZStd::string& abilityId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->ActivateAbility(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                abilityId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "ActivateAbility failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "ability");
        return true;
    }

    bool GameCoreSystemComponent::RequestDuel(const AZStd::string& targetCharacterId, const AZStd::string& targetName)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->RequestDuel(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                targetCharacterId,
                targetName,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "RequestDuel failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "duel_request");
        return true;
    }

    bool GameCoreSystemComponent::AcceptDuel(const AZStd::string& duelId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->AcceptDuel(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                duelId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "AcceptDuel failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "duel_accept");
        return true;
    }

    bool GameCoreSystemComponent::DeclineDuel(const AZStd::string& duelId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->DeclineDuel(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                duelId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "DeclineDuel failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "duel_decline");
        return true;
    }

    bool GameCoreSystemComponent::CancelDuel(const AZStd::string& duelId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->CancelDuel(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                duelId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "CancelDuel failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "duel_cancel");
        return true;
    }

    bool GameCoreSystemComponent::SurrenderDuel(const AZStd::string& duelId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->SurrenderDuel(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                duelId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "SurrenderDuel failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "duel_surrender");
        return true;
    }

    bool GameCoreSystemComponent::LearnTrainerAbility(const AZStd::string& trainerId, const AZStd::string& abilityId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->LearnTrainerAbility(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                trainerId,
                abilityId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "LearnTrainerAbility failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "trainer_learn");
        return true;
    }

    bool GameCoreSystemComponent::SelectTalent(const AZStd::string& talentId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->SelectTalent(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                talentId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "SelectTalent failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "talent_select");
        return true;
    }

    bool GameCoreSystemComponent::AssignActionBarSlot(int slotIndex, const AZStd::string& abilityId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->AssignActionBarSlot(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                slotIndex,
                abilityId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "AssignActionBarSlot failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "action_bar_assign");
        return true;
    }

    bool GameCoreSystemComponent::MoveActionBarSlot(int fromSlotIndex, int toSlotIndex)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->MoveActionBarSlot(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                fromSlotIndex,
                toSlotIndex,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "MoveActionBarSlot failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "action_bar_move");
        return true;
    }

    bool GameCoreSystemComponent::ClearActionBarSlot(int slotIndex)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->ClearActionBarSlot(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                slotIndex,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "ClearActionBarSlot failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "action_bar_clear");
        return true;
    }

    bool GameCoreSystemComponent::MoveInventorySlot(int fromSlotIndex, int toSlotIndex)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->MoveInventorySlot(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                fromSlotIndex,
                toSlotIndex,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "MoveInventorySlot failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "inventory_move");
        return true;
    }

    bool GameCoreSystemComponent::BrowseAuctions(
        const AZStd::string& search,
        const AZStd::string& itemType,
        const AZStd::string& sort)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::AuctionStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->BrowseAuctions(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                search,
                itemType,
                sort,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "BrowseAuctions failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyAuctionStateResponse(AZStd::move(response), "auction_browse");
        return true;
    }

    bool GameCoreSystemComponent::ListAuctionItem(int slotIndex, int stackCount, int buyoutCopper, AZ::s64 durationSeconds)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::AuctionStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->ListAuctionItem(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                slotIndex,
                stackCount,
                buyoutCopper,
                durationSeconds,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "ListAuctionItem failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyAuctionStateResponse(AZStd::move(response), "auction_list");
        (void)PollWorldState();
        return true;
    }

    bool GameCoreSystemComponent::BuyoutAuction(const AZStd::string& auctionId)
    {
        if (!m_worldState.m_worldConnected || auctionId.empty())
        {
            return false;
        }

        NetClient::AuctionStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->BuyoutAuction(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                auctionId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "BuyoutAuction failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyAuctionStateResponse(AZStd::move(response), "auction_buyout");
        (void)PollWorldState();
        return true;
    }

    bool GameCoreSystemComponent::CancelAuction(const AZStd::string& auctionId)
    {
        if (!m_worldState.m_worldConnected || auctionId.empty())
        {
            return false;
        }

        NetClient::AuctionStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->CancelAuction(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                auctionId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "CancelAuction failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyAuctionStateResponse(AZStd::move(response), "auction_cancel");
        (void)PollWorldState();
        return true;
    }

    bool GameCoreSystemComponent::SubmitChatMessage(
        const AZStd::string& channel,
        const AZStd::string& targetName,
        const AZStd::string& messageText)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->SendChat(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                channel,
                targetName,
                messageText,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "SendChat failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "chat_send");
        return true;
    }

    bool GameCoreSystemComponent::AddFriend(const AZStd::string& name)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->AddFriend(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                name,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "AddFriend failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "friend_add");
        return true;
    }

    bool GameCoreSystemComponent::RemoveFriend(const AZStd::string& name)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->RemoveFriend(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                name,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "RemoveFriend failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "friend_remove");
        return true;
    }

    bool GameCoreSystemComponent::InviteParty(const AZStd::string& targetName, const AZStd::string& targetCharacterId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->InviteParty(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                targetName,
                targetCharacterId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "InviteParty failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "party_invite");
        return true;
    }

    bool GameCoreSystemComponent::AcceptPartyInvite(const AZStd::string& inviteId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->AcceptPartyInvite(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                inviteId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "AcceptPartyInvite failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "party_accept");
        return true;
    }

    bool GameCoreSystemComponent::DeclinePartyInvite(const AZStd::string& inviteId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->DeclinePartyInvite(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                inviteId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "DeclinePartyInvite failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "party_decline");
        return true;
    }

    bool GameCoreSystemComponent::LeaveParty()
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->LeaveParty(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "LeaveParty failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "party_leave");
        return true;
    }

    bool GameCoreSystemComponent::DisbandParty()
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->DisbandParty(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "DisbandParty failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "party_disband");
        return true;
    }

    bool GameCoreSystemComponent::CreateGuild(const AZStd::string& guildName)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->CreateGuild(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                guildName,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "CreateGuild failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "guild_create");
        return true;
    }

    bool GameCoreSystemComponent::InviteGuild(const AZStd::string& targetName)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->InviteGuild(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                targetName,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "InviteGuild failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "guild_invite");
        return true;
    }

    bool GameCoreSystemComponent::AcceptGuildInvite(const AZStd::string& inviteId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->AcceptGuildInvite(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                inviteId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "AcceptGuildInvite failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "guild_accept");
        return true;
    }

    bool GameCoreSystemComponent::DeclineGuildInvite(const AZStd::string& inviteId)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->DeclineGuildInvite(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                inviteId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "DeclineGuildInvite failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "guild_decline");
        return true;
    }

    bool GameCoreSystemComponent::LeaveGuild()
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->LeaveGuild(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "LeaveGuild failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "guild_leave");
        return true;
    }

    bool GameCoreSystemComponent::DisbandGuild()
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->DisbandGuild(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "DisbandGuild failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "guild_disband");
        return true;
    }

    bool GameCoreSystemComponent::PromoteGuildMember(const AZStd::string& targetName)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->PromoteGuildMember(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                targetName,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "PromoteGuildMember failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "guild_promote");
        return true;
    }

    bool GameCoreSystemComponent::DemoteGuildMember(const AZStd::string& targetName)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->DemoteGuildMember(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                targetName,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "DemoteGuildMember failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "guild_demote");
        return true;
    }

    bool GameCoreSystemComponent::RemoveGuildMember(const AZStd::string& targetName)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->RemoveGuildMember(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                targetName,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "RemoveGuildMember failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "guild_remove");
        return true;
    }

    bool GameCoreSystemComponent::SetGuildMessageOfTheDay(const AZStd::string& messageOfTheDay)
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->SetGuildMessageOfTheDay(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                messageOfTheDay,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "SetGuildMessageOfTheDay failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "guild_motd");
        return true;
    }

    bool GameCoreSystemComponent::DisconnectWorld()
    {
        if (!m_worldState.m_worldConnected)
        {
            return true;
        }

        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->Disconnect(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "Disconnect failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_worldConnected = false;
        AZ_Printf("amandacore", "client.world_disconnected token=%s", m_worldState.m_session.m_worldSessionToken.c_str());
        return true;
    }

    bool GameCoreSystemComponent::ReconnectWorld()
    {
        if (m_worldState.m_session.m_worldSessionToken.empty())
        {
            return false;
        }

        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->Reconnect(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "Reconnect failed: %s", error.c_str());
            return false;
        }

        ApplyWorldSessionResponse(AZStd::move(response), "reconnect");
        m_worldState.m_worldConnected = true;
        PollSocialState();
        AZ_Printf(
            "amandacore",
            "client.world_connected reconnect=true token=%s position=(%.3f, %.3f, %.3f)",
            m_worldState.m_session.m_worldSessionToken.c_str(),
            m_worldState.m_session.m_position.m_x,
            m_worldState.m_session.m_position.m_y,
            m_worldState.m_session.m_position.m_z);
        return true;
    }

    void GameCoreSystemComponent::ParseLaunchOptions()
    {
        const AzFramework::CommandLine* commandLine = nullptr;
        AzFramework::ApplicationRequests::Bus::BroadcastResult(
            commandLine,
            &AzFramework::ApplicationRequests::Bus::Events::GetApplicationCommandLine);

        if (!commandLine)
        {
            AZ_Warning("amandacore", false, "Client launch arguments were not available.");
            return;
        }

        if (commandLine->HasSwitch("join-ticket"))
        {
            m_launchOptions.m_joinTicketId = commandLine->GetSwitchValue("join-ticket");
        }

        if (commandLine->HasSwitch("world-endpoint"))
        {
            m_launchOptions.m_worldEndpoint = commandLine->GetSwitchValue("world-endpoint");
        }

        m_worldState.m_launchOptionsPresent = m_launchOptions.IsValid();
    }

    void GameCoreSystemComponent::MarkLevelReady(const char* levelName)
    {
        m_levelReady = true;
        if (!m_worldConnectStartLogged)
        {
            m_worldConnectStartLogged = true;
            AZ_Printf(
                "amandacore",
                "client.world_connect_started endpoint=%s joinTicketPresent=%s",
                m_launchOptions.m_worldEndpoint.c_str(),
                m_launchOptions.m_joinTicketId.empty() ? "false" : "true");
        }

        if (m_levelReadyLogged)
        {
            return;
        }

        m_levelReadyLogged = true;
        const char* resolvedLevelName = (levelName && levelName[0] != '\0') ? levelName : "unknown";
        AZ_Printf("amandacore", "client.level_ready level=%s", resolvedLevelName);

        if (!m_worldState.m_connectAttempted)
        {
            m_worldState.m_connectAttempted = true;
            AttemptInitialWorldConnect();
        }
    }

    void GameCoreSystemComponent::AttemptInitialWorldConnect()
    {
        if (!m_launchOptions.IsValid())
        {
            m_worldState.m_errorMessage = "Missing --join-ticket or --world-endpoint.";
            AZ_Warning("amandacore", false, "%s", m_worldState.m_errorMessage.c_str());
            return;
        }

        auto* httpClient = NetClient::IWorldHttpClient::Get();
        if (!httpClient)
        {
            m_worldState.m_errorMessage = "NetClient interface is unavailable.";
            AZ_Warning("amandacore", false, "%s", m_worldState.m_errorMessage.c_str());
            return;
        }

        NetClient::WorldSessionResponse session;
        AZStd::string error;
        if (!httpClient->Connect(m_launchOptions.m_worldEndpoint, m_launchOptions.m_joinTicketId, session, error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "World connect failed: %s", error.c_str());
            return;
        }

        NetClient::WorldBootstrapResponse bootstrap;
        if (!httpClient->Bootstrap(m_launchOptions.m_worldEndpoint, bootstrap, error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "Bootstrap failed: %s", error.c_str());
            return;
        }

        const bool isStonewakeValeBootstrap = bootstrap.m_zoneId == "stonewake_vale" && bootstrap.m_cellId == "stonewake_vale";
        const bool isLegacyStonewakeBootstrap = bootstrap.m_zoneId == "sunset_frontier" && bootstrap.m_cellId == "stonewake_vale";
        const bool isLegacyWestApproachBootstrap = bootstrap.m_zoneId == "sunset_frontier" && bootstrap.m_cellId == "west_approach";
        if (!isStonewakeValeBootstrap && !isLegacyStonewakeBootstrap && !isLegacyWestApproachBootstrap)
        {
            m_worldState.m_errorMessage = "Bootstrap zone mapping did not match the playable slice contract.";
            AZ_Warning(
                "amandacore",
                false,
                "%s zone=%s cell=%s",
                m_worldState.m_errorMessage.c_str(),
                bootstrap.m_zoneId.c_str(),
                bootstrap.m_cellId.c_str());
            return;
        }

        ApplyWorldSessionResponse(AZStd::move(session), "connect");
        m_worldState.m_bootstrap = AZStd::move(bootstrap);
        m_worldState.m_bootstrapReady = true;
        m_worldState.m_worldConnected = true;
        m_worldState.m_errorMessage.clear();
        PollSocialState();
        AZ_Printf(
            "amandacore",
            "client.world_bootstrap_applied zone=%s cell=%s revision=%s motd=%s",
            m_worldState.m_bootstrap.m_zoneId.c_str(),
            m_worldState.m_bootstrap.m_cellId.c_str(),
            m_worldState.m_bootstrap.m_revision.c_str(),
            m_worldState.m_bootstrap.m_motd.c_str());

        AZ_Printf(
            "amandacore",
            "client.world_connected reconnect=false token=%s",
            m_worldState.m_session.m_worldSessionToken.c_str());
        AZ_Printf(
            "amandacore",
            "client.player_spawned character=%s position=(%.3f, %.3f, %.3f)",
            m_worldState.m_session.m_displayName.c_str(),
            m_worldState.m_session.m_position.m_x,
            m_worldState.m_session.m_position.m_y,
            m_worldState.m_session.m_position.m_z);
        AZ_Printf(
            "amandacore",
            "client.input_help move=WASD camera=RMB interact=right_click_npc bag=B spellbook=P settings=ESC disconnect=X reconnect=R quit=Q");
    }

    bool GameCoreSystemComponent::ApplyWorldSessionResponse(NetClient::WorldSessionResponse&& response, const char* source)
    {
        const NetClient::WorldSessionResponse previousSession = m_worldState.m_session;
        m_worldState.m_session = AZStd::move(response);
        EnsureAbilityPresentationDefaults(m_worldState.m_session, source);
        LogCombatStateIfChanged(previousSession, source);
        LogAbilityStateIfChanged(previousSession, source);
        LogQuestStateIfChanged(previousSession, source);
        LogTrainerStateIfChanged(previousSession, source);
        return true;
    }

    bool GameCoreSystemComponent::ApplySocialStateResponse(NetClient::SocialStateResponse&& response, const char* source)
    {
        for (const auto& message : response.m_chatMessages)
        {
            const bool alreadyPresent = AZStd::find_if(
                m_worldState.m_social.m_chatMessages.begin(),
                m_worldState.m_social.m_chatMessages.end(),
                [&message](const NetClient::ChatMessageState& existing)
                {
                    return existing.m_messageId == message.m_messageId;
                }) != m_worldState.m_social.m_chatMessages.end();
            if (alreadyPresent)
            {
                continue;
            }

            m_worldState.m_social.m_chatMessages.push_back(message);
            if (!message.m_messageId.empty())
            {
                m_lastChatMessageId = message.m_messageId;
            }
        }

        while (m_worldState.m_social.m_chatMessages.size() > 120)
        {
            m_worldState.m_social.m_chatMessages.erase(m_worldState.m_social.m_chatMessages.begin());
        }

        m_worldState.m_social.m_friends = AZStd::move(response.m_friends);
        m_worldState.m_social.m_hasParty = response.m_hasParty;
        m_worldState.m_social.m_party = AZStd::move(response.m_party);
        m_worldState.m_social.m_partyInvites = AZStd::move(response.m_partyInvites);
        m_worldState.m_social.m_hasGuild = response.m_hasGuild;
        m_worldState.m_social.m_guild = AZStd::move(response.m_guild);
        m_worldState.m_social.m_guildInvites = AZStd::move(response.m_guildInvites);

        AZ_Printf(
            "amandacore",
            "client.social_state_applied source=%s messages=%zu friends=%zu party=%s partyInvites=%zu guild=%s guildInvites=%zu",
            source,
            m_worldState.m_social.m_chatMessages.size(),
            m_worldState.m_social.m_friends.size(),
            m_worldState.m_social.m_hasParty ? "true" : "false",
            m_worldState.m_social.m_partyInvites.size(),
            m_worldState.m_social.m_hasGuild ? "true" : "false",
            m_worldState.m_social.m_guildInvites.size());
        return true;
    }

    bool GameCoreSystemComponent::ApplyAuctionStateResponse(NetClient::AuctionStateResponse&& response, const char* source)
    {
        m_worldState.m_auction = AZStd::move(response);
        AZ_Printf(
            "amandacore",
            "client.auction_state_applied source=%s listings=%zu mine=%zu mail=%zu",
            source,
            m_worldState.m_auction.m_listings.size(),
            m_worldState.m_auction.m_myAuctions.size(),
            m_worldState.m_auction.m_mail.size());
        return true;
    }

    void GameCoreSystemComponent::EnsureAbilityPresentationDefaults(
        NetClient::WorldSessionResponse& session,
        const char* source)
    {
        AZStd::vector<AZStd::string> normalizedLearnedAbilityIds;
        normalizedLearnedAbilityIds.reserve(session.m_learnedAbilityIds.size());
        for (const AZStd::string& learnedAbilityId : session.m_learnedAbilityIds)
        {
            const AZStd::string normalizedAbilityId = NormalizeAbilityId(learnedAbilityId);
            if (normalizedAbilityId.empty())
            {
                continue;
            }

            if (AZStd::find(
                    normalizedLearnedAbilityIds.begin(),
                    normalizedLearnedAbilityIds.end(),
                    normalizedAbilityId) != normalizedLearnedAbilityIds.end())
            {
                continue;
            }

            normalizedLearnedAbilityIds.push_back(normalizedAbilityId);
        }
        session.m_learnedAbilityIds = AZStd::move(normalizedLearnedAbilityIds);

        if (session.m_learnedAbilityIds.empty())
        {
            return;
        }

        const bool rebuiltSpellbook = SpellbookPayloadLooksEmpty(session);
        const bool rebuiltActionBar = ActionBarPayloadLooksEmpty(session);
        if (!rebuiltSpellbook && !rebuiltActionBar)
        {
            return;
        }

        auto knowsAbility = [&](const char* abilityId)
        {
            return AZStd::find(
                       session.m_learnedAbilityIds.begin(),
                       session.m_learnedAbilityIds.end(),
                       AZStd::string(abilityId)) != session.m_learnedAbilityIds.end();
        };

        if (rebuiltSpellbook)
        {
            session.m_spellbookEntries.clear();
            session.m_spellbookEntries.reserve(AZ_ARRAY_SIZE(WarriorAbilityCatalog));
            for (const AbilityPresentationDefinition& definition : WarriorAbilityCatalog)
            {
                NetClient::SpellbookEntryState entry;
                entry.m_id = definition.m_id;
                entry.m_displayName = definition.m_displayName;
                entry.m_description = definition.m_description;
                entry.m_requirementText = definition.m_requirementText;
                entry.m_iconKind = definition.m_iconKind;
                entry.m_requiredLevel = definition.m_requiredLevel;
                entry.m_learned = knowsAbility(definition.m_id);
                session.m_spellbookEntries.push_back(AZStd::move(entry));
            }
        }

        if (rebuiltActionBar)
        {
            session.m_actionBarSlots.clear();
            session.m_actionBarSlots.reserve(48);
            for (int slotIndex = 0; slotIndex < 48; ++slotIndex)
            {
                NetClient::ActionBarSlotState slot;
                slot.m_slotIndex = slotIndex;
                session.m_actionBarSlots.push_back(AZStd::move(slot));
            }

            for (const AbilityPresentationDefinition& definition : WarriorAbilityCatalog)
            {
                if (definition.m_actionBarSlot < 0 || definition.m_actionBarSlot >= 48 || !knowsAbility(definition.m_id))
                {
                    continue;
                }

                NetClient::ActionBarSlotState& slot = session.m_actionBarSlots[definition.m_actionBarSlot];
                slot.m_slotIndex = definition.m_actionBarSlot;
                slot.m_hotkey = definition.m_actionBarHotkey;
                slot.m_abilityId = definition.m_id;
                slot.m_displayName = definition.m_displayName;
                slot.m_buttonLabel = definition.m_actionBarLabel;
                slot.m_iconKind = definition.m_iconKind;
                slot.m_requiresTarget = definition.m_requiresTarget;
                slot.m_learned = true;
            }
        }

        AZ_Printf(
            "amandacore",
            "client.ability_presentation_rehydrated source=%s learned=%zu spellbook=%zu actionBar=%zu",
            source,
            session.m_learnedAbilityIds.size(),
            session.m_spellbookEntries.size(),
            session.m_actionBarSlots.size());
    }

    bool GameCoreSystemComponent::PollWorldState()
    {
        NetClient::WorldSessionResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->State(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "World state poll failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplyWorldSessionResponse(AZStd::move(response), "state");
        return true;
    }

    bool GameCoreSystemComponent::PollSocialState()
    {
        if (!m_worldState.m_worldConnected)
        {
            return false;
        }

        NetClient::SocialStateResponse response;
        AZStd::string error;
        if (!NetClient::IWorldHttpClient::Get() ||
            !NetClient::IWorldHttpClient::Get()->SocialState(
                m_launchOptions.m_worldEndpoint,
                m_worldState.m_session.m_worldSessionToken,
                m_lastChatMessageId,
                response,
                error))
        {
            m_worldState.m_errorMessage = error;
            AZ_Warning("amandacore", false, "Social state poll failed: %s", error.c_str());
            return false;
        }

        m_worldState.m_errorMessage.clear();
        ApplySocialStateResponse(AZStd::move(response), "social_state");
        return true;
    }

    void GameCoreSystemComponent::LogCombatStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source)
    {
        const bool sessionChanged =
            previousSession.m_health != m_worldState.m_session.m_health ||
            previousSession.m_resource != m_worldState.m_session.m_resource ||
            previousSession.m_alive != m_worldState.m_session.m_alive ||
            previousSession.m_currentTargetId != m_worldState.m_session.m_currentTargetId ||
            previousSession.m_autoAttackActive != m_worldState.m_session.m_autoAttackActive ||
            previousSession.m_globalCooldownEndsAt != m_worldState.m_session.m_globalCooldownEndsAt ||
            previousSession.m_castEndsAt != m_worldState.m_session.m_castEndsAt ||
            previousSession.m_castingAbilityId != m_worldState.m_session.m_castingAbilityId;

        bool entityChanged = previousSession.m_entities.size() != m_worldState.m_session.m_entities.size();
        if (!entityChanged)
        {
            for (size_t index = 0; index < previousSession.m_entities.size(); ++index)
            {
                const auto& previousEntity = previousSession.m_entities[index];
                const auto& currentEntity = m_worldState.m_session.m_entities[index];
                if (previousEntity.m_id != currentEntity.m_id ||
                    previousEntity.m_health != currentEntity.m_health ||
                    previousEntity.m_alive != currentEntity.m_alive ||
                    previousEntity.m_aiState != currentEntity.m_aiState ||
                    previousEntity.m_targetable != currentEntity.m_targetable)
                {
                    entityChanged = true;
                    break;
                }
            }
        }

        if (!sessionChanged && !entityChanged)
        {
            return;
        }

        AZ_Printf(
            "amandacore",
            "client.authoritative_combat_state_applied source=%s health=%.1f resource=%.1f alive=%s targetId=%s autoAttack=%s castAbility=%s castEndsAt=%lld gcdEndsAt=%lld",
            source,
            m_worldState.m_session.m_health,
            m_worldState.m_session.m_resource,
            m_worldState.m_session.m_alive ? "true" : "false",
            m_worldState.m_session.m_currentTargetId.c_str(),
            m_worldState.m_session.m_autoAttackActive ? "true" : "false",
            m_worldState.m_session.m_castingAbilityId.c_str(),
            static_cast<long long>(m_worldState.m_session.m_castEndsAt),
            static_cast<long long>(m_worldState.m_session.m_globalCooldownEndsAt));
    }

    void GameCoreSystemComponent::LogQuestStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source)
    {
        const auto& previousQuest = previousSession.m_quest;
        const auto& currentQuest = m_worldState.m_session.m_quest;
        if (previousSession.m_experience == m_worldState.m_session.m_experience &&
            previousQuest.m_id == currentQuest.m_id &&
            previousQuest.m_state == currentQuest.m_state &&
            previousQuest.m_currentCount == currentQuest.m_currentCount &&
            previousQuest.m_targetCount == currentQuest.m_targetCount)
        {
            return;
        }

        AZ_Printf(
            "amandacore",
            "client.quest_state_applied source=%s questId=%s state=%s progress=%d/%d experience=%d currency=%dg %ds %dc",
            source,
            currentQuest.m_id.c_str(),
            currentQuest.m_state.c_str(),
            currentQuest.m_currentCount,
            currentQuest.m_targetCount,
            m_worldState.m_session.m_experience,
            m_worldState.m_session.m_currency.m_gold,
            m_worldState.m_session.m_currency.m_silver,
            m_worldState.m_session.m_currency.m_copper);
    }

    void GameCoreSystemComponent::LogAbilityStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source)
    {
        if (previousSession.m_learnedAbilityIds == m_worldState.m_session.m_learnedAbilityIds &&
            previousSession.m_spellbookEntries.size() == m_worldState.m_session.m_spellbookEntries.size() &&
            previousSession.m_actionBarSlots.size() == m_worldState.m_session.m_actionBarSlots.size())
        {
            return;
        }

        int filledActionBarSlots = 0;
        for (const auto& slot : m_worldState.m_session.m_actionBarSlots)
        {
            if (!slot.m_abilityId.empty())
            {
                ++filledActionBarSlots;
            }
        }

        AZ_Printf(
            "amandacore",
            "client.ability_state_applied source=%s learned=%zu spellbookEntries=%zu actionBarFilled=%d",
            source,
            m_worldState.m_session.m_learnedAbilityIds.size(),
            m_worldState.m_session.m_spellbookEntries.size(),
            filledActionBarSlots);
    }

    void GameCoreSystemComponent::LogTrainerStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source)
    {
        const bool offersChanged =
            previousSession.m_trainer.m_offers.size() != m_worldState.m_session.m_trainer.m_offers.size();
        if (previousSession.m_trainer.m_id == m_worldState.m_session.m_trainer.m_id &&
            previousSession.m_trainer.m_inRange == m_worldState.m_session.m_trainer.m_inRange &&
            !offersChanged)
        {
            return;
        }

        AZ_Printf(
            "amandacore",
            "client.trainer_state_applied source=%s trainerId=%s inRange=%s offers=%zu",
            source,
            m_worldState.m_session.m_trainer.m_id.c_str(),
            m_worldState.m_session.m_trainer.m_inRange ? "true" : "false",
            m_worldState.m_session.m_trainer.m_offers.size());
    }
}
