package router

import (
	"sort"
	"strings"
)

type nodeType int

const (
	nodeStatic nodeType = iota
	nodeParam
	nodeWildcard
)

type node struct {
	typ      nodeType
	label    string
	prefix   string
	parent   *node
	children []*node
	entry    *routeEntry
}

type routerTrie struct {
	root *node
}

type matchResult struct {
	Entry  *routeEntry
	Params map[string]string
	Rest   string
}

func newRouterTrie() *routerTrie {
	return &routerTrie{
		root: &node{
			typ: nodeStatic,
		},
	}
}

func (t *routerTrie) Insert(pattern string, entry routeEntry) {
	if pattern == "" {
		pattern = "/"
	}
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	segments := strings.Split(strings.Trim(pattern, "/"), "/")
	if pattern == "/" {
		segments = []string{}
	}

	curr := t.root
	for _, seg := range segments {
		if seg == "" {
			continue
		}

		var child *node
		typ := nodeStatic
		label := seg
		prefix := seg

		if strings.HasPrefix(seg, ":") {
			typ = nodeParam
			label = strings.TrimPrefix(seg, ":")
			prefix = ":"
		} else if strings.HasPrefix(seg, "*") {
			typ = nodeWildcard
			label = strings.TrimPrefix(seg, "*")
			prefix = "*"
		}

		for _, c := range curr.children {
			if c.typ == typ && c.label == label {
				child = c
				break
			}
		}

		if child == nil {
			child = &node{
				typ:    typ,
				label:  label,
				prefix: prefix,
				parent: curr,
			}
			curr.children = append(curr.children, child)

			sort.Slice(curr.children, func(i, j int) bool {
				return curr.children[i].typ < curr.children[j].typ
			})
		}
		curr = child
	}

	curr.entry = &entry
}

func (t *routerTrie) Match(path string) *matchResult {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var bestMatch *matchResult

	var search func(n *node, pathIdx int, params map[string]string)
	search = func(n *node, pathIdx int, params map[string]string) {
		if bestMatch != nil {
			return
		}

		if pathIdx >= len(path) {
			if n.entry != nil {
				bestMatch = &matchResult{
					Entry:  n.entry,
					Params: params,
				}
				return
			}
			for _, child := range n.children {
				if child.typ == nodeWildcard && child.entry != nil {
					bestMatch = &matchResult{
						Entry:  child.entry,
						Params: params,
						Rest:   "/",
					}
					return
				}
			}
			return
		}

		if path[pathIdx] == '/' {
			pathIdx++
		}

		if pathIdx >= len(path) {
			if n.entry != nil {
				bestMatch = &matchResult{
					Entry:  n.entry,
					Params: params,
				}
				return
			}
			for _, child := range n.children {
				if child.typ == nodeWildcard && child.entry != nil {
					bestMatch = &matchResult{
						Entry:  child.entry,
						Params: params,
						Rest:   "/",
					}
					return
				}
			}
			return
		}

		end := strings.IndexByte(path[pathIdx:], '/')
		if end == -1 {
			end = len(path)
		} else {
			end += pathIdx
		}
		seg := path[pathIdx:end]
		nextPathIdx := end

		for _, child := range n.children {
			switch child.typ {
			case nodeStatic:
				if child.label == seg {
					search(child, nextPathIdx, copyParams(params))
					if bestMatch != nil {
						return
					}
				}
			case nodeParam:
				newParams := copyParams(params)
				newParams[child.label] = seg
				search(child, nextPathIdx, newParams)
				if bestMatch != nil {
					return
				}
			case nodeWildcard:
				newParams := copyParams(params)
				rest := path[pathIdx:]
				if child.label != "" {
					newParams[child.label] = rest
				}
				if child.entry != nil {
					bestMatch = &matchResult{
						Entry:  child.entry,
						Params: newParams,
						Rest:   "/" + rest,
					}
					return
				}
			}
		}
	}

	search(t.root, 0, make(map[string]string))
	return bestMatch
}

func copyParams(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
