package server

import (
	"strings"
	"sync"

	"github.com/eleven-am/pondlive/internal/protocol"
	pond "github.com/eleven-am/pondsocket/go/pondsocket"
)

const (
	pubsubLobbyPattern    = "pondlive:*"
	pubsubChannelPrefix   = "pondlive:"
	channelTokenAssignKey = "channel.token"
)

type PubSubLobby struct {
	lobby    *pond.Lobby
	endpoint *pond.Endpoint
	registry *SessionRegistry
	mu       sync.RWMutex
}

type pubsubJoinPayload struct {
	Token string `json:"token"`
}

func NewPubSubLobby(endpoint *pond.Endpoint, registry *SessionRegistry) *PubSubLobby {
	p := &PubSubLobby{
		endpoint: endpoint,
		registry: registry,
	}

	p.lobby = endpoint.CreateChannel(pubsubLobbyPattern, p.handleJoin)
	p.lobby.OnOutgoing("*", p.handleOutgoing)
	p.lobby.OnLeave(p.handleLeave)
	p.lobby.OnMessage("*", p.handleMessage)

	return p
}

func (p *PubSubLobby) handleJoin(ctx *pond.JoinContext) error {
	var payload pubsubJoinPayload
	if err := ctx.ParsePayload(&payload); err != nil {
		return ctx.Decline(pond.StatusBadRequest, "invalid join payload")
	}

	if payload.Token == "" {
		return ctx.Decline(pond.StatusUnauthorized, "missing channel token")
	}

	user := ctx.GetUser()
	if user == nil {
		return ctx.Decline(pond.StatusUnauthorized, "no user context")
	}

	sess, _, ok := p.registry.LookupByConnection(user.UserID)
	if !ok || sess == nil {
		return ctx.Decline(pond.StatusUnauthorized, "session not found")
	}

	channelName := extractAppChannelName(ctx.Channel.Name())
	if channelName == "" {
		return ctx.Decline(pond.StatusBadRequest, "invalid channel name")
	}

	channelManager := sess.ChannelManager()
	if channelManager == nil {
		return ctx.Decline(pond.StatusInternalServerError, "channel manager not available")
	}

	if !channelManager.ValidateToken(payload.Token, channelName) {
		return ctx.Decline(pond.StatusUnauthorized, "invalid channel token")
	}

	ctx.Accept()
	if errStr := ctx.Error(); errStr != "" {
		return nil
	}

	presence := ctx.GetAllPresence()

	channelManager.SetPondChannel(channelName, ctx.Channel)

	if bus := sess.Bus(); bus != nil {
		bus.PublishChannelJoined(channelName, presence)
		bus.PublishChannelPresenceSync(channelName, presence)
	}

	return nil
}

func (p *PubSubLobby) handleOutgoing(ctx *pond.OutgoingContext) error {
	event := ctx.GetEvent()

	if event == "ACKNOWLEDGE" || event == "EXIT_ACKNOWLEDGE" {
		return nil
	}

	user := ctx.User
	if user == nil {
		ctx.Block()
		return nil
	}

	sess, _, ok := p.registry.LookupByConnection(user.UserID)
	if !ok || sess == nil {
		ctx.Block()
		return nil
	}

	bus := sess.Bus()
	if bus == nil {
		ctx.Block()
		return nil
	}

	channelName := extractAppChannelName(ctx.Channel.Name())
	if channelName == "" {
		ctx.Block()
		return nil
	}

	payload := ctx.GetPayload()

	switch {
	case strings.HasPrefix(event, "presence:"):
		p.routePresenceEvent(bus, channelName, event, payload)
	default:
		bus.PublishChannelMessage(channelName, event, payload)
	}

	ctx.Block()
	return nil
}

func (p *PubSubLobby) routePresenceEvent(bus *protocol.Bus, channelName, event string, payload interface{}) {
	switch event {
	case "presence:join":
		if data, ok := payload.(map[string]interface{}); ok {
			userID, _ := data["user_id"].(string)
			presence := data["presence"]
			bus.PublishChannelPresenceJoin(channelName, userID, presence)
		}
	case "presence:leave":
		if data, ok := payload.(map[string]interface{}); ok {
			userID, _ := data["user_id"].(string)
			bus.PublishChannelPresenceLeave(channelName, userID)
		}
	case "presence:update":
		if data, ok := payload.(map[string]interface{}); ok {
			userID, _ := data["user_id"].(string)
			presence := data["presence"]
			bus.PublishChannelPresenceUpdate(channelName, userID, presence)
		}
	}
}

func (p *PubSubLobby) handleMessage(ctx *pond.EventContext) error {
	return nil
}

func (p *PubSubLobby) handleLeave(ctx *pond.LeaveContext) {
	if ctx == nil || ctx.User == nil {
		return
	}

	sess, _, ok := p.registry.LookupByConnection(ctx.User.UserID)
	if !ok || sess == nil {
		return
	}

	channelName := extractAppChannelName(ctx.Channel.Name())
	if channelName == "" {
		return
	}

	channelManager := sess.ChannelManager()
	if channelManager != nil {
		channelManager.HandleDisconnect(channelName)
	}

	if bus := sess.Bus(); bus != nil {
		bus.PublishChannelLeft(channelName)
	}
}

func extractAppChannelName(pondChannelName string) string {
	if !strings.HasPrefix(pondChannelName, pubsubChannelPrefix) {
		return ""
	}
	return strings.TrimPrefix(pondChannelName, pubsubChannelPrefix)
}

func PondChannelName(appChannelName string) string {
	return pubsubChannelPrefix + appChannelName
}
