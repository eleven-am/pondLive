package pkg

import (
	"github.com/eleven-am/pondlive/internal/router"
)

func Routes(defs ...RouteDef) Node {
	return router.Routes(defs...)
}

func Route(path string, component func() Node) RouteDef {
	return router.Route(path, component)
}

func Link(props LinkProps) Node {
	return router.Link(props)
}

func NavLink(props NavLinkProps) Node {
	return router.NavLink(props)
}

func Redirect(props RedirectProps) Node {
	return router.Redirect(props)
}
