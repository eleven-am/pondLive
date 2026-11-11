package runtimetest

import (
	"encoding/json"

	"golang.org/x/net/html"

	"github.com/eleven-am/pondlive/internal/protocol"
)

func (a *App) element(id uint32) *html.Node {
	a.T.Helper()
	n := a.Model.Node(id)
	if n == nil || n.Type != html.ElementNode {
		a.T.Fatalf("runtimetest: no element with id %d", id)
	}
	return n
}

func (a *App) specFor(id uint32, events ...string) (protocol.HandlerSpec, bool) {
	raw, ok := attrOf(a.element(id), protocol.AttrHandlers)
	if !ok {
		return protocol.HandlerSpec{}, false
	}
	var specs []protocol.HandlerSpec
	if err := json.Unmarshal([]byte(raw), &specs); err != nil {
		a.T.Fatalf("runtimetest: element %d has an invalid data-pon: %v", id, err)
	}
	for _, name := range events {
		for _, s := range specs {
			if s.Event == name {
				return s, true
			}
		}
	}
	return protocol.HandlerSpec{}, false
}

func (a *App) FindByAttr(key, value string) uint32 {
	a.T.Helper()
	for _, id := range a.Model.IDs() {
		n := a.Model.Node(id)
		if n == nil || n.Type != html.ElementNode {
			continue
		}
		if v, ok := attrOf(n, key); ok && v == value {
			return id
		}
	}
	a.T.Fatalf("runtimetest: no element with %s=%q", key, value)
	return 0
}

func (a *App) browserEvent(id uint32, names []string, payload func(props string) map[string]any) []protocol.Edit {
	a.T.Helper()
	spec, ok := a.specFor(id, names...)
	if !ok {
		return nil
	}
	var err error
	a.On(func() { err = a.S.DispatchHandler(id, spec.Event, payload(spec.Fields)) })
	if err != nil {
		a.T.Fatalf("runtimetest: %s on element %d: %v", spec.Event, id, err)
	}
	edits := a.Pump()
	a.CheckReset()
	return edits
}

func (a *App) Type(id uint32, text string) []protocol.Edit {
	a.T.Helper()
	n := a.element(id)
	setAttr(n, "value", text)
	if n.Data == "textarea" {
		a.Model.setBody(n, text)
	}
	return a.browserEvent(id, []string{"~change", "input", "change"}, func(string) map[string]any {
		return map[string]any{"value": text}
	})
}

func (a *App) Check(id uint32, on bool) []protocol.Edit {
	a.T.Helper()
	n := a.element(id)
	if on {
		setAttr(n, "checked", "")
	} else {
		delAttr(n, "checked")
	}
	return a.browserEvent(id, []string{"~change", "change", "input"}, func(string) map[string]any {
		return map[string]any{"checked": on}
	})
}

func (a *App) Toggle(id uint32, open bool) []protocol.Edit {
	a.T.Helper()
	n := a.element(id)
	if open {
		setAttr(n, "open", "")
	} else {
		delAttr(n, "open")
	}
	return a.browserEvent(id, []string{"toggle", "close"}, func(string) map[string]any {
		return map[string]any{"open": open}
	})
}

func (a *App) Choose(id uint32, values ...string) []protocol.Edit {
	a.T.Helper()
	n := a.element(id)
	if _, multiple := attrOf(n, "multiple"); multiple {
		encoded, _ := json.Marshal(append([]string{}, values...))
		a.Model.selectOptions(n, protocol.AttrSelection, string(encoded))
		setAttr(n, protocol.AttrSelection, string(encoded))
	} else {
		if len(values) != 1 {
			a.T.Fatalf("runtimetest: a single select takes one value, got %d", len(values))
		}
		a.Model.selectOptions(n, "value", values[0])
		setAttr(n, "value", values[0])
	}
	return a.browserEvent(id, []string{"~change", "change", "input"}, func(string) map[string]any {
		if _, multiple := attrOf(n, "multiple"); multiple {
			list := make([]any, len(values))
			for i, v := range values {
				list[i] = v
			}
			return map[string]any{"selected": list}
		}
		return map[string]any{"value": values[0]}
	})
}

func (a *App) PushValue(id uint32, prop string, v any) []protocol.Edit {
	a.T.Helper()
	a.element(id)
	a.On(func() {
		a.S.Bus.Publish(protocol.DOMHandler, string(protocol.DOMValueAction), protocol.DOMValuePayload{ID: id, Prop: prop, Value: v})
	})
	return a.Pump()
}
