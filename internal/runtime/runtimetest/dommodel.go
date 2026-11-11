package runtimetest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/eleven-am/pondlive/internal/htmlutil"
	"github.com/eleven-am/pondlive/internal/protocol"
)

type Model struct {
	root  *html.Node
	frag  *html.Node
	nodes map[uint32]*html.Node
	start map[uint32]*html.Node
	end   map[uint32]*html.Node
}

type EditError struct {
	Edit   protocol.Edit
	Detail string
}

func (e *EditError) Error() string {
	return fmt.Sprintf("edit %s id=%d: %s", e.Edit.Op, e.Edit.ID, e.Detail)
}

func NewModel(markup string) (*Model, error) {
	m := &Model{}
	lower := strings.ToLower(strings.TrimSpace(markup))
	if strings.HasPrefix(lower, "<html") || strings.HasPrefix(lower, "<!doctype") {
		doc, err := html.Parse(strings.NewReader(markup))
		if err != nil {
			return nil, err
		}
		m.root = doc
	} else {
		body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
		nodes, err := html.ParseFragment(strings.NewReader(markup), body)
		if err != nil {
			return nil, err
		}
		for _, n := range nodes {
			body.AppendChild(n)
		}
		doc := &html.Node{Type: html.DocumentNode}
		doc.AppendChild(body)
		m.root = doc
		m.frag = body
	}
	return m, m.Index()
}

func (m *Model) Index() error {
	m.nodes = map[uint32]*html.Node{}
	m.start = map[uint32]*html.Node{}
	m.end = map[uint32]*html.Node{}
	return m.indexSubtree(m.root)
}

func (m *Model) indexSubtree(root *html.Node) error {
	var walk func(n *html.Node) error
	walk = func(n *html.Node) error {
		switch n.Type {
		case html.ElementNode:
			if v, ok := attrOf(n, protocol.AttrID); ok {
				id, err := parseID(v)
				if err != nil {
					return fmt.Errorf("invalid data-pi %q", v)
				}
				if _, dup := m.nodes[id]; dup {
					return fmt.Errorf("duplicate id %d", id)
				}
				m.nodes[id] = n
			}
			if _, raw := attrOf(n, protocol.AttrRaw); raw {
				return nil
			}
		case html.CommentNode:
			if kind, id, ok := marker(n.Data); ok {
				switch kind {
				case protocol.MarkerText:
					text := n.NextSibling
					if text == nil || text.Type != html.TextNode {
						text = &html.Node{Type: html.TextNode}
						n.Parent.InsertBefore(text, n.NextSibling)
					}
					if _, dup := m.nodes[id]; dup {
						return fmt.Errorf("duplicate id %d", id)
					}
					m.nodes[id] = text
				case protocol.MarkerRegion:
					if _, dup := m.nodes[id]; dup {
						return fmt.Errorf("duplicate id %d", id)
					}
					m.start[id] = n
					m.nodes[id] = n
				case "/r":
					m.end[id] = n
				}
			}
		}
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			if err := walk(c); err != nil {
				return err
			}
			c = next
		}
		return nil
	}
	return walk(root)
}

func parseID(v string) (uint32, error) {
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("bad id")
	}
	return uint32(n), nil
}

func marker(data string) (kind string, id uint32, ok bool) {
	kind, id, ok = protocol.ParseMarker(data)
	if !ok || id == 0 {
		return "", 0, false
	}
	return kind, id, true
}

func attrOf(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func setAttr(n *html.Node, key, val string) {
	for i := range n.Attr {
		if n.Attr[i].Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func delAttr(n *html.Node, key string) {
	for i := range n.Attr {
		if n.Attr[i].Key == key {
			n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
			return
		}
	}
}

func (m *Model) Has(id uint32) bool {
	_, ok := m.nodes[id]
	return ok
}

func (m *Model) IDs() []uint32 {
	ids := make([]uint32, 0, len(m.nodes))
	for id := range m.nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (m *Model) Node(id uint32) *html.Node { return m.nodes[id] }

func (m *Model) unitNodes(id uint32) ([]*html.Node, error) {
	if s, ok := m.start[id]; ok {
		e, ok := m.end[id]
		if !ok || e.Parent != s.Parent {
			return nil, fmt.Errorf("region %d has no matching end", id)
		}
		var out []*html.Node
		for n := s; n != nil; n = n.NextSibling {
			out = append(out, n)
			if n == e {
				return out, nil
			}
		}
		return nil, fmt.Errorf("region %d end precedes start", id)
	}
	n := m.nodes[id]
	if n == nil || n.Type != html.ElementNode {
		return nil, fmt.Errorf("unit %d is not an element or region", id)
	}
	return []*html.Node{n}, nil
}

func (m *Model) forget(nodes []*html.Node) {
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for id, x := range m.nodes {
			if x == n {
				delete(m.nodes, id)
				delete(m.start, id)
				delete(m.end, id)
			}
		}
		if n.Type == html.CommentNode {
			if kind, id, ok := marker(n.Data); ok && kind == "/r" {
				delete(m.end, id)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for _, n := range nodes {
		walk(n)
	}
}

func (m *Model) regionInfo(region uint32) (parent, end *html.Node, err error) {
	s, ok := m.start[region]
	if !ok {
		return nil, nil, fmt.Errorf("region %d is unknown", region)
	}
	e, ok := m.end[region]
	if !ok || e.Parent != s.Parent {
		return nil, nil, fmt.Errorf("region %d has no matching end", region)
	}
	return s.Parent, e, nil
}

func (m *Model) beforeFor(region, ref uint32) (*html.Node, error) {
	_, end, err := m.regionInfo(region)
	if err != nil {
		return nil, err
	}
	if ref == 0 {
		return end, nil
	}
	nodes, err := m.unitNodes(ref)
	if err != nil {
		return nil, err
	}
	first := nodes[0]
	inside := false
	for n := m.start[region].NextSibling; n != nil && n != end; n = n.NextSibling {
		if n == first {
			inside = true
			break
		}
	}
	if !inside {
		return nil, fmt.Errorf("ref %d is not a unit of region %d", ref, region)
	}
	return first, nil
}

func (m *Model) Apply(edits []protocol.Edit) error {
	for _, e := range edits {
		if err := m.applyOne(e); err != nil {
			return &EditError{Edit: e, Detail: err.Error()}
		}
	}
	return nil
}

func (m *Model) applyOne(e protocol.Edit) error {
	switch e.Op {
	case protocol.OpText:
		n := m.nodes[e.ID]
		switch {
		case n == nil:
			return fmt.Errorf("unresolved")
		case n.Type == html.TextNode:
			n.Data = e.Value
		case n.Type == html.ElementNode:
			for c := n.FirstChild; c != nil; c = n.FirstChild {
				m.forget([]*html.Node{c})
				n.RemoveChild(c)
			}
			if e.Value != "" {
				n.AppendChild(&html.Node{Type: html.TextNode, Data: e.Value})
			}
		default:
			return fmt.Errorf("not a text slot or element")
		}
	case protocol.OpAttr:
		n := m.element(e.ID)
		if n == nil {
			return fmt.Errorf("unresolved element")
		}
		setAttr(n, e.Name, e.Value)
		if n.Data == "select" && (e.Name == "value" || e.Name == protocol.AttrSelection) {
			m.selectOptions(n, e.Name, e.Value)
		}
		if n.Data == "textarea" && e.Name == "value" {
			m.setBody(n, e.Value)
		}
	case protocol.OpRemoveAttr:
		n := m.element(e.ID)
		if n == nil {
			return fmt.Errorf("unresolved element")
		}
		delAttr(n, e.Name)
	case protocol.OpStyle:
		n := m.element(e.ID)
		if n == nil {
			return fmt.Errorf("unresolved element")
		}
		setStyle(n, e.Name, e.Value)
	case protocol.OpHTML:
		n := m.element(e.ID)
		if n == nil {
			return fmt.Errorf("unresolved element")
		}
		if _, raw := attrOf(n, protocol.AttrRaw); !raw {
			return fmt.Errorf("not a raw host")
		}
		for c := n.FirstChild; c != nil; c = n.FirstChild {
			n.RemoveChild(c)
		}
		nodes, err := html.ParseFragment(strings.NewReader(e.Value), n)
		if err != nil {
			return err
		}
		for _, c := range nodes {
			n.AppendChild(c)
		}
	case protocol.OpInsert:
		parent, _, err := m.regionInfo(e.ID)
		if err != nil {
			return err
		}
		before, err := m.beforeFor(e.ID, e.Ref)
		if err != nil {
			return err
		}
		nodes, err := html.ParseFragment(strings.NewReader(e.Value), parent)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			return fmt.Errorf("empty unit")
		}
		holder := &html.Node{Type: html.ElementNode, Data: "template"}
		for _, c := range nodes {
			holder.AppendChild(c)
		}
		if err := m.indexSubtree(holder); err != nil {
			return err
		}
		for c := holder.FirstChild; c != nil; c = holder.FirstChild {
			holder.RemoveChild(c)
			parent.InsertBefore(c, before)
		}
	case protocol.OpMove:
		nodes, err := m.unitNodes(e.ID)
		if err != nil {
			return err
		}
		parent, _, err := m.regionInfo(e.Region)
		if err != nil {
			return err
		}
		if e.Ref == e.ID {
			return fmt.Errorf("unit moved before itself")
		}
		before, err := m.beforeFor(e.Region, e.Ref)
		if err != nil {
			return err
		}
		if nodes[0].Parent != parent {
			return fmt.Errorf("unit is not in region %d", e.Region)
		}
		for _, n := range nodes {
			parent.RemoveChild(n)
		}
		for _, n := range nodes {
			parent.InsertBefore(n, before)
		}
	case protocol.OpRemove:
		nodes, err := m.unitNodes(e.ID)
		if err != nil {
			return err
		}
		m.forget(nodes)
		for _, n := range nodes {
			n.Parent.RemoveChild(n)
		}
	case protocol.OpClear:
		parent, end, err := m.regionInfo(e.ID)
		if err != nil {
			return err
		}
		for n := m.start[e.ID].NextSibling; n != nil && n != end; {
			next := n.NextSibling
			m.forget([]*html.Node{n})
			parent.RemoveChild(n)
			n = next
		}
	default:
		return fmt.Errorf("unknown op")
	}
	return nil
}

func (m *Model) element(id uint32) *html.Node {
	n := m.nodes[id]
	if n == nil || n.Type != html.ElementNode {
		return nil
	}
	return n
}

func parseStyle(v string) [][2]string {
	var out [][2]string
	for _, part := range strings.Split(v, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, val, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		out = append(out, [2]string{strings.TrimSpace(k), strings.TrimSpace(val)})
	}
	return out
}

func setStyle(n *html.Node, prop, value string) {
	cur, _ := attrOf(n, "style")
	decls := parseStyle(cur)
	found := -1
	for i, d := range decls {
		if d[0] == prop {
			found = i
		}
	}
	switch {
	case value == "" && found >= 0:
		decls = append(decls[:found], decls[found+1:]...)
	case value == "":
	case found >= 0:
		decls[found][1] = value
	default:
		decls = append(decls, [2]string{prop, value})
	}
	var parts []string
	for _, d := range decls {
		parts = append(parts, d[0]+": "+d[1]+";")
	}
	setAttr(n, "style", strings.Join(parts, " "))
}

func (m *Model) top() *html.Node {
	if m.frag != nil {
		return m.frag
	}
	return m.root
}

func (m *Model) HTML() string {
	var buf bytes.Buffer
	for c := m.top().FirstChild; c != nil; c = c.NextSibling {
		writeCanonical(&buf, c, false)
	}
	return buf.String()
}

func (m *Model) Normalized() string {
	var buf bytes.Buffer
	for c := m.top().FirstChild; c != nil; c = c.NextSibling {
		writeCanonical(&buf, c, true)
	}
	return buf.String()
}

func Canonical(markup string) (string, error) {
	m, err := NewModel(markup)
	if err != nil {
		return "", err
	}
	return m.HTML(), nil
}

func NormalizedOf(markup string) (string, error) {
	m, err := NewModel(markup)
	if err != nil {
		return "", err
	}
	return m.Normalized(), nil
}

func writeCanonical(buf *bytes.Buffer, n *html.Node, strip bool) {
	switch n.Type {
	case html.TextNode:
		if n.Data == "" {
			return
		}
		parent := ""
		if n.Parent != nil {
			parent = n.Parent.Data
		}
		if parent == "script" || parent == "style" {
			buf.Write(htmlutil.AppendRawText(nil, parent, n.Data))
			return
		}
		buf.Write(htmlutil.AppendText(nil, n.Data))
	case html.CommentNode:
		if strip {
			if _, _, ok := marker(n.Data); ok {
				return
			}
		}
		buf.WriteString("<!--" + n.Data + "-->")
	case html.ElementNode:
		buf.WriteByte('<')
		buf.WriteString(n.Data)
		attrs := append([]html.Attribute(nil), n.Attr...)
		sort.Slice(attrs, func(i, j int) bool { return attrs[i].Key < attrs[j].Key })
		for _, a := range attrs {
			if strip && a.Key == protocol.AttrID {
				continue
			}
			buf.WriteString(" " + a.Key + `="`)
			buf.Write(htmlutil.AppendAttr(nil, a.Val))
			buf.WriteByte('"')
		}
		buf.WriteByte('>')
		if htmlutil.IsVoid(n.Data) {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeCanonical(buf, c, strip)
		}
		buf.WriteString("</" + n.Data + ">")
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeCanonical(buf, c, strip)
		}
	}
}

func (m *Model) selectOptions(sel *html.Node, name, value string) {
	wanted := map[string]bool{}
	if name == "value" {
		wanted[value] = true
	} else {
		var list []string
		_ = json.Unmarshal([]byte(value), &list)
		for _, v := range list {
			wanted[v] = true
		}
	}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "option" {
				v := optionText(c)
				for _, a := range c.Attr {
					if a.Key == "value" {
						v = a.Val
					}
				}
				if wanted[v] {
					setAttr(c, "selected", "")
				} else {
					delAttr(c, "selected")
				}
			}
			walk(c)
		}
	}
	walk(sel)
}

func optionText(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	}
	return b.String()
}

func (m *Model) setBody(n *html.Node, text string) {
	for c := n.FirstChild; c != nil; c = n.FirstChild {
		m.forget([]*html.Node{c})
		n.RemoveChild(c)
	}
	if text != "" {
		n.AppendChild(&html.Node{Type: html.TextNode, Data: text})
	}
}
