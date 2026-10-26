package highrisebot

import (
	"context"
	"sync"
	"time"
)

// ScheduledTaskFunc represents scheduled work.
type ScheduledTaskFunc func(ctx context.Context) error

// ScheduledTask encapsulates a periodic scheduled job.
type ScheduledTask struct {
	ID       string
	Interval time.Duration
	Action   ScheduledTaskFunc
	cancel   context.CancelFunc
}

// BotScheduler manages scheduled periodic background tasks.
type BotScheduler struct {
	mu    sync.RWMutex
	tasks map[string]*ScheduledTask
	ctx   context.Context
	stop  context.CancelFunc
	wg    sync.WaitGroup
}

// NewBotScheduler creates a background task scheduler.
func NewBotScheduler() *BotScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &BotScheduler{
		tasks: make(map[string]*ScheduledTask),
		ctx:   ctx,
		stop:  cancel,
	}
}

// Every registers and starts a recurring task at fixed intervals.
func (s *BotScheduler) Every(id string, interval time.Duration, fn ScheduledTaskFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if old, exists := s.tasks[id]; exists {
		old.cancel()
	}

	taskCtx, taskCancel := context.WithCancel(s.ctx)
	t := &ScheduledTask{
		ID:       id,
		Interval: interval,
		Action:   fn,
		cancel:   taskCancel,
	}
	s.tasks[id] = t

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-taskCtx.Done():
				return
			case <-ticker.C:
				_ = fn(taskCtx)
			}
		}
	}()
}

// Cancel terminates a specific scheduled task.
func (s *BotScheduler) Cancel(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, exists := s.tasks[id]; exists {
		t.cancel()
		delete(s.tasks, id)
	}
}

// Stop terminates all scheduled tasks.
func (s *BotScheduler) Stop() {
	s.stop()
	s.wg.Wait()
}
