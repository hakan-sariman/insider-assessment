package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hakan-sariman/insider-assessment/internal/model"
	"github.com/hakan-sariman/insider-assessment/internal/outbound"
)

type fakeStore struct {
	msgs []model.Message
	sent int
}

func (f *fakeStore) FetchUnsentForUpdate(ctx context.Context, n int) ([]model.Message, error) {
	if len(f.msgs) < n {
		return f.msgs, nil
	}
	return f.msgs[:n], nil
}
func (f *fakeStore) MarkSent(ctx context.Context, id string, providerID *string, sentAt time.Time) error {
	f.sent++
	return nil
}
func (f *fakeStore) IncrementAttempt(ctx context.Context, id string, lastErr *string) error {
	return nil
}

type fakeSender struct{}

func (f fakeSender) Send(ctx context.Context, req outbound.SendRequest) (*string, error) {
	ok := "id"
	return &ok, nil
}

func TestTick_SendsTwo(t *testing.T) {
	// Prepare 5 unsent messages
	var msgs []model.Message
	for i := 0; i < 5; i++ {
		msgs = append(msgs, model.Message{ID: uuid.New(), To: "x", Content: "y"})
	}
	store := &fakeStore{msgs: msgs}
	cfg := Config{Enabled: true, Interval: time.Hour, BatchSize: 2}
	s := &Scheduler{cfg: cfg, store: store, cache: nil, send: fakeSender{}}
	s.tick(context.Background())
	if store.sent != 2 {
		t.Fatalf("expected 2 sent, got %d", store.sent)
	}
}
