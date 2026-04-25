#pragma once

#include <AzCore/Component/Component.h>

namespace MovementPhysics
{
    class MovementPhysicsSystemComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(MovementPhysicsSystemComponent, "{A8C41A83-95A2-4DCB-94D4-BEFCC17D55CE}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;

    private:
        AZ::Entity* m_localPlayerEntity = nullptr;
    };
}
