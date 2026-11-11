package htmlutil

import (
	"slices"
	"strings"
)

var voidTags = []string{
	"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr", "keygen", "command",
}

func IsVoid(tag string) bool { return slices.Contains(voidTags, tag) }

func EscapeClose(s string) string { return string(appendClosing(nil, "</", s)) }

func AppendRawText(dst []byte, tag, s string) []byte { return appendClosing(dst, "</"+tag, s) }

func appendClosing(dst []byte, closer, s string) []byte {
	for {
		i := IndexFold(s, closer)
		if i < 0 {
			return append(dst, s...)
		}
		dst = append(dst, s[:i]...)
		dst = append(dst, "<\\/"...)
		dst = append(dst, s[i+2:i+len(closer)]...)
		s = s[i+len(closer):]
	}
}

func AppendText(dst []byte, s string) []byte { return appendEscaped(dst, s, false) }

func AppendAttr(dst []byte, s string) []byte { return appendEscaped(dst, s, true) }

func appendEscaped(dst []byte, s string, quotes bool) []byte {
	start := 0
	for i := 0; i < len(s); i++ {
		var rep string
		switch s[i] {
		case '&':
			rep = "&amp;"
		case '"':
			if !quotes {
				continue
			}
			rep = "&quot;"
		case '<':
			rep = "&lt;"
		case '>':
			rep = "&gt;"
		default:
			continue
		}
		dst = append(dst, s[start:i]...)
		dst = append(dst, rep...)
		start = i + 1
	}
	return append(dst, s[start:]...)
}

func IndexFold(s, substr string) int {
	n := len(substr)
	for i := 0; i+n <= len(s); i++ {
		if strings.EqualFold(s[i:i+n], substr) {
			return i
		}
	}
	return -1
}

func LastIndexFold(s, substr string) int {
	n := len(substr)
	if n == 0 || len(s) < n {
		return -1
	}
	for i := len(s) - n; i >= 0; i-- {
		if strings.EqualFold(s[i:i+n], substr) {
			return i
		}
	}
	return -1
}

func UniqueClasses(s string) string {
	tokens := strings.Fields(s)
	kept := tokens[:0:0]
	for _, t := range tokens {
		if !slices.Contains(kept, t) {
			kept = append(kept, t)
		}
	}
	if len(kept) == len(tokens) {
		return s
	}
	return strings.Join(kept, " ")
}
