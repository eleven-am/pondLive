package protocol

type RefKind struct {
	Methods []string
	Set     []string
	Query   []string
	Observe []string
}

const BaseRefKind = "ElementRef"

var RefKinds = map[string]RefKind{
	"ElementRef": {
		Methods: []string{"blur", "checkVisibility", "click", "focus", "getComputedStyle", "matches", "scrollIntoView"},
		Set:     []string{"scrollLeft", "scrollTop"},
		Observe: []string{"focused", "rect", "scroll"},
	},
	"InputRef": {
		Methods: []string{"select", "setRangeText", "setSelectionRange"},
		Query:   []string{"selectionDirection", "selectionEnd", "selectionStart"},
	},
	"FormRef": {
		Methods: []string{"checkValidity", "reportValidity", "requestSubmit", "reset", "submit"},
	},
	"MediaRef": {
		Methods: []string{"load", "pause", "play"},
		Set:     []string{"currentTime", "muted", "playbackRate", "volume"},
		Query:   []string{"networkState", "playbackRate", "readyState"},
		Observe: []string{"currentTime", "duration", "ended", "muted", "paused", "volume"},
	},
	"DialogRef": {
		Methods: []string{"close", "show", "showModal"},
		Query:   []string{"open", "returnValue"},
	},
	"CanvasRef": {
		Methods: []string{"toDataURL"},
		Set:     []string{"height", "width"},
		Query:   []string{"height", "width"},
	},
	"DetailsRef": {
		Set:   []string{"open"},
		Query: []string{"open"},
	},
}

type RefOp string

const (
	RefMethod  RefOp = "method"
	RefSet     RefOp = "set"
	RefQuery   RefOp = "query"
	RefObserve RefOp = "observe"
)

func refNames(k RefKind, op RefOp) []string {
	switch op {
	case RefMethod:
		return k.Methods
	case RefSet:
		return k.Set
	case RefQuery:
		return k.Query
	}
	return k.Observe
}

func RefAllows(kind string, op RefOp, name string) bool {
	k, ok := RefKinds[kind]
	if !ok {
		return false
	}
	for _, kk := range []RefKind{k, RefKinds[BaseRefKind]} {
		for _, n := range refNames(kk, op) {
			if n == name {
				return true
			}
		}
	}
	return false
}
