package css

import "strings"

func render(ss *Stylesheet) string {
	if ss == nil {
		return ""
	}
	var b strings.Builder
	block := func(sel string, decls []Declaration) {
		b.WriteString(sel)
		b.WriteByte('{')
		for i, d := range decls {
			if i > 0 {
				b.WriteByte(';')
			}
			b.WriteString(d.Property)
			b.WriteByte(':')
			b.WriteString(d.Value)
		}
		b.WriteByte('}')
	}
	for _, r := range ss.Rules {
		block(r.Selector, r.Decls)
	}
	for _, m := range ss.MediaRules {
		b.WriteString("@media ")
		b.WriteString(m.Query)
		b.WriteByte('{')
		for _, r := range m.Rules {
			block(r.Selector, r.Decls)
		}
		b.WriteByte('}')
	}
	for _, kf := range ss.Keyframes {
		b.WriteString("@keyframes ")
		b.WriteString(kf.Name)
		b.WriteByte('{')
		for _, step := range kf.Steps {
			block(step.Selector, step.Decls)
		}
		b.WriteByte('}')
	}
	for _, other := range ss.OtherBlocks {
		b.WriteString(other)
	}
	return b.String()
}
