package html

import (
	"fmt"
	"strconv"
)

func payloadFloat(payload map[string]any, key string, fallback float64) float64 {
	if payload == nil {
		return fallback
	}
	raw, ok := payload[key]
	if !ok {
		return fallback
	}
	switch v := raw.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case string:
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			return val
		}
	}
	return fallback
}

func payloadBool(payload map[string]any, key string, fallback bool) bool {
	if payload == nil {
		return fallback
	}
	raw, ok := payload[key]
	if !ok {
		return fallback
	}
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		if val, err := strconv.ParseBool(v); err == nil {
			return val
		}
	case float64:
		return v != 0
	case float32:
		return v != 0
	case int:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	}
	return fallback
}

func payloadInt(payload map[string]any, key string, fallback int) int {
	if payload == nil {
		return fallback
	}
	raw, ok := payload[key]
	if !ok {
		return fallback
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case uint:
		return int(v)
	case uint64:
		return int(v)
	case uint32:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
	}
	return fallback
}

// extractDetail extracts the detail map from payload if present, otherwise returns the payload itself.
// Client sends events as {name: "eventName", detail: {...}} so we need to unwrap the detail.
func extractDetail(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	if detail, ok := payload["detail"].(map[string]any); ok {
		return detail
	}
	return payload
}

func PayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return fallback
	}
	switch v := raw.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case []rune:
		return string(v)
	case fmt.Stringer:
		return v.String()
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	default:
		return fmt.Sprint(v)
	}
}
