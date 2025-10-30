package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/hakan-sariman/insider-assessment/internal/cache"
	"github.com/hakan-sariman/insider-assessment/internal/model"
	"github.com/hakan-sariman/insider-assessment/internal/outbound"

	"go.uber.org/zap"
)

// Store is the store interface for the scheduler
type Store interface {
	// FetchUnsentForUpdate fetches unsent messages for update
	FetchUnsentForUpdate(ctx context.Context, n int) ([]model.Message, error)
	// MarkSent marks a message as sent
	MarkSent(ctx context.Context, id string, sentAt time.Time) error
	// IncrementAttempt increments the attempt count for a message
	IncrementAttempt(ctx context.Context, id string, lastErr *string) error
}

// Config is the configuration for the scheduler
type Config struct {
	Enabled   bool
	Interval  time.Duration
	BatchSize int
}

// Scheduler is the scheduler
type Scheduler struct {
	cfg    Config
	store  Store
	cache  *cache.Redis
	sender outbound.Sender
	log    *zap.Logger

	mtx       sync.Mutex
	ctxCancel context.CancelCauseFunc
	running   bool
}

// New creates a new scheduler
func New(cfg Config, store Store, cache *cache.Redis, sender outbound.Sender, log *zap.Logger) *Scheduler {
	return &Scheduler{
		cfg:    cfg,
		store:  store,
		cache:  cache,
		sender: sender,
		log:    log,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.mtx.Lock()
	if s.running {
		s.mtx.Unlock()
		s.log.Info("scheduler already running")
		return
	}

	var sCtx context.Context
	sCtx, s.ctxCancel = context.WithCancelCause(ctx)
	s.running = true
	s.mtx.Unlock()

	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	s.log.Info("scheduler started", zap.Duration("interval", s.cfg.Interval), zap.Int("batch", s.cfg.BatchSize))
	go func() {
		for {
			select {
			case <-sCtx.Done():
				s.log.Info("scheduler context done", zap.Error(context.Cause(sCtx)))
				return
			case <-ticker.C:
				s.tick(sCtx)
			}
		}
	}()
}

// Stop stops the scheduler
func (s *Scheduler) Stop(reason error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if !s.running {
		s.log.Info("scheduler not running")
		return
	}
	s.running = false
	s.ctxCancel(reason)
}

// tick processes the unsent messages
func (s *Scheduler) tick(ctx context.Context) {
	msgs, err := s.store.FetchUnsentForUpdate(ctx, s.cfg.BatchSize)
	if err != nil {
		s.log.Error("fetch unsent", zap.Error(err))
		return
	}
	if len(msgs) == 0 {
		s.log.Info("tick: no messages to process")
		return
	}
	s.log.Info("tick: processing messages", zap.Int("count", len(msgs)))
	now := time.Now().UTC()
	for _, m := range msgs {

		if ctx.Err() != nil {
			s.log.Info("tick: context done", zap.Error(ctx.Err()))
			return
		}

		s.log.Info("tick: sending message", zap.String("id", m.ID.String()), zap.String("to", m.To))
		providerID, err := s.sender.Send(ctx, outbound.SendRequest{To: m.To, Content: m.Content})
		if err != nil {
			s.log.Warn("tick: send error, will increment attempt", zap.String("id", m.ID.String()), zap.Error(err))
			_ = s.store.IncrementAttempt(ctx, m.ID.String(), strPtr(err.Error()))
			continue
		}
		if err := s.store.MarkSent(ctx, m.ID.String(), now); err != nil {
			s.log.Error("tick: mark sent failed", zap.String("id", m.ID.String()), zap.Error(err))
			continue
		}
		s.log.Info("tick: message marked sent", zap.String("id", m.ID.String()), zap.String("provider_id", providerID))
		if s.cache != nil && providerID != "" {
			err = s.cache.SetSent(ctx, "message:"+providerID, now, 24*time.Hour)
			if err != nil {
				s.log.Error("tick: cache set sent failed", zap.String("id", m.ID.String()), zap.Error(err))
			}

		}
	}
}

func strPtr(s string) *string { return &s }
