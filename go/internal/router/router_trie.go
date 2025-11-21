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

type RouterTrie struct {
	root *node
}

func NewRouterTrie() *RouterTrie {
	return &RouterTrie{
		root: &node{
			typ: nodeStatic,
		},
	}
}

func (t *RouterTrie) Insert(pattern string, entry routeEntry) {

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

type MatchResult struct {
	Entry  *routeEntry
	Params map[string]string
	Rest   string
}

func (t *RouterTrie) Match(path string) *MatchResult {

	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var bestMatch *MatchResult

	// Recursive search
	// pathIdx is the current position in the path string
	var search func(n *node, pathIdx int, params map[string]string)
	search = func(n *node, pathIdx int, params map[string]string) {
		if bestMatch != nil {
			return
		}

		if pathIdx >= len(path) {

			if n.entry != nil {
				bestMatch = &MatchResult{
					Entry:  n.entry,
					Params: params,
				}
			}
			return
		}

		if path[pathIdx] == '/' {
			pathIdx++
		}

		if pathIdx >= len(path) {

			if n.entry != nil {
				bestMatch = &MatchResult{
					Entry:  n.entry,
					Params: params,
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
					bestMatch = &MatchResult{
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
