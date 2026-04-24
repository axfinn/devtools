package handlers

import (
	"net/http"
	"testing"
)

func TestCloneHeadersStripsHopByHopHeaders(t *testing.T) {
	src := http.Header{}
	src.Set("Authorization", "Bearer secret")
	src.Set("Connection", "upgrade")
	src.Set("Proxy-Connection", "keep-alive")
	src.Set("Upgrade", "websocket")
	src.Set("Keep-Alive", "timeout=5")
	src.Set("TE", "trailers")
	src.Set("Trailer", "Foo")
	src.Set("Transfer-Encoding", "chunked")
	src.Set("Host", "example.com")
	src.Set("Content-Length", "123")
	src.Set("Content-Type", "application/json")
	src.Set("X-Test", "ok")

	cloned := cloneHeaders(src)

	for _, key := range []string{
		"Authorization",
		"Connection",
		"Proxy-Connection",
		"Upgrade",
		"Keep-Alive",
		"TE",
		"Trailer",
		"Transfer-Encoding",
		"Host",
		"Content-Length",
	} {
		if cloned.Get(key) != "" {
			t.Fatalf("expected header %s to be stripped, got %q", key, cloned.Get(key))
		}
	}

	if cloned.Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type to be preserved, got %q", cloned.Get("Content-Type"))
	}
	if cloned.Get("X-Test") != "ok" {
		t.Fatalf("expected X-Test to be preserved, got %q", cloned.Get("X-Test"))
	}
}
