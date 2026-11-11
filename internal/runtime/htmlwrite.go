package runtime

import (
	"strconv"
	"strings"

	"github.com/eleven-am/pondlive/internal/htmlutil"
	"github.com/eleven-am/pondlive/internal/protocol"
)

type skelEntry struct {
	name   string
	parent int
}

type writer struct {
	buf   []byte
	skel  []skelEntry
	stack []int
	rec   bool
	tags  []string
	nl    bool
}

func newWriter(rec bool) *writer {
	return &writer{rec: rec}
}

func (w *writer) String() string { return string(w.buf) }

func (w *writer) raw(s string) { w.buf = append(w.buf, s...) }

func (w *writer) byte(b byte) { w.buf = append(w.buf, b) }

func (w *writer) uint(n uint32) { w.buf = strconv.AppendUint(w.buf, uint64(n), 10) }

func (w *writer) parentTag() string {
	if len(w.tags) == 0 {
		return ""
	}
	return w.tags[len(w.tags)-1]
}

func (w *writer) parentIndex() int {
	if len(w.stack) == 0 {
		return -1
	}
	return w.stack[len(w.stack)-1]
}

func (w *writer) enter(tag string) {
	w.tags = append(w.tags, tag)
	if w.rec {
		w.skel = append(w.skel, skelEntry{name: tag, parent: w.parentIndex()})
		w.stack = append(w.stack, len(w.skel)-1)
	}
}

func (w *writer) leave() {
	w.tags = w.tags[:len(w.tags)-1]
	if w.rec {
		w.stack = w.stack[:len(w.stack)-1]
	}
}

func (w *writer) marker(name string) {
	w.nl = false
	if w.rec {
		w.skel = append(w.skel, skelEntry{name: name, parent: w.parentIndex()})
	}
}

func (w *writer) openTag(tag string) {
	w.nl = false
	w.byte('<')
	w.raw(tag)
	w.enter(tag)
}

func (w *writer) closeOpen() { w.byte('>') }

func (w *writer) closeTag(tag string) {
	w.raw("</")
	w.raw(tag)
	w.byte('>')
	w.leave()
}

func (w *writer) attr(name, value string) {
	w.byte(' ')
	w.raw(name)
	if strings.IndexByte(value, '"') >= 0 && strings.IndexByte(value, '\'') < 0 {
		w.raw(`='`)
		w.buf = htmlutil.AppendText(w.buf, value)
		w.byte('\'')
		return
	}
	w.raw(`="`)
	w.buf = htmlutil.AppendAttr(w.buf, value)
	w.byte('"')
}

func (w *writer) attrUint(name string, v uint32) {
	w.byte(' ')
	w.raw(name)
	w.raw(`="`)
	w.uint(v)
	w.byte('"')
}

func (w *writer) commentMarker(kind string, id uint32) {
	w.raw("<!--")
	w.raw(kind)
	w.byte(':')
	w.uint(id)
	w.raw("-->")
	if w.rec && kind != protocol.MarkerRegionEnd {
		w.marker(protocol.Marker(kind, id))
	}
}

func (w *writer) regionStart(id uint32) {
	w.commentMarker(protocol.MarkerRegion, id)
}

func (w *writer) regionEnd(id uint32) {
	w.commentMarker(protocol.MarkerRegionEnd, id)
}

func (w *writer) textMarker(id uint32) {
	w.commentMarker(protocol.MarkerText, id)
}

func isVoid(tag string) bool { return htmlutil.IsVoid(tag) }

func isRawText(tag string) bool { return tag == "style" || tag == "script" }

func (w *writer) text(parentTag, s string) {
	if w.nl && strings.HasPrefix(s, "\n") {
		w.byte('\n')
	}
	w.nl = false
	if isRawText(parentTag) {
		w.buf = htmlutil.AppendRawText(w.buf, parentTag, s)
		return
	}
	w.buf = htmlutil.AppendText(w.buf, s)
}

func sanitizeComment(s string) string {
	if protocol.IsMarker(s) {
		s = " " + s
	}
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "- -")
	}
	if strings.HasPrefix(s, ">") || strings.HasPrefix(s, "->") {
		s = " " + s
	}
	if strings.HasSuffix(s, "-") {
		s += " "
	}
	return strings.ReplaceAll(s, "<!-", "< !-")
}
