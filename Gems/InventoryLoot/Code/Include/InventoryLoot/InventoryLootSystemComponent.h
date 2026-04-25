#pragma once

#include <AzCore/Component/Component.h>

namespace InventoryLoot
{
    class InventoryLootSystemComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(InventoryLootSystemComponent, "{8241D00B-8439-4D97-B30B-3F94727B40D5}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
    };
}
