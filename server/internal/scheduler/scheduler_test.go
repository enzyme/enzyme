package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestTaskRunsAtInterval(t *testing.T) {
	var count atomic.Int32

	s := New()
	s.Register(Task{
		Name:     "counter",
		Interval: 10 * time.Millisecond,
		Fn: func(ctx context.Context) error {
			count.Add(1)
			return nil
		},
	})

	ctx := context.Background()
	s.Start(ctx)

	time.Sleep(55 * time.Millisecond)
	s.Stop(context.Background())

	got := count.Load()
	if got < 3 || got > 7 {
		t.Errorf("expected 3-7 runs in 55ms at 10ms interval, got %d", got)
	}
}

func TestRunOnStart(t *testing.T) {
	var ran atomic.Bool

	s := New()
	s.Register(Task{
		Name:       "immediate",
		Interval:   1 * time.Hour, // won't tick in test
		RunOnStart: true,
		Fn: func(ctx context.Context) error {
			ran.Store(true)
			return nil
		},
	})

	ctx := context.Background()
	s.Start(ctx)

	time.Sleep(20 * time.Millisecond)
	s.Stop(context.Background())

	if !ran.Load() {
		t.Error("RunOnStart task did not execute")
	}
}

func TestStopWaitsForInFlight(t *testing.T) {
	var finished atomic.Bool

	s := New()
	s.Register(Task{
		Name:       "slow",
		Interval:   1 * time.Hour,
		RunOnStart: true,
		Fn: func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			finished.Store(true)
			return nil
		},
	})

	ctx := context.Background()
	s.Start(ctx)

	// Give the RunOnStart task time to begin
	time.Sleep(10 * time.Millisecond)
	s.Stop(context.Background())

	if !finished.Load() {
		t.Error("Stop() returned before in-flight task finished")
	}
}

func TestStopRespectsContextDeadline(t *testing.T) {
	s := New()
	s.Register(Task{
		Name:       "blocked",
		Interval:   1 * time.Hour,
		RunOnStart: true,
		Fn: func(ctx context.Context) error {
			// Block until context is cancelled, then take a while to "finish"
			<-ctx.Done()
			time.Sleep(500 * time.Millisecond)
			return nil
		},
	})

	s.Start(context.Background())
	time.Sleep(10 * time.Millisecond)

	// Stop with a short deadline — should return before the task finishes
	stopCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	s.Stop(stopCtx)
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Errorf("Stop() blocked too long (%v), should have timed out at ~50ms", elapsed)
	}
}

func TestErrorDoesNotStopScheduler(t *testing.T) {
	var count atomic.Int32

	s := New()
	s.Register(Task{
		Name:     "failing",
		Interval: 10 * time.Millisecond,
		Fn: func(ctx context.Context) error {
			count.Add(1)
			return errors.New("oops")
		},
	})

	ctx := context.Background()
	s.Start(ctx)

	time.Sleep(55 * time.Millisecond)
	s.Stop(context.Background())

	got := count.Load()
	if got < 3 {
		t.Errorf("expected task to keep running despite errors, got %d runs", got)
	}
}

func TestPanicRecovery(t *testing.T) {
	var count atomic.Int32

	s := New()
	s.Register(Task{
		Name:     "panicky",
		Interval: 10 * time.Millisecond,
		Fn: func(ctx context.Context) error {
			count.Add(1)
			panic("boom")
		},
	})

	ctx := context.Background()
	s.Start(ctx)

	time.Sleep(55 * time.Millisecond)
	s.Stop(context.Background())

	got := count.Load()
	if got < 3 {
		t.Errorf("expected task to keep running despite panics, got %d runs", got)
	}
}

func TestRegisterValidation(t *testing.T) {
	s := New()

	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: expected panic", name)
			}
		}()
		fn()
	}

	assertPanics("zero interval", func() {
		s.Register(Task{Name: "bad", Interval: 0, Fn: func(ctx context.Context) error { return nil }})
	})

	assertPanics("nil fn", func() {
		s.Register(Task{Name: "bad", Interval: time.Second, Fn: nil})
	})
}
