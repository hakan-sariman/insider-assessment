//go:build integration

package postgres

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hakan-sariman/insider-assessment/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func runMigrations(t *testing.T, pool *pgxpool.Pool) {
	upPath := filepath.Join("..", "..", "migrations", "001_init.up.sql")
	b, err := os.ReadFile(upPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(b)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
}

func TestPostgres_CRUD(t *testing.T) {
	url := os.Getenv("PG_URL")
	if url == "" {
		t.Skip("PG_URL not set")
	}
	ctx := context.Background()
	log := zap.NewNop()
	p, err := New(ctx, url, 3, log)
	if err != nil {
		t.Fatalf("new pg: %v", err)
	}
	defer p.Close()
	runMigrations(t, p.pool)

	msg, _ := model.NewMessage("to", "content")
	if err := p.InsertMessage(ctx, msg); err != nil {
		t.Fatalf("insert: %v", err)
	}

	unsent, err := p.FetchUnsentForUpdate(ctx, 10)
	if err != nil || len(unsent) == 0 {
		t.Fatalf("fetch unsent: %v %d", err, len(unsent))
	}

	sentAt := time.Now().UTC()
	if err := p.MarkSent(ctx, msg.ID.String(), sentAt); err != nil {
		t.Fatalf("mark sent: %v", err)
	}

	list, err := p.ListSent(ctx, 10, 0)
	if err != nil || len(list) == 0 {
		t.Fatalf("list sent: %v %d", err, len(list))
	}

	if err := p.IncrementAttempt(ctx, msg.ID.String(), nil); err != nil {
		t.Fatalf("inc attempt: %v", err)
	}
}
