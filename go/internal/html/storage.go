package html

// StorageEvent represents Web Storage (localStorage/sessionStorage) change events.
type StorageEvent struct {
	Event
	Key         string // Key being changed
	OldValue    string // Previous value
	NewValue    string // New value
	URL         string // URL of document that changed
	StorageArea string // Storage area (localStorage or sessionStorage)
}

// Props returns the list of properties this event needs from the client.
func (StorageEvent) props() []string {
	return []string{
		"event.key",
		"event.oldValue",
		"event.newValue",
		"event.url",
	}
}

// StorageHandler provides storage event handlers.
type StorageHandler struct {
	ref RefListener
}

// NewStorageHandler creates a new StorageHandler.
func NewStorageHandler(ref RefListener) *StorageHandler {
	return &StorageHandler{ref: ref}
}

// OnStorage registers a handler for the "storage" event.
func (h *StorageHandler) OnStorage(handler func(StorageEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildStorageEvent(evt)) }
	h.ref.AddListener("storage", wrapped, StorageEvent{}.props())
}

func buildStorageEvent(evt Event) StorageEvent {
	detail := extractDetail(evt.Payload)
	return StorageEvent{
		Event:       evt,
		Key:         PayloadString(detail, "event.key", ""),
		OldValue:    PayloadString(detail, "event.oldValue", ""),
		NewValue:    PayloadString(detail, "event.newValue", ""),
		URL:         PayloadString(detail, "event.url", ""),
		StorageArea: PayloadString(detail, "event.storageArea", ""),
	}
}
