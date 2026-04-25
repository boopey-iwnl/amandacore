package worlds

import (
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/simcore"
)

func (s *worldServer) enqueueRuntimeCommandLocked(envelope simcore.CommandEnvelope) {
	if s.runtime == nil {
		return
	}

	commandKind := "unknown"
	if envelope.Command != nil {
		commandKind = string(envelope.Command.CommandKind())
	}

	queued, err := s.runtime.Enqueue(envelope)
	if err != nil {
		observability.LogEvent("world-service", observability.EventWorldCommandRejected, map[string]any{
			"commandKind": commandKind,
			"actorId":     string(envelope.ActorID),
			"zoneId":      string(envelope.ZoneID),
			"reason":      err.Error(),
		})
		return
	}

	observability.LogEvent("world-service", observability.EventWorldCommandEnqueued, map[string]any{
		"commandId":   string(queued.CommandID),
		"commandKind": string(queued.Command.CommandKind()),
		"sequence":    queued.Sequence,
		"actorId":     string(queued.ActorID),
		"zoneId":      string(queued.ZoneID),
		"queueDepth":  s.runtime.PendingCommandCount(),
	})
}

func (s *worldServer) runRuntimeTickLocked(now time.Time) TickResult {
	if s.runtime == nil {
		return TickResult{}
	}

	result := s.runtime.RunTick(now)
	if result.CommandsProcessed == 0 && !result.Slow {
		return result
	}

	fields := map[string]any{
		"tickId":               uint64(result.Tick.ID),
		"tickIntervalMs":       result.Tick.Interval.Milliseconds(),
		"queueDepthBeforeTick": result.QueueDepthBeforeTick,
		"commandsProcessed":    result.CommandsProcessed,
	}
	observability.LogEvent("world-service", observability.EventWorldTickStarted, fields)

	completedFields := map[string]any{
		"tickId":            uint64(result.Tick.ID),
		"durationMs":        float64(result.Duration.Microseconds()) / 1000.0,
		"commandsProcessed": result.CommandsProcessed,
		"eventsEmitted":     len(result.Events),
	}
	observability.LogEvent("world-service", observability.EventWorldTickCompleted, completedFields)

	if result.Slow {
		observability.LogEvent("world-service", observability.EventWorldTickSlow, completedFields)
	}
	return result
}
