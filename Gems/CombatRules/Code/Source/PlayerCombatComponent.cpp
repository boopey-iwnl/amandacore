#include <CombatRules/PlayerCombatComponent.h>

#include <AzCore/Serialization/SerializeContext.h>
#include <AzFramework/Input/Channels/InputChannel.h>
#include <AzFramework/Input/Devices/Keyboard/InputDeviceKeyboard.h>
#include <GameCore/GameCoreInterface.h>

namespace CombatRules
{
    namespace
    {
        const NetClient::ActionBarSlotState* FindActionBarSlot(
            const NetClient::WorldSessionResponse& session,
            int slotIndex)
        {
            for (const auto& slot : session.m_actionBarSlots)
            {
                if (slot.m_slotIndex == slotIndex)
                {
                    return &slot;
                }
            }

            return nullptr;
        }

        bool ActivateActionBarSlot(GameCore::IGameCoreRequests* gameCore, int slotIndex)
        {
            if (!gameCore)
            {
                return false;
            }

            if (const auto* slot = FindActionBarSlot(gameCore->GetClientWorldState().m_session, slotIndex);
                slot && !slot->m_abilityId.empty())
            {
                const bool activated = slot->m_abilityId == "auto_attack"
                    ? gameCore->SetAutoAttack(!gameCore->GetClientWorldState().m_session.m_autoAttackActive)
                    : gameCore->ActivateAbility(slot->m_abilityId);
                if (activated)
                {
                    AZ_Printf("amandacore", "client.ability_requested abilityId=%s slot=%d", slot->m_abilityId.c_str(), slotIndex);
                    return true;
                }
            }

            return false;
        }
    }

    PlayerCombatComponent::PlayerCombatComponent()
        : AzFramework::InputChannelEventListener(AzFramework::InputChannelEventListener::GetPriorityFirst())
    {
    }

    void PlayerCombatComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<PlayerCombatComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void PlayerCombatComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("PlayerCombatService"));
    }

    void PlayerCombatComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("PlayerCombatService"));
    }

    void PlayerCombatComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void PlayerCombatComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void PlayerCombatComponent::Activate()
    {
        AzFramework::InputChannelEventListener::Connect();
        AZ::TickBus::Handler::BusConnect();
    }

    void PlayerCombatComponent::Deactivate()
    {
        AZ::TickBus::Handler::BusDisconnect();
        AzFramework::InputChannelEventListener::Disconnect();
    }

    void PlayerCombatComponent::OnTick(float, AZ::ScriptTimePoint)
    {
        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return;
        }

        const auto& worldState = gameCore->GetClientWorldState();
        if (!worldState.m_worldConnected)
        {
            return;
        }

        if (!m_loggedReady)
        {
            m_loggedReady = true;
            AZ_Printf("amandacore", "client.combat_ready entity=LocalPlayer");
        }
        if (!m_loggedHelp)
        {
            m_loggedHelp = true;
            AZ_Printf("amandacore", "client.combat_input_help autoAttack=F strike=1 defense=2 trainerStrike=3 spellbook=P trainer=right_click_npc");
        }

        if (worldState.m_session.m_autoAttackActive != m_lastAutoAttackState)
        {
            if (worldState.m_session.m_autoAttackActive)
            {
                AZ_Printf("amandacore", "client.auto_attack_started targetId=%s", worldState.m_session.m_currentTargetId.c_str());
            }
            else
            {
                AZ_Printf("amandacore", "client.auto_attack_stopped");
            }
            m_lastAutoAttackState = worldState.m_session.m_autoAttackActive;
        }

        if (worldState.m_session.m_castingAbilityId != m_lastCastingAbilityId)
        {
            if (!worldState.m_session.m_castingAbilityId.empty())
            {
                AZ_Printf(
                    "amandacore",
                    "client.cast_started abilityId=%s castEndsAt=%lld",
                    worldState.m_session.m_castingAbilityId.c_str(),
                    static_cast<long long>(worldState.m_session.m_castEndsAt));
            }
            else if (!m_lastCastingAbilityId.empty())
            {
                AZ_Printf("amandacore", "client.cast_completed");
            }
            m_lastCastingAbilityId = worldState.m_session.m_castingAbilityId;
        }
    }

    bool PlayerCombatComponent::OnInputChannelEventFiltered(const AzFramework::InputChannel& inputChannel)
    {
        if (!inputChannel.IsStateBegan())
        {
            return false;
        }

        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return false;
        }

        // Keyboard action-slot activation is owned by UiClient so local keybinding
        // changes are respected consistently across all four action bars.
        return false;
    }
} // namespace CombatRules
