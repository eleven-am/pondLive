package pkg

func (r *InputRef) Select() { r.ref.Call("select") }

func (r *InputRef) SetSelectionRange(start, end int, direction ...string) {
	if len(direction) > 0 {
		r.ref.Call("setSelectionRange", start, end, direction[0])
		return
	}
	r.ref.Call("setSelectionRange", start, end)
}

func (r *InputRef) SetRangeText(text string, startEnd ...int) {
	if len(startEnd) >= 2 {
		r.ref.Call("setRangeText", text, startEnd[0], startEnd[1])
		return
	}
	r.ref.Call("setRangeText", text)
}

func (r *InputRef) Selection(fn func(start, end int, direction string, err error)) {
	if fn == nil {
		return
	}
	r.ref.Query(func(values map[string]any, err error) {
		if err != nil {
			fn(0, 0, "", err)
			return
		}
		fn(toInt(values["selectionStart"]), toInt(values["selectionEnd"]), asString(values["selectionDirection"]), nil)
	}, "selectionStart", "selectionEnd", "selectionDirection")
}
