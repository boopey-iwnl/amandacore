#include <InventoryLoot/InventoryLootSystemComponent.h>

#include <AzCore/Serialization/SerializeContext.h>

namespace InventoryLoot
{
    void InventoryLootSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<InventoryLootSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void InventoryLootSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("InventoryLootService"));
    }

    void InventoryLootSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("InventoryLootService"));
    }

    void InventoryLootSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
        required.push_back(AZ_CRC_CE("StatsProgressionService"));
    }

    void InventoryLootSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void InventoryLootSystemComponent::Activate()
    {
    }

    void InventoryLootSystemComponent::Deactivate()
    {
    }
}
