package headers

import (
	"net/http"
	"time"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

const headersScript = `
function(element, transport) {
	transport.on('set', (data) => {
		fetch(data.url, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				'X-Header-Action-Token': data.token
			},
		}).catch(err => {
			console.error('Failed to set header:', err);
		});
	});
	return () => {};
}
`

type actionRequest struct {
	Name         string
	Value        string
	Token        string
	URL          string
	DeleteAction bool
	Options      *cookieOptions
}

type cookieOptions struct {
	Path     string
	Domain   string
	MaxAge   int
	Expires  time.Time
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

type Manager struct {
	controller *RequestController
	script     *runtime.ScriptHandle
	handler    *runtime.HandlerHandle
	ctx        runtime.Ctx
	actions    []actionRequest
}

var headersManagerCtx = runtime.CreateContext[*Manager](nil)

func UseHeadersManager(ctx runtime.Ctx) *Manager {
	return headersManagerCtx.Use(ctx)
}

func ProvideHeadersManager(ctx runtime.Ctx, render func(runtime.Ctx) *dom.StructuredNode) *dom.StructuredNode {
	manager := useInternalHeaders(ctx)
	return headersManagerCtx.Provide(ctx, manager, render)
}

func useInternalHeaders(ctx runtime.Ctx) *Manager {
	controller := UseRequestController(ctx)
	script := runtime.UseScript(ctx, headersScript)

	manager := &Manager{
		controller: controller,
		script:     &script,
		ctx:        ctx,
	}

	handler := runtime.UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
		token := r.Header.Get("X-Header-Action-Token")
		if token == "" {
			http.Error(w, "server: missing action token", http.StatusBadRequest)
			return nil
		}

		action, ok := manager.getAction(token)
		if !ok || action == nil {
			http.Error(w, "server: invalid action token", http.StatusBadRequest)
			return nil
		}

		if !action.DeleteAction {
			cookie := &http.Cookie{
				Name:  action.Name,
				Value: action.Value,
			}
			if action.Options != nil {
				cookie.Path = action.Options.Path
				cookie.Domain = action.Options.Domain
				cookie.MaxAge = action.Options.MaxAge
				cookie.Expires = action.Options.Expires
				cookie.Secure = action.Options.Secure
				cookie.HttpOnly = action.Options.HttpOnly
				cookie.SameSite = action.Options.SameSite
			}
			http.SetCookie(w, cookie)
		} else {
			http.SetCookie(w, &http.Cookie{
				Name:   action.Name,
				Value:  "",
				MaxAge: -1,
			})
		}

		w.WriteHeader(http.StatusOK)
		return nil
	})

	manager.handler = &handler
	return manager
}

func (m *Manager) Get(name string) (string, bool) {
	if m.controller == nil {
		return "", false
	}
	return m.controller.Get(name)
}

func (m *Manager) SetCookie(name, value string) {
	m.SetCookieWithOptions(name, value, CookieOptions{})
}

func (m *Manager) SetCookieWithOptions(name, value string, opts CookieOptions) {
	if m.controller == nil {
		return
	}

	if !m.ctx.IsLive() {
		cookie := &http.Cookie{
			Name:     name,
			Value:    value,
			Path:     opts.Path,
			Domain:   opts.Domain,
			MaxAge:   opts.MaxAge,
			Expires:  opts.Expires,
			Secure:   opts.Secure,
			HttpOnly: opts.HttpOnly,
			SameSite: opts.SameSite,
		}
		m.controller.Set("Set-Cookie", cookie.String())
		return
	}

	var options *cookieOptions
	if opts.Path != "" || opts.Domain != "" || opts.MaxAge != 0 || !opts.Expires.IsZero() || opts.Secure || opts.HttpOnly || opts.SameSite != 0 {
		options = &cookieOptions{
			Path:     opts.Path,
			Domain:   opts.Domain,
			MaxAge:   opts.MaxAge,
			Expires:  opts.Expires,
			Secure:   opts.Secure,
			HttpOnly: opts.HttpOnly,
			SameSite: opts.SameSite,
		}
	}

	act := actionRequest{
		Name:    name,
		Value:   value,
		Token:   m.handler.GenerateToken(),
		URL:     m.handler.URL(),
		Options: options,
	}

	m.replaceAction(act)
	m.script.Send("set", act)

	m.controller.UpdateCookie(name, value)
}

func (m *Manager) DeleteCookie(name string) {
	if m.controller == nil {
		return
	}

	if !m.ctx.IsLive() {
		m.controller.Set("Set-Cookie", name+"=; Max-Age=-1")
		return
	}

	act := actionRequest{
		Name:         name,
		DeleteAction: true,
		Token:        m.handler.GenerateToken(),
		URL:          m.handler.URL(),
	}

	m.replaceAction(act)
	m.script.Send("set", act)

	m.controller.DeleteCookieFromRequest(name)
}

func (m *Manager) getAction(token string) (*actionRequest, bool) {
	for _, action := range m.actions {
		if action.Token == token {
			return &action, true
		}
	}

	return nil, false
}

func (m *Manager) replaceAction(request actionRequest) {
	for i, action := range m.actions {
		if action.Name == request.Name {
			m.actions[i] = request
			return
		}
	}
	m.actions = append(m.actions, request)
}

func (m *Manager) AttachTo(node *dom.StructuredNode) {
	if m.script != nil {
		m.script.AttachTo(node)
	}
}
