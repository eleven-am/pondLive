package dom

import "strings"

var defaultEventPresets = map[string]EventOptions{
	"input": {
		Listen: []string{"change"},
		Props:  []string{"target.value"},
	},
	"change": {
		Props: []string{"target.value"},
	},
	"submit": {
		Props: []string{"event.submitter"},
	},
	"timeupdate": {
		Listen: []string{"play", "pause"},
		Props:  []string{"target.currentTime", "target.duration", "target.paused"},
	},
	"play": {
		Props: []string{"target.paused", "target.currentTime"},
	},
	"pause": {
		Props: []string{"target.paused", "target.currentTime"},
	},
	"seeking": {
		Props: []string{"target.currentTime"},
	},
	"seeked": {
		Props: []string{"target.currentTime"},
	},
}

// Event represents a DOM event payload delivered to the server.
type Event struct {
	Name    string
	Value   string
	Payload map[string]any
	Form    map[string]string
	Mods    Modifiers
}

// Modifiers captures keyboard and mouse modifier state for an event.
type Modifiers struct {
	Ctrl   bool
	Meta   bool
	Shift  bool
	Alt    bool
	Button int
}

// Updates marks a handler return value that can trigger rerenders.
type Updates interface{ isUpdates() }

// EventHandler represents a server-side event handler for a DOM event.
type EventHandler func(Event) Updates

// EventOptions configures additional metadata for a DOM event handler.
type EventOptions struct {
	Key    string
	Listen []string
	Props  []string
}

// EventBinding stores a handler together with its metadata so the runtime can
// describe how the browser should subscribe to and capture the event payload.
type EventBinding struct {
	Handler EventHandler
	Listen  []string
	Props   []string
}

// EventAssignment captures the runtime metadata associated with a handler ID.
type EventAssignment struct {
	ID     string
	Listen []string
	Props  []string
}

func (b EventBinding) WithOptions(opts EventOptions, primary string) EventBinding {
	b.Key = opts.Key
	b.Listen = sanitizeEventList(primary, opts.Listen)
	b.Props = sanitizeSelectorList(opts.Props)
	return b
}

func MergeEventOptions(base, extra EventOptions) EventOptions {
	merged := EventOptions{}
	if extra.Key != "" {
		merged.Key = extra.Key
	} else {
		merged.Key = base.Key
	}
	if len(base.Listen) > 0 {
		merged.Listen = append(merged.Listen, base.Listen...)
	}
	if len(extra.Listen) > 0 {
		merged.Listen = append(merged.Listen, extra.Listen...)
	}
	if len(base.Props) > 0 {
		merged.Props = append(merged.Props, base.Props...)
	}
	if len(extra.Props) > 0 {
		merged.Props = append(merged.Props, extra.Props...)
	}
	return merged
}

func DefaultEventOptions(event string) EventOptions {
	if event == "" {
		return EventOptions{}
	}
	preset, ok := defaultEventPresets[strings.ToLower(event)]
	if !ok {
		return EventOptions{}
	}

	out := EventOptions{}
	if len(preset.Listen) > 0 {
		out.Listen = append(out.Listen, preset.Listen...)
	}
	if len(preset.Props) > 0 {
		out.Props = append(out.Props, preset.Props...)
	}
	return out
}

func MergeEventBinding(existing, addition EventBinding) EventBinding {
	merged := existing
	if merged.Handler == nil {
		merged.Handler = addition.Handler
	} else if addition.Handler != nil {
		first := merged.Handler
		second := addition.Handler
		merged.Handler = func(ev Event) Updates {
			var result Updates
			if first != nil {
				if out := first(ev); out != nil {
					result = out
				}
			}
			if second != nil {
				if out := second(ev); out != nil {
					result = out
				}
			}
			return result
		}
	}
	merged.Listen = mergeStringSet(existing.Listen, addition.Listen, true)
	merged.Props = mergeStringSet(existing.Props, addition.Props, false)
	if merged.Key == "" {
		merged.Key = addition.Key
	}
	return merged
}

func mergeStringSet(base, extra []string, fold bool) []string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(base)+len(extra))
	add := func(values []string) {
		for _, v := range values {
			if v == "" {
				continue
			}
			key := v
			if fold {
				key = strings.ToLower(v)
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, v)
		}
	}
	add(base)
	add(extra)
	if len(out) == 0 {
		return nil
	}
	return out
}

func sanitizeEventList(primary string, events []string) []string {
	if len(events) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	cleaned := make([]string, 0, len(events))
	for _, evt := range events {
		evt = strings.TrimSpace(evt)
		if evt == "" || strings.EqualFold(evt, primary) {
			continue
		}
		key := strings.ToLower(evt)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, evt)
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

func sanitizeSelectorList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	cleaned := make([]string, 0, len(values))
	for _, sel := range values {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		if _, ok := seen[sel]; ok {
			continue
		}
		seen[sel] = struct{}{}
		cleaned = append(cleaned, sel)
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

type rerender struct{}

func (rerender) isUpdates() {}

// Rerender signals that the component tree should be rendered again.
func Rerender() Updates { return rerender{} }
