package protocol

import (
	"strconv"
	"unicode/utf8"
)

type Op string

const (
	OpText       Op = "t"
	OpAttr       Op = "a"
	OpRemoveAttr Op = "ra"
	OpStyle      Op = "st"
	OpHTML       Op = "h"
	OpInsert     Op = "ins"
	OpMove       Op = "mv"
	OpRemove     Op = "rm"
	OpClear      Op = "clr"
)

type Edit struct {
	Op     Op
	ID     uint32
	Ref    uint32
	Region uint32
	Name   string
	Value  string
}

func TextEdit(id uint32, text string) Edit { return Edit{Op: OpText, ID: id, Value: text} }

func AttrEdit(id uint32, name, value string) Edit {
	return Edit{Op: OpAttr, ID: id, Name: name, Value: value}
}

func RemoveAttrEdit(id uint32, name string) Edit { return Edit{Op: OpRemoveAttr, ID: id, Name: name} }

func StyleEdit(id uint32, prop, value string) Edit {
	return Edit{Op: OpStyle, ID: id, Name: prop, Value: value}
}

func HTMLEdit(id uint32, html string) Edit { return Edit{Op: OpHTML, ID: id, Value: html} }

func InsertEdit(region, ref uint32, html string) Edit {
	return Edit{Op: OpInsert, ID: region, Ref: ref, Value: html}
}

func MoveEdit(unit, region, ref uint32) Edit {
	return Edit{Op: OpMove, ID: unit, Region: region, Ref: ref}
}

func RemoveEdit(unit uint32) Edit { return Edit{Op: OpRemove, ID: unit} }

func ClearEdit(region uint32) Edit { return Edit{Op: OpClear, ID: region} }

func (e Edit) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, 24+len(e.Name)+len(e.Value))
	b = append(b, '[')
	b = AppendJSONString(b, string(e.Op))
	b = append(b, ',')
	b = strconv.AppendUint(b, uint64(e.ID), 10)
	switch e.Op {
	case OpText, OpHTML:
		b = append(b, ',')
		b = AppendJSONString(b, e.Value)
	case OpAttr, OpStyle:
		b = append(b, ',')
		b = AppendJSONString(b, e.Name)
		b = append(b, ',')
		b = AppendJSONString(b, e.Value)
	case OpRemoveAttr:
		b = append(b, ',')
		b = AppendJSONString(b, e.Name)
	case OpInsert:
		b = append(b, ',')
		b = strconv.AppendUint(b, uint64(e.Ref), 10)
		b = append(b, ',')
		b = AppendJSONString(b, e.Value)
	case OpMove:
		b = append(b, ',')
		b = strconv.AppendUint(b, uint64(e.Region), 10)
		b = append(b, ',')
		b = strconv.AppendUint(b, uint64(e.Ref), 10)
	}
	return append(b, ']'), nil
}

const hexDigits = "0123456789abcdef"

func AppendJSONString(b []byte, s string) []byte {
	b = append(b, '"')
	start := 0
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			if c >= 0x20 && c != '"' && c != '\\' {
				i++
				continue
			}
			b = append(b, s[start:i]...)
			switch c {
			case '"', '\\':
				b = append(b, '\\', c)
			case '\n':
				b = append(b, '\\', 'n')
			case '\r':
				b = append(b, '\\', 'r')
			case '\t':
				b = append(b, '\\', 't')
			default:
				b = append(b, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			}
			i++
			start = i
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			b = append(b, s[start:i]...)
			b = append(b, `�`...)
			i += size
			start = i
			continue
		}
		i += size
	}
	b = append(b, s[start:]...)
	return append(b, '"')
}
