#include <NetClient/NetClientSystemComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace NetClient
{
    class NetClientModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(NetClientModule, "{F575D2BD-9052-423D-B6D5-2330CBAC2BC9}", AZ::Module);
        AZ_CLASS_ALLOCATOR(NetClientModule, AZ::SystemAllocator);

        NetClientModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                NetClientSystemComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<NetClientSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_NetClient, NetClient::NetClientModule)
