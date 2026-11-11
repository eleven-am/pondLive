package session

import (
	"sync"
	"sync/atomic"
	"testing"

	"go.uber.org/goleak"

	"github.com/eleven-am/pondlive/internal/protocol"
)

var testClientSeq sync.Map

func deliver(ls *LiveSession, evt protocol.ClientEvt) error {
	v, _ := testClientSeq.LoadOrStore(ls, new(atomic.Uint64))
	evt.CS = v.(*atomic.Uint64).Add(1)
	_, err := ls.Deliver(ls.Generation(), evt)
	return err
}

func leakCheck(t *testing.T) {
	t.Helper()
	opt := goleak.IgnoreCurrent()
	t.Cleanup(func() { goleak.VerifyNone(t, opt) })
}
