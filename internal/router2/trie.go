package router2

import (
	"sort"
	"strings"
)

// nodeType defines the type of a trie node.
type nodeType int

const (
	nodeStatic   nodeType = iota // Static segment (exact match)
	nodeParam                    // Parameter segment (e.g., :id)
	nodeWildcard                 // Wildcard segment (e.g., *rest)
)

// node represents a single node in the route trie.
type node struct {
	typ      nodeType
	label    string // The segment label (without prefix like : or *)
	prefix   string // The original prefix (e.g., ":", "*", or the static segment)
	parent   *node
	children []*node
	entry    *routeEntry // Route entry if this node is a terminal
}

// RouterTrie implements a prefix tree for efficient route matching.
// Supports static segments, parameters (:id), and wildcards (*rest).
type RouterTrie struct {
	root *node
}

// NewRouterTrie creates a new empty router trie.
func NewRouterTrie() *RouterTrie {
	return &RouterTrie{
		root: &node{
			typ: nodeStatic,
		},
	}
}

// Insert adds a route pattern and its entry to the trie.
// Patterns are split into segments and inserted as a path from root to leaf.
// Priority: static > param > wildcard (ensured by sorting children).
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

// MatchResult contains the result of a successful route match.
type MatchResult struct {
	Entry   *routeEntry       // The matched route entry
	Params  map[string]string // Extracted route parameters
	Pattern string            // The matched pattern
	Path    string            // The matched path
	Rest    string            // Remaining path for wildcard matches
}

// Match attempts to match a path against the trie.
// Returns MatchResult if a route matches, nil otherwise.
// Priority: static > param > wildcard (first match wins).
func (t *RouterTrie) Match(path string) *MatchResult {

	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var bestMatch *MatchResult

	var search func(n *node, pathIdx int, params map[string]string)
	search = func(n *node, pathIdx int, params map[string]string) {

		if bestMatch != nil {
			return
		}

		if pathIdx >= len(path) {
			if n.entry != nil {
				bestMatch = &MatchResult{
					Entry:   n.entry,
					Params:  params,
					Pattern: n.entry.pattern,
					Path:    path,
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
					Entry:   n.entry,
					Params:  params,
					Pattern: n.entry.pattern,
					Path:    path,
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
						Entry:   child.entry,
						Params:  newParams,
						Pattern: child.entry.pattern,
						Path:    path,
						Rest:    "/" + rest,
					}
					return
				}
			}
		}
	}

	search(t.root, 0, make(map[string]string))
	return bestMatch
}

// copyParams creates a shallow copy of a params map.
// Used during recursive matching to avoid mutation.
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
