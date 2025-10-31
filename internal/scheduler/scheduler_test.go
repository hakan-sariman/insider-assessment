package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hakan-sariman/insider-assessment/internal/model"
	"github.com/hakan-sariman/insider-assessment/internal/outbound"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type fakeStore struct {
	msgs          []model.Message
	sent          int
	incAttempts   int
	fetchErr      error
	markSentErr   error
	incAttemptErr error
}

func (f *fakeStore) FetchUnsentForUpdate(ctx context.Context, n int) ([]model.Message, error) {
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}
	if len(f.msgs) < n {
		return f.msgs, nil
	}
	return f.msgs[:n], nil
}
func (f *fakeStore) MarkSent(ctx context.Context, id string, sentAt time.Time) error {
	if f.markSentErr != nil {
		return f.markSentErr
	}
	f.sent++
	return nil
}
func (f *fakeStore) IncrementAttempt(ctx context.Context, id string, lastErr *string) error {
	f.incAttempts++
	if f.incAttemptErr != nil {
		return f.incAttemptErr
	}
	return nil
}

type fakeSender struct{}

func (f fakeSender) Send(ctx context.Context, req outbound.SendRequest) (string, error) {
	return "id", nil
}

func TestTick_SendsTwo(t *testing.T) {
	// Prepare 5 unsent messages
	var msgs []model.Message
	for i := 0; i < 5; i++ {
		msgs = append(msgs, model.Message{ID: uuid.New(), To: "x", Content: "y"})
	}
	store := &fakeStore{msgs: msgs}
	cfg := Config{Enabled: true, Interval: time.Hour, BatchSize: 2}
	s := &Scheduler{cfg: cfg, store: store, cache: nil, sender: fakeSender{}, log: zap.NewNop()}
	s.tick(context.Background())
	if store.sent != 2 {
		t.Fatalf("expected 2 sent, got %d", store.sent)
	}
}

// sender that can be configured per test via a function
type funcSender struct {
	fn func(ctx context.Context, req outbound.SendRequest) (string, error)
}

func (f funcSender) Send(ctx context.Context, req outbound.SendRequest) (string, error) {
	return f.fn(ctx, req)
}

func TestTick_NoMessages(t *testing.T) {
	store := &fakeStore{msgs: nil}
	cfg := Config{Enabled: true, Interval: time.Hour, BatchSize: 10}
	s := &Scheduler{cfg: cfg, store: store, cache: nil, sender: fakeSender{}, log: zap.NewNop()}
	s.tick(context.Background())
	if store.sent != 0 || store.incAttempts != 0 {
		t.Fatalf("expected no operations, got sent=%d inc=%d", store.sent, store.incAttempts)
	}
}

func TestTick_SendError_IncrementsAttempt(t *testing.T) {
	msgs := []model.Message{{ID: uuid.New(), To: "a", Content: "b"}}
	store := &fakeStore{msgs: msgs}
	sendErr := errors.New("send failed")
	sender := funcSender{fn: func(ctx context.Context, req outbound.SendRequest) (string, error) {
		return "", sendErr
	}}
	cfg := Config{Enabled: true, Interval: time.Hour, BatchSize: 5}
	s := &Scheduler{cfg: cfg, store: store, cache: nil, sender: sender, log: zap.NewNop()}
	s.tick(context.Background())
	if store.incAttempts != 1 {
		t.Fatalf("expected 1 increment attempt, got %d", store.incAttempts)
	}
	if store.sent != 0 {
		t.Fatalf("expected 0 sent on send error, got %d", store.sent)
	}
}

func TestTick_MarkSentError_DoesNotCountAsSent(t *testing.T) {
	msgs := []model.Message{{ID: uuid.New(), To: "a", Content: "b"}}
	store := &fakeStore{msgs: msgs, markSentErr: errors.New("db error")}
	sender := funcSender{fn: func(ctx context.Context, req outbound.SendRequest) (string, error) { return "mid", nil }}
	cfg := Config{Enabled: true, Interval: time.Hour, BatchSize: 5}
	s := &Scheduler{cfg: cfg, store: store, cache: nil, sender: sender, log: zap.NewNop()}
	s.tick(context.Background())
	if store.sent != 0 {
		t.Fatalf("expected 0 marked sent due to error, got %d", store.sent)
	}
}

func TestTick_RespectsAvailableLessThanBatch(t *testing.T) {
	// Only 1 message available but batch size is 3
	msgs := []model.Message{{ID: uuid.New(), To: "x", Content: "y"}}
	store := &fakeStore{msgs: msgs}
	cfg := Config{Enabled: true, Interval: time.Hour, BatchSize: 3}
	s := &Scheduler{cfg: cfg, store: store, cache: nil, sender: fakeSender{}, log: zap.NewNop()}
	s.tick(context.Background())
	if store.sent != 1 {
		t.Fatalf("expected 1 sent, got %d", store.sent)
	}
}

func TestTick_ContextCancelledStopsProcessing(t *testing.T) {
	// 3 messages, but context will be cancelled by sender after first send
	var msgs []model.Message
	for i := 0; i < 3; i++ {
		msgs = append(msgs, model.Message{ID: uuid.New(), To: "x", Content: "y"})
	}
	store := &fakeStore{msgs: msgs}

	ctx, cancel := context.WithCancel(context.Background())
	sender := funcSender{fn: func(c context.Context, req outbound.SendRequest) (string, error) {
		// cancel as soon as first send is attempted
		cancel()
		return "id", nil
	}}
	cfg := Config{Enabled: true, Interval: time.Hour, BatchSize: 3}
	s := &Scheduler{cfg: cfg, store: store, cache: nil, sender: sender, log: zap.NewNop()}
	s.tick(ctx)
	if store.sent != 1 {
		t.Fatalf("expected only 1 processed before cancel, got %d", store.sent)
	}
}

func TestTick_FetchError(t *testing.T) {
	store := &fakeStore{fetchErr: errors.New("fetch boom")}
	cfg := Config{Enabled: true, Interval: time.Hour, BatchSize: 2}
	s := &Scheduler{cfg: cfg, store: store, cache: nil, sender: fakeSender{}, log: zap.NewNop()}
	// should not panic and should not mark anything sent
	s.tick(context.Background())
	if store.sent != 0 && store.incAttempts != 0 {
		t.Fatalf("expected no operations on fetch error, got sent=%d inc=%d", store.sent, store.incAttempts)
	}
}
