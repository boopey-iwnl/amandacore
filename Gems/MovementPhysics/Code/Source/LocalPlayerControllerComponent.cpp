#include <MovementPhysics/LocalPlayerControllerComponent.h>

#include <Atom/RPI.Public/AuxGeom/AuxGeomDraw.h>
#include <Atom/RPI.Public/AuxGeom/AuxGeomFeatureProcessorInterface.h>
#include <Atom/RPI.Public/Scene.h>
#include <AzCore/Component/TransformBus.h>
#include <AzCore/Component/Entity.h>
#include <AzCore/Interface/Interface.h>
#include <AzCore/Math/Aabb.h>
#include <AzCore/Math/Color.h>
#include <AzCore/Math/MathUtils.h>
#include <AzCore/Math/Transform.h>
#include <AzCore/Math/Vector2.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <AzFramework/Physics/Common/PhysicsSceneQueries.h>
#include <AzFramework/Physics/PhysicsScene.h>
#include <AzFramework/API/ApplicationAPI.h>
#include <AzFramework/Input/Channels/InputChannel.h>
#include <AzFramework/Input/Devices/Keyboard/InputDeviceKeyboard.h>
#include <AzFramework/Input/Devices/Mouse/InputDeviceMouse.h>
#include <AzFramework/Physics/CharacterBus.h>
#include <AzFramework/Physics/SystemBus.h>
#include <AzFramework/Viewport/ViewportBus.h>
#include <Atom/RPI.Public/ViewportContext.h>
#include <Atom/RPI.Public/ViewportContextBus.h>
#include <GameCore/GameCoreInterface.h>
#include <PhysX/CharacterGameplayBus.h>
#include <imgui/imgui.h>

namespace MovementPhysics
{
    namespace
    {
        constexpr float ValidationFloorZ = 0.0f;
        constexpr float ValidationFloorExtent = 480.0f;
        constexpr float ValidationFloorExtentY = 300.0f;
        constexpr float ValidationMarkerZ = 0.08f;
        constexpr float ValidationSpawnMarkerRadius = 0.18f;
        constexpr float ValidationSpawnX = 10.0f;
        constexpr float ValidationSpawnY = 10.0f;
        constexpr float EncounterAnchorX = 322.0f;
        constexpr float EncounterAnchorY = 174.0f;
        constexpr float MoveSpeedUnitsPerSecond = 6.0f;
        constexpr float BackpedalSpeedFactor = 0.62f;
        constexpr float SubmitIntervalSeconds = 0.10f;
        constexpr float CorrectionSnapDistance = 1.25f;
        constexpr float CorrectionBlendRate = 5.25f;
        constexpr float CorrectionDeadZoneDistance = 0.045f;
        constexpr float CorrectionEpsilon = 0.002f;
        constexpr float CharacterBaseSnapZ = 0.05f;
        constexpr float AvatarTurnRate = 2.75f;
        constexpr float CameraEncounterAnchorX = 34.0f;
        constexpr float CameraEncounterAnchorY = 18.0f;
        constexpr float CameraPivotOffsetZ = 1.42f;
        constexpr float CameraLookAheadDistance = 0.82f;
        constexpr float CameraLookLiftZ = 0.16f;
        constexpr float CameraYawSensitivity = 0.01f;
        constexpr float CameraPitchSensitivity = 0.01f;
        constexpr float CameraDefaultFollowDistance = 4.65f;
        constexpr float CameraMinFollowDistance = 4.2f;
        constexpr float CameraMaxFollowDistance = 6.0f;
        constexpr float CameraDefaultPitchRadians = -0.44f;
        constexpr float CameraMinPitchRadians = -0.95f;
        constexpr float CameraMaxPitchRadians = -0.26f;
        constexpr float CameraFloorFallbackZ = ValidationFloorZ + 0.30f;
        constexpr float CameraCollisionSafetyDistance = 0.18f;
        constexpr float CameraMinimumResolvedDistance = 1.10f;
        constexpr float AvatarFootHeight = 0.08f;
        constexpr float AvatarAnkleHeight = 0.30f;
        constexpr float AvatarKneeHeight = 0.70f;
        constexpr float AvatarHipHeight = 1.00f;
        constexpr float AvatarWaistHeight = 1.16f;
        constexpr float AvatarChestHeight = 1.40f;
        constexpr float AvatarShoulderHeight = 1.54f;
        constexpr float AvatarHeadHeight = 1.82f;
        constexpr float AvatarFootSpacing = 0.18f;
        constexpr float AvatarHipSpacing = 0.12f;
        constexpr float AvatarShoulderOffset = 0.27f;
        constexpr float AvatarPelvisRadius = 0.18f;
        constexpr float AvatarTorsoRadius = 0.23f;
        constexpr float AvatarChestRadius = 0.28f;
        constexpr float AvatarHeadRadius = 0.17f;
        constexpr float AvatarLimbRadius = 0.10f;
        constexpr float AvatarArmRadius = 0.09f;
        constexpr float AvatarSelectionRingRadius = 0.68f;
        constexpr float AvatarSelectionRingSphereRadius = 0.06f;
        constexpr int AvatarSelectionRingSegments = 12;
        constexpr float AvatarStrideRate = 8.5f;
        constexpr float AvatarStrideReach = 0.22f;
        constexpr float AvatarIdleTurnRate = 4.6f;
        constexpr float AvatarMoveTurnRate = 10.0f;
        constexpr float AvatarStrideBlendRate = 7.5f;
        constexpr float PoseLogDistanceThreshold = 0.75f;

        bool IsInsideWestApproachObstacle(const AZ::Vector3& position)
        {
            return position.GetX() >= 72.0f && position.GetX() <= 80.0f &&
                position.GetY() >= 28.0f && position.GetY() <= 46.0f;
        }

        AZ::Vector3 ResolveMovementCollision(const AZ::Vector3& currentPosition, const AZ::Vector3& requestedPosition)
        {
            AZ::Vector3 clampedPosition = requestedPosition;
            clampedPosition.SetX(AZ::GetClamp(clampedPosition.GetX(), 0.0f, ValidationFloorExtent));
            clampedPosition.SetY(AZ::GetClamp(clampedPosition.GetY(), 0.0f, ValidationFloorExtentY));
            return IsInsideWestApproachObstacle(clampedPosition) ? currentPosition : clampedPosition;
        }

        float WrapAngleRadians(float angleRadians)
        {
            while (angleRadians > AZ::Constants::Pi)
            {
                angleRadians -= AZ::Constants::TwoPi;
            }
            while (angleRadians < -AZ::Constants::Pi)
            {
                angleRadians += AZ::Constants::TwoPi;
            }
            return angleRadians;
        }

        float StepAngleTowards(float currentRadians, float targetRadians, float maxStepRadians)
        {
            const float deltaRadians = WrapAngleRadians(targetRadians - currentRadians);
            return WrapAngleRadians(currentRadians + AZ::GetClamp(deltaRadians, -maxStepRadians, maxStepRadians));
        }

        AZ::Vector3 ForwardFromYaw(float yawRadians)
        {
            return AZ::Vector3(AZStd::cos(yawRadians), AZStd::sin(yawRadians), 0.0f);
        }

        AZ::Vector3 RightFromYaw(float yawRadians)
        {
            return AZ::Vector3(-AZStd::sin(yawRadians), AZStd::cos(yawRadians), 0.0f);
        }

        AZ::Vector3 BuildCameraForwardFromYaw(float yawRadians)
        {
            return AZ::Vector3(AZStd::cos(yawRadians), AZStd::sin(yawRadians), 0.0f);
        }
    } // namespace

    LocalPlayerControllerComponent::LocalPlayerControllerComponent()
        : AzFramework::InputChannelEventListener(AzFramework::InputChannelEventListener::GetPriorityFirst())
    {
    }

    void LocalPlayerControllerComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<LocalPlayerControllerComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void LocalPlayerControllerComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("LocalPlayerControllerService"));
    }

    void LocalPlayerControllerComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("LocalPlayerControllerService"));
    }

    void LocalPlayerControllerComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("TransformService"));
    }

    void LocalPlayerControllerComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void LocalPlayerControllerComponent::Activate()
    {
        AzFramework::InputChannelEventListener::Connect();
        m_inputListening = true;
        AZ::TickBus::Handler::BusConnect();
    }

    void LocalPlayerControllerComponent::Deactivate()
    {
        AZ::TickBus::Handler::BusDisconnect();
        AzFramework::InputChannelEventListener::Disconnect();
    }

    void LocalPlayerControllerComponent::OnTick(float deltaTime, AZ::ScriptTimePoint)
    {
        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return;
        }

        InitializeFromWorldState();
        UpdateGroundingState();

        if (m_requestDisconnect)
        {
            AZ_Printf("amandacore", "client.disconnect_requested");
            gameCore->DisconnectWorld();
            m_requestDisconnect = false;
            m_pendingServerDelta = AZ::Vector2::CreateZero();
            m_pendingServerCorrection = AZ::Vector2::CreateZero();
            m_cachedFinalPoseValid = false;
            return;
        }

        if (m_requestReconnect)
        {
            AZ_Printf("amandacore", "client.reconnect_requested");
            if (gameCore->ReconnectWorld())
            {
                const auto& session = gameCore->GetClientWorldState().m_session;
                ApplyWorldPosition(
                    static_cast<float>(session.m_position.m_x),
                    static_cast<float>(session.m_position.m_y),
                    static_cast<float>(session.m_position.m_z));
                AZ_Printf(
                    "amandacore",
                    "client.reconnect_completed token=%s position=(%.3f, %.3f, %.3f)",
                    session.m_worldSessionToken.c_str(),
                    session.m_position.m_x,
                    session.m_position.m_y,
                    session.m_position.m_z);
                m_pendingCameraReset = true;
                m_avatarStridePhase = 0.0f;
                m_avatarStrideBlend = 0.0f;
            }
            m_requestReconnect = false;
            m_pendingServerDelta = AZ::Vector2::CreateZero();
            m_pendingServerCorrection = AZ::Vector2::CreateZero();
            return;
        }

        if (m_requestQuit)
        {
            AZ_Printf("amandacore", "client.quit_requested");
            AzFramework::ApplicationRequests::Bus::Broadcast(
                &AzFramework::ApplicationRequests::Bus::Events::ExitMainLoop);
            m_requestQuit = false;
            return;
        }

        const auto& worldState = gameCore->GetClientWorldState();
        if (!worldState.m_worldConnected)
        {
            return;
        }

        if (worldState.m_session.m_worldSessionToken != m_lastWorldSessionToken)
        {
            m_lastWorldSessionToken = worldState.m_session.m_worldSessionToken;
            m_pendingServerDelta = AZ::Vector2::CreateZero();
            m_pendingServerCorrection = AZ::Vector2::CreateZero();
            ApplyWorldPosition(
                static_cast<float>(worldState.m_session.m_position.m_x),
                static_cast<float>(worldState.m_session.m_position.m_y),
                static_cast<float>(worldState.m_session.m_position.m_z));
            m_pendingCameraReset = true;
            m_avatarStridePhase = 0.0f;
            m_avatarStrideBlend = 0.0f;
        }

        const bool orbiting = m_cameraOrbitModeActive;
        const bool dualMouseMoveActive = orbiting && m_leftMouseHeld;
        float cameraYawRadians = orbiting
            ? WrapAngleRadians(m_avatarFacingRadians + m_cameraOrbitYawRadians)
            : m_avatarFacingRadians;
        m_cameraYawRadians = cameraYawRadians;

        const float turnInput = (m_strafeLeft ? 1.0f : 0.0f) - (m_strafeRight ? 1.0f : 0.0f);
        AZ::Vector2 movementIntent = AZ::Vector2::CreateZero();
        if (m_moveForward)
        {
            movementIntent += AZ::Vector2(1.0f, 0.0f);
        }
        if (m_moveBackward)
        {
            movementIntent += AZ::Vector2(-1.0f, 0.0f);
        }
        if (dualMouseMoveActive)
        {
            movementIntent += AZ::Vector2(1.0f, 0.0f);
        }

        if (orbiting)
        {
            if (m_strafeLeft)
            {
                movementIntent += AZ::Vector2(0.0f, 1.0f);
            }
            if (m_strafeRight)
            {
                movementIntent += AZ::Vector2(0.0f, -1.0f);
            }

            if (movementIntent.GetLengthSq() > 0.0f)
            {
                m_avatarFacingRadians = cameraYawRadians;
                m_cameraOrbitYawRadians = 0.0f;
                cameraYawRadians = m_avatarFacingRadians;
                m_cameraYawRadians = cameraYawRadians;
                m_chaseLockActive = true;
            }
            else
            {
                m_chaseLockActive = false;
            }
        }
        else if (AZ::GetAbs(turnInput) > 0.001f)
        {
            m_avatarFacingRadians = WrapAngleRadians(m_avatarFacingRadians + (turnInput * AvatarTurnRate * deltaTime));
            cameraYawRadians = m_avatarFacingRadians;
            m_cameraYawRadians = cameraYawRadians;
            m_chaseLockActive = true;
        }
        else if (!orbiting)
        {
            m_chaseLockActive = true;
        }

        if (!orbiting)
        {
            m_cameraOrbitYawRadians = 0.0f;
            cameraYawRadians = m_avatarFacingRadians;
            m_cameraYawRadians = cameraYawRadians;
        }

        const AZ::Vector3 groundedPosition = GetPresentationPosition();
        AZ::Vector3 currentPosition = groundedPosition;
        if (m_cachedFinalPoseValid)
        {
            currentPosition.SetX(m_cachedFinalPresentationPosition.GetX());
            currentPosition.SetY(m_cachedFinalPresentationPosition.GetY());
        }

        AZ::Vector3 finalPresentationPosition = currentPosition;
        AZ::Vector2 visualPlanarDelta = AZ::Vector2::CreateZero();

        if (movementIntent.GetLengthSq() > 0.0f)
        {
            if (!m_loggedFirstMovementInput)
            {
                m_loggedFirstMovementInput = true;
                AZ_Printf("amandacore", "client.first_movement_input_received");
            }

            AZ::Vector2 move2d = AZ::Vector2::CreateZero();
            if (orbiting)
            {
                movementIntent.Normalize();
                const AZ::Vector2 forward(AZStd::cos(cameraYawRadians), AZStd::sin(cameraYawRadians));
                const AZ::Vector2 right(-AZStd::sin(cameraYawRadians), AZStd::cos(cameraYawRadians));
                move2d = ((forward * movementIntent.GetX()) + (right * movementIntent.GetY())) *
                    (MoveSpeedUnitsPerSecond * deltaTime);
            }
            else
            {
                const float moveIntent = movementIntent.GetX();
                const AZ::Vector2 facingForward(AZStd::cos(m_avatarFacingRadians), AZStd::sin(m_avatarFacingRadians));
                const float moveSpeed = moveIntent > 0.0f ? MoveSpeedUnitsPerSecond : (MoveSpeedUnitsPerSecond * BackpedalSpeedFactor);
                move2d = facingForward * (moveIntent * moveSpeed * deltaTime);
            }

            AZ::Vector3 requestedPosition = currentPosition + AZ::Vector3(move2d.GetX(), move2d.GetY(), 0.0f);
            requestedPosition = ResolveMovementCollision(currentPosition, requestedPosition);
            const AZ::Vector3 appliedDelta = requestedPosition - currentPosition;
            visualPlanarDelta = AZ::Vector2(appliedDelta.GetX(), appliedDelta.GetY());
            m_pendingServerDelta += visualPlanarDelta;
            finalPresentationPosition = requestedPosition;
        }

        if (m_pendingServerCorrection.GetLengthSq() > CorrectionEpsilon * CorrectionEpsilon)
        {
            const float correctionBlend = AZ::GetClamp(deltaTime * CorrectionBlendRate, 0.0f, 1.0f);
            const AZ::Vector2 correctionStep = m_pendingServerCorrection * correctionBlend;
            finalPresentationPosition.SetX(finalPresentationPosition.GetX() + correctionStep.GetX());
            finalPresentationPosition.SetY(finalPresentationPosition.GetY() + correctionStep.GetY());
            m_pendingServerCorrection -= correctionStep;
            if (m_pendingServerCorrection.GetLengthSq() <= CorrectionEpsilon * CorrectionEpsilon)
            {
                m_pendingServerCorrection = AZ::Vector2::CreateZero();
            }
        }

        m_submitAccumulator += deltaTime;
        if (m_submitAccumulator >= SubmitIntervalSeconds && m_pendingServerDelta.GetLengthSq() >= 0.0001f)
        {
            m_submitAccumulator = AZStd::fmod(m_submitAccumulator, SubmitIntervalSeconds);
            const AZ::Vector2 submittedDelta = m_pendingServerDelta;
            m_pendingServerDelta = AZ::Vector2::CreateZero();

            if (gameCore->SubmitMove(submittedDelta.GetX(), submittedDelta.GetY()))
            {
                const auto& authoritativeSession = gameCore->GetClientWorldState().m_session;
                const AZ::Vector2 authoritativePosition(
                    static_cast<float>(authoritativeSession.m_position.m_x),
                    static_cast<float>(authoritativeSession.m_position.m_y));

                const AZ::Vector3 locallyPredictedPosition = finalPresentationPosition;
                const AZ::Vector2 localPlanarPosition(locallyPredictedPosition.GetX(), locallyPredictedPosition.GetY());
                const AZ::Vector2 correctionVector = authoritativePosition - localPlanarPosition;
                const float correctionDistance = correctionVector.GetLength();
                if (correctionDistance > CorrectionSnapDistance)
                {
                    finalPresentationPosition.SetX(authoritativePosition.GetX());
                    finalPresentationPosition.SetY(authoritativePosition.GetY());
                    finalPresentationPosition.SetZ(locallyPredictedPosition.GetZ());
                    m_pendingServerCorrection = AZ::Vector2::CreateZero();
                    SetCharacterBasePosition(finalPresentationPosition);
                    AZ_Printf(
                        "amandacore",
                        "client.planar_reconciliation_applied localXY=(%.3f, %.3f) authoritativeXY=(%.3f, %.3f) mode=snap",
                        localPlanarPosition.GetX(),
                        localPlanarPosition.GetY(),
                        authoritativePosition.GetX(),
                        authoritativePosition.GetY());
                }
                else if (correctionDistance > CorrectionDeadZoneDistance)
                {
                    m_pendingServerCorrection = correctionVector;
                    AZ_Printf(
                        "amandacore",
                        "client.planar_reconciliation_applied localXY=(%.3f, %.3f) authoritativeXY=(%.3f, %.3f) mode=queued_blend",
                        localPlanarPosition.GetX(),
                        localPlanarPosition.GetY(),
                        authoritativePosition.GetX(),
                        authoritativePosition.GetY());
                }
                else
                {
                    m_pendingServerCorrection = AZ::Vector2::CreateZero();
                }
            }
        }

        m_cachedFinalPresentationPosition = finalPresentationPosition;
        m_cachedFinalAvatarFacingRadians = m_avatarFacingRadians;
        m_cachedFinalPoseValid = true;

        if (visualPlanarDelta.GetLengthSq() > 0.0001f)
        {
            if (!m_loggedMovementTranslationApplied)
            {
                m_loggedMovementTranslationApplied = true;
                AZ_Printf(
                    "amandacore",
                    "client.movement_translation_applied delta=(%.3f, %.3f, %.3f)",
                    visualPlanarDelta.GetX(),
                    visualPlanarDelta.GetY(),
                    0.0f);
            }
        }
        else
        {
            m_loggedMovementTranslationApplied = false;
        }

        const float finalBodyPoseDistance = m_loggedFinalBodyPose
            ? (m_cachedFinalPresentationPosition - m_lastLoggedFinalBodyPose).GetLength()
            : PoseLogDistanceThreshold;
        if (!m_loggedFinalBodyPose || finalBodyPoseDistance >= PoseLogDistanceThreshold)
        {
            m_loggedFinalBodyPose = true;
            m_lastLoggedFinalBodyPose = m_cachedFinalPresentationPosition;
            AZ_Printf(
                "amandacore",
                "client.final_body_pose pos=(%.3f, %.3f, %.3f) yaw=%.3f",
                m_cachedFinalPresentationPosition.GetX(),
                m_cachedFinalPresentationPosition.GetY(),
                m_cachedFinalPresentationPosition.GetZ(),
                m_cachedFinalAvatarFacingRadians);
        }

        UpdateAvatarPresentation(deltaTime, visualPlanarDelta);
        SetCharacterBasePosition(m_cachedFinalPresentationPosition);
        SyncEntityTransformToCharacterBase(m_cachedFinalPresentationPosition);
        UpdateCameraComponent();
        DrawValidationArena();
        DrawLocalPlayerProxy();
    }

    bool LocalPlayerControllerComponent::OnInputChannelEventFiltered(const AzFramework::InputChannel& inputChannel)
    {
        const auto& channelId = inputChannel.GetInputChannelId();
        const bool active = inputChannel.IsActive();

        if (ImGui::GetIO().WantTextInput)
        {
            if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericW)
            {
                m_moveForward = false;
            }
            else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericS)
            {
                m_moveBackward = false;
            }
            else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericA)
            {
                m_strafeLeft = false;
            }
            else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericD)
            {
                m_strafeRight = false;
            }
            return false;
        }

        if (channelId == AzFramework::InputDeviceMouse::Button::Left)
        {
            m_leftMouseHeld = active;
            return false;
        }

        if (channelId == AzFramework::InputDeviceMouse::Button::Right)
        {
            const bool wasOrbiting = m_cameraOrbitModeActive;
            m_cameraOrbitModeActive = active;
            if (wasOrbiting && !m_cameraOrbitModeActive)
            {
                m_cameraOrbitYawRadians = 0.0f;
                m_cameraPitchRadians = CameraDefaultPitchRadians;
                m_cameraYawRadians = m_avatarFacingRadians;
                m_chaseLockActive = true;
            }
            else if (m_cameraOrbitModeActive)
            {
                m_chaseLockActive = false;
            }
            return !inputChannel.IsStateBegan();
        }

        if (channelId == AzFramework::InputDeviceMouse::Movement::X)
        {
            if (m_cameraOrbitModeActive)
            {
                m_cameraOrbitYawRadians = WrapAngleRadians(m_cameraOrbitYawRadians - (inputChannel.GetValue() * CameraYawSensitivity));
                m_cameraYawRadians = WrapAngleRadians(m_avatarFacingRadians + m_cameraOrbitYawRadians);
            }
            return true;
        }

        if (channelId == AzFramework::InputDeviceMouse::Movement::Y)
        {
            if (m_cameraOrbitModeActive)
            {
                m_cameraPitchRadians = AZ::GetClamp(
                    m_cameraPitchRadians - (inputChannel.GetValue() * CameraPitchSensitivity),
                    CameraMinPitchRadians,
                    CameraMaxPitchRadians);
            }
            return true;
        }

        if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericW)
        {
            m_moveForward = active;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericS)
        {
            m_moveBackward = active;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericA)
        {
            m_strafeLeft = active;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericD)
        {
            m_strafeRight = active;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericX && inputChannel.IsStateBegan())
        {
            m_requestDisconnect = true;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericR && inputChannel.IsStateBegan())
        {
            m_requestReconnect = true;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericQ && inputChannel.IsStateBegan())
        {
            m_requestQuit = true;
        }

        return false;
    }

    void LocalPlayerControllerComponent::InitializeFromWorldState()
    {
        if (m_initialized)
        {
            return;
        }

        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return;
        }

        const auto& worldState = gameCore->GetClientWorldState();
        if (!worldState.m_worldConnected)
        {
            return;
        }

        ApplyWorldPosition(
            static_cast<float>(worldState.m_session.m_position.m_x),
            static_cast<float>(worldState.m_session.m_position.m_y),
            static_cast<float>(worldState.m_session.m_position.m_z));
        if (!m_loggedInputReady)
        {
            m_loggedInputReady = true;
            AZ_Printf(
                "amandacore",
                "client.input_ready entity=LocalPlayer listening=%s priority=%d focusHint=click_game_window_if_input_is_still_inactive",
                m_inputListening ? "true" : "false",
                GetPriority());
        }
        if (!m_loggedInputHelp)
        {
            m_loggedInputHelp = true;
            AZ_Printf(
                "amandacore",
                "client.input_help move=W/S turn=A/D camera=RMB target=Tab/LMB dualMove=LMB+RMB combat=F/1/2 disconnect=X reconnect=R quit=Q");
        }
        if (!m_loggedLocomotionMode)
        {
            m_loggedLocomotionMode = true;
            AZ_Printf(
                "amandacore",
                "client.locomotion_mode_applied mode=avatar_chase turn=A/D orbitMove=RMB+WASD freeFly=false");
        }
        if (!m_loggedStableLocomotionMode)
        {
            m_loggedStableLocomotionMode = true;
            AZ_Printf("amandacore", "client.locomotion_mode_active mode=avatar_chase_stable");
        }
        if (!m_loggedGrounded)
        {
            m_loggedGrounded = true;
            AZ_Printf(
                "amandacore",
                "client.player_grounded spawn=(%.3f, %.3f, %.3f) floorZ=%.3f",
                static_cast<float>(worldState.m_session.m_position.m_x),
                static_cast<float>(worldState.m_session.m_position.m_y),
                ValidationFloorZ,
                ValidationFloorZ);
        }
        m_initialized = true;
    }

    void LocalPlayerControllerComponent::ResetCameraToEncounterFrame(const AZ::Vector3& playerPosition)
    {
        const AZ::Vector2 encounterDirection(
            CameraEncounterAnchorX - playerPosition.GetX(),
            CameraEncounterAnchorY - playerPosition.GetY());
        if (encounterDirection.GetLengthSq() > 0.001f)
        {
            m_avatarFacingRadians = WrapAngleRadians(AZStd::atan2(encounterDirection.GetY(), encounterDirection.GetX()));
        }

        m_cameraOrbitYawRadians = 0.0f;
        m_cameraYawRadians = m_avatarFacingRadians;
        m_cameraPitchRadians = CameraDefaultPitchRadians;
        m_cameraFollowDistance = CameraDefaultFollowDistance;
        m_cameraOrbitModeActive = false;
        m_chaseLockActive = true;
        m_loggedCameraSource = false;

        AZ_Printf(
            "amandacore",
            "client.camera_spawn_frame_locked yawRadians=%.3f pitchRadians=%.3f followDistance=%.2f",
            m_cameraYawRadians,
            m_cameraPitchRadians,
            m_cameraFollowDistance);
    }

    void LocalPlayerControllerComponent::UpdateCameraComponent()
    {
        if (m_defaultViewportId == AzFramework::InvalidViewportId)
        {
            if (auto* viewportRequests = AZ::RPI::ViewportContextRequests::Get())
            {
                if (AZ::RPI::ViewportContextPtr viewportContext = viewportRequests->GetDefaultViewportContext())
                {
                    m_defaultViewportId = viewportContext->GetId();
                }
            }
        }

        if (m_defaultViewportId == AzFramework::InvalidViewportId)
        {
            return;
        }

        if (!m_loggedCameraAttachment)
        {
            m_loggedCameraAttachment = true;
            AZ_Printf("amandacore", "client.camera_attached entity=LocalPlayer attached=true freeFly=false");
        }

        if (!m_loggedCameraMode)
        {
            m_loggedCameraMode = true;
            AZ_Printf("amandacore", "client.camera_mode_applied mode=third_person_chase");
        }

        const AZ::Vector3 playerPosition = m_cachedFinalPoseValid ? m_cachedFinalPresentationPosition : GetPresentationPosition();
        if (m_pendingCameraReset)
        {
            ResetCameraToEncounterFrame(playerPosition);
            m_pendingCameraReset = false;
        }

        AZ::Transform playerWorldTransform = AZ::Transform::CreateIdentity();
        playerWorldTransform.SetTranslation(playerPosition);
        playerWorldTransform.SetRotation(AZ::Quaternion::CreateRotationZ(m_cachedFinalPoseValid ? m_cachedFinalAvatarFacingRadians : m_avatarFacingRadians));

        m_cameraFollowDistance = AZ::GetClamp(m_cameraFollowDistance, CameraMinFollowDistance, CameraMaxFollowDistance);
        const AZ::Vector3 localForward = BuildCameraForwardFromYaw(m_cameraOrbitYawRadians);
        const AZ::Vector3 pivotLocal(0.0f, 0.0f, CameraPivotOffsetZ);
        const AZ::Vector3 targetLocal =
            pivotLocal + (localForward * CameraLookAheadDistance) + AZ::Vector3(0.0f, 0.0f, CameraLookLiftZ);
        const float horizontalDistance = AZStd::cos(-m_cameraPitchRadians) * m_cameraFollowDistance;
        const float verticalDistance = AZStd::sin(-m_cameraPitchRadians) * m_cameraFollowDistance;
        const AZ::Vector3 desiredCameraLocalPosition =
            pivotLocal - (localForward * horizontalDistance) + AZ::Vector3(0.0f, 0.0f, 0.50f + verticalDistance);

        const AZ::Vector3 anchorWorldPosition = playerWorldTransform.TransformPoint(pivotLocal);
        AZ::Vector3 resolvedCameraWorldPosition = playerWorldTransform.TransformPoint(desiredCameraLocalPosition);
        const AZ::Vector3 desiredCameraWorldPosition = resolvedCameraWorldPosition;
        const AZ::Vector3 cameraDelta = desiredCameraWorldPosition - anchorWorldPosition;
        const float desiredDistance = cameraDelta.GetLength();

        bool sceneQueryResolved = false;
        if (desiredDistance > 0.001f)
        {
            AzPhysics::SceneHandle sceneHandle = AzPhysics::InvalidSceneHandle;
            Physics::DefaultWorldBus::BroadcastResult(
                sceneHandle,
                &Physics::DefaultWorldRequests::GetDefaultSceneHandle);

            if (sceneHandle != AzPhysics::InvalidSceneHandle)
            {
                if (AzPhysics::SceneInterface* sceneInterface = AZ::Interface<AzPhysics::SceneInterface>::Get())
                {
                    const AZ::Vector3 cameraDirection = cameraDelta / desiredDistance;
                    AzPhysics::RayCastRequest rayRequest;
                    rayRequest.m_start = anchorWorldPosition;
                    rayRequest.m_direction = cameraDirection;
                    rayRequest.m_distance = desiredDistance;
                    rayRequest.m_queryType = AzPhysics::SceneQuery::QueryType::StaticAndDynamic;
                    rayRequest.m_hitFlags = AzPhysics::SceneQuery::HitFlags::Position | AzPhysics::SceneQuery::HitFlags::Normal;
                    rayRequest.m_filterCallback = [entityId = GetEntityId()](const AzPhysics::SimulatedBody* body, const Physics::Shape*)
                    {
                        if (body && body->GetEntityId() == entityId)
                        {
                            return AzPhysics::SceneQuery::QueryHitType::None;
                        }
                        return AzPhysics::SceneQuery::QueryHitType::Block;
                    };

                    AzPhysics::SceneQueryHits queryHits;
                    if (sceneInterface->QueryScene(sceneHandle, &rayRequest, queryHits) && !queryHits.m_hits.empty())
                    {
                        const AzPhysics::SceneQueryHit& hit = queryHits.m_hits.front();
                        if ((hit.m_resultFlags & AzPhysics::SceneQuery::ResultFlags::Distance) != AzPhysics::SceneQuery::ResultFlags::Invalid)
                        {
                            const float resolvedDistance = AZ::GetClamp(
                                hit.m_distance - CameraCollisionSafetyDistance,
                                CameraMinimumResolvedDistance,
                                desiredDistance);
                            if (resolvedDistance < desiredDistance - 0.001f)
                            {
                                resolvedCameraWorldPosition = anchorWorldPosition + (cameraDirection * resolvedDistance);
                                sceneQueryResolved = true;
                                AZ_Printf(
                                    "amandacore",
                                    "client.camera_scene_query_resolved hit=true resolvedDistance=%.3f",
                                    resolvedDistance);
                            }
                        }
                    }
                }
            }
        }

        if (!sceneQueryResolved && resolvedCameraWorldPosition.GetZ() < CameraFloorFallbackZ)
        {
            const float candidateZ = resolvedCameraWorldPosition.GetZ();
            resolvedCameraWorldPosition.SetZ(CameraFloorFallbackZ);
            AZ_Printf(
                "amandacore",
                "client.camera_floor_clamp_applied candidateZ=%.3f resolvedZ=%.3f floorZ=%.3f",
                candidateZ,
                resolvedCameraWorldPosition.GetZ(),
                ValidationFloorZ);
        }

        const AZ::Transform inversePlayerWorld = playerWorldTransform.GetInverse();
        const AZ::Vector3 resolvedCameraLocalPosition = inversePlayerWorld.TransformPoint(resolvedCameraWorldPosition);
        const AZ::Transform cameraLocalTransform = AZ::Transform::CreateLookAt(
            resolvedCameraLocalPosition,
            targetLocal,
            AZ::Transform::Axis::YPositive);
        const AZ::Transform finalCameraWorldTransform = playerWorldTransform * cameraLocalTransform;
        AzFramework::ViewportRequestBus::Event(
            m_defaultViewportId,
            &AzFramework::ViewportRequestBus::Events::SetCameraTransform,
            finalCameraWorldTransform);

        if (auto* gameCore = GameCore::IGameCoreRequests::Get())
        {
            GameCore::ClientCameraState cameraState;
            cameraState.m_ready = true;
            cameraState.m_worldTransform = finalCameraWorldTransform;
            cameraState.m_verticalFovDegrees = 60.0f;
            gameCore->SetCameraState(cameraState);
        }

        if (!m_loggedCameraReady)
        {
            m_loggedCameraReady = true;
            AZ_Printf(
                "amandacore",
                "client.camera_ready entity=LocalPlayer viewportId=%d active=true attached=true yawRadians=%.3f",
                m_defaultViewportId,
                m_cameraYawRadians);
        }

        if (!m_loggedCameraEntity)
        {
            m_loggedCameraEntity = true;
            AZ_Printf(
                "amandacore",
                "client.active_camera_entity name=DefaultViewport id=%d parent=LocalPlayer",
                m_defaultViewportId);
        }

        if (!m_loggedCameraAnchor)
        {
            m_loggedCameraAnchor = true;
            const AZStd::string anchorEntityId = GetEntityId().ToString();
            AZ_Printf(
                "amandacore",
                "client.camera_anchor_entity name=LocalPlayer id=%s",
                anchorEntityId.c_str());
        }

        if (!m_loggedDetachedCameraDisabled)
        {
            m_loggedDetachedCameraDisabled = true;
            AZ_Printf("amandacore", "client.detached_camera_path_active=false");
        }

        if (!m_loggedViewportCameraOwner)
        {
            m_loggedViewportCameraOwner = true;
            AZ_Printf("amandacore", "client.viewport_camera_owned_by=LocalPlayer");
        }

        if (!m_loggedCameraTransformWriter)
        {
            m_loggedCameraTransformWriter = true;
            AZ_Printf("amandacore", "client.camera_transform_writer=LocalPlayer");
        }

        if (!m_loggedCameraSource)
        {
            m_loggedCameraSource = true;
            if (!sceneQueryResolved)
            {
                AZ_Printf(
                    "amandacore",
                    "client.camera_scene_query_resolved hit=false resolvedDistance=%.3f",
                    desiredDistance);
            }
            AZ_Printf(
                "amandacore",
                "client.avatar_root_transform pos=(%.3f, %.3f, %.3f) yaw=%.3f",
                playerPosition.GetX(),
                playerPosition.GetY(),
                playerPosition.GetZ(),
                m_avatarFacingRadians);
            AZ_Printf(
                "amandacore",
                "client.camera_source_transform anchorPos=(%.3f, %.3f, %.3f) anchorYaw=%.3f",
                anchorWorldPosition.GetX(),
                anchorWorldPosition.GetY(),
                anchorWorldPosition.GetZ(),
                m_cameraYawRadians);
        }

        const float cameraSourceDistance = m_loggedCameraSourceFollow
            ? (anchorWorldPosition - m_lastLoggedCameraSourcePosition).GetLength()
            : PoseLogDistanceThreshold;
        if (!m_loggedCameraSourceFollow || cameraSourceDistance >= PoseLogDistanceThreshold)
        {
            m_loggedCameraSourceFollow = true;
            m_lastLoggedCameraSourcePosition = anchorWorldPosition;
            AZ_Printf(
                "amandacore",
                "client.camera_source_transform pos=(%.3f, %.3f, %.3f) yaw=%.3f",
                anchorWorldPosition.GetX(),
                anchorWorldPosition.GetY(),
                anchorWorldPosition.GetZ(),
                m_cameraYawRadians);
        }

        if (!m_loggedMouseLookState || m_lastLoggedMouseLookState != m_cameraOrbitModeActive)
        {
            m_loggedMouseLookState = true;
            m_lastLoggedMouseLookState = m_cameraOrbitModeActive;
            AZ_Printf(
                "amandacore",
                "client.mouse_look_active=%s",
                m_cameraOrbitModeActive ? "true" : "false");
        }

        if (!m_loggedOrbitState || m_lastLoggedOrbitState != m_cameraOrbitModeActive)
        {
            m_loggedOrbitState = true;
            m_lastLoggedOrbitState = m_cameraOrbitModeActive;
            AZ_Printf(
                "amandacore",
                "client.orbit_mode_active=%s",
                m_cameraOrbitModeActive ? "true" : "false");
        }

        if (!m_loggedChaseLockState || m_lastLoggedChaseLockState != m_chaseLockActive)
        {
            m_loggedChaseLockState = true;
            m_lastLoggedChaseLockState = m_chaseLockActive;
            AZ_Printf(
                "amandacore",
                "client.chase_lock_active=%s",
                m_chaseLockActive ? "true" : "false");
        }
    }

    void LocalPlayerControllerComponent::UpdateGroundingState()
    {
        bool onGround = false;
        PhysX::CharacterGameplayRequestBus::EventResult(
            onGround,
            GetEntityId(),
            &PhysX::CharacterGameplayRequestBus::Events::IsOnGround);

        if (onGround && !m_loggedGroundMovementReady)
        {
            m_loggedGroundMovementReady = true;
            AZ_Printf("amandacore", "client.grounded_movement_ready onGround=true");
        }
    }

    void LocalPlayerControllerComponent::UpdateAvatarPresentation(
        float deltaTime,
        const AZ::Vector2& planarDelta)
    {
        const float deltaTimeSafe = deltaTime > 0.0001f ? deltaTime : 0.0001f;
        const float planarSpeed = planarDelta.GetLength() / deltaTimeSafe;
        const float strideTarget = planarSpeed > 0.05f ? 1.0f : 0.0f;
        m_avatarStrideBlend += (strideTarget - m_avatarStrideBlend) *
            AZ::GetClamp(deltaTime * AvatarStrideBlendRate, 0.0f, 1.0f);

        if (planarSpeed > 0.05f)
        {
            const float strideScale = AZ::GetClamp(planarSpeed / MoveSpeedUnitsPerSecond, 0.55f, 1.35f);
            m_avatarStridePhase = WrapAngleRadians(m_avatarStridePhase + (deltaTime * AvatarStrideRate * strideScale));
            return;
        }

    }

    AZ::Vector3 LocalPlayerControllerComponent::GetPresentationPosition() const
    {
        AZ::Vector3 basePosition = AZ::Vector3::CreateZero();
        Physics::CharacterRequestBus::EventResult(
            basePosition,
            GetEntityId(),
            &Physics::CharacterRequestBus::Events::GetBasePosition);

        if (basePosition.GetLengthSq() > 0.0f)
        {
            return basePosition;
        }

        AZ::Vector3 transformPosition = AZ::Vector3::CreateZero();
        AZ::TransformBus::EventResult(transformPosition, GetEntityId(), &AZ::TransformBus::Events::GetWorldTranslation);
        return transformPosition;
    }

    void LocalPlayerControllerComponent::SyncEntityTransformToCharacterBase(const AZ::Vector3& basePosition)
    {
        AZ::Transform worldTransform = AZ::Transform::CreateIdentity();
        worldTransform.SetTranslation(basePosition);
        worldTransform.SetRotation(AZ::Quaternion::CreateRotationZ(m_avatarFacingRadians));
        AZ::TransformBus::Event(GetEntityId(), &AZ::TransformBus::Events::SetWorldTM, worldTransform);

        const float entitySyncDistance = m_loggedEntitySync
            ? (basePosition - m_lastLoggedEntitySyncPosition).GetLength()
            : PoseLogDistanceThreshold;
        if (!m_loggedEntitySync || entitySyncDistance >= PoseLogDistanceThreshold)
        {
            m_loggedEntitySync = true;
            m_lastLoggedEntitySyncPosition = basePosition;
            AZ_Printf(
                "amandacore",
                "client.entity_sync_applied pos=(%.3f, %.3f, %.3f) yaw=%.3f",
                basePosition.GetX(),
                basePosition.GetY(),
                basePosition.GetZ(),
                m_avatarFacingRadians);
        }
    }

    void LocalPlayerControllerComponent::SetCharacterBasePosition(const AZ::Vector3& basePosition)
    {
        Physics::CharacterRequestBus::Event(
            GetEntityId(),
            &Physics::CharacterRequestBus::Events::SetBasePosition,
            basePosition);
    }

    void LocalPlayerControllerComponent::ApplyWorldPosition(float x, float y, float z)
    {
        AZ::Vector3 currentBasePosition = AZ::Vector3::CreateZero();
        Physics::CharacterRequestBus::EventResult(
            currentBasePosition,
            GetEntityId(),
            &Physics::CharacterRequestBus::Events::GetBasePosition);

        const AZ::Vector3 requestedPosition(
            x,
            y,
            currentBasePosition.GetZ() == 0.0f ? CharacterBaseSnapZ : currentBasePosition.GetZ());
        if (AZ::GetAbs(z - ValidationFloorZ) > 0.001f)
        {
            AZ_Printf(
                "amandacore",
                "client.ground_snap_applied fromZ=%.3f toZ=%.3f",
                z,
                ValidationFloorZ);
        }

        SetCharacterBasePosition(requestedPosition);
        m_cachedFinalPresentationPosition = requestedPosition;
        m_cachedFinalAvatarFacingRadians = m_avatarFacingRadians;
        m_cachedFinalPoseValid = true;
    }

    void LocalPlayerControllerComponent::DrawValidationArena()
    {
        AZ::RPI::Scene* scene = AZ::RPI::Scene::GetSceneForEntityId(GetEntityId());
        if (!scene)
        {
            return;
        }

        auto auxGeom = AZ::RPI::AuxGeomFeatureProcessorInterface::GetDrawQueueForScene(scene);
        if (!auxGeom)
        {
            return;
        }

        if (!m_loggedValidationFloor)
        {
            m_loggedValidationFloor = true;
            AZ_Printf(
                "amandacore",
                "client.world_material_coverage_visible center=(%.1f, %.1f, %.1f) extent=%.1f spawn=(%.1f, %.1f, %.1f) materials=stonewake_0_2",
                ValidationFloorExtent * 0.5f,
                ValidationFloorExtentY * 0.5f,
                ValidationFloorZ,
                ValidationFloorExtent,
                ValidationSpawnX,
                ValidationSpawnY,
                ValidationFloorZ);
        }

        const AZ::Color commandColor(0.28f, 0.74f, 0.78f, 1.0f);
        const AZ::Color pathColor(0.78f, 0.60f, 0.28f, 1.0f);
        const AZ::Color obstacleColor(0.38f, 0.39f, 0.43f, 1.0f);
        const AZ::Color encounterColor(0.90f, 0.42f, 0.22f, 1.0f);
        const AZ::Color groundBaseColor(0.24f, 0.33f, 0.23f, 1.0f);
        const AZ::Color groundTileLight(0.34f, 0.47f, 0.25f, 1.0f);
        const AZ::Color groundTileDark(0.24f, 0.36f, 0.22f, 1.0f);
        const AZ::Color mossColor(0.20f, 0.41f, 0.23f, 1.0f);
        const AZ::Color rockyGroundColor(0.34f, 0.36f, 0.34f, 1.0f);
        const AZ::Color ridgeColor(0.30f, 0.31f, 0.34f, 1.0f);
        const AZ::Color horizonColor(0.23f, 0.34f, 0.44f, 1.0f);
        const AZ::Color roadColor(0.50f, 0.37f, 0.22f, 1.0f);
        const AZ::Color roadPebbleColor(0.62f, 0.55f, 0.43f, 1.0f);
        const AZ::Color fieldColor(0.50f, 0.43f, 0.22f, 1.0f);
        const AZ::Color cropColor(0.74f, 0.62f, 0.25f, 1.0f);
        const AZ::Color plasterColor(0.78f, 0.70f, 0.58f, 1.0f);
        const AZ::Color woodColor(0.42f, 0.28f, 0.16f, 1.0f);
        const AZ::Color roofColor(0.50f, 0.40f, 0.20f, 1.0f);
        const AZ::Color cutStoneColor(0.42f, 0.43f, 0.40f, 1.0f);
        const AZ::Color trunkColor(0.33f, 0.22f, 0.13f, 1.0f);
        const AZ::Color canopyColor(0.16f, 0.43f, 0.23f, 1.0f);
        const AZ::Color trainingRingColor(0.70f, 0.55f, 0.28f, 1.0f);
        const AZ::Color waterColor(0.12f, 0.39f, 0.57f, 1.0f);
        const AZ::Color wetShoreColor(0.45f, 0.39f, 0.30f, 1.0f);
        const AZ::Color runeColor(0.28f, 0.74f, 0.92f, 1.0f);

        auxGeom->DrawAabb(
            AZ::Aabb::CreateCenterHalfExtents(
                AZ::Vector3(ValidationFloorExtent * 0.5f, ValidationFloorExtentY * 0.5f, -0.22f),
                AZ::Vector3(ValidationFloorExtent * 0.5f, ValidationFloorExtentY * 0.5f, 0.22f)),
            groundBaseColor,
            AZ::RPI::AuxGeomDraw::DrawStyle::Solid);

        for (int tileY = 0; tileY < 25; ++tileY)
        {
            for (int tileX = 0; tileX < 40; ++tileX)
            {
                const float centerX = (static_cast<float>(tileX) * 12.0f) + 6.0f;
                const float centerY = (static_cast<float>(tileY) * 12.0f) + 6.0f;
                const int materialSelector = ((tileX * 17) + (tileY * 31)) % 9;
                const AZ::Color tileColor = materialSelector == 0
                    ? mossColor
                    : (materialSelector == 1 ? rockyGroundColor : (materialSelector <= 4 ? groundTileLight : groundTileDark));
                auxGeom->DrawAabb(
                    AZ::Aabb::CreateCenterHalfExtents(
                        AZ::Vector3(centerX, centerY, -0.03f),
                        AZ::Vector3(5.9f, 5.9f, 0.03f)),
                    tileColor,
                    AZ::RPI::AuxGeomDraw::DrawStyle::Solid);

                if (((tileX * 5) + tileY) % 11 == 0)
                {
                    auxGeom->DrawAabb(
                        AZ::Aabb::CreateCenterHalfExtents(
                            AZ::Vector3(centerX - 2.2f, centerY + 1.8f, 0.035f),
                            AZ::Vector3(1.1f, 0.08f, 0.025f)),
                        groundTileLight,
                        AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
                }
            }
        }

        const AZ::Vector3 streamCenters[] = {
            AZ::Vector3(248.0f, 158.0f, 0.04f),
            AZ::Vector3(296.0f, 184.0f, 0.04f),
            AZ::Vector3(352.0f, 215.0f, 0.04f),
            AZ::Vector3(420.0f, 248.0f, 0.04f),
        };
        const AZ::Vector3 streamExtents[] = {
            AZ::Vector3(30.0f, 2.8f, 0.04f),
            AZ::Vector3(34.0f, 3.1f, 0.04f),
            AZ::Vector3(38.0f, 3.3f, 0.04f),
            AZ::Vector3(30.0f, 3.5f, 0.04f),
        };
        for (size_t streamIndex = 0; streamIndex < AZ_ARRAY_SIZE(streamCenters); ++streamIndex)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(streamCenters[streamIndex], streamExtents[streamIndex]),
                wetShoreColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    streamCenters[streamIndex] + AZ::Vector3(0.0f, 0.0f, 0.035f),
                    streamExtents[streamIndex] - AZ::Vector3(0.0f, 1.2f, 0.0f)),
                waterColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        const AZ::Vector3 roadCenters[] = {
            AZ::Vector3(22.0f, 14.0f, 0.015f),
            AZ::Vector3(55.0f, 26.0f, 0.015f),
            AZ::Vector3(96.0f, 44.0f, 0.015f),
            AZ::Vector3(154.0f, 82.0f, 0.015f),
            AZ::Vector3(232.0f, 122.0f, 0.015f),
            AZ::Vector3(314.0f, 172.0f, 0.015f),
            AZ::Vector3(382.0f, 214.0f, 0.015f),
            AZ::Vector3(438.0f, 246.0f, 0.015f),
        };
        const AZ::Vector3 roadExtents[] = {
            AZ::Vector3(16.0f, 1.35f, 0.035f),
            AZ::Vector3(24.0f, 1.45f, 0.035f),
            AZ::Vector3(30.0f, 1.55f, 0.035f),
            AZ::Vector3(34.0f, 1.70f, 0.035f),
            AZ::Vector3(40.0f, 1.80f, 0.035f),
            AZ::Vector3(38.0f, 1.90f, 0.035f),
            AZ::Vector3(34.0f, 1.95f, 0.035f),
            AZ::Vector3(18.0f, 2.05f, 0.035f),
        };
        for (size_t roadIndex = 0; roadIndex < AZ_ARRAY_SIZE(roadCenters); ++roadIndex)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(roadCenters[roadIndex], roadExtents[roadIndex]),
                roadColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);

            for (int pebbleIndex = 0; pebbleIndex < 5; ++pebbleIndex)
            {
                const float offsetX = (static_cast<float>(pebbleIndex) - 2.0f) * (roadExtents[roadIndex].GetX() * 0.34f);
                const float offsetY = (pebbleIndex % 2 == 0 ? 0.75f : -0.75f);
                auxGeom->DrawSphere(
                    roadCenters[roadIndex] + AZ::Vector3(offsetX, offsetY, 0.08f),
                    0.16f,
                    roadPebbleColor);
            }
        }

        const AZ::Vector3 fieldCenters[] = {
            AZ::Vector3(142.0f, 72.0f, 0.02f),
            AZ::Vector3(156.0f, 78.0f, 0.02f),
            AZ::Vector3(170.0f, 84.0f, 0.02f),
            AZ::Vector3(184.0f, 90.0f, 0.02f),
            AZ::Vector3(164.0f, 104.0f, 0.02f),
        };
        for (const AZ::Vector3& fieldCenter : fieldCenters)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(fieldCenter, AZ::Vector3(9.0f, 0.28f, 0.04f)),
                fieldColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);

            for (int cropIndex = 0; cropIndex < 6; ++cropIndex)
            {
                auxGeom->DrawAabb(
                    AZ::Aabb::CreateCenterHalfExtents(
                        fieldCenter + AZ::Vector3((static_cast<float>(cropIndex) - 2.5f) * 2.6f, 0.0f, 0.22f),
                        AZ::Vector3(0.10f, 0.16f, 0.22f)),
                    cropColor,
                    AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            }
        }

        const AZ::Vector3 buildingCenters[] = {
            AZ::Vector3(7.0f, 7.0f, 1.0f),
            AZ::Vector3(18.0f, 7.0f, 0.9f),
            AZ::Vector3(8.0f, 29.0f, 0.85f),
            AZ::Vector3(22.0f, 29.0f, 0.85f),
            AZ::Vector3(42.0f, 22.0f, 0.65f),
            AZ::Vector3(76.0f, 37.0f, 1.1f),
            AZ::Vector3(232.0f, 118.0f, 1.2f),
            AZ::Vector3(322.0f, 174.0f, 2.6f),
            AZ::Vector3(420.0f, 224.0f, 1.0f),
            AZ::Vector3(438.0f, 246.0f, 1.7f),
        };
        const AZ::Vector3 buildingExtents[] = {
            AZ::Vector3(2.8f, 2.0f, 1.0f),
            AZ::Vector3(2.0f, 1.8f, 0.9f),
            AZ::Vector3(2.4f, 1.6f, 0.85f),
            AZ::Vector3(3.0f, 1.8f, 0.85f),
            AZ::Vector3(7.0f, 0.35f, 0.65f),
            AZ::Vector3(6.0f, 0.9f, 1.1f),
            AZ::Vector3(2.4f, 2.4f, 1.2f),
            AZ::Vector3(4.4f, 4.4f, 2.6f),
            AZ::Vector3(3.8f, 1.6f, 1.0f),
            AZ::Vector3(6.5f, 1.2f, 1.7f),
        };
        for (size_t buildingIndex = 0; buildingIndex < AZ_ARRAY_SIZE(buildingCenters); ++buildingIndex)
        {
            const AZ::Color wallColor = buildingIndex == 6 || buildingIndex == 7
                ? cutStoneColor
                : (buildingIndex == 4 || buildingIndex == 8 || buildingIndex == 9 ? woodColor : plasterColor);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(buildingCenters[buildingIndex], buildingExtents[buildingIndex]),
                wallColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    buildingCenters[buildingIndex] + AZ::Vector3(0.0f, 0.0f, buildingExtents[buildingIndex].GetZ() + 0.18f),
                    AZ::Vector3(buildingExtents[buildingIndex].GetX() + 0.35f, buildingExtents[buildingIndex].GetY() + 0.35f, 0.18f)),
                buildingIndex == 6 || buildingIndex == 7 ? ridgeColor : roofColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        const AZ::Vector3 horizonCenters[] = {
            AZ::Vector3(ValidationFloorExtent * 0.5f, -5.0f, 3.0f),
            AZ::Vector3(ValidationFloorExtent * 0.5f, ValidationFloorExtentY + 5.0f, 3.0f),
            AZ::Vector3(-5.0f, ValidationFloorExtentY * 0.5f, 3.0f),
            AZ::Vector3(ValidationFloorExtent + 5.0f, ValidationFloorExtentY * 0.5f, 3.0f),
        };
        const AZ::Vector3 horizonHalfExtents[] = {
            AZ::Vector3(ValidationFloorExtent * 0.5f, 0.5f, 3.0f),
            AZ::Vector3(ValidationFloorExtent * 0.5f, 0.5f, 3.0f),
            AZ::Vector3(0.5f, ValidationFloorExtentY * 0.5f, 3.0f),
            AZ::Vector3(0.5f, ValidationFloorExtentY * 0.5f, 3.0f),
        };
        for (size_t horizonIndex = 0; horizonIndex < AZ_ARRAY_SIZE(horizonCenters); ++horizonIndex)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(horizonCenters[horizonIndex], horizonHalfExtents[horizonIndex]),
                horizonColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        const AZ::Vector3 ridgeCenters[] = {
            AZ::Vector3(5.0f, 6.0f, 1.2f),
            AZ::Vector3(18.0f, 4.0f, 1.0f),
            AZ::Vector3(42.0f, 5.0f, 1.3f),
            AZ::Vector3(72.0f, 7.0f, 1.1f),
            AZ::Vector3(104.0f, 18.0f, 1.2f),
            AZ::Vector3(120.0f, 48.0f, 1.4f),
            AZ::Vector3(190.0f, 92.0f, 1.1f),
            AZ::Vector3(230.0f, 112.0f, 1.4f),
            AZ::Vector3(268.0f, 142.0f, 1.2f),
            AZ::Vector3(330.0f, 184.0f, 1.3f),
            AZ::Vector3(402.0f, 232.0f, 1.1f),
            AZ::Vector3(460.0f, 270.0f, 1.3f),
        };
        for (const AZ::Vector3& ridgeCenter : ridgeCenters)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    ridgeCenter,
                    AZ::Vector3(1.8f, 1.4f, 1.1f)),
                ridgeColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
        }

        const AZ::Vector3 treeCenters[] = {
            AZ::Vector3(29.0f, 48.0f, 0.0f),
            AZ::Vector3(41.0f, 50.0f, 0.0f),
            AZ::Vector3(90.0f, 63.0f, 0.0f),
            AZ::Vector3(112.0f, 83.0f, 0.0f),
            AZ::Vector3(188.0f, 118.0f, 0.0f),
            AZ::Vector3(210.0f, 136.0f, 0.0f),
            AZ::Vector3(270.0f, 154.0f, 0.0f),
            AZ::Vector3(350.0f, 162.0f, 0.0f),
            AZ::Vector3(396.0f, 244.0f, 0.0f),
        };
        for (const AZ::Vector3& treeBase : treeCenters)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    treeBase + AZ::Vector3(0.0f, 0.0f, 0.65f),
                    AZ::Vector3(0.18f, 0.18f, 0.65f)),
                trunkColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawSphere(treeBase + AZ::Vector3(0.0f, 0.0f, 1.75f), 0.82f, canopyColor);
            auxGeom->DrawSphere(treeBase + AZ::Vector3(0.42f, 0.12f, 1.55f), 0.55f, canopyColor);
            auxGeom->DrawSphere(treeBase + AZ::Vector3(-0.38f, -0.16f, 1.55f), 0.55f, canopyColor);
        }

        for (int segmentIndex = 0; segmentIndex < 18; ++segmentIndex)
        {
            const float angleRadians = (AZ::Constants::TwoPi / 18.0f) * static_cast<float>(segmentIndex);
            auxGeom->DrawSphere(
                AZ::Vector3(
                    58.0f + (AZStd::cos(angleRadians) * 9.0f),
                    32.0f + (AZStd::sin(angleRadians) * 7.0f),
                    ValidationMarkerZ),
                0.14f,
                trainingRingColor);
        }

        for (int segmentIndex = 0; segmentIndex < 10; ++segmentIndex)
        {
            const float angleRadians = (AZ::Constants::TwoPi / 10.0f) * static_cast<float>(segmentIndex);
            const AZ::Vector3 ringOffset(
                AZStd::cos(angleRadians) * 2.5f,
                AZStd::sin(angleRadians) * 2.5f,
                ValidationMarkerZ);
            auxGeom->DrawSphere(
                AZ::Vector3(ValidationSpawnX, ValidationSpawnY, 0.0f) + ringOffset,
                ValidationSpawnMarkerRadius,
                commandColor);
        }

        const AZ::Vector3 commandPostColumns[] = {
            AZ::Vector3(10.0f, 10.0f, ValidationMarkerZ),
            AZ::Vector3(14.0f, 10.0f, ValidationMarkerZ),
            AZ::Vector3(10.0f, 14.0f, ValidationMarkerZ),
            AZ::Vector3(14.0f, 14.0f, ValidationMarkerZ)};
        for (const AZ::Vector3& columnBase : commandPostColumns)
        {
            for (int segmentIndex = 0; segmentIndex < 4; ++segmentIndex)
            {
                auxGeom->DrawSphere(
                    columnBase + AZ::Vector3(0.0f, 0.0f, 0.28f + (segmentIndex * 0.24f)),
                    0.12f,
                    commandColor);
            }
        }

        const AZ::Vector3 trailMarkers[] = {
            AZ::Vector3(22.0f, 14.0f, ValidationMarkerZ),
            AZ::Vector3(52.0f, 26.0f, ValidationMarkerZ),
            AZ::Vector3(84.0f, 36.0f, ValidationMarkerZ),
            AZ::Vector3(134.0f, 64.0f, ValidationMarkerZ),
            AZ::Vector3(184.0f, 96.0f, ValidationMarkerZ),
            AZ::Vector3(232.0f, 118.0f, ValidationMarkerZ),
            AZ::Vector3(322.0f, 174.0f, ValidationMarkerZ),
            AZ::Vector3(438.0f, 246.0f, ValidationMarkerZ)};
        for (const AZ::Vector3& marker : trailMarkers)
        {
            auxGeom->DrawSphere(marker, 0.18f, pathColor);
        }

        const AZ::Vector3 boulderCluster[] = {
            AZ::Vector3(72.0f, 28.0f, 0.30f),
            AZ::Vector3(76.0f, 36.0f, 0.55f),
            AZ::Vector3(80.0f, 44.0f, 0.42f),
            AZ::Vector3(74.0f, 46.0f, 0.38f)};
        for (const AZ::Vector3& boulder : boulderCluster)
        {
            auxGeom->DrawSphere(boulder, 0.75f, obstacleColor);
        }

        const AZ::Vector3 standingStones[] = {
            AZ::Vector3(228.0f, 114.0f, 1.0f),
            AZ::Vector3(233.0f, 113.0f, 1.4f),
            AZ::Vector3(237.0f, 117.0f, 1.0f),
            AZ::Vector3(235.0f, 123.0f, 1.2f),
            AZ::Vector3(229.0f, 122.0f, 0.9f),
        };
        for (const AZ::Vector3& stone : standingStones)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(stone, AZ::Vector3(0.42f, 0.28f, stone.GetZ())),
                cutStoneColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
            auxGeom->DrawSphere(stone + AZ::Vector3(0.0f, 0.0f, stone.GetZ() + 0.22f), 0.16f, runeColor);
        }

        const AZ::Vector3 quarryBlocks[] = {
            AZ::Vector3(310.0f, 166.0f, 0.45f),
            AZ::Vector3(318.0f, 170.0f, 0.55f),
            AZ::Vector3(328.0f, 176.0f, 0.50f),
            AZ::Vector3(336.0f, 181.0f, 0.38f),
        };
        for (const AZ::Vector3& block : quarryBlocks)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(block, AZ::Vector3(2.6f, 1.1f, block.GetZ())),
                ridgeColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
        }

        const AZ::Vector3 dockPlanks[] = {
            AZ::Vector3(410.0f, 252.0f, 0.18f),
            AZ::Vector3(418.0f, 253.0f, 0.18f),
            AZ::Vector3(426.0f, 254.0f, 0.18f),
            AZ::Vector3(434.0f, 255.0f, 0.18f),
        };
        for (const AZ::Vector3& plank : dockPlanks)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(plank, AZ::Vector3(3.6f, 0.42f, 0.08f)),
                woodColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        for (int tierIndex = 0; tierIndex < 4; ++tierIndex)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(452.0f, 258.0f, 0.45f + (tierIndex * 0.62f)),
                    AZ::Vector3(1.15f - (tierIndex * 0.10f), 1.15f - (tierIndex * 0.10f), 0.30f)),
                cutStoneColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
        }

        for (int segmentIndex = 0; segmentIndex < 12; ++segmentIndex)
        {
            const float angleRadians = (AZ::Constants::TwoPi / 12.0f) * static_cast<float>(segmentIndex);
            auxGeom->DrawSphere(
                AZ::Vector3(
                    EncounterAnchorX + (AZStd::cos(angleRadians) * 6.1f),
                    EncounterAnchorY + (AZStd::sin(angleRadians) * 4.8f),
                    ValidationMarkerZ),
                0.15f,
                encounterColor);
        }
    }

    void LocalPlayerControllerComponent::DrawLocalPlayerProxy()
    {
        AZ::RPI::Scene* scene = AZ::RPI::Scene::GetSceneForEntityId(GetEntityId());
        if (!scene)
        {
            return;
        }

        auto auxGeom = AZ::RPI::AuxGeomFeatureProcessorInterface::GetDrawQueueForScene(scene);
        if (!auxGeom)
        {
            return;
        }

        const AZ::Vector3 position = m_cachedFinalPoseValid ? m_cachedFinalPresentationPosition : GetPresentationPosition();

        const float avatarVisiblePoseDistance = m_loggedAvatarVisiblePose
            ? (position - m_lastLoggedAvatarVisiblePose).GetLength()
            : PoseLogDistanceThreshold;
        if (!m_loggedAvatarVisiblePose || avatarVisiblePoseDistance >= PoseLogDistanceThreshold)
        {
            m_loggedAvatarVisiblePose = true;
            m_lastLoggedAvatarVisiblePose = position;
            AZ_Printf(
                "amandacore",
                "client.avatar_visible_pose pos=(%.3f, %.3f, %.3f) yaw=%.3f",
                position.GetX(),
                position.GetY(),
                position.GetZ(),
                m_cachedFinalPoseValid ? m_cachedFinalAvatarFacingRadians : m_avatarFacingRadians);
        }

        const float facingRadians = m_cachedFinalPoseValid ? m_cachedFinalAvatarFacingRadians : m_avatarFacingRadians;
        const AZ::Vector3 forward = ForwardFromYaw(facingRadians);
        const AZ::Vector3 right = RightFromYaw(facingRadians);
        const float strideSwing = AZStd::sin(m_avatarStridePhase) * AvatarStrideReach * m_avatarStrideBlend;
        const float oppositeStrideSwing = -strideSwing;
        const float armSwing = strideSwing * 0.8f;
        const float leanForward = 0.05f * m_avatarStrideBlend;

        const AZ::Color clothColor(0.19f, 0.34f, 0.59f, 1.0f);
        const AZ::Color trimColor(0.85f, 0.71f, 0.31f, 1.0f);
        const AZ::Color skinColor(0.92f, 0.90f, 0.84f, 1.0f);
        const AZ::Color shadowColor(0.10f, 0.16f, 0.22f, 1.0f);
        const AZ::Color ringColor(0.88f, 0.91f, 0.94f, 1.0f);

        const AZ::Vector3 pelvis = position + AZ::Vector3(0.0f, 0.0f, AvatarHipHeight);
        const AZ::Vector3 waist = position + (forward * leanForward) + AZ::Vector3(0.0f, 0.0f, AvatarWaistHeight);
        const AZ::Vector3 chest = position + (forward * (0.08f + leanForward)) + AZ::Vector3(0.0f, 0.0f, AvatarChestHeight);
        const AZ::Vector3 head = chest + (forward * 0.08f) + AZ::Vector3(0.0f, 0.0f, AvatarHeadHeight - AvatarChestHeight);

        auxGeom->DrawSphere(pelvis, AvatarPelvisRadius, clothColor);
        auxGeom->DrawSphere(waist, AvatarTorsoRadius, clothColor);
        auxGeom->DrawSphere(chest, AvatarChestRadius, clothColor);
        auxGeom->DrawSphere(chest - (forward * 0.18f), 0.18f, shadowColor);
        auxGeom->DrawSphere(head, AvatarHeadRadius, skinColor);
        auxGeom->DrawSphere(head + (forward * 0.14f), 0.05f, trimColor);

        const AZ::Vector3 leftHip = pelvis + (right * AvatarHipSpacing);
        const AZ::Vector3 rightHip = pelvis - (right * AvatarHipSpacing);
        const AZ::Vector3 leftKnee = position + (right * AvatarHipSpacing) + (forward * (-strideSwing * 0.20f)) + AZ::Vector3(0.0f, 0.0f, AvatarKneeHeight);
        const AZ::Vector3 rightKnee = position - (right * AvatarHipSpacing) + (forward * (-oppositeStrideSwing * 0.20f)) + AZ::Vector3(0.0f, 0.0f, AvatarKneeHeight);
        const AZ::Vector3 leftAnkle = position + (right * AvatarFootSpacing) + (forward * (strideSwing * 0.35f)) + AZ::Vector3(0.0f, 0.0f, AvatarAnkleHeight);
        const AZ::Vector3 rightAnkle = position - (right * AvatarFootSpacing) + (forward * (oppositeStrideSwing * 0.35f)) + AZ::Vector3(0.0f, 0.0f, AvatarAnkleHeight);
        const AZ::Vector3 leftFoot = position + (right * AvatarFootSpacing) + (forward * (strideSwing * 0.45f)) + AZ::Vector3(0.0f, 0.0f, AvatarFootHeight);
        const AZ::Vector3 rightFoot = position - (right * AvatarFootSpacing) + (forward * (oppositeStrideSwing * 0.45f)) + AZ::Vector3(0.0f, 0.0f, AvatarFootHeight);

        auxGeom->DrawSphere(leftHip, AvatarLimbRadius, clothColor);
        auxGeom->DrawSphere(rightHip, AvatarLimbRadius, clothColor);
        auxGeom->DrawSphere(leftKnee, AvatarLimbRadius, clothColor);
        auxGeom->DrawSphere(rightKnee, AvatarLimbRadius, clothColor);
        auxGeom->DrawSphere(leftAnkle, AvatarLimbRadius * 0.92f, trimColor);
        auxGeom->DrawSphere(rightAnkle, AvatarLimbRadius * 0.92f, trimColor);
        auxGeom->DrawSphere(leftFoot, AvatarLimbRadius * 0.85f, trimColor);
        auxGeom->DrawSphere(rightFoot, AvatarLimbRadius * 0.85f, trimColor);

        const AZ::Vector3 leftShoulder = chest + (right * AvatarShoulderOffset) + AZ::Vector3(0.0f, 0.0f, AvatarShoulderHeight - AvatarChestHeight);
        const AZ::Vector3 rightShoulder = chest - (right * AvatarShoulderOffset) + AZ::Vector3(0.0f, 0.0f, AvatarShoulderHeight - AvatarChestHeight);
        const AZ::Vector3 leftElbow = chest + (right * (AvatarShoulderOffset + 0.10f)) + (forward * (-armSwing * 0.24f)) + AZ::Vector3(0.0f, 0.0f, 1.16f);
        const AZ::Vector3 rightElbow = chest - (right * (AvatarShoulderOffset + 0.10f)) + (forward * (armSwing * 0.24f)) + AZ::Vector3(0.0f, 0.0f, 1.16f);
        const AZ::Vector3 leftHand = chest + (right * (AvatarShoulderOffset + 0.14f)) + (forward * (-armSwing * 0.34f)) + AZ::Vector3(0.0f, 0.0f, 0.84f);
        const AZ::Vector3 rightHand = chest - (right * (AvatarShoulderOffset + 0.14f)) + (forward * (armSwing * 0.34f)) + AZ::Vector3(0.0f, 0.0f, 0.84f);

        auxGeom->DrawSphere(leftShoulder, AvatarArmRadius, trimColor);
        auxGeom->DrawSphere(rightShoulder, AvatarArmRadius, trimColor);
        auxGeom->DrawSphere(leftElbow, AvatarArmRadius, clothColor);
        auxGeom->DrawSphere(rightElbow, AvatarArmRadius, clothColor);
        auxGeom->DrawSphere(leftHand, AvatarArmRadius * 0.85f, skinColor);
        auxGeom->DrawSphere(rightHand, AvatarArmRadius * 0.85f, skinColor);

        for (int segmentIndex = 0; segmentIndex < AvatarSelectionRingSegments; ++segmentIndex)
        {
            const float angleRadians = (AZ::Constants::TwoPi / static_cast<float>(AvatarSelectionRingSegments)) *
                static_cast<float>(segmentIndex);
            const AZ::Vector3 ringOffset(
                AZStd::cos(angleRadians) * AvatarSelectionRingRadius,
                AZStd::sin(angleRadians) * AvatarSelectionRingRadius,
                0.06f);
            auxGeom->DrawSphere(position + ringOffset, AvatarSelectionRingSphereRadius, ringColor);
        }
    }
} // namespace MovementPhysics
