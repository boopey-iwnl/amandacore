package worlds

import "time"

type DomainEvent struct {
	EventName    string         `json:"eventName,omitempty"`
	Type         string         `json:"type"`
	OccurredAtMs int64          `json:"occurredAtMs"`
	CharacterID  string         `json:"characterId,omitempty"`
	EntityID     string         `json:"entityId,omitempty"`
	ZoneID       string         `json:"zoneId,omitempty"`
	Fields       map[string]any `json:"fields,omitempty"`
}

type StateDiff struct {
	DiffType     string         `json:"diffType,omitempty"`
	Type         string         `json:"type"`
	OccurredAtMs int64          `json:"occurredAtMs"`
	CharacterID  string         `json:"characterId,omitempty"`
	EntityID     string         `json:"entityId,omitempty"`
	ZoneID       string         `json:"zoneId,omitempty"`
	Fields       map[string]any `json:"fields,omitempty"`
}

func newDomainEvent(eventName string, args ...any) DomainEvent {
	fields := map[string]any{}
	characterID := ""
	entityID := ""
	zoneID := ""
	if len(args) == 1 {
		if typed, ok := args[0].(map[string]any); ok {
			fields = typed
		}
	}
	if len(args) == 4 {
		characterID, _ = args[0].(string)
		entityID, _ = args[1].(string)
		zoneID, _ = args[2].(string)
		if typed, ok := args[3].(map[string]any); ok {
			fields = typed
		}
	}
	cloned := cloneEventFields(fields)
	if characterID != "" {
		cloned["characterId"] = characterID
	}
	if entityID != "" {
		cloned["entityId"] = entityID
	}
	if zoneID != "" {
		cloned["zoneId"] = zoneID
	}
	return DomainEvent{EventName: eventName, Type: eventName, OccurredAtMs: nowMillis(), CharacterID: characterID, EntityID: entityID, ZoneID: zoneID, Fields: cloned}
}

func newStateDiff(diffType string, characterID string, entityID string, zoneID string, fields map[string]any) StateDiff {
	return StateDiff{
		DiffType:     diffType,
		Type:         diffType,
		OccurredAtMs: nowMillis(),
		CharacterID:  characterID,
		EntityID:     entityID,
		ZoneID:       zoneID,
		Fields:       cloneEventFields(fields),
	}
}

func cloneDomainEvents(source []DomainEvent) []DomainEvent {
	if len(source) == 0 {
		return []DomainEvent{}
	}
	cloned := make([]DomainEvent, len(source))
	for index, event := range source {
		cloned[index] = DomainEvent{EventName: event.EventName, Type: event.Type, OccurredAtMs: event.OccurredAtMs, CharacterID: event.CharacterID, EntityID: event.EntityID, ZoneID: event.ZoneID, Fields: cloneEventFields(event.Fields)}
	}
	return cloned
}

func cloneStateDiffs(source []StateDiff) []StateDiff {
	if len(source) == 0 {
		return []StateDiff{}
	}
	cloned := make([]StateDiff, len(source))
	for index, diff := range source {
		cloned[index] = StateDiff{DiffType: diff.DiffType, Type: diff.Type, OccurredAtMs: diff.OccurredAtMs, CharacterID: diff.CharacterID, EntityID: diff.EntityID, ZoneID: diff.ZoneID, Fields: cloneEventFields(diff.Fields)}
	}
	return cloned
}

func cloneEventFields(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func eventTimeFromMs(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}
