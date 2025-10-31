package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hakan-sariman/insider-assessment/internal/model"
	"go.uber.org/zap"
)

type fakeStorage struct {
	insertErr error
	listErr   error
	inserted  *model.Message
	listed    []model.Message
}

func (f *fakeStorage) InsertMessage(ctx context.Context, m *model.Message) error {
	f.inserted = m
	return f.insertErr
}

func (f *fakeStorage) ListSent(ctx context.Context, limit, offset int) ([]model.Message, error) {
	return f.listed, f.listErr
}

func (f *fakeStorage) FetchUnsentForUpdate(ctx context.Context, n int) ([]model.Message, error) {
	return nil, nil
}
func (f *fakeStorage) MarkSent(ctx context.Context, id string, sentAt time.Time) error { return nil }
func (f *fakeStorage) IncrementAttempt(ctx context.Context, id string, lastErr *string) error {
	return nil
}
func (f *fakeStorage) Close() {}

func TestMessageService_CreateMessage_Success(t *testing.T) {
	store := &fakeStorage{}
	svc := NewMessageService(store, zap.NewNop(), nil, nil)
	msg, err := svc.CreateMessage(context.Background(), CreateMessageRequest{To: "x", Content: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil || msg.Status != model.StatusUnsent {
		t.Fatalf("unexpected msg: %#v", msg)
	}
	if store.inserted == nil {
		t.Fatalf("expected insert called")
	}
}

func TestMessageService_CreateMessage_ValidationError(t *testing.T) {
	store := &fakeStorage{}
	svc := NewMessageService(store, zap.NewNop(), nil, nil)
	// content > 140 chars
	long := make([]byte, model.MaxContentLength+1)
	_, err := svc.CreateMessage(context.Background(), CreateMessageRequest{To: "x", Content: string(long)})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if store.inserted != nil {
		t.Fatalf("insert should not be called on validation error")
	}
}

func TestMessageService_CreateMessage_DBError(t *testing.T) {
	store := &fakeStorage{insertErr: errors.New("db")}
	svc := NewMessageService(store, zap.NewNop(), nil, nil)
	_, err := svc.CreateMessage(context.Background(), CreateMessageRequest{To: "x", Content: "hi"})
	if err == nil {
		t.Fatalf("expected db error")
	}
}

func TestMessageService_ListSent(t *testing.T) {
	expected := []model.Message{{To: "a"}, {To: "b"}}
	store := &fakeStorage{listed: expected}
	svc := NewMessageService(store, zap.NewNop(), nil, nil)
	msgs, err := svc.ListSentMessages(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("unexpected len: %d", len(msgs))
	}
}

func TestMessageService_ListSent_Error(t *testing.T) {
	store := &fakeStorage{listErr: errors.New("db")}
	svc := NewMessageService(store, zap.NewNop(), nil, nil)
	_, err := svc.ListSentMessages(context.Background(), 10, 0)
	if err == nil {
		t.Fatalf("expected error")
	}
}
