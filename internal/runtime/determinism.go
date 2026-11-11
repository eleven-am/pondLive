package runtime

import (
	"fmt"
	"strings"

	"github.com/eleven-am/pondlive/internal/protocol"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type SelfCheckError struct {
	Where  string
	Offset int
	Mount  string
	Serial string
}

func (e *SelfCheckError) Error() string {
	return fmt.Sprintf("runtime: %s: serialize output differs from what was sent at byte %d: sent %q, serialized %q",
		e.Where, e.Offset, e.Mount, e.Serial)
}

type ParityError struct {
	Where string
	Index int
	Want  string
	Got   string
}

func (e *ParityError) Error() string {
	return fmt.Sprintf("runtime: %s: the browser parser would reshape this markup at node %d: emitted %s, parsed %s",
		e.Where, e.Index, e.Want, e.Got)
}

func firstDiff(a, b string) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func snippet(s string, at int) string {
	lo := max(at-24, 0)
	hi := min(at+24, len(s))
	return s[lo:hi]
}

func (s *Session) selfCheckDocument(html string, w *writer) error {
	if !s.devMode {
		return nil
	}
	if err := parityCheck("document", "", html, w.skel); err != nil {
		return err
	}
	if s.hasWork() {
		return nil
	}
	again, err := s.DocumentHTML()
	if err != nil {
		return err
	}
	if again != html {
		at := firstDiff(html, again)
		return &SelfCheckError{Where: "mount self-check", Offset: at, Mount: snippet(html, at), Serial: snippet(again, at)}
	}
	return nil
}

func (us *unitSet[K]) checkUnit(w *writer, sent string) {
	if err := parityCheck(us.kd.String()+" unit", us.tag, sent, w.skel); err != nil {
		panic(err)
	}
}

func (us *unitSet[K]) selfCheck(r *unitRow[K], sent string) {
	if !us.s.devMode {
		return
	}
	var again string
	defer us.s.ensureFeedCaches()()
	us.s.graph.Serialize(func() {
		w := newWriter(false)
		w.tags = append(w.tags, us.tag)
		us.serializeRow(w, r)
		again = w.String()
	})
	if again != sent {
		at := firstDiff(sent, again)
		panic(&SelfCheckError{Where: us.kd.String() + " unit self-check", Offset: at, Mount: snippet(sent, at), Serial: snippet(again, at)})
	}
}

func parityCheck(where, contextTag, markup string, want []skelEntry) error {
	var roots []*html.Node
	lower := strings.ToLower(strings.TrimSpace(markup))
	switch {
	case contextTag == "" && strings.HasPrefix(lower, "<html"), contextTag == "" && strings.HasPrefix(lower, "<!doctype"):
		doc, err := html.Parse(strings.NewReader(markup))
		if err != nil {
			return err
		}
		roots = []*html.Node{doc}
	default:
		tag := contextTag
		if tag == "" {
			tag = "body"
		}
		ctxNode := &html.Node{Type: html.ElementNode, Data: tag, DataAtom: atom.Lookup([]byte(tag))}
		nodes, err := html.ParseFragment(strings.NewReader(markup), ctxNode)
		if err != nil {
			return err
		}
		roots = nodes
	}
	var got []skelEntry
	index := make(map[*html.Node]int)
	var walk func(n *html.Node, parent int)
	walk = func(n *html.Node, parent int) {
		next := parent
		switch n.Type {
		case html.ElementNode:
			got = append(got, skelEntry{name: n.Data, parent: parent})
			next = len(got) - 1
			index[n] = next
			if _, raw := attrOfNode(n, protocol.AttrRaw); raw {
				return
			}
		case html.CommentNode:
			if protocol.IsMarker(n.Data) && !strings.HasPrefix(n.Data, "/") {
				got = append(got, skelEntry{name: n.Data, parent: parent})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, next)
		}
	}
	for _, r := range roots {
		walk(r, -1)
	}
	for i := 0; i < len(want) || i < len(got); i++ {
		switch {
		case i >= len(want):
			return &ParityError{Where: where, Index: i, Want: "end", Got: got[i].name}
		case i >= len(got):
			return &ParityError{Where: where, Index: i, Want: want[i].name, Got: "end"}
		case want[i] != got[i]:
			return &ParityError{Where: where, Index: i,
				Want: fmt.Sprintf("%s in %d", want[i].name, want[i].parent),
				Got:  fmt.Sprintf("%s in %d", got[i].name, got[i].parent)}
		}
	}
	return nil
}

func attrOfNode(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}
