package runtime

import "github.com/eleven-am/pondlive/go/internal/protocol"

// Ctx is passed to component functions during render.
// It provides access to hooks and runtime functionality.
type Ctx struct {
	instance  *Instance // Which component is rendering
	session   *Session  // Access to session
	hookIndex int       // Current hook call position
}

// GetBus returns the session's Bus, or nil if not available (e.g., SSR).
// This provides controlled access to the Bus without exposing the full Session.
// Internal packages can use this for pub/sub communication.
func GetBus(ctx *Ctx) *protocol.Bus {
	if ctx == nil || ctx.session == nil {
		return nil
	}
	return ctx.session.Bus
}

// ComponentID returns the unique ID for the current component instance.
func (c *Ctx) ComponentID() string {
	if c == nil || c.instance == nil {
		return ""
	}
	return c.instance.ID
}

// ComponentDepth returns the depth of the current component in the tree (root = 0).
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
