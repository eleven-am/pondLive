package pkg

func (r *FormRef) Submit() { r.ref.Call("submit") }

func (r *FormRef) Reset() { r.ref.Call("reset") }

func (r *FormRef) RequestSubmit() { r.ref.Call("requestSubmit") }

func (r *FormRef) CheckValidity(fn func(bool, error)) {
	result(r.ElementRef, "checkValidity", asBool, fn)
}

func (r *FormRef) ReportValidity(fn func(bool, error)) {
	result(r.ElementRef, "reportValidity", asBool, fn)
}

func (r *DialogRef) Show() { r.ref.Call("show") }

func (r *DialogRef) ShowModal() { r.ref.Call("showModal") }

func (r *DialogRef) Close(returnValue ...string) {
	if len(returnValue) > 0 {
		r.ref.Call("close", returnValue[0])
		return
	}
	r.ref.Call("close")
}

func (r *DialogRef) Open(fn func(bool, error)) { property(r.ElementRef, "open", asBool, fn) }

func (r *DialogRef) ReturnValue(fn func(string, error)) {
	property(r.ElementRef, "returnValue", asString, fn)
}

func (r *CanvasRef) ToDataURL(fn func(string, error), mimeType ...string) {
	args := make([]any, 0, 1)
	if len(mimeType) > 0 {
		args = append(args, mimeType[0])
	}
	result(r.ElementRef, "toDataURL", asString, fn, args...)
}

func (r *CanvasRef) ToDataURLWithQuality(fn func(string, error), mimeType string, quality float64) {
	result(r.ElementRef, "toDataURL", asString, fn, mimeType, quality)
}

func (r *CanvasRef) Width(fn func(int, error)) { property(r.ElementRef, "width", toInt, fn) }

func (r *CanvasRef) Height(fn func(int, error)) { property(r.ElementRef, "height", toInt, fn) }

func (r *CanvasRef) SetWidth(width int) { r.ref.Set("width", width) }

func (r *CanvasRef) SetHeight(height int) { r.ref.Set("height", height) }

func (r *DetailsRef) SetOpen(open bool) { r.ref.Set("open", open) }

func (r *DetailsRef) Open(fn func(bool, error)) { property(r.ElementRef, "open", asBool, fn) }

func (r *DetailsRef) Toggle() {
	r.Open(func(open bool, err error) {
		if err == nil {
			r.SetOpen(!open)
		}
	})
}
