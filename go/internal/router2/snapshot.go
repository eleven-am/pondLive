package router2

import (
	"encoding/json"
	"time"
)

// SnapshotPayload encodes router snapshot data for SSR/hydration.
type SnapshotPayload struct {
	Location Location          `json:"location"`
	Params   map[string]string `json:"params"`
	History  []NavEventPayload `json:"history"`
}

// NavEventPayload serializes NavEvent.
type NavEventPayload struct {
	Seq    uint64   `json:"seq"`
	Kind   NavKind  `json:"kind"`
	Target Location `json:"target"`
	Source string   `json:"source"`
	Time   int64    `json:"time"`
}

// ToPayload converts a snapshot to JSON bytes.
func (s Snapshot) ToPayload() ([]byte, error) {
	payload := SnapshotPayload{
		Location: s.Location,
		Params:   cloneParams(s.Params),
	}
	if len(s.History) > 0 {
		payload.History = make([]NavEventPayload, len(s.History))
		for i, event := range s.History {
			payload.History[i] = NavEventPayload{
				Seq:    event.Seq,
				Kind:   event.Kind,
				Target: event.Target,
				Source: event.Source,
				Time:   event.Time.UnixMilli(),
			}
		}
	}
	return json.Marshal(payload)
}

// SnapshotFromPayload decodes JSON snapshot bytes.
func SnapshotFromPayload(data []byte) (Snapshot, error) {
	if len(data) == 0 {
		return Snapshot{}, nil
	}
	var payload SnapshotPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return Snapshot{}, err
	}
	snap := Snapshot{
		Location: payload.Location,
		Params:   cloneParams(payload.Params),
	}
	if len(payload.History) > 0 {
		snap.History = make([]NavEvent, len(payload.History))
		for i, event := range payload.History {
			snap.History[i] = NavEvent{
				Seq:    event.Seq,
				Kind:   event.Kind,
				Target: event.Target,
				Source: event.Source,
				Time:   time.UnixMilli(event.Time),
			}
		}
	}
	return snap, nil
}
