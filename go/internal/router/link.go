package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

func Link(ctx *runtime.Ctx, props LinkProps, children []work.Node) work.Node {
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

func NavLink(ctx *runtime.Ctx, props NavLinkProps, children []work.Node) work.Node {
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

type NavLinkProps struct {
	To          string
	Replace     bool
	ClassName   string
	ActiveClass string
	End         bool
}

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
