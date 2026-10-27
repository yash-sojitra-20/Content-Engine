package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/yash-sojitra-20/content-engine/internal/observability"
	"github.com/yash-sojitra-20/content-engine/internal/pipeline"
)

// Scheduler manages recurring cron execution of the pipeline.
type Scheduler struct {
	cron        *cron.Cron
	pipeline    *pipeline.Pipeline
	schedule    string
	location    *time.Location
	isBusy      atomic.Bool
	activeRunWg sync.WaitGroup
	activeRunCancel context.CancelFunc
	cancelMu    sync.Mutex
}

// NewScheduler creates a Scheduler configured with timezone and schedule expression.
func NewScheduler(scheduleExpr string, loc *time.Location, p *pipeline.Pipeline) (*Scheduler, error) {
	if loc == nil {
		loc = time.UTC
	}

	c := cron.New(cron.WithLocation(loc))

	s := &Scheduler{
		cron:     c,
		pipeline: p,
		schedule: scheduleExpr,
		location: loc,
	}

	_, err := c.AddFunc(scheduleExpr, s.onTick)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", scheduleExpr, err)
	}

	return s, nil
}

// Start begins scheduler background loop.
func (s *Scheduler) Start() {
	slog.Info("Starting cron scheduler",
		slog.String("schedule", s.schedule),
		slog.String("timezone", s.location.String()),
	)
	s.cron.Start()
}

// onTick executes when cron schedule triggers.
func (s *Scheduler) onTick() {
	// Prevent overlapping runs
	if !s.isBusy.CompareAndSwap(false, true) {
		slog.Warn("Skipping scheduled run: previous run is still active and in progress",
			observability.StageAttr("scheduler_skip"),
		)
		return
	}
	defer s.isBusy.Store(false)

	s.activeRunWg.Add(1)
	defer s.activeRunWg.Done()

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelMu.Lock()
	s.activeRunCancel = cancel
	s.cancelMu.Unlock()

	defer func() {
		s.cancelMu.Lock()
		s.activeRunCancel = nil
		s.cancelMu.Unlock()
		cancel()
	}()

	slog.Info("Triggering scheduled pipeline run", observability.StageAttr("scheduler_tick"))

	_, err := s.pipeline.Execute(ctx)
	if err != nil {
		// Log error, but DO NOT stop the scheduler!
		slog.Error("Scheduled pipeline run encountered an error",
			observability.StageAttr("scheduler_run_failed"),
			observability.ErrorAttr(err),
		)
		return
	}

	slog.Info("Scheduled pipeline run finished successfully", observability.StageAttr("scheduler_run_succeeded"))
}

// Stop gracefully stops the scheduler and waits for active runs up to gracePeriod.
func (s *Scheduler) Stop(gracePeriod time.Duration) {
	slog.Info("Stopping cron scheduler", slog.Duration("grace_period", gracePeriod))
	cronCtx := s.cron.Stop()

	// Wait for cron internal loop to stop
	<-cronCtx.Done()

	// Wait for active pipeline run or cancel after gracePeriod
	done := make(chan struct{})
	go func() {
		s.activeRunWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("All active scheduler runs completed cleanly")
	case <-time.After(gracePeriod):
		slog.Warn("Grace period expired, canceling in-progress pipeline run",
			slog.Duration("grace_period", gracePeriod),
		)
		s.cancelMu.Lock()
		if s.activeRunCancel != nil {
			s.activeRunCancel()
		}
		s.cancelMu.Unlock()

		// Wait briefly for worker to exit
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			slog.Error("Active run failed to exit within cancellation window")
		}
	}
}

// IsBusy returns whether a scheduled run is actively executing.
func (s *Scheduler) IsBusy() bool {
	return s.isBusy.Load()
}
