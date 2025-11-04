package testh

import "github.com/eleven-am/liveui/internal/diff"

type recorder struct {
	html string
	ops  []diff.Op
}

// NewRecorder constructs a Recorder backed by an in-memory snapshot.
func NewRecorder() Recorder {
	return &recorder{}
}

func (r *recorder) SnapshotHTML(html string) {
	if r == nil {
		return
	}
	r.html = html
}

func (r *recorder) SnapshotOps(ops []diff.Op) {
	if r == nil {
		return
	}
	r.ops = cloneOps(ops)
}

func (r *recorder) HTML() string {
	if r == nil {
		return ""
	}
	return r.html
}

func (r *recorder) Ops() []diff.Op {
	if r == nil {
		return nil
	}
	return cloneOps(r.ops)
}

func (r *recorder) ResetOps() {
	if r == nil {
		return
	}
	r.ops = nil
}
