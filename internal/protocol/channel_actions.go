package protocol

import "strings"

const ChannelTopicPrefix = "channel:"

type ChannelServerAction string
type ChannelClientAction string

const (
	ChannelJoinedAction         ChannelServerAction = "joined"
	ChannelLeftAction           ChannelServerAction = "left"
	ChannelMessageAction        ChannelServerAction = "message"
	ChannelPresenceJoinAction   ChannelServerAction = "presence_join"
	ChannelPresenceLeaveAction  ChannelServerAction = "presence_leave"
	ChannelPresenceUpdateAction ChannelServerAction = "presence_update"
	ChannelPresenceSyncAction   ChannelServerAction = "presence_sync"
)

const (
	ChannelJoinAction  ChannelClientAction = "join"
	ChannelLeaveAction ChannelClientAction = "leave"
)

func ChannelTopic(channelName string) Topic {
	return Topic(ChannelTopicPrefix + channelName)
}

func IsChannelTopic(topic Topic) bool {
	return strings.HasPrefix(string(topic), ChannelTopicPrefix)
}

func ExtractChannelName(topic Topic) string {
	if !IsChannelTopic(topic) {
		return ""
	}
	return strings.TrimPrefix(string(topic), ChannelTopicPrefix)
}

type ChannelJoinPayload struct {
	Channel string `json:"channel"`
	Token   string `json:"token"`
}

type ChannelLeavePayload struct {
	Channel string `json:"channel"`
}

type ChannelJoinedPayload struct {
	Channel  string                 `json:"channel"`
	Presence map[string]interface{} `json:"presence"`
}

type ChannelLeftPayload struct {
	Channel string `json:"channel"`
}

type ChannelMessagePayload struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Data    interface{} `json:"data"`
}

type ChannelPresencePayload struct {
	Channel  string      `json:"channel"`
	UserID   string      `json:"userId"`
	Presence interface{} `json:"presence,omitempty"`
}

type ChannelPresenceSyncPayload struct {
	Channel  string                 `json:"channel"`
	Presence map[string]interface{} `json:"presence"`
}

func (b *Bus) PublishChannelJoined(channelName string, presence map[string]interface{}) {
	b.Publish(ChannelTopic(channelName), string(ChannelJoinedAction), ChannelJoinedPayload{
		Channel:  channelName,
		Presence: presence,
	})
}

func (b *Bus) PublishChannelLeft(channelName string) {
	b.Publish(ChannelTopic(channelName), string(ChannelLeftAction), ChannelLeftPayload{
		Channel: channelName,
	})
}

func (b *Bus) PublishChannelMessage(channelName, event string, data interface{}) {
	b.Publish(ChannelTopic(channelName), string(ChannelMessageAction), ChannelMessagePayload{
		Channel: channelName,
		Event:   event,
		Data:    data,
	})
}

func (b *Bus) PublishChannelPresenceJoin(channelName, userID string, presence interface{}) {
	b.Publish(ChannelTopic(channelName), string(ChannelPresenceJoinAction), ChannelPresencePayload{
		Channel:  channelName,
		UserID:   userID,
		Presence: presence,
	})
}

func (b *Bus) PublishChannelPresenceLeave(channelName, userID string) {
	b.Publish(ChannelTopic(channelName), string(ChannelPresenceLeaveAction), ChannelPresencePayload{
		Channel: channelName,
		UserID:  userID,
	})
}

func (b *Bus) PublishChannelPresenceUpdate(channelName, userID string, presence interface{}) {
	b.Publish(ChannelTopic(channelName), string(ChannelPresenceUpdateAction), ChannelPresencePayload{
		Channel:  channelName,
		UserID:   userID,
		Presence: presence,
	})
}

func (b *Bus) PublishChannelPresenceSync(channelName string, presence map[string]interface{}) {
	b.Publish(ChannelTopic(channelName), string(ChannelPresenceSyncAction), ChannelPresenceSyncPayload{
		Channel:  channelName,
		Presence: presence,
	})
}
