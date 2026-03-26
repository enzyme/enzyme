package scheduler

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// Task defines a periodic background task.
type Task struct {
	Name       string
	Interval   time.Duration
	Fn         func(ctx context.Context) error
	RunOnStart bool // run immediately on Start(), before the first tick
}

// Scheduler manages periodic background tasks with consistent logging
// and graceful shutdown.
type Scheduler struct {
	tasks   []Task
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

// New creates a new Scheduler.
func New() *Scheduler {
	return &Scheduler{}
}

// Register adds a task to the scheduler. Must be called before Start.
func (s *Scheduler) Register(task Task) {
	if s.started {
		panic("scheduler: Register called after Start")
	}
	if task.Interval <= 0 {
		panic("scheduler: task " + task.Name + " has non-positive interval")
	}
	if task.Fn == nil {
		panic("scheduler: task " + task.Name + " has nil Fn")
	}
	s.tasks = append(s.tasks, task)
}

// Start launches a goroutine for each registered task.
func (s *Scheduler) Start(ctx context.Context) {
	if s.started {
		panic("scheduler: Start called twice")
	}
	s.started = true
	ctx, s.cancel = context.WithCancel(ctx)

	for _, task := range s.tasks {
		s.wg.Add(1)
		go s.run(ctx, task)
	}

	slog.Info("scheduler started", "tasks", len(s.tasks))
}

// Stop signals all tasks to stop and waits for in-flight executions to finish.
// It respects the provided context's deadline to avoid blocking shutdown indefinitely.
func (s *Scheduler) Stop(ctx context.Context) {
	if s.cancel != nil {
		s.cancel()
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("scheduler stopped")
	case <-ctx.Done():
		slog.Warn("scheduler stop timed out, some tasks may still be running")
	}
}

func (s *Scheduler) run(ctx context.Context, task Task) {
	defer s.wg.Done()

	if task.RunOnStart {
		s.execute(ctx, task)
	}

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.execute(ctx, task)
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, task Task) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("scheduler task panicked", "task", task.Name, "panic", r, "stack", string(debug.Stack()))
		}
	}()

	err := task.Fn(ctx)
	if err != nil {
		slog.Error("scheduler task failed", "task", task.Name, "error", err)
	}
}
