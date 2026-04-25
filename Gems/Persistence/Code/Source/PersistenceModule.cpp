#include <Persistence/PersistenceSystemComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace Persistence
{
    class PersistenceModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(PersistenceModule, "{19AA5B15-BF9C-458F-B67F-0879DC0BD912}", AZ::Module);
        AZ_CLASS_ALLOCATOR(PersistenceModule, AZ::SystemAllocator);

        PersistenceModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                PersistenceSystemComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<PersistenceSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_Persistence, Persistence::PersistenceModule)
