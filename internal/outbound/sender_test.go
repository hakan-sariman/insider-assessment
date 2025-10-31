package outbound

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newSender(t *testing.T, handler http.HandlerFunc) Sender {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cfg := Config{
		URL:          server.URL,
		Timeout:      2 * time.Second,
		MaxRetries:   3,
		ExpectStatus: http.StatusOK,
		AuthHeader:   "X-Auth",
		AuthValue:    "token",
	}
	return NewHTTP(cfg, zap.NewNop())
}

func TestSend_Success(t *testing.T) {
	s := newSender(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type not set")
		}
		if r.Header.Get("X-Auth") != "token" {
			t.Fatalf("auth header missing")
		}
		b, _ := io.ReadAll(r.Body)
		var m map[string]string
		_ = json.Unmarshal(b, &m)
		if m["to"] == "" || m["content"] == "" {
			t.Fatalf("body not marshaled correctly")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "messageId": "mid-1"})
	})
	msgId, err := s.Send(context.Background(), SendRequest{To: "a", Content: "b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgId != "mid-1" {
		t.Fatalf("unexpected message id: %s", msgId)
	}
}

func TestSend_RetriesOnBadStatusThenSucceeds(t *testing.T) {
	var calls int32
	s := newSender(t, func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&calls, 1)
		if c < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("fail"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "messageId": "mid-2"})
	})
	msgId, err := s.Send(context.Background(), SendRequest{To: "a", Content: "b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgId != "mid-2" {
		t.Fatalf("unexpected message id: %s", msgId)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected at least 2 attempts, got %d", calls)
	}
}

func TestSend_InvalidJSONResponse(t *testing.T) {
	s := newSender(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	})
	_, err := s.Send(context.Background(), SendRequest{To: "a", Content: "b"})
	if err == nil {
		t.Fatalf("expected error for invalid json")
	}
}

func TestSend_UnexpectedStatus(t *testing.T) {
	// Build sender with ExpectStatus 200 but return 202
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "messageId": "mid"})
	}))
	defer server.Close()
	s := NewHTTP(Config{URL: server.URL, Timeout: time.Second, MaxRetries: 1, ExpectStatus: http.StatusOK}, zap.NewNop())
	_, err := s.Send(context.Background(), SendRequest{To: "a", Content: "b"})
	if err == nil {
		t.Fatalf("expected error for unexpected status")
	}
}
