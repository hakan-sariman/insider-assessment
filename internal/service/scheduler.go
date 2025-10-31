package service

import (
	"context"

	"github.com/hakan-sariman/insider-assessment/internal/scheduler"

	"go.uber.org/zap"
)

// Scheduler is the scheduler service interface
type Scheduler interface {
	Start(ctx context.Context)
	Stop(reason error)
}

// sched is the scheduler service wrapper
type sched struct {
	sched *scheduler.Scheduler
	log   *zap.Logger
}

// NewScheduler creates a new scheduler service wrapper
func NewScheduler(s *scheduler.Scheduler, log *zap.Logger) Scheduler {
	return &sched{sched: s, log: log}
}

// Start starts the scheduler
func (s *sched) Start(ctx context.Context) {
	s.sched.Start(ctx)
}

// Stop stops the scheduler
func (s *sched) Stop(reason error) {
	s.sched.Stop(reason)
}
