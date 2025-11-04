package runtime

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// NewInMemoryPubsubProvider returns a PubsubProvider implementation that keeps
// subscriptions and fan-out completely in-memory. It is primarily intended for
// tests and single-process setups where PondSocket is unnecessary.
func NewInMemoryPubsubProvider() PubsubProvider {
	return &inMemoryPubsub{
		subscribers: make(map[SessionID]map[string]map[string]PubsubHandler),
	}
}

type inMemoryPubsub struct {
	mu          sync.RWMutex
	subscribers map[SessionID]map[string]map[string]PubsubHandler
	seq         uint64
}

func (p *inMemoryPubsub) Subscribe(session *LiveSession, topic string, handler PubsubHandler) (string, error) {
	if session == nil {
		return "", ErrPubsubUnavailable
	}
	if handler == nil {
		return "", ErrPubsubUnavailable
	}

	if topic == "" {
		return "", ErrPubsubUnavailable
	}

	token := fmt.Sprintf("%s-%d", topic, atomic.AddUint64(&p.seq, 1))

	p.mu.Lock()
	defer p.mu.Unlock()

	sid := session.ID()
	topicSubs, ok := p.subscribers[sid]
	if !ok {
		topicSubs = make(map[string]map[string]PubsubHandler)
		p.subscribers[sid] = topicSubs
	}

	listeners, ok := topicSubs[topic]
	if !ok {
		listeners = make(map[string]PubsubHandler)
		topicSubs[topic] = listeners
	}

	listeners[token] = handler
	return token, nil
}

func (p *inMemoryPubsub) Unsubscribe(session *LiveSession, token string) error {
	if session == nil || token == "" {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	topicSubs, ok := p.subscribers[session.ID()]
	if !ok {
		return nil
	}

	for topic, listeners := range topicSubs {
		if _, exists := listeners[token]; exists {
			delete(listeners, token)
			if len(listeners) == 0 {
				delete(topicSubs, topic)
			}
			break
		}
	}

	if len(topicSubs) == 0 {
		delete(p.subscribers, session.ID())
	}

	return nil
}

func (p *inMemoryPubsub) Publish(topic string, payload []byte, meta map[string]string) error {
	if topic == "" {
		return ErrPubsubUnavailable
	}

	p.mu.RLock()
	if len(p.subscribers) == 0 {
		p.mu.RUnlock()
		return ErrPubsubUnavailable
	}

	var handlers []PubsubHandler
	for _, topicSubs := range p.subscribers {
		if topicSubs == nil {
			continue
		}
		listeners := topicSubs[topic]
		if len(listeners) == 0 {
			continue
		}
		for _, handler := range listeners {
			if handler != nil {
				handlers = append(handlers, handler)
			}
		}
	}
	p.mu.RUnlock()

	if len(handlers) == 0 {
		return ErrPubsubUnavailable
	}

	for _, handler := range handlers {
		h := handler
		go h(topic, append([]byte(nil), payload...), copyStringMap(meta))
	}

	return nil
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
