package runtime

import (
	"strings"
	"sync"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
)

func TestNewChannelManager(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	if cm == nil {
		t.Fatal("expected non-nil ChannelManager")
	}

	if cm.sessionID != "session-123" {
		t.Errorf("expected sessionID 'session-123', got '%s'", cm.sessionID)
	}

	if len(cm.secret) != 32 {
		t.Errorf("expected 32-byte secret, got %d bytes", len(cm.secret))
	}

	if cm.bus != bus {
		t.Error("expected bus to be set")
	}

	if cm.refs == nil {
		t.Error("expected refs map to be initialized")
	}

	if cm.counts == nil {
		t.Error("expected counts map to be initialized")
	}
}

func TestJoin_FirstJoin_PublishesInstruction(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	var publishedTopic protocol.Topic
	var publishedEvent string
	done := make(chan struct{})

	sub := bus.Subscribe(protocol.ChannelTopic("chat-room"), func(event string, data interface{}) {
		publishedTopic = protocol.ChannelTopic("chat-room")
		publishedEvent = event
		close(done)
	})
	defer sub.Unsubscribe()

	ref := cm.Join("chat-room")

	if ref == nil {
		t.Fatal("expected non-nil ChannelRef")
	}

	<-done

	if publishedTopic != protocol.ChannelTopic("chat-room") {
		t.Errorf("expected topic 'channel:chat-room', got '%s'", publishedTopic)
	}

	if publishedEvent != string(protocol.ChannelJoinAction) {
		t.Errorf("expected event 'join', got '%s'", publishedEvent)
	}

	if cm.counts["chat-room"] != 1 {
		t.Errorf("expected count 1, got %d", cm.counts["chat-room"])
	}

	if cm.refs["chat-room"] != ref {
		t.Error("expected ref to be stored in refs map")
	}
}

func TestJoin_SubsequentJoin_NoInstruction(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	cm.Join("chat-room")

	publishCount := 0
	sub := bus.Subscribe(protocol.ChannelTopic("chat-room"), func(event string, data interface{}) {
		publishCount++
	})
	defer sub.Unsubscribe()

	ref2 := cm.Join("chat-room")

	if ref2 == nil {
		t.Fatal("expected non-nil ChannelRef")
	}

	if publishCount != 0 {
		t.Errorf("expected no publish on subsequent join, got %d publishes", publishCount)
	}

	if cm.counts["chat-room"] != 2 {
		t.Errorf("expected count 2, got %d", cm.counts["chat-room"])
	}
}

func TestJoin_ReturnsSameRef(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	ref1 := cm.Join("chat-room")
	ref2 := cm.Join("chat-room")

	if ref1 != ref2 {
		t.Error("expected same ChannelRef for same channel")
	}
}

func TestLeave_LastLeave_PublishesInstruction(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	events := make(chan string, 2)

	sub := bus.Subscribe(protocol.ChannelTopic("chat-room"), func(event string, data interface{}) {
		events <- event
	})
	defer sub.Unsubscribe()

	cm.Join("chat-room")
	<-events

	cm.Leave("chat-room")
	leaveEvent := <-events

	if leaveEvent != string(protocol.ChannelLeaveAction) {
		t.Errorf("expected event 'leave', got '%s'", leaveEvent)
	}

	if _, exists := cm.counts["chat-room"]; exists {
		t.Error("expected channel to be removed from counts")
	}

	if _, exists := cm.refs["chat-room"]; exists {
		t.Error("expected ref to be removed from refs")
	}
}

func TestLeave_NotLastLeave_NoInstruction(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	cm.Join("chat-room")
	cm.Join("chat-room")

	publishCount := 0
	sub := bus.Subscribe(protocol.ChannelTopic("chat-room"), func(event string, data interface{}) {
		publishCount++
	})
	defer sub.Unsubscribe()

	cm.Leave("chat-room")

	if publishCount != 0 {
		t.Errorf("expected no publish on non-last leave, got %d publishes", publishCount)
	}

	if cm.counts["chat-room"] != 1 {
		t.Errorf("expected count 1, got %d", cm.counts["chat-room"])
	}
}

func TestLeave_NonExistentChannel(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	cm.Leave("non-existent")
}

func TestGetRef(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	ref := cm.Join("chat-room")
	gotRef := cm.GetRef("chat-room")

	if gotRef != ref {
		t.Error("expected GetRef to return same ref")
	}

	nilRef := cm.GetRef("non-existent")
	if nilRef != nil {
		t.Error("expected nil for non-existent ref")
	}
}

func TestGenerateToken_ValidFormat(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	token := cm.GenerateToken("chat-room")

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		t.Errorf("expected token with 2 parts, got %d", len(parts))
	}
}

func TestValidateToken_Valid(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	token := cm.GenerateToken("chat-room")
	valid := cm.ValidateToken(token, "chat-room")

	if !valid {
		t.Error("expected token to be valid")
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	token := cm.GenerateToken("chat-room")
	tamperedToken := token[:len(token)-5] + "XXXXX"

	valid := cm.ValidateToken(tamperedToken, "chat-room")

	if valid {
		t.Error("expected tampered token to be invalid")
	}
}

func TestValidateToken_WrongChannel(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	token := cm.GenerateToken("chat-room")
	valid := cm.ValidateToken(token, "other-room")

	if valid {
		t.Error("expected token to be invalid for wrong channel")
	}
}

func TestValidateToken_WrongSession(t *testing.T) {
	bus := protocol.NewBus()
	cm1 := NewChannelManager("session-123", bus)
	cm2 := NewChannelManager("session-456", bus)

	token := cm1.GenerateToken("chat-room")
	valid := cm2.ValidateToken(token, "chat-room")

	if valid {
		t.Error("expected token to be invalid for different session")
	}
}

func TestValidateToken_InvalidFormat(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	testCases := []string{
		"",
		"no-dot",
		"too.many.dots",
		"invalid-base64.!!!",
	}

	for _, tc := range testCases {
		if cm.ValidateToken(tc, "chat-room") {
			t.Errorf("expected token '%s' to be invalid", tc)
		}
	}
}

func TestSetPondChannel(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	ref := cm.Join("chat-room")

	if ref.PondChannel() != nil {
		t.Error("expected nil pond channel initially")
	}

	cm.SetPondChannel("chat-room", nil)

	cm.SetPondChannel("non-existent", nil)
}

func TestHandleDisconnect(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	ref := cm.Join("chat-room")
	cm.SetPondChannel("chat-room", nil)

	cm.HandleDisconnect("chat-room")

	if ref.Connected() {
		t.Error("expected not connected after HandleDisconnect")
	}

	if ref.PondChannel() != nil {
		t.Error("expected nil pond channel after HandleDisconnect")
	}

	cm.HandleDisconnect("non-existent")
}

func TestClose(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	cm.Join("room1")
	cm.Join("room2")

	cm.Close()

	if len(cm.refs) != 0 {
		t.Errorf("expected empty refs after Close, got %d", len(cm.refs))
	}

	if len(cm.counts) != 0 {
		t.Errorf("expected empty counts after Close, got %d", len(cm.counts))
	}
}

func TestActiveChannels(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	channels := cm.ActiveChannels()
	if len(channels) != 0 {
		t.Error("expected no active channels initially")
	}

	cm.Join("room1")
	cm.Join("room2")

	channels = cm.ActiveChannels()
	if len(channels) != 2 {
		t.Errorf("expected 2 active channels, got %d", len(channels))
	}

	hasRoom1 := false
	hasRoom2 := false
	for _, ch := range channels {
		if ch == "room1" {
			hasRoom1 = true
		}
		if ch == "room2" {
			hasRoom2 = true
		}
	}

	if !hasRoom1 || !hasRoom2 {
		t.Error("expected both room1 and room2 in active channels")
	}
}

func TestNilChannelManager(t *testing.T) {
	var cm *ChannelManager

	if cm.Join("room") != nil {
		t.Error("expected nil from nil manager Join")
	}

	cm.Leave("room")

	if cm.GetRef("room") != nil {
		t.Error("expected nil from nil manager GetRef")
	}

	cm.SetPondChannel("room", nil)
	cm.HandleDisconnect("room")

	if cm.GenerateToken("room") != "" {
		t.Error("expected empty token from nil manager")
	}

	if cm.ValidateToken("token", "room") {
		t.Error("expected false from nil manager ValidateToken")
	}

	cm.Close()

	if cm.ActiveChannels() != nil {
		t.Error("expected nil from nil manager ActiveChannels")
	}
}

func TestConcurrentAccess(t *testing.T) {
	bus := protocol.NewBus()
	cm := NewChannelManager("session-123", bus)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cm.Join("chat-room")
		}()
	}

	wg.Wait()

	if cm.counts["chat-room"] != 100 {
		t.Errorf("expected count 100, got %d", cm.counts["chat-room"])
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cm.Leave("chat-room")
		}()
	}

	wg.Wait()

	if _, exists := cm.refs["chat-room"]; exists {
		t.Error("expected ref to be removed after all leaves")
	}
}
