package storage

import (
	"context"
	"time"

	"github.com/hakan-sariman/insider-assessment/internal/model"
)

type Storage interface {
	InsertMessage(ctx context.Context, m *model.Message) error
	ListSent(ctx context.Context, limit, offset int) ([]model.Message, error)
	FetchUnsent(ctx context.Context, n int) ([]model.Message, error)
	MarkSent(ctx context.Context, id string, sentAt time.Time) error
	IncrementAttempt(ctx context.Context, id string, lastErr *string) error
	Close()
}
