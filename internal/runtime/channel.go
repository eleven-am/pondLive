package runtime

import (
	"sync"

	"github.com/eleven-am/pondlive/internal/protocol"
)

type ChannelMessageHandler func(event string, data interface{})
type ChannelPresenceHandler func(userID string, presence interface{})
type ChannelPresenceLeaveHandler func(userID string)
type ChannelJoinHandler func(presence map[string]interface{})
type ChannelLeaveHandler func()
type ChannelPresenceSyncHandler func(presence map[string]interface{})

type Channel struct {
	ref *ChannelRef

	onMessage        ChannelMessageHandler
	onPresenceJoin   ChannelPresenceHandler
	onPresenceLeave  ChannelPresenceLeaveHandler
	onPresenceUpdate ChannelPresenceHandler
	onPresenceSync   ChannelPresenceSyncHandler
	onJoin           ChannelJoinHandler
	onLeave          ChannelLeaveHandler

	mu sync.RWMutex
}

func newChannel(ref *ChannelRef) *Channel {
	return &Channel{
		ref: ref,
	}
}

func (c *Channel) Send(event string, payload interface{}) error {
	if c == nil || c.ref == nil {
		return nil
	}
	return c.ref.Send(event, payload)
}

func (c *Channel) SendTo(event string, payload interface{}, userIDs ...string) error {
	if c == nil || c.ref == nil {
		return nil
	}
	return c.ref.SendTo(event, payload, userIDs...)
}

func (c *Channel) Track(presence interface{}) error {
	if c == nil || c.ref == nil {
		return nil
	}
	return c.ref.Track(presence)
}

func (c *Channel) UpdatePresence(presence interface{}) error {
	if c == nil || c.ref == nil {
		return nil
	}
	return c.ref.UpdatePresence(presence)
}

func (c *Channel) Untrack() error {
	if c == nil || c.ref == nil {
		return nil
	}
	return c.ref.Untrack()
}

func (c *Channel) Connected() bool {
	if c == nil || c.ref == nil {
		return false
	}
	return c.ref.Connected()
}

func (c *Channel) ChannelName() string {
	if c == nil || c.ref == nil {
		return ""
	}
	return c.ref.ChannelName()
}

func (c *Channel) OnMessage(handler ChannelMessageHandler) *Channel {
	if c == nil {
		return c
	}
	c.mu.Lock()
	c.onMessage = handler
	c.mu.Unlock()
	return c
}

func (c *Channel) OnPresenceJoin(handler ChannelPresenceHandler) *Channel {
	if c == nil {
		return c
	}
	c.mu.Lock()
	c.onPresenceJoin = handler
	c.mu.Unlock()
	return c
}

func (c *Channel) OnPresenceLeave(handler ChannelPresenceLeaveHandler) *Channel {
	if c == nil {
		return c
	}
	c.mu.Lock()
	c.onPresenceLeave = handler
	c.mu.Unlock()
	return c
}

func (c *Channel) OnPresenceUpdate(handler ChannelPresenceHandler) *Channel {
	if c == nil {
		return c
	}
	c.mu.Lock()
	c.onPresenceUpdate = handler
	c.mu.Unlock()
	return c
}

func (c *Channel) OnPresenceSync(handler ChannelPresenceSyncHandler) *Channel {
	if c == nil {
		return c
	}
	c.mu.Lock()
	c.onPresenceSync = handler
	c.mu.Unlock()
	return c
}

func (c *Channel) OnJoin(handler ChannelJoinHandler) *Channel {
	if c == nil {
		return c
	}
	c.mu.Lock()
	c.onJoin = handler
	c.mu.Unlock()
	return c
}

func (c *Channel) OnLeave(handler ChannelLeaveHandler) *Channel {
	if c == nil {
		return c
	}
	c.mu.Lock()
	c.onLeave = handler
	c.mu.Unlock()
	return c
}

func (c *Channel) resetHandlers() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.onMessage = nil
	c.onPresenceJoin = nil
	c.onPresenceLeave = nil
	c.onPresenceUpdate = nil
	c.onPresenceSync = nil
	c.onJoin = nil
	c.onLeave = nil
	c.mu.Unlock()
}

func (c *Channel) subscribeToBus(bus *protocol.Bus) *protocol.Subscription {
	if c == nil || c.ref == nil || bus == nil {
		return nil
	}

	topic := protocol.ChannelTopic(c.ref.channelName)

	return bus.Subscribe(topic, func(event string, data interface{}) {
		c.mu.RLock()
		onMessage := c.onMessage
		onPresenceJoin := c.onPresenceJoin
		onPresenceLeave := c.onPresenceLeave
		onPresenceUpdate := c.onPresenceUpdate
		onPresenceSync := c.onPresenceSync
		onJoin := c.onJoin
		onLeave := c.onLeave
		c.mu.RUnlock()

		switch protocol.ChannelServerAction(event) {
		case protocol.ChannelMessageAction:
			if onMessage != nil {
				if payload, ok := protocol.DecodePayload[protocol.ChannelMessagePayload](data); ok {
					onMessage(payload.Event, payload.Data)
				}
			}
		case protocol.ChannelPresenceJoinAction:
			if onPresenceJoin != nil {
				if payload, ok := protocol.DecodePayload[protocol.ChannelPresencePayload](data); ok {
					onPresenceJoin(payload.UserID, payload.Presence)
				}
			}
		case protocol.ChannelPresenceLeaveAction:
			if onPresenceLeave != nil {
				if payload, ok := protocol.DecodePayload[protocol.ChannelPresencePayload](data); ok {
					onPresenceLeave(payload.UserID)
				}
			}
		case protocol.ChannelPresenceUpdateAction:
			if onPresenceUpdate != nil {
				if payload, ok := protocol.DecodePayload[protocol.ChannelPresencePayload](data); ok {
					onPresenceUpdate(payload.UserID, payload.Presence)
				}
			}
		case protocol.ChannelPresenceSyncAction:
			if onPresenceSync != nil {
				if payload, ok := protocol.DecodePayload[protocol.ChannelPresenceSyncPayload](data); ok {
					onPresenceSync(payload.Presence)
				}
			}
		case protocol.ChannelJoinedAction:
			if onJoin != nil {
				if payload, ok := protocol.DecodePayload[protocol.ChannelJoinedPayload](data); ok {
					onJoin(payload.Presence)
				}
			}
		case protocol.ChannelLeftAction:
			if onLeave != nil {
				onLeave()
			}
		}
	})
}
