package protocol

import "maps"

func Number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func Strings(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

var RectKeys = []string{"x", "y", "width", "height", "top", "right", "bottom", "left"}

var ScrollKeys = []string{"scrollTop", "scrollLeft", "scrollHeight", "scrollWidth", "clientHeight", "clientWidth"}

func Floats(m map[string]any, keys []string) []float64 {
	out := make([]float64, len(keys))
	for i, k := range keys {
		out[i], _ = Number(m[k])
	}
	return out
}

func FloatMap(keys []string, values ...float64) map[string]any {
	out := make(map[string]any, len(keys))
	for i, k := range keys {
		out[k] = values[i]
	}
	return out
}

func DiagMeta(sessionID string, extra map[string]any) map[string]any {
	meta := make(map[string]any, len(extra)+2)
	maps.Copy(meta, extra)
	meta["session_id"] = sessionID
	return meta
}
