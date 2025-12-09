package runtime

import (
	"sync"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"
)

type ChannelRef struct {
	channelName string
	sessionID   string
	pondChannel *pond.Channel
	mu          sync.RWMutex
}

func NewChannelRef(channelName, sessionID string) *ChannelRef {
	return &ChannelRef{
		channelName: channelName,
		sessionID:   sessionID,
	}
}

func (c *ChannelRef) ChannelName() string {
	if c == nil {
		return ""
	}
	return c.channelName
}

func (c *ChannelRef) Connected() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	connected := c.pondChannel != nil
	c.mu.RUnlock()
	return connected
}

func (c *ChannelRef) SetPondChannel(channel *pond.Channel) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.pondChannel = channel
	c.mu.Unlock()
}

func (c *ChannelRef) PondChannel() *pond.Channel {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	ch := c.pondChannel
	c.mu.RUnlock()
	return ch
}

func (c *ChannelRef) Send(event string, payload interface{}) error {
	if c == nil {
		return nil
	}

	ch := c.PondChannel()
	if ch == nil {
		return nil
	}

	return ch.BroadcastFrom(event, payload, c.sessionID)
}

func (c *ChannelRef) SendTo(event string, payload interface{}, userIDs ...string) error {
	if c == nil {
		return nil
	}

	ch := c.PondChannel()
	if ch == nil {
		return nil
	}

	return ch.BroadcastTo(event, payload, userIDs...)
}

func (c *ChannelRef) Track(presence interface{}) error {
	if c == nil {
		return nil
	}

	ch := c.PondChannel()
	if ch == nil {
		return nil
	}

	return ch.Track(c.sessionID, presence)
}

func (c *ChannelRef) UpdatePresence(presence interface{}) error {
	if c == nil {
		return nil
	}

	ch := c.PondChannel()
	if ch == nil {
		return nil
	}

	return ch.UpdatePresence(c.sessionID, presence)
}

func (c *ChannelRef) Untrack() error {
	if c == nil {
		return nil
	}

	ch := c.PondChannel()
	if ch == nil {
		return nil
	}

	return ch.UnTrack(c.sessionID)
}

func (c *ChannelRef) Close() {
	if c == nil {
		return
	}

	c.mu.Lock()
	c.pondChannel = nil
	c.mu.Unlock()
}
