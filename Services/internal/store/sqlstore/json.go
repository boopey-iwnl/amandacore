package sqlstore

import (
	"encoding/json"
	"fmt"
)

func encodeJSON(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodeJSON(payload string, target any) error {
	if payload == "" {
		payload = "null"
	}
	if err := json.Unmarshal([]byte(payload), target); err != nil {
		return fmt.Errorf("decode json payload: %w", err)
	}
	return nil
}
