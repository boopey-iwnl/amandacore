#include <ZoneStreaming/ZoneStreamingSystemComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace ZoneStreaming
{
    class ZoneStreamingModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(ZoneStreamingModule, "{1155920F-7C40-46F8-9F78-9A99FE051033}", AZ::Module);
        AZ_CLASS_ALLOCATOR(ZoneStreamingModule, AZ::SystemAllocator);

        ZoneStreamingModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                ZoneStreamingSystemComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<ZoneStreamingSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_ZoneStreaming, ZoneStreaming::ZoneStreamingModule)
