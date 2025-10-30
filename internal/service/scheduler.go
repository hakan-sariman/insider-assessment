package service

import (
	"context"

	"github.com/hakan-sariman/insider-assessment/internal/scheduler"

	"go.uber.org/zap"
)

type Scheduler interface {
	Start(ctx context.Context)
	Stop(reason error)
}

type sched struct {
	sched *scheduler.Scheduler
	log   *zap.Logger
}

func NewScheduler(s *scheduler.Scheduler, log *zap.Logger) Scheduler {
	return &sched{sched: s, log: log}
}

func (s *sched) Start(ctx context.Context) {
	s.sched.Start(ctx)
}

func (s *sched) Stop(reason error) {
	s.sched.Stop(reason)
}
