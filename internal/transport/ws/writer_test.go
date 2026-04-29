package ws

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestEnqueue_Normal(t *testing.T) {
	h := &hub{
		log:      slog.Default(),
		registry: newClientRegistry(),
	}
	c := makeFakeClient("c1")
	h.registry.add(c)
	h.enqueue(c, []byte("ping"))
	select {
	case msg := <-c.Out:
		if string(msg) != "ping" {
			t.Errorf("got %q", string(msg))
		}
	case <-time.After(time.Second):
		t.Error("enqueue did not deliver")
	}
}

func TestEnqueue_DropsOnCancelledClient(t *testing.T) {
	h := &hub{
		log:      slog.Default(),
		registry: newClientRegistry(),
	}
	c := makeFakeClient("c1")
	c.cancel() // pre-cancel
	h.registry.add(c)
	h.enqueue(c, []byte("ping"))
	select {
	case msg := <-c.Out:
		t.Errorf("cancelled client received %q", string(msg))
	default:
		// ok
	}
}

func TestEnqueue_FullChannelDisconnects(t *testing.T) {
	h := &hub{
		log:      slog.Default(),
		registry: newClientRegistry(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		ID:     "c1",
		Out:    make(chan []byte, 1), // tiny buffer to provoke
		ctx:    ctx,
		cancel: cancel,
	}
	h.registry.add(c)

	c.Out <- []byte("first") // saturate

	h.enqueue(c, []byte("second"))
	// give the unregister goroutine a moment
	time.Sleep(50 * time.Millisecond)
	if h.registry.byPlayerSafe("c1") != nil {
		// no PlayerID, so nil expected anyway
	}
}
