package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/hakan-sariman/insider-assessment/internal/model"
	"github.com/hakan-sariman/insider-assessment/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Ensure Postgres implements Storage interface
var _ storage.Storage = (*Postgres)(nil)

// Postgres is the postgres storage implementation
type Postgres struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// New creates a new postgres storage
func New(ctx context.Context, url string, maxOpen int, logger *zap.Logger) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		logger.Error("pgx parse config error", zap.Error(err))
		return nil, err
	}
	// set max connections
	cfg.MaxConns = int32(maxOpen)
	// create pool
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		logger.Error("pgx pool error", zap.Error(err))
		return nil, err
	}
	return &Postgres{pool: pool, logger: logger}, nil
}

// Close closes the postgres storage
func (p *Postgres) Close() { p.pool.Close() }

// InsertMessage inserts a new message into the database
func (p *Postgres) InsertMessage(ctx context.Context, m *model.Message) error {
	p.logger.Info("InsertMessage", zap.String("to", m.To), zap.String("content", m.Content))
	_, err := p.pool.Exec(ctx, `
		INSERT INTO messages (id, "to", content, status, attempt_count, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, m.ID, m.To, m.Content, m.Status, m.AttemptCount, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		p.logger.Error("InsertMessage fail", zap.Error(err))
	}
	return err
}

// ListSent lists sent messages
func (p *Postgres) ListSent(ctx context.Context, limit, offset int) ([]model.Message, error) {
	p.logger.Info("ListSent", zap.Int("limit", limit), zap.Int("offset", offset))
	rows, err := p.pool.Query(ctx, `
		SELECT id, "to", content, status, attempt_count, created_at, updated_at, sent_at, last_error
		FROM messages
		WHERE status='sent'
		ORDER BY sent_at DESC NULLS LAST
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		p.logger.Error("ListSent query fail", zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	var out []model.Message
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.To, &m.Content, &m.Status, &m.AttemptCount, &m.CreatedAt, &m.UpdatedAt, &m.SentAt, &m.LastError); err != nil {
			p.logger.Error("ListSent scan fail", zap.Error(err))
			return nil, err
		}
		out = append(out, m)
	}
	p.logger.Info("ListSent - fetched", zap.Int("results", len(out)))
	return out, rows.Err()
}

// FetchUnsent fetches unsent messages for update
func (p *Postgres) FetchUnsent(ctx context.Context, n int) ([]model.Message, error) {
	p.logger.Debug("FetchUnsent", zap.Int("batch", n))
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		p.logger.Error("FetchUnsent: begin fail", zap.Error(err))
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	rows, err := tx.Query(ctx, `
		SELECT id, "to", content, status, attempt_count, created_at, updated_at, sent_at, last_error
		FROM messages
		WHERE status='unsent'
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`, n)
	if err != nil {
		p.logger.Error("FetchUnsent: query fail", zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	var out []model.Message
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.To, &m.Content, &m.Status, &m.AttemptCount, &m.CreatedAt, &m.UpdatedAt, &m.SentAt, &m.LastError); err != nil {
			p.logger.Error("FetchUnsent: scan fail", zap.Error(err))
			return nil, err
		}
		out = append(out, m)
	}
	p.logger.Debug("FetchUnsent - fetched", zap.Int("count", len(out)))
	if err := tx.Commit(ctx); err != nil {
		p.logger.Error("FetchUnsent: commit fail", zap.Error(err))
		return nil, err
	}
	return out, nil
}

// MarkSent marks a message as sent
func (p *Postgres) MarkSent(ctx context.Context, id string, sentAt time.Time) error {
	p.logger.Info("MarkSent", zap.String("id", id), zap.Time("sentAt", sentAt))
	ct, err := p.pool.Exec(ctx, `
		UPDATE messages SET status='sent', sent_at=$2, updated_at=now()
		WHERE id=$1 AND status='unsent'
	`, id, sentAt)
	if err != nil {
		p.logger.Error("MarkSent update fail", zap.Error(err))
		return err
	}
	p.logger.Info("MarkSent RowsAffected", zap.Int64("rows_affected", ct.RowsAffected()))
	if ct.RowsAffected() == 0 {
		p.logger.Warn("MarkSent: no rows updated, possibly already sent", zap.String("id", id))
		return errors.New("no rows updated (possibly already sent)")
	}
	return nil
}

// IncrementAttempt increments the attempt count for a message
func (p *Postgres) IncrementAttempt(ctx context.Context, id string, lastErr *string) error {
	p.logger.Info("IncrementAttempt", zap.String("id", id), zap.Stringp("lastErr", lastErr))
	_, err := p.pool.Exec(ctx, `
		UPDATE messages SET attempt_count = attempt_count + 1, last_error=$2, updated_at=now()
		WHERE id=$1
	`, id, lastErr)
	if err != nil {
		p.logger.Error("IncrementAttempt update fail", zap.Error(err))
	}
	return err
}
