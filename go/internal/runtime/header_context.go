package runtime

import "net/http"

var headerContext = NewContext[HeaderState](noopHeaderState{})

// UseHeader returns the shared header state for the current session.
func UseHeader(ctx Ctx) HeaderState {
	state := headerContext.Use(ctx)
	if state == nil {
		if sess := ctx.Session(); sess != nil {
			return sess.currentHeaderState()
		}
		return noopHeaderState{}
	}
	return state
}

// provideHeaderState attaches the header state to the root component so hooks can access it.
func provideHeaderState(sess *ComponentSession, state HeaderState) {
	if sess == nil {
		return
	}
	if state == nil {
		state = noopHeaderState{}
	}
	sess.assignHeaderState(state)
	sess.installHeaderProvider()
}

func (s *ComponentSession) installHeaderProvider() {
	if s == nil || s.root == nil {
		return
	}
	root := s.root
	root.mu.Lock()
	defer root.mu.Unlock()
	if root.providers == nil {
		root.providers = make(map[contextID]any)
	}
	entry := &providerEntry[HeaderState]{
		get: func() HeaderState { return s.currentHeaderState() },
		set: func(next HeaderState) { s.assignHeaderState(next) },
		assign: func(next HeaderState) {
			s.assignHeaderState(next)
		},
		owner:  root,
		active: true,
	}
	root.providers[headerContext.id] = entry
}

type noopHeaderState struct{}

func (noopHeaderState) GetHeader(string) (string, bool)       { return "", false }
func (noopHeaderState) SetHeader(string, string)              {}
func (noopHeaderState) DeleteHeader(string)                   {}
func (noopHeaderState) AllHeaders() http.Header               { return http.Header{} }
func (noopHeaderState) GetCookie(string) (*http.Cookie, bool) { return nil, false }
func (noopHeaderState) SetCookie(*http.Cookie)                {}
func (noopHeaderState) DeleteCookie(string)                   {}
func (noopHeaderState) AllCookies() []*http.Cookie            { return nil }
func (noopHeaderState) RequestID() string                     { return "" }
func (noopHeaderState) SetMeta(string, any)                   {}
func (noopHeaderState) Meta(string) (any, bool)               { return nil, false }
