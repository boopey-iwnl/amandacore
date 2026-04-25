#pragma once

#include <AzCore/Math/MathUtils.h>
#include <AzCore/Math/Vector3.h>
#include <AzCore/std/string/string.h>
#include <AzCore/std/string/string_view.h>

namespace GameCore
{
    namespace MobProxyGeometry
    {
        constexpr float BodyHeight = 1.0f;
        constexpr float BodyRadius = 0.78f;
        constexpr float HeadHeight = 1.95f;
        constexpr float HeadRadius = 0.34f;
        constexpr float BaseRingRadius = 1.05f;
        constexpr float BaseRingSphereRadius = 0.09f;
        constexpr int BaseRingSegments = 16;
        constexpr float InstancePipHeight = 3.05f;
        constexpr float InstancePipSpacing = 0.30f;
        constexpr float InstancePipRadius = 0.13f;
        constexpr float TwoPi = 6.28318530717958647692f;

        inline int GetMobOrdinal(const AZStd::string& mobId)
        {
            const size_t separatorIndex = mobId.find_last_of('_');
            if (separatorIndex == AZStd::string::npos || separatorIndex + 1 >= mobId.size())
            {
                return 0;
            }

            const AZStd::string_view suffix(mobId.data() + separatorIndex + 1, mobId.size() - separatorIndex - 1);
            if (suffix == "01")
            {
                return 1;
            }
            if (suffix == "02")
            {
                return 2;
            }
            if (suffix == "03")
            {
                return 3;
            }

            return 0;
        }

        template<class Callback>
        inline void VisitProxySpheres(const AZStd::string& mobId, bool alive, Callback&& callback)
        {
            callback(AZ::Vector3(0.0f, 0.0f, BodyHeight), BodyRadius);
            callback(AZ::Vector3(0.0f, 0.0f, HeadHeight), HeadRadius);

            if (!alive)
            {
                return;
            }

            for (int segmentIndex = 0; segmentIndex < BaseRingSegments; ++segmentIndex)
            {
                const float angleRadians = (TwoPi / static_cast<float>(BaseRingSegments)) *
                    static_cast<float>(segmentIndex);
                callback(
                    AZ::Vector3(
                        AZStd::cos(angleRadians) * BaseRingRadius,
                        AZStd::sin(angleRadians) * BaseRingRadius,
                        0.10f),
                    BaseRingSphereRadius);
            }

            const int mobOrdinal = GetMobOrdinal(mobId);
            if (mobOrdinal <= 0)
            {
                return;
            }

            const float centerOffset = (static_cast<float>(mobOrdinal - 1) * InstancePipSpacing) * 0.5f;
            for (int pipIndex = 0; pipIndex < mobOrdinal; ++pipIndex)
            {
                const float xOffset = (static_cast<float>(pipIndex) * InstancePipSpacing) - centerOffset;
                callback(AZ::Vector3(xOffset, 0.0f, InstancePipHeight), InstancePipRadius);
            }
        }

        template<class Callback>
        inline void VisitWorldProxySpheres(
            const AZStd::string& mobId,
            const AZ::Vector3& worldPosition,
            bool alive,
            Callback&& callback)
        {
            VisitProxySpheres(
                mobId,
                alive,
                [&worldPosition, &callback](const AZ::Vector3& localOffset, float radius)
                {
                    callback(worldPosition + localOffset, radius);
                });
        }
    } // namespace MobProxyGeometry
} // namespace GameCore
