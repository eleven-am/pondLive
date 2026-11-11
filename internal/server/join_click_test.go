package server

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/eleven-am/pondlive/internal/session"
)

func joinRaw(h *liveHarness, b bootDoc) (*websocket.Conn, error) {
	c, _, err := h.dial(h.srv.URL, nil)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c, c.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"t":"join","sid":%q,"tok":%q,"ver":%d,"bs":0}`, b.SID, b.Tok, b.Ver)))
}

func TestConcurrentJoinsWithClicksDoNotRace(t *testing.T) {
	h := newHarness(t, TransportConfig{})
	defer h.close()
	b := h.boot()
	hid := b.handlerID(t)

	const workers, cycles = 4, 25
	var wg sync.WaitGroup
	var clicks atomic.Int64
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for cycle := 0; cycle < cycles; cycle++ {
				c, err := joinRaw(h, b)
				if err != nil {
					if c != nil {
						c.CloseNow()
					}
					time.Sleep(dialBackoff)
					continue
				}
				go func() {
					for {
						if _, _, err := c.Read(context.Background()); err != nil {
							return
						}
					}
				}()
				for i := 0; i < 30; i++ {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					err := c.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"t":"h","i":%d,"sid":%q,"a":"click","p":{},"cs":%d}`, hid, b.SID, i+1)))
					cancel()
					if err != nil {
						break
					}
					clicks.Add(1)
				}
				c.CloseNow()
			}
		}()
	}
	wg.Wait()
	waitUntilWithin(t, DefaultJoinDeadline+5*time.Second, "endpoint handlers to finish", func() bool { return liveHandlers(h.app.endpoint) == 0 })

	c, err := joinRaw(h, b)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	m := readFrame(t, c)
	if m["t"] != "joined" || !strings.Contains(fmt.Sprint(m), "tok") {
		t.Fatalf("join after the click storm failed: %v", m)
	}
	if clicks.Load() == 0 {
		t.Fatal("no clicks were sent")
	}
}

func TestSSRSessionsRunOnAStartedActor(t *testing.T) {
	h := newHarness(t, TransportConfig{})
	defer h.close()
	b := h.boot()
	live, ok := h.app.registry.Lookup(session.SessionID(b.SID))
	if !ok {
		t.Fatal("session not registered")
	}
	if live.Session().Done() == nil {
		t.Fatal("SSR created an unstarted session")
	}
}
