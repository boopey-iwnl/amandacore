#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/Component/EntityId.h>
#include <AzCore/Math/Transform.h>
#include <AzCore/std/string/string.h>

namespace MovementPhysics
{
    class ThirdPersonCameraComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(ThirdPersonCameraComponent, "{484FB3A0-7E95-4E05-82E2-C52F6D955154}");

        ThirdPersonCameraComponent();
        ~ThirdPersonCameraComponent() override = default;

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;

        bool IsCameraReady() const;
        AZ::EntityId GetCameraEntityId() const;
        AZStd::string GetCameraEntityName() const;
        void ApplyLocalCameraTransform(const AZ::Transform& cameraLocalTransform);

    private:
        void EnsureCameraEntity();

        AZ::Entity* m_cameraEntity = nullptr;
        bool m_loggedActivation = false;
    };
} // namespace MovementPhysics
