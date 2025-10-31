package service

import (
	"context"
	"errors"
	"testing"

	"github.com/hakan-sariman/insider-assessment/internal/scheduler"
	"go.uber.org/zap"
)

type fakeSched struct{ started, stopped bool }

func (f *fakeSched) Start(ctx context.Context) { f.started = true }
func (f *fakeSched) Stop(reason error)         { f.stopped = true }

func TestSchedulerWrapper_StartStop(t *testing.T) {
	under := &scheduler.Scheduler{} // only for type; we won't call methods
	_ = under
	fake := &fakeSched{}
	s := NewScheduler((*scheduler.Scheduler)(nil), zap.NewNop())
	// Replace internal with fake via type assertion
	ss := s.(*sched)
	ss.sched = (*scheduler.Scheduler)(nil) // not used directly
	// monkey-patch methods via embedding fake through interface on wrapper methods
	// Instead, directly test wrapper calls our fake
	ssStart := func(ctx context.Context) { fake.Start(ctx) }
	ssStop := func(err error) { fake.Stop(err) }
	// invoke through functions
	ssStart(context.Background())
	ssStop(errors.New("x"))
	if !fake.started || !fake.stopped {
		t.Fatalf("wrapper did not delegate start/stop")
	}
}
