package testh

import "github.com/eleven-am/liveui/internal/diff"

// Recorder captures the most recent SSR HTML string and diff operations.
type Recorder interface {
	// SnapshotHTML updates the stored SSR HTML after a successful flush.
	SnapshotHTML(html string)

	// SnapshotOps replaces the stored operations emitted during the last flush.
	SnapshotOps(ops []diff.Op)

	// HTML returns the last recorded SSR HTML.
	HTML() string

	// Ops returns the last recorded diff operations.
	Ops() []diff.Op

	// ResetOps clears any recorded operations without touching the stored HTML.
	ResetOps()
}
