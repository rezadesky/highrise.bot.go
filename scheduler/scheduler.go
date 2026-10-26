package scheduler

import (
	"context"
	"sync"
	"time"
)

// TaskFunc represents scheduled work.
type TaskFunc func(ctx context.Context) error

// Task encapsulates a periodic scheduled job.
type Task struct {
	ID       string
	Interval time.Duration
	Action   TaskFunc
	cancel   context.CancelFunc
}

// Scheduler manages scheduled periodic background tasks.
type Scheduler struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	ctx   context.Context
	stop  context.CancelFunc
	wg    sync.WaitGroup
}

// NewScheduler creates a background task scheduler.
func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		tasks: make(map[string]*Task),
		ctx:   ctx,
		stop:  cancel,
	}
}

// Every registers and starts a recurring task at fixed intervals.
func (s *Scheduler) Every(id string, interval time.Duration, fn TaskFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel existing if ID exists
	if old, exists := s.tasks[id]; exists {
		old.cancel()
	}

	taskCtx, taskCancel := context.WithCancel(s.ctx)
	t := &Task{
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
func (s *Scheduler) Cancel(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, exists := s.tasks[id]; exists {
		t.cancel()
		delete(s.tasks, id)
	}
}

// Stop terminates all scheduled tasks.
func (s *Scheduler) Stop() {
	s.stop()
	s.wg.Wait()
}
