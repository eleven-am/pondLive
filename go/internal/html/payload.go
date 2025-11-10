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
	default:
		if str, ok := v.(fmt.Stringer); ok {
			return str.String()
		}
		return fmt.Sprint(v)
	}
	return fallback
}
