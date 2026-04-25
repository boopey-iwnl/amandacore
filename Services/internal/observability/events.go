package observability

import (
	"encoding/json"
	"log"
	"time"
)

func LogEvent(service string, event string, fields map[string]any) {
	payload := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"service":   service,
		"event":     event,
	}

	for key, value := range fields {
		payload[key] = value
	}

	serialized, err := json.Marshal(payload)
	if err != nil {
		log.Printf("{\"timestamp\":\"%s\",\"service\":\"%s\",\"event\":\"logging_failed\",\"message\":%q}", time.Now().UTC().Format(time.RFC3339Nano), service, err.Error())
		return
	}

	log.Print(string(serialized))
}
