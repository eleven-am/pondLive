package html

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

func (b EventBinding) withOptions(opts EventOptions, primary string) EventBinding {
	b.Listen = sanitizeEventList(primary, opts.Listen)
	b.Props = sanitizeSelectorList(opts.Props)
	return b
}

func mergeEventOptions(base, extra EventOptions) EventOptions {
	merged := EventOptions{}
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

func defaultEventOptions(event string) EventOptions {
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
