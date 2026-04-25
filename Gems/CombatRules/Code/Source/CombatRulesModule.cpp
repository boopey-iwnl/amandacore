#include <CombatRules/CombatRulesSystemComponent.h>
#include <CombatRules/PlayerCombatComponent.h>
#include <CombatRules/PlayerTargetingComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace CombatRules
{
    class CombatRulesModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(CombatRulesModule, "{2BB9870A-96DA-47A1-85D0-F341CF53B2C5}", AZ::Module);
        AZ_CLASS_ALLOCATOR(CombatRulesModule, AZ::SystemAllocator);

        CombatRulesModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                CombatRulesSystemComponent::CreateDescriptor(),
                PlayerTargetingComponent::CreateDescriptor(),
                PlayerCombatComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<CombatRulesSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_CombatRules, CombatRules::CombatRulesModule)
