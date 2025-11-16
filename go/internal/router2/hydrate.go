package router2

// StoreSnapshotPayload serializes the router store snapshot into JSON bytes for
// SSR boot payloads.
func StoreSnapshotPayload(store *RouterStore) ([]byte, error) {
	if store == nil {
		return nil, nil
	}
	return store.Snapshot().ToPayload()
}

// ApplySnapshotPayload decodes JSON bytes into a snapshot and primes the store.
func ApplySnapshotPayload(store *RouterStore, data []byte) error {
	if store == nil {
		return nil
	}
	snap, err := SnapshotFromPayload(data)
	if err != nil {
		return err
	}
	store.ApplySnapshot(snap)
	return nil
}
