package runtime

import (
	"context"
	"strconv"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
)

type Ctx struct {
	s *Session
	o *reactive.Owner
}

func (c *Ctx) Context() context.Context {
	if c == nil || c.o == nil {
		return cancelledContext
	}
	return c.o.Context()
}

func SessionOf(c *Ctx) *Session {
	if c == nil {
		return nil
	}
	return c.s
}

func RunAttributed(c *Ctx, fn func()) {
	if c == nil || c.s == nil || fn == nil {
		return
	}
	c.s.runAttributed(c.o, fn)
}

func Invoke(c *Ctx, fn func()) {
	if c == nil || c.s == nil || fn == nil {
		return
	}
	c.s.invokeHandler(c.o, fn)
}

func RunOnActor(c *Ctx, fn func()) {
	if c == nil || fn == nil {
		return
	}
	if c.s == nil {
		fn()
		return
	}
	s, o := c.s, c.o
	s.RunOnActor(func() { s.runAttributed(o, fn) })
}

func CtxOnActor(c *Ctx) bool { return c != nil && c.s != nil && c.s.graph.OnActor() }

func (c *Ctx) After(d time.Duration, fn func()) *Timer {
	if c == nil || c.s == nil {
		return &Timer{dead: true}
	}
	s, o := c.s, c.o
	if !s.allowCreate("After") {
		return &Timer{dead: true}
	}
	tm := s.After(d, func() { s.invokeHandler(o, fn) })
	o.OnCleanup(tm.Stop)
	return tm
}

func GetBus(ctx *Ctx) *protocol.Bus {
	if ctx == nil || ctx.s == nil {
		return nil
	}
	return ctx.s.Bus
}

func (c *Ctx) ComponentID() string {
	if c == nil || c.o == nil {
		return ""
	}
	return c.o.Info.Name + "#" + strconv.FormatUint(uint64(c.o.Base), 10)
}

func (c *Ctx) ComponentName() string {
	if c == nil || c.o == nil {
		return ""
	}
	return c.o.Info.Name
}

func (c *Ctx) SessionID() string {
	if c == nil || c.s == nil {
		return ""
	}
	return c.s.SessionID
}

func (s *Session) newCtx(o *reactive.Owner) *Ctx { return &Ctx{s: s, o: o} }

func NewCtxForTest(s *Session, o *reactive.Owner) *Ctx { return &Ctx{s: s, o: o} }
