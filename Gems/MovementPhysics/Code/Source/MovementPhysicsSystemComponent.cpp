#include <MovementPhysics/MovementPhysicsSystemComponent.h>

#include <AzCore/Debug/Trace.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <AzFramework/Components/TransformComponent.h>
#include <AzFramework/Entity/GameEntityContextBus.h>
#include <AzFramework/Physics/CharacterBus.h>
#include <CombatRules/PlayerCombatComponent.h>
#include <CombatRules/PlayerTargetingComponent.h>
#include <MovementPhysics/LocalPlayerControllerComponent.h>
#include <PhysX/CharacterControllerBus.h>
#include <PhysX/CharacterGameplayBus.h>
#include <PhysXCharacters/Components/CharacterControllerComponent.h>
#include <PhysXCharacters/Components/CharacterGameplayComponent.h>

namespace MovementPhysics
{
    void MovementPhysicsSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<MovementPhysicsSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void MovementPhysicsSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("MovementPhysicsService"));
    }

    void MovementPhysicsSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("MovementPhysicsService"));
    }

    void MovementPhysicsSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
    }

    void MovementPhysicsSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void MovementPhysicsSystemComponent::Activate()
    {
        AzFramework::GameEntityContextRequestBus::BroadcastResult(
            m_localPlayerEntity,
            &AzFramework::GameEntityContextRequestBus::Events::CreateGameEntity,
            "LocalPlayer");
        if (!m_localPlayerEntity)
        {
            AZ_Warning("amandacore", false, "Unable to create LocalPlayer in game entity context");
            return;
        }

        auto* transformComponent = m_localPlayerEntity->CreateComponent<AzFramework::TransformComponent>();
        auto* characterController = m_localPlayerEntity->CreateComponent<PhysX::CharacterControllerComponent>();
        auto* characterGameplay = m_localPlayerEntity->CreateComponent<PhysX::CharacterGameplayComponent>();
        auto* localPlayerController = m_localPlayerEntity->CreateComponent<LocalPlayerControllerComponent>();
        auto* targetingComponent = m_localPlayerEntity->CreateComponent<CombatRules::PlayerTargetingComponent>();
        auto* combatComponent = m_localPlayerEntity->CreateComponent<CombatRules::PlayerCombatComponent>();
        m_localPlayerEntity->Init();
        AzFramework::GameEntityContextRequestBus::Broadcast(
            &AzFramework::GameEntityContextRequestBus::Events::AddGameEntity,
            m_localPlayerEntity);
        AzFramework::GameEntityContextRequestBus::Broadcast(
            &AzFramework::GameEntityContextRequestBus::Events::ActivateGameEntity,
            m_localPlayerEntity->GetId());

        Physics::CharacterRequestBus::Event(
            m_localPlayerEntity->GetId(),
            &Physics::CharacterRequestBus::Events::SetMaximumSpeed,
            6.0f);
        Physics::CharacterRequestBus::Event(
            m_localPlayerEntity->GetId(),
            &Physics::CharacterRequestBus::Events::SetStepHeight,
            0.35f);
        Physics::CharacterRequestBus::Event(
            m_localPlayerEntity->GetId(),
            &Physics::CharacterRequestBus::Events::SetSlopeLimitDegrees,
            45.0f);
        PhysX::CharacterControllerRequestBus::Event(
            m_localPlayerEntity->GetId(),
            &PhysX::CharacterControllerRequestBus::Events::SetHeight,
            1.8f);
        PhysX::CharacterControllerRequestBus::Event(
            m_localPlayerEntity->GetId(),
            &PhysX::CharacterControllerRequestBus::Events::SetRadius,
            0.4f);
        PhysX::CharacterGameplayRequestBus::Event(
            m_localPlayerEntity->GetId(),
            &PhysX::CharacterGameplayRequestBus::Events::SetGravityMultiplier,
            1.0f);

        AZ_Printf(
            "amandacore",
            "MovementPhysics spawned local player entity transform=%s physxController=%s physxGameplay=%s controller=%s camera=%s targeting=%s combat=%s",
            transformComponent ? "true" : "false",
            characterController ? "true" : "false",
            characterGameplay ? "true" : "false",
            localPlayerController ? "true" : "false",
            "false",
            targetingComponent ? "true" : "false",
            combatComponent ? "true" : "false");
    }

    void MovementPhysicsSystemComponent::Deactivate()
    {
        if (m_localPlayerEntity)
        {
            AzFramework::GameEntityContextRequestBus::Broadcast(
                &AzFramework::GameEntityContextRequestBus::Events::DestroyGameEntity,
                m_localPlayerEntity->GetId());
            m_localPlayerEntity = nullptr;
        }
    }
}
