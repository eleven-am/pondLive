package runtime

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"

	"github.com/eleven-am/pondlive/internal/protocol"
	pond "github.com/eleven-am/pondsocket/go/pondsocket"
)

type ChannelManager struct {
	sessionID string
	secret    []byte
	bus       *protocol.Bus

	refs   map[string]*ChannelRef
	counts map[string]int
	mu     sync.Mutex
}

type channelTokenPayload struct {
	SessionID   string `json:"sid"`
	ChannelName string `json:"ch"`
}

func NewChannelManager(sessionID string, bus *protocol.Bus) *ChannelManager {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)

	return &ChannelManager{
		sessionID: sessionID,
		secret:    secret,
		bus:       bus,
		refs:      make(map[string]*ChannelRef),
		counts:    make(map[string]int),
	}
}

func (m *ChannelManager) Join(channelName string) *ChannelRef {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.counts[channelName]++

	if m.counts[channelName] == 1 {
		ref := NewChannelRef(channelName, m.sessionID)
		m.refs[channelName] = ref
		m.publishJoinInstruction(channelName)
	}

	return m.refs[channelName]
}

func (m *ChannelManager) Leave(channelName string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.counts[channelName] <= 0 {
		return
	}

	m.counts[channelName]--

	if m.counts[channelName] == 0 {
		m.publishLeaveInstruction(channelName)

		if ref := m.refs[channelName]; ref != nil {
			ref.Close()
		}
		delete(m.refs, channelName)
		delete(m.counts, channelName)
	}
}

func (m *ChannelManager) GetRef(channelName string) *ChannelRef {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.refs[channelName]
}

func (m *ChannelManager) SetPondChannel(channelName string, channel *pond.Channel) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if ref := m.refs[channelName]; ref != nil {
		ref.SetPondChannel(channel)
	}
}

func (m *ChannelManager) HandleDisconnect(channelName string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if ref := m.refs[channelName]; ref != nil {
		ref.SetPondChannel(nil)
	}
}

func (m *ChannelManager) GenerateToken(channelName string) string {
	if m == nil {
		return ""
	}

	payload := channelTokenPayload{
		SessionID:   m.sessionID,
		ChannelName: channelName,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return ""
	}

	h := hmac.New(sha256.New, m.secret)
	h.Write(payloadBytes)
	signature := h.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(payloadBytes) + "." +
		base64.RawURLEncoding.EncodeToString(signature)
}

func (m *ChannelManager) ValidateToken(token, channelName string) bool {
	if m == nil {
		return false
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	signatureBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	h := hmac.New(sha256.New, m.secret)
	h.Write(payloadBytes)
	expectedSig := h.Sum(nil)

	if !hmac.Equal(signatureBytes, expectedSig) {
		return false
	}

	var payload channelTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return false
	}

	return payload.SessionID == m.sessionID && payload.ChannelName == channelName
}

func (m *ChannelManager) Close() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for name, ref := range m.refs {
		if ref != nil {
			ref.Close()
		}
		delete(m.refs, name)
	}

	m.counts = make(map[string]int)
}

func (m *ChannelManager) ActiveChannels() []string {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	names := make([]string, 0, len(m.refs))
	for name := range m.refs {
		names = append(names, name)
	}

	return names
}

func (m *ChannelManager) publishJoinInstruction(channelName string) {
	if m.bus == nil {
		return
	}

	token := m.GenerateToken(channelName)
	m.bus.Publish(protocol.ChannelTopic(channelName), string(protocol.ChannelJoinAction), protocol.ChannelJoinPayload{
		Channel: channelName,
		Token:   token,
	})
}

func (m *ChannelManager) publishLeaveInstruction(channelName string) {
	if m.bus == nil {
		return
	}

	m.bus.Publish(protocol.ChannelTopic(channelName), string(protocol.ChannelLeaveAction), protocol.ChannelLeavePayload{
		Channel: channelName,
	})
}
