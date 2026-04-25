#include <StatsProgression/StatsProgressionSystemComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace StatsProgression
{
    class StatsProgressionModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(StatsProgressionModule, "{DF520A8E-8107-4623-9C60-8971423EAA09}", AZ::Module);
        AZ_CLASS_ALLOCATOR(StatsProgressionModule, AZ::SystemAllocator);

        StatsProgressionModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                StatsProgressionSystemComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<StatsProgressionSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_StatsProgression, StatsProgression::StatsProgressionModule)
