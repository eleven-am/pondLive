package runtime

import (
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
)

func TestNewChannel(t *testing.T) {
	ref := &ChannelRef{channelName: "test-channel"}
	ch := newChannel(ref)

	if ch == nil {
		t.Fatal("expected non-nil Channel")
	}

	if ch.ref != ref {
		t.Error("expected ref to be set")
	}
}

func TestChannel_NilChannel(t *testing.T) {
	var ch *Channel

	if err := ch.Send("event", "data"); err != nil {
		t.Errorf("expected nil error from nil channel Send, got %v", err)
	}

	if err := ch.SendTo("event", "data", "user1"); err != nil {
		t.Errorf("expected nil error from nil channel SendTo, got %v", err)
	}

	if err := ch.Track("presence"); err != nil {
		t.Errorf("expected nil error from nil channel Track, got %v", err)
	}

	if err := ch.UpdatePresence("presence"); err != nil {
		t.Errorf("expected nil error from nil channel UpdatePresence, got %v", err)
	}

	if err := ch.Untrack(); err != nil {
		t.Errorf("expected nil error from nil channel Untrack, got %v", err)
	}

	if ch.Connected() {
		t.Error("expected false from nil channel Connected")
	}

	if ch.ChannelName() != "" {
		t.Error("expected empty string from nil channel ChannelName")
	}

	if ch.OnMessage(nil) != nil {
		t.Error("expected nil from nil channel OnMessage")
	}

	if ch.OnPresenceJoin(nil) != nil {
		t.Error("expected nil from nil channel OnPresenceJoin")
	}

	if ch.OnPresenceLeave(nil) != nil {
		t.Error("expected nil from nil channel OnPresenceLeave")
	}

	if ch.OnPresenceUpdate(nil) != nil {
		t.Error("expected nil from nil channel OnPresenceUpdate")
	}

	if ch.OnPresenceSync(nil) != nil {
		t.Error("expected nil from nil channel OnPresenceSync")
	}

	if ch.OnJoin(nil) != nil {
		t.Error("expected nil from nil channel OnJoin")
	}

	if ch.OnLeave(nil) != nil {
		t.Error("expected nil from nil channel OnLeave")
	}

	ch.resetHandlers()

	if ch.subscribeToBus(nil) != nil {
		t.Error("expected nil from nil channel subscribeToBus")
	}
}

func TestChannel_NilRef(t *testing.T) {
	ch := &Channel{ref: nil}

	if err := ch.Send("event", "data"); err != nil {
		t.Errorf("expected nil error from nil ref Send, got %v", err)
	}

	if err := ch.SendTo("event", "data", "user1"); err != nil {
		t.Errorf("expected nil error from nil ref SendTo, got %v", err)
	}

	if err := ch.Track("presence"); err != nil {
		t.Errorf("expected nil error from nil ref Track, got %v", err)
	}

	if err := ch.UpdatePresence("presence"); err != nil {
		t.Errorf("expected nil error from nil ref UpdatePresence, got %v", err)
	}

	if err := ch.Untrack(); err != nil {
		t.Errorf("expected nil error from nil ref Untrack, got %v", err)
	}

	if ch.Connected() {
		t.Error("expected false from nil ref Connected")
	}

	if ch.ChannelName() != "" {
		t.Error("expected empty string from nil ref ChannelName")
	}

	if ch.subscribeToBus(protocol.NewBus()) != nil {
		t.Error("expected nil from nil ref subscribeToBus")
	}
}

func TestChannel_OnHandlers(t *testing.T) {
	ref := &ChannelRef{channelName: "test-channel"}
	ch := newChannel(ref)

	messageHandlerCalled := false
	presenceJoinCalled := false
	presenceLeaveCalled := false
	presenceUpdateCalled := false
	presenceSyncCalled := false
	joinCalled := false
	leaveCalled := false

	ch.OnMessage(func(event string, data interface{}) {
		messageHandlerCalled = true
	})

	ch.OnPresenceJoin(func(userID string, presence interface{}) {
		presenceJoinCalled = true
	})

	ch.OnPresenceLeave(func(userID string) {
		presenceLeaveCalled = true
	})

	ch.OnPresenceUpdate(func(userID string, presence interface{}) {
		presenceUpdateCalled = true
	})

	ch.OnPresenceSync(func(presence map[string]interface{}) {
		presenceSyncCalled = true
	})

	ch.OnJoin(func(presence map[string]interface{}) {
		joinCalled = true
	})

	ch.OnLeave(func() {
		leaveCalled = true
	})

	if ch.onMessage == nil {
		t.Error("expected onMessage to be set")
	}
	if ch.onPresenceJoin == nil {
		t.Error("expected onPresenceJoin to be set")
	}
	if ch.onPresenceLeave == nil {
		t.Error("expected onPresenceLeave to be set")
	}
	if ch.onPresenceUpdate == nil {
		t.Error("expected onPresenceUpdate to be set")
	}
	if ch.onPresenceSync == nil {
		t.Error("expected onPresenceSync to be set")
	}
	if ch.onJoin == nil {
		t.Error("expected onJoin to be set")
	}
	if ch.onLeave == nil {
		t.Error("expected onLeave to be set")
	}

	ch.onMessage("test", nil)
	ch.onPresenceJoin("user", nil)
	ch.onPresenceLeave("user")
	ch.onPresenceUpdate("user", nil)
	ch.onPresenceSync(nil)
	ch.onJoin(nil)
	ch.onLeave()

	if !messageHandlerCalled {
		t.Error("expected messageHandler to be called")
	}
	if !presenceJoinCalled {
		t.Error("expected presenceJoin to be called")
	}
	if !presenceLeaveCalled {
		t.Error("expected presenceLeave to be called")
	}
	if !presenceUpdateCalled {
		t.Error("expected presenceUpdate to be called")
	}
	if !presenceSyncCalled {
		t.Error("expected presenceSync to be called")
	}
	if !joinCalled {
		t.Error("expected join to be called")
	}
	if !leaveCalled {
		t.Error("expected leave to be called")
	}
}

func TestChannel_OnHandlersChaining(t *testing.T) {
	ref := &ChannelRef{channelName: "test-channel"}
	ch := newChannel(ref)

	result := ch.
		OnMessage(func(event string, data interface{}) {}).
		OnPresenceJoin(func(userID string, presence interface{}) {}).
		OnPresenceLeave(func(userID string) {}).
		OnPresenceUpdate(func(userID string, presence interface{}) {}).
		OnPresenceSync(func(presence map[string]interface{}) {}).
		OnJoin(func(presence map[string]interface{}) {}).
		OnLeave(func() {})

	if result != ch {
		t.Error("expected chaining to return same channel")
	}
}

func TestChannel_ResetHandlers(t *testing.T) {
	ref := &ChannelRef{channelName: "test-channel"}
	ch := newChannel(ref)

	ch.OnMessage(func(event string, data interface{}) {})
	ch.OnPresenceJoin(func(userID string, presence interface{}) {})
	ch.OnPresenceLeave(func(userID string) {})
	ch.OnPresenceUpdate(func(userID string, presence interface{}) {})
	ch.OnPresenceSync(func(presence map[string]interface{}) {})
	ch.OnJoin(func(presence map[string]interface{}) {})
	ch.OnLeave(func() {})

	ch.resetHandlers()

	if ch.onMessage != nil {
		t.Error("expected onMessage to be nil after reset")
	}
	if ch.onPresenceJoin != nil {
		t.Error("expected onPresenceJoin to be nil after reset")
	}
	if ch.onPresenceLeave != nil {
		t.Error("expected onPresenceLeave to be nil after reset")
	}
	if ch.onPresenceUpdate != nil {
		t.Error("expected onPresenceUpdate to be nil after reset")
	}
	if ch.onPresenceSync != nil {
		t.Error("expected onPresenceSync to be nil after reset")
	}
	if ch.onJoin != nil {
		t.Error("expected onJoin to be nil after reset")
	}
	if ch.onLeave != nil {
		t.Error("expected onLeave to be nil after reset")
	}
}

func TestChannel_SubscribeToBus(t *testing.T) {
	bus := protocol.NewBus()
	ref := &ChannelRef{channelName: "test-channel"}
	ch := newChannel(ref)

	var wg sync.WaitGroup
	messageReceived := false
	presenceJoinReceived := false
	presenceLeaveReceived := false
	presenceUpdateReceived := false
	presenceSyncReceived := false
	joinReceived := false
	leaveReceived := false

	ch.OnMessage(func(event string, data interface{}) {
		messageReceived = true
		wg.Done()
	})

	ch.OnPresenceJoin(func(userID string, presence interface{}) {
		presenceJoinReceived = true
		wg.Done()
	})

	ch.OnPresenceLeave(func(userID string) {
		presenceLeaveReceived = true
		wg.Done()
	})

	ch.OnPresenceUpdate(func(userID string, presence interface{}) {
		presenceUpdateReceived = true
		wg.Done()
	})

	ch.OnPresenceSync(func(presence map[string]interface{}) {
		presenceSyncReceived = true
		wg.Done()
	})

	ch.OnJoin(func(presence map[string]interface{}) {
		joinReceived = true
		wg.Done()
	})

	ch.OnLeave(func() {
		leaveReceived = true
		wg.Done()
	})

	sub := ch.subscribeToBus(bus)
	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}
	defer sub.Unsubscribe()

	topic := protocol.ChannelTopic("test-channel")

	wg.Add(1)
	bus.Publish(topic, string(protocol.ChannelMessageAction), protocol.ChannelMessagePayload{
		Event: "test",
		Data:  "data",
	})

	wg.Add(1)
	bus.Publish(topic, string(protocol.ChannelPresenceJoinAction), protocol.ChannelPresencePayload{
		UserID:   "user1",
		Presence: "online",
	})

	wg.Add(1)
	bus.Publish(topic, string(protocol.ChannelPresenceLeaveAction), protocol.ChannelPresencePayload{
		UserID: "user1",
	})

	wg.Add(1)
	bus.Publish(topic, string(protocol.ChannelPresenceUpdateAction), protocol.ChannelPresencePayload{
		UserID:   "user1",
		Presence: "away",
	})

	wg.Add(1)
	bus.Publish(topic, string(protocol.ChannelPresenceSyncAction), protocol.ChannelPresenceSyncPayload{
		Presence: map[string]interface{}{"user1": "online"},
	})

	wg.Add(1)
	bus.Publish(topic, string(protocol.ChannelJoinedAction), protocol.ChannelJoinedPayload{
		Presence: map[string]interface{}{"user1": "online"},
	})

	wg.Add(1)
	bus.Publish(topic, string(protocol.ChannelLeftAction), nil)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("timeout waiting for events")
	}

	if !messageReceived {
		t.Error("expected message to be received")
	}
	if !presenceJoinReceived {
		t.Error("expected presence join to be received")
	}
	if !presenceLeaveReceived {
		t.Error("expected presence leave to be received")
	}
	if !presenceUpdateReceived {
		t.Error("expected presence update to be received")
	}
	if !presenceSyncReceived {
		t.Error("expected presence sync to be received")
	}
	if !joinReceived {
		t.Error("expected join to be received")
	}
	if !leaveReceived {
		t.Error("expected leave to be received")
	}
}

func TestChannel_SubscribeToBus_NilBus(t *testing.T) {
	ref := &ChannelRef{channelName: "test-channel"}
	ch := newChannel(ref)

	sub := ch.subscribeToBus(nil)
	if sub != nil {
		t.Error("expected nil subscription for nil bus")
	}
}

func TestChannel_SubscribeToBus_NoHandlers(t *testing.T) {
	bus := protocol.NewBus()
	ref := &ChannelRef{channelName: "test-channel"}
	ch := newChannel(ref)

	sub := ch.subscribeToBus(bus)
	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}
	defer sub.Unsubscribe()

	topic := protocol.ChannelTopic("test-channel")
	bus.Publish(topic, string(protocol.ChannelMessageAction), protocol.ChannelMessagePayload{
		Event: "test",
		Data:  "data",
	})

	time.Sleep(10 * time.Millisecond)
}

func TestChannel_ConcurrentHandlerAccess(t *testing.T) {
	ref := &ChannelRef{channelName: "test-channel"}
	ch := newChannel(ref)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch.OnMessage(func(event string, data interface{}) {})
		}()
	}

	wg.Wait()
}

func TestChannelSendNilRef(t *testing.T) {
	ch := &Channel{ref: nil}
	err := ch.Send("event", "data")
	if err != nil {
		t.Errorf("expected nil error for nil ref, got %v", err)
	}
}

func TestChannelSendToNilRef(t *testing.T) {
	ch := &Channel{ref: nil}
	err := ch.SendTo("event", "data", "user1")
	if err != nil {
		t.Errorf("expected nil error for nil ref, got %v", err)
	}
}

func TestChannelTrackNilRef(t *testing.T) {
	ch := &Channel{ref: nil}
	err := ch.Track(map[string]string{"status": "online"})
	if err != nil {
		t.Errorf("expected nil error for nil ref, got %v", err)
	}
}

func TestChannelUpdatePresenceNilRef(t *testing.T) {
	ch := &Channel{ref: nil}
	err := ch.UpdatePresence(map[string]string{"status": "away"})
	if err != nil {
		t.Errorf("expected nil error for nil ref, got %v", err)
	}
}

func TestChannelUntrackNilRef(t *testing.T) {
	ch := &Channel{ref: nil}
	err := ch.Untrack()
	if err != nil {
		t.Errorf("expected nil error for nil ref, got %v", err)
	}
}

func TestChannelConnectedNilRef(t *testing.T) {
	ch := &Channel{ref: nil}
	if ch.Connected() {
		t.Error("expected false for nil ref")
	}
}

func TestChannelNameNilRef(t *testing.T) {
	ch := &Channel{ref: nil}
	if ch.ChannelName() != "" {
		t.Errorf("expected empty string for nil ref, got %q", ch.ChannelName())
	}
}

func TestChannelNilChannel(t *testing.T) {
	var ch *Channel
	if ch.Send("event", "data") != nil {
		t.Error("expected nil error for nil channel Send")
	}
	if ch.SendTo("event", "data", "user") != nil {
		t.Error("expected nil error for nil channel SendTo")
	}
	if ch.Track(nil) != nil {
		t.Error("expected nil error for nil channel Track")
	}
	if ch.UpdatePresence(nil) != nil {
		t.Error("expected nil error for nil channel UpdatePresence")
	}
	if ch.Untrack() != nil {
		t.Error("expected nil error for nil channel Untrack")
	}
	if ch.Connected() {
		t.Error("expected false for nil channel Connected")
	}
	if ch.ChannelName() != "" {
		t.Error("expected empty string for nil channel ChannelName")
	}
}

func TestChannelWithValidRef(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)
	ref := cm.Join("test-channel")
	ch := newChannel(ref)

	if ch.ChannelName() != "test-channel" {
		t.Errorf("expected channel name 'test-channel', got %q", ch.ChannelName())
	}

	_ = ch.Connected()

	if err := ch.Send("event", "data"); err != nil {
		t.Errorf("Send returned error: %v", err)
	}

	if err := ch.SendTo("event", "data", "user1"); err != nil {
		t.Errorf("SendTo returned error: %v", err)
	}

	if err := ch.Track(map[string]string{"status": "online"}); err != nil {
		t.Errorf("Track returned error: %v", err)
	}

	if err := ch.UpdatePresence(map[string]string{"status": "away"}); err != nil {
		t.Errorf("UpdatePresence returned error: %v", err)
	}

	if err := ch.Untrack(); err != nil {
		t.Errorf("Untrack returned error: %v", err)
	}
}
