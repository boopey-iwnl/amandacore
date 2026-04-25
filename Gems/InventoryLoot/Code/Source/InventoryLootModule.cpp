#include <InventoryLoot/InventoryLootSystemComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace InventoryLoot
{
    class InventoryLootModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(InventoryLootModule, "{9C5DFE11-6A7D-4379-8F38-63595AAE7ADF}", AZ::Module);
        AZ_CLASS_ALLOCATOR(InventoryLootModule, AZ::SystemAllocator);

        InventoryLootModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                InventoryLootSystemComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<InventoryLootSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_InventoryLoot, InventoryLoot::InventoryLootModule)
