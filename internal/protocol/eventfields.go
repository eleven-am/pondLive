package protocol

import "fmt"

var EventFields = []string{
	"detail", "button", "buttons",
	"clientX", "clientY", "offsetX", "offsetY", "pageX", "pageY", "screenX", "screenY", "movementX", "movementY",
	"altKey", "ctrlKey", "shiftKey", "metaKey",
	"key", "code", "location", "repeat", "isComposing",
	"pointerType", "pointerId", "width", "height", "pressure", "tiltX", "tiltY", "isPrimary",
	"deltaX", "deltaY", "deltaZ", "deltaMode",
	"inputType", "data",
	"animationName", "elapsedTime", "pseudoElement", "propertyName",
	"loaded", "total", "lengthComputable",
	"scrollTop", "scrollLeft", "scrollHeight", "scrollWidth", "clientHeight", "clientWidth",
	"currentTime", "duration", "paused", "ended", "volume", "muted", "playbackRate",
	"selectionStart", "selectionEnd", "open", "naturalWidth", "naturalHeight",
	"value", "checked", "selected", "files", "touches", "text", "fullscreen", "returnValue",
}

var ComputedFields = []string{"value", "selected", "files", "touches", "text", "fullscreen", "returnValue"}

var knownFields = func() map[string]struct{} {
	m := make(map[string]struct{}, len(EventFields))
	for _, f := range EventFields {
		m[f] = struct{}{}
	}
	return m
}()

func ValidateFields(names ...string) error {
	for _, n := range names {
		if _, ok := knownFields[n]; !ok {
			return fmt.Errorf("protocol: %q is not an event field", n)
		}
	}
	return nil
}
