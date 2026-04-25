#include <NpcAi/NpcAiSystemComponent.h>
#include <NpcAi/MobCombatStateComponent.h>
#include <NpcAi/MobPresentationComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace NpcAi
{
    class NpcAiModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(NpcAiModule, "{96229752-E9F6-4305-BD2E-B749E87E6F1D}", AZ::Module);
        AZ_CLASS_ALLOCATOR(NpcAiModule, AZ::SystemAllocator);

        NpcAiModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                NpcAiSystemComponent::CreateDescriptor(),
                MobCombatStateComponent::CreateDescriptor(),
                MobPresentationComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<NpcAiSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_NpcAi, NpcAi::NpcAiModule)
