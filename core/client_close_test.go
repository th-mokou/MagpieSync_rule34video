//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"net/http"
	"testing"
)

type idleTrackingTransport struct {
	closed bool
}

func (t *idleTrackingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic("unexpected request")
}

func (t *idleTrackingTransport) CloseIdleConnections() {
	t.closed = true
}

func TestClientCloseClosesIdleConnections(t *testing.T) {
	transport := &idleTrackingTransport{}
	c := &Client{http: &http.Client{Transport: transport}}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if !transport.closed {
		t.Fatal("idle connections were not closed")
	}
}

func TestAdapterCloseClosesClientConnections(t *testing.T) {
	transport := &idleTrackingTransport{}
	adapter := NewAdapter(&Client{http: &http.Client{Transport: transport}})
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
	if !transport.closed {
		t.Fatal("adapter did not close its client")
	}
}
