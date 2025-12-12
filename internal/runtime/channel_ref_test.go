package runtime

import (
	"testing"
)

func TestNewChannelRef(t *testing.T) {
	ref := NewChannelRef("my-channel", "session-123")

	if ref == nil {
		t.Fatal("expected non-nil ChannelRef")
	}
	if ref.channelName != "my-channel" {
		t.Errorf("expected channelName 'my-channel', got %q", ref.channelName)
	}
	if ref.sessionID != "session-123" {
		t.Errorf("expected sessionID 'session-123', got %q", ref.sessionID)
	}
	if ref.pondChannel != nil {
		t.Error("expected pondChannel to be nil initially")
	}
}

func TestChannelRef_ChannelName(t *testing.T) {
	t.Run("returns channel name", func(t *testing.T) {
		ref := NewChannelRef("test-channel", "sess-1")
		if ref.ChannelName() != "test-channel" {
			t.Errorf("expected 'test-channel', got %q", ref.ChannelName())
		}
	})

	t.Run("returns empty for nil ref", func(t *testing.T) {
		var ref *ChannelRef
		if ref.ChannelName() != "" {
			t.Errorf("expected empty string for nil ref, got %q", ref.ChannelName())
		}
	})
}

func TestChannelRef_Connected(t *testing.T) {
	t.Run("returns false when no pond channel", func(t *testing.T) {
		ref := NewChannelRef("ch", "sess")
		if ref.Connected() {
			t.Error("expected Connected to return false when pondChannel is nil")
		}
	})

	t.Run("returns false for nil ref", func(t *testing.T) {
		var ref *ChannelRef
		if ref.Connected() {
			t.Error("expected Connected to return false for nil ref")
		}
	})
}

func TestChannelRef_SetPondChannel_Nil(t *testing.T) {
	var ref *ChannelRef
	ref.SetPondChannel(nil)
}

func TestChannelRef_PondChannel(t *testing.T) {
	t.Run("returns nil for nil ref", func(t *testing.T) {
		var ref *ChannelRef
		if ref.PondChannel() != nil {
			t.Error("expected nil for nil ref")
		}
	})

	t.Run("returns nil when not set", func(t *testing.T) {
		ref := NewChannelRef("ch", "sess")
		if ref.PondChannel() != nil {
			t.Error("expected nil when pondChannel not set")
		}
	})
}

func TestChannelRef_Send_Nil(t *testing.T) {
	var ref *ChannelRef
	err := ref.Send("event", "data")
	if err != nil {
		t.Errorf("expected nil error for nil ref, got %v", err)
	}
}

func TestChannelRef_Send_NoPondChannel(t *testing.T) {
	ref := NewChannelRef("ch", "sess")
	err := ref.Send("event", "data")
	if err != nil {
		t.Errorf("expected nil error when pondChannel is nil, got %v", err)
	}
}

func TestChannelRef_SendTo_Nil(t *testing.T) {
	var ref *ChannelRef
	err := ref.SendTo("event", "data", "user1", "user2")
	if err != nil {
		t.Errorf("expected nil error for nil ref, got %v", err)
	}
}

func TestChannelRef_SendTo_NoPondChannel(t *testing.T) {
	ref := NewChannelRef("ch", "sess")
	err := ref.SendTo("event", "data", "user1")
	if err != nil {
		t.Errorf("expected nil error when pondChannel is nil, got %v", err)
	}
}

func TestChannelRef_Track_Nil(t *testing.T) {
	var ref *ChannelRef
	err := ref.Track(map[string]string{"status": "online"})
	if err != nil {
		t.Errorf("expected nil error for nil ref, got %v", err)
	}
}

func TestChannelRef_Track_NoPondChannel(t *testing.T) {
	ref := NewChannelRef("ch", "sess")
	err := ref.Track(map[string]string{"status": "online"})
	if err != nil {
		t.Errorf("expected nil error when pondChannel is nil, got %v", err)
	}
}

func TestChannelRef_UpdatePresence_Nil(t *testing.T) {
	var ref *ChannelRef
	err := ref.UpdatePresence(map[string]string{"status": "away"})
	if err != nil {
		t.Errorf("expected nil error for nil ref, got %v", err)
	}
}

func TestChannelRef_UpdatePresence_NoPondChannel(t *testing.T) {
	ref := NewChannelRef("ch", "sess")
	err := ref.UpdatePresence(map[string]string{"status": "away"})
	if err != nil {
		t.Errorf("expected nil error when pondChannel is nil, got %v", err)
	}
}

func TestChannelRef_Untrack_Nil(t *testing.T) {
	var ref *ChannelRef
	err := ref.Untrack()
	if err != nil {
		t.Errorf("expected nil error for nil ref, got %v", err)
	}
}

func TestChannelRef_Untrack_NoPondChannel(t *testing.T) {
	ref := NewChannelRef("ch", "sess")
	err := ref.Untrack()
	if err != nil {
		t.Errorf("expected nil error when pondChannel is nil, got %v", err)
	}
}

func TestChannelRef_Close(t *testing.T) {
	t.Run("nil ref", func(t *testing.T) {
		var ref *ChannelRef
		ref.Close()
	})

	t.Run("clears pond channel", func(t *testing.T) {
		ref := NewChannelRef("ch", "sess")
		ref.Close()
		if ref.pondChannel != nil {
			t.Error("expected pondChannel to be nil after Close")
		}
	})
}
