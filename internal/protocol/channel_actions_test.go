package protocol

import (
	"sync"
	"testing"
	"time"
)

func TestChannelTopic(t *testing.T) {
	topic := ChannelTopic("my-channel")
	if topic != "channel:my-channel" {
		t.Errorf("ChannelTopic() = %q, want %q", topic, "channel:my-channel")
	}
}

func TestIsChannelTopic(t *testing.T) {
	tests := []struct {
		topic Topic
		want  bool
	}{
		{Topic("channel:foo"), true},
		{Topic("channel:"), true},
		{Topic("frame"), false},
		{Topic("router"), false},
		{Topic(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.topic), func(t *testing.T) {
			got := IsChannelTopic(tt.topic)
			if got != tt.want {
				t.Errorf("IsChannelTopic(%q) = %v, want %v", tt.topic, got, tt.want)
			}
		})
	}
}

func TestExtractChannelName(t *testing.T) {
	tests := []struct {
		topic Topic
		want  string
	}{
		{Topic("channel:my-channel"), "my-channel"},
		{Topic("channel:"), ""},
		{Topic("frame"), ""},
		{Topic("router"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.topic), func(t *testing.T) {
			got := ExtractChannelName(tt.topic)
			if got != tt.want {
				t.Errorf("ExtractChannelName(%q) = %q, want %q", tt.topic, got, tt.want)
			}
		})
	}
}

func TestPublishChannelJoined(t *testing.T) {
	bus := NewBus()
	var wg sync.WaitGroup
	var received ChannelJoinedPayload
	var mu sync.Mutex

	wg.Add(1)
	bus.Subscribe(ChannelTopic("test"), func(event string, data interface{}) {
		mu.Lock()
		if event == string(ChannelJoinedAction) {
			if payload, ok := data.(ChannelJoinedPayload); ok {
				received = payload
			}
		}
		mu.Unlock()
		wg.Done()
	})

	presence := map[string]interface{}{"user1": "online"}
	bus.PublishChannelJoined("test", presence)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	mu.Lock()
	if received.Channel != "test" {
		t.Errorf("Channel = %q, want %q", received.Channel, "test")
	}
	mu.Unlock()
}

func TestPublishChannelLeft(t *testing.T) {
	bus := NewBus()
	var wg sync.WaitGroup
	var received ChannelLeftPayload
	var mu sync.Mutex

	wg.Add(1)
	bus.Subscribe(ChannelTopic("test"), func(event string, data interface{}) {
		mu.Lock()
		if event == string(ChannelLeftAction) {
			if payload, ok := data.(ChannelLeftPayload); ok {
				received = payload
			}
		}
		mu.Unlock()
		wg.Done()
	})

	bus.PublishChannelLeft("test")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	mu.Lock()
	if received.Channel != "test" {
		t.Errorf("Channel = %q, want %q", received.Channel, "test")
	}
	mu.Unlock()
}

func TestPublishChannelMessage(t *testing.T) {
	bus := NewBus()
	var wg sync.WaitGroup
	var received ChannelMessagePayload
	var mu sync.Mutex

	wg.Add(1)
	bus.Subscribe(ChannelTopic("chat"), func(event string, data interface{}) {
		mu.Lock()
		if event == string(ChannelMessageAction) {
			if payload, ok := data.(ChannelMessagePayload); ok {
				received = payload
			}
		}
		mu.Unlock()
		wg.Done()
	})

	bus.PublishChannelMessage("chat", "new_message", "hello")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	mu.Lock()
	if received.Event != "new_message" {
		t.Errorf("Event = %q, want %q", received.Event, "new_message")
	}
	mu.Unlock()
}

func TestPublishChannelPresenceJoin(t *testing.T) {
	bus := NewBus()
	var wg sync.WaitGroup
	var received ChannelPresencePayload
	var mu sync.Mutex

	wg.Add(1)
	bus.Subscribe(ChannelTopic("room"), func(event string, data interface{}) {
		mu.Lock()
		if event == string(ChannelPresenceJoinAction) {
			if payload, ok := data.(ChannelPresencePayload); ok {
				received = payload
			}
		}
		mu.Unlock()
		wg.Done()
	})

	bus.PublishChannelPresenceJoin("room", "user123", "online")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	mu.Lock()
	if received.UserID != "user123" {
		t.Errorf("UserID = %q, want %q", received.UserID, "user123")
	}
	mu.Unlock()
}

func TestPublishChannelPresenceLeave(t *testing.T) {
	bus := NewBus()
	var wg sync.WaitGroup
	var received ChannelPresencePayload
	var mu sync.Mutex

	wg.Add(1)
	bus.Subscribe(ChannelTopic("room"), func(event string, data interface{}) {
		mu.Lock()
		if event == string(ChannelPresenceLeaveAction) {
			if payload, ok := data.(ChannelPresencePayload); ok {
				received = payload
			}
		}
		mu.Unlock()
		wg.Done()
	})

	bus.PublishChannelPresenceLeave("room", "user456")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	mu.Lock()
	if received.UserID != "user456" {
		t.Errorf("UserID = %q, want %q", received.UserID, "user456")
	}
	mu.Unlock()
}

func TestPublishChannelPresenceUpdate(t *testing.T) {
	bus := NewBus()
	var wg sync.WaitGroup
	var received ChannelPresencePayload
	var mu sync.Mutex

	wg.Add(1)
	bus.Subscribe(ChannelTopic("room"), func(event string, data interface{}) {
		mu.Lock()
		if event == string(ChannelPresenceUpdateAction) {
			if payload, ok := data.(ChannelPresencePayload); ok {
				received = payload
			}
		}
		mu.Unlock()
		wg.Done()
	})

	bus.PublishChannelPresenceUpdate("room", "user789", "away")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	mu.Lock()
	if received.UserID != "user789" {
		t.Errorf("UserID = %q, want %q", received.UserID, "user789")
	}
	mu.Unlock()
}

func TestPublishChannelPresenceSync(t *testing.T) {
	bus := NewBus()
	var wg sync.WaitGroup
	var received ChannelPresenceSyncPayload
	var mu sync.Mutex

	wg.Add(1)
	bus.Subscribe(ChannelTopic("room"), func(event string, data interface{}) {
		mu.Lock()
		if event == string(ChannelPresenceSyncAction) {
			if payload, ok := data.(ChannelPresenceSyncPayload); ok {
				received = payload
			}
		}
		mu.Unlock()
		wg.Done()
	})

	presence := map[string]interface{}{"user1": "online", "user2": "away"}
	bus.PublishChannelPresenceSync("room", presence)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	mu.Lock()
	if len(received.Presence) != 2 {
		t.Errorf("Presence length = %d, want 2", len(received.Presence))
	}
	mu.Unlock()
}
