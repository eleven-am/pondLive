package runtime

import (
	"bytes"
	"strings"

	"github.com/eleven-am/pondlive/internal/htmlutil"
	"github.com/eleven-am/pondlive/internal/protocol"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func normalizeRaw(tag, src string) string {
	if src == "" {
		return ""
	}
	context := &html.Node{Type: html.ElementNode, Data: tag, DataAtom: atom.Lookup([]byte(tag))}
	nodes, err := html.ParseFragment(strings.NewReader(src), context)
	if err != nil {
		var sb bytes.Buffer
		sb.Write(htmlutil.AppendText(nil, src))
		return sb.String()
	}
	var buf bytes.Buffer
	for _, n := range nodes {
		stripReserved(n)
		if n.Type == html.CommentNode && protocol.IsMarker(n.Data) {
			continue
		}
		_ = html.Render(&buf, n)
	}
	return buf.String()
}

func stripReserved(n *html.Node) {
	if n.Type == html.ElementNode {
		kept := n.Attr[:0]
		for _, a := range n.Attr {
			if !protocol.IsReservedAttr(a.Key) {
				kept = append(kept, a)
			}
		}
		n.Attr = kept
	}
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == html.CommentNode && protocol.IsMarker(c.Data) {
			n.RemoveChild(c)
		} else {
			stripReserved(c)
		}
		c = next
	}
}
