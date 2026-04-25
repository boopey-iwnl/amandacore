#include <MovementPhysics/ThirdPersonCameraComponent.h>

#include <AzCore/Component/TransformBus.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <AzFramework/Components/TransformComponent.h>
#include <AzFramework/Components/CameraBus.h>
#include <AzFramework/Entity/GameEntityContextBus.h>
#include <Atom/RPI.Public/RPISystemInterface.h>
#include <Atom/Component/DebugCamera/CameraComponent.h>

namespace MovementPhysics
{
    ThirdPersonCameraComponent::ThirdPersonCameraComponent()
    {
    }

    void ThirdPersonCameraComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<ThirdPersonCameraComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void ThirdPersonCameraComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("ThirdPersonCameraService"));
    }

    void ThirdPersonCameraComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("ThirdPersonCameraService"));
    }

    void ThirdPersonCameraComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("TransformService"));
    }

    void ThirdPersonCameraComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void ThirdPersonCameraComponent::Activate()
    {
        EnsureCameraEntity();
    }

    void ThirdPersonCameraComponent::Deactivate()
    {
        if (m_cameraEntity)
        {
            AzFramework::GameEntityContextRequestBus::Broadcast(
                &AzFramework::GameEntityContextRequestBus::Events::DestroyGameEntity,
                m_cameraEntity->GetId());
            m_cameraEntity = nullptr;
        }
    }

    bool ThirdPersonCameraComponent::IsCameraReady() const
    {
        return m_cameraEntity != nullptr;
    }

    AZ::EntityId ThirdPersonCameraComponent::GetCameraEntityId() const
    {
        return m_cameraEntity ? m_cameraEntity->GetId() : AZ::EntityId();
    }

    AZStd::string ThirdPersonCameraComponent::GetCameraEntityName() const
    {
        return m_cameraEntity ? AZStd::string(m_cameraEntity->GetName()) : AZStd::string();
    }

    void ThirdPersonCameraComponent::ApplyLocalCameraTransform(const AZ::Transform& cameraLocalTransform)
    {
        EnsureCameraEntity();
        if (!m_cameraEntity)
        {
            return;
        }

        AZ::TransformBus::Event(m_cameraEntity->GetId(), &AZ::TransformBus::Events::SetLocalTM, cameraLocalTransform);
    }

    void ThirdPersonCameraComponent::EnsureCameraEntity()
    {
        if (m_cameraEntity || !AZ::RPI::RPISystemInterface::Get())
        {
            return;
        }

        AzFramework::GameEntityContextRequestBus::BroadcastResult(
            m_cameraEntity,
            &AzFramework::GameEntityContextRequestBus::Events::CreateGameEntity,
            "LocalPlayerCameraSocket");
        if (!m_cameraEntity)
        {
            AZ_Warning("amandacore", false, "Unable to create LocalPlayerCameraSocket in game entity context");
            return;
        }

        m_cameraEntity->CreateComponent<AzFramework::TransformComponent>();
        m_cameraEntity->CreateComponent<AZ::Debug::CameraComponent>();
        m_cameraEntity->Init();
        AzFramework::GameEntityContextRequestBus::Broadcast(
            &AzFramework::GameEntityContextRequestBus::Events::AddGameEntity,
            m_cameraEntity);
        AzFramework::GameEntityContextRequestBus::Broadcast(
            &AzFramework::GameEntityContextRequestBus::Events::ActivateGameEntity,
            m_cameraEntity->GetId());
        AZ::TransformBus::Event(m_cameraEntity->GetId(), &AZ::TransformBus::Events::SetParent, GetEntityId());

        Camera::CameraRequestBus::Event(m_cameraEntity->GetId(), &Camera::CameraRequestBus::Events::SetFovDegrees, 60.0f);
        Camera::CameraRequestBus::Event(m_cameraEntity->GetId(), &Camera::CameraRequestBus::Events::SetNearClipDistance, 0.05f);
        Camera::CameraRequestBus::Event(m_cameraEntity->GetId(), &Camera::CameraRequestBus::Events::SetFarClipDistance, 500.0f);
        Camera::CameraRequestBus::Event(m_cameraEntity->GetId(), &Camera::CameraRequestBus::Events::MakeActiveView);

        if (!m_loggedActivation)
        {
            m_loggedActivation = true;
            AZ_Printf(
                "amandacore",
                "client.camera_activated entity=LocalPlayerCameraSocket active=true");
        }
    }
} // namespace MovementPhysics
