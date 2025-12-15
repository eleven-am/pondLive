package runtime

import (
	"context"

	"github.com/eleven-am/pondlive/internal/protocol"
)

type Ctx struct {
	instance  *Instance
	session   *Session
	hookIndex int
}

func (c *Ctx) Context() context.Context {
	return context.Background()
}

func GetBus(ctx *Ctx) *protocol.Bus {
	if ctx == nil || ctx.session == nil {
		return nil
	}
	return ctx.session.Bus
}

func (c *Ctx) ComponentID() string {
	if c == nil || c.instance == nil {
		return ""
	}
	return c.instance.ID
}

func (c *Ctx) ComponentDepth() int {
	if c == nil || c.instance == nil {
		return 0
	}
	depth := 0
	for p := c.instance.Parent; p != nil; p = p.Parent {
		depth++
	}
	return depth
}

func NewCtxForTest(inst *Instance, sess *Session) *Ctx {
	return &Ctx{instance: inst, session: sess}
}

func (c *Ctx) SessionID() string {
	if c == nil || c.session == nil {
		return ""
	}
	return c.session.SessionID
}
