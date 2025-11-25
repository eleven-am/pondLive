package router2

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// Link renders a simple anchor element with server-side navigation.
// Click events are handled server-side via the Bus - client stays naive.
//
// Usage:
//
//	router.Link(ctx, router.LinkProps{To: "/about"}, h.Text("About"))
func Link(ctx *runtime2.Ctx, props LinkProps, children []work.Node) work.Node {
	location := LocationContext.UseContextValue(ctx)
	base := &Location{Path: "/", Query: url.Values{}}
	if location != nil {
		base = location
	}

	target := resolveHref(base, props.To)
	href := buildHref(target.Path, target.Query, target.Hash)

	clickHandler := work.Handler{
		EventOptions: metadata.EventOptions{
			Prevent: true,
		},
		Fn: func(e work.Event) work.Updates {
			if props.Replace {
				Replace(ctx, props.To)
			} else {
				Navigate(ctx, props.To)
			}
			return nil
		},
	}

	return &work.Element{
		Tag: "a",
		Attrs: map[string][]string{
			"href": {href},
		},
		Handlers: map[string]work.Handler{
			"click": clickHandler,
		},
		Children: children,
	}
}

// NavLink is like Link but adds an "active" class when the link matches the current path.
//
// Usage:
//
//	router.NavLink(ctx, router.NavLinkProps{
//	    To:          "/about",
//	    ActiveClass: "nav-active",
//	}, h.Text("About"))
func NavLink(ctx *runtime2.Ctx, props NavLinkProps, children []work.Node) work.Node {
	location := LocationContext.UseContextValue(ctx)
	base := &Location{Path: "/", Query: url.Values{}}
	if location != nil {
		base = location
	}

	target := resolveHref(base, props.To)
	href := buildHref(target.Path, target.Query, target.Hash)

	isActive := false
	if props.End {
		isActive = location != nil && location.Path == target.Path
	} else {
		isActive = location != nil && matchesPrefix(location.Path, target.Path)
	}

	var classes []string
	if props.ClassName != "" {
		classes = append(classes, props.ClassName)
	}
	if isActive && props.ActiveClass != "" {
		classes = append(classes, props.ActiveClass)
	}

	attrs := map[string][]string{
		"href": {href},
	}

	if len(classes) > 0 {
		attrs["class"] = classes
	}

	if isActive {
		attrs["aria-current"] = []string{"page"}
	}

	clickHandler := work.Handler{
		EventOptions: metadata.EventOptions{
			Prevent: true,
		},
		Fn: func(e work.Event) work.Updates {
			if props.Replace {
				Replace(ctx, props.To)
			} else {
				Navigate(ctx, props.To)
			}
			return nil
		},
	}

	return &work.Element{
		Tag:   "a",
		Attrs: attrs,
		Handlers: map[string]work.Handler{
			"click": clickHandler,
		},
		Children: children,
	}
}

// NavLinkProps extends LinkProps with styling options for active state.
type NavLinkProps struct {
	To          string
	Replace     bool
	ClassName   string
	ActiveClass string
	End         bool
}

// matchesPrefix checks if the current path starts with the target path.
// Used for NavLink active state detection.
func matchesPrefix(currentPath, targetPath string) bool {
	if targetPath == "/" {
		return currentPath == "/"
	}

	if len(currentPath) < len(targetPath) {
		return false
	}

	if currentPath[:len(targetPath)] != targetPath {
		return false
	}

	if len(currentPath) > len(targetPath) {
		return currentPath[len(targetPath)] == '/'
	}
	return true
}
