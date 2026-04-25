#pragma once

#include <AzCore/Component/Component.h>

namespace ZoneStreaming
{
    class ZoneStreamingSystemComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(ZoneStreamingSystemComponent, "{54B217C3-527A-40F0-B0D0-8FA7A3232A3B}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
    };
}
