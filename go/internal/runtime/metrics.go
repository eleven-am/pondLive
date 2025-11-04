package runtime

import (
	"sync"
	"time"
)

// FrameRecord captures metadata about a frame delivered to a client.
type FrameRecord struct {
	SessionID         SessionID
	Sequence          int
	Ops               int
	Effects           int
	Nav               bool
	RenderDuration    time.Duration
	EffectDuration    time.Duration
	MaxEffectDuration time.Duration
	SlowEffects       int
}

// MetricsRecorder receives frame records for observability.
type MetricsRecorder interface {
	RecordFrame(FrameRecord)
}

var (
	metricsMu       sync.RWMutex
	metricsRecorder MetricsRecorder
)

// RegisterMetricsRecorder installs a process-wide metrics recorder.
func RegisterMetricsRecorder(recorder MetricsRecorder) {
	metricsMu.Lock()
	metricsRecorder = recorder
	metricsMu.Unlock()
}

func recordFrameMetrics(record FrameRecord) {
	metricsMu.RLock()
	recorder := metricsRecorder
	metricsMu.RUnlock()
	if recorder != nil {
		recorder.RecordFrame(record)
	}
}
