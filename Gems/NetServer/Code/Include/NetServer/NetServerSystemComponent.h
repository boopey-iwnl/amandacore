#pragma once

#include <AzCore/Component/Component.h>

namespace NetServer
{
    class NetServerSystemComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(NetServerSystemComponent, "{DF77A3D7-2B08-4C22-8B47-06ECAAA8665D}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
    };
}
