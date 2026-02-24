package scheduled

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/testutil"
)

type mockSender struct {
	mu          sync.Mutex
	sentIDs     []string
	err         error
	failedNotif []string // IDs notified as failed
}

func (m *mockSender) ExecuteScheduledSend(_ context.Context, msg *ScheduledMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.sentIDs = append(m.sentIDs, msg.ID)
	return nil
}

func (m *mockSender) NotifyScheduledMessageFailed(_ context.Context, msg *ScheduledMessage, _ string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedNotif = append(m.failedNotif, msg.ID)
}

func (m *mockSender) getSentIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.sentIDs...)
}

func (m *mockSender) getFailedNotifIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.failedNotif...)
}

func TestWorker_ProcessDue(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "alice@example.com", "Alice")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	// Create a due message
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Due now",
		ScheduledFor: time.Now().Add(-1 * time.Minute),
	})

	// Create a future message
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Not yet",
		ScheduledFor: time.Now().Add(1 * time.Hour),
	})

	sender := &mockSender{}
	worker := NewWorker(repo, sender)
	worker.processDue(ctx)

	sent := sender.getSentIDs()
	if len(sent) != 1 {
		t.Fatalf("processDue sent %d messages, want 1", len(sent))
	}
}

func TestWorker_ProcessDue_ContinuesOnError(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "alice@example.com", "Alice")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	// Create two due messages
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "First",
		ScheduledFor: time.Now().Add(-2 * time.Minute),
	})
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Second",
		ScheduledFor: time.Now().Add(-1 * time.Minute),
	})

	// Sender fails on all messages
	sender := &mockSender{err: context.DeadlineExceeded}
	worker := NewWorker(repo, sender)
	worker.processDue(ctx)

	// Messages should be retried (status back to pending, retry_count incremented)
	due, _ := repo.ListDue(ctx)
	if len(due) != 2 {
		t.Errorf("after failed processDue, %d messages remain due, want 2", len(due))
	}
	for _, m := range due {
		if m.RetryCount != 1 {
			t.Errorf("RetryCount = %d, want 1", m.RetryCount)
		}
	}
}

func TestWorker_ProcessDue_PermanentError(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "alice@example.com", "Alice")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Perm fail",
		ScheduledFor: time.Now().Add(-1 * time.Minute),
	}
	repo.Create(ctx, msg)

	sender := &mockSender{err: &PermanentError{Err: fmt.Errorf("channel is archived")}}
	worker := NewWorker(repo, sender)
	worker.processDue(ctx)

	// Message should be immediately marked as failed (no retries)
	got, _ := repo.GetByID(ctx, msg.ID)
	if got.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", got.Status, StatusFailed)
	}
	if got.LastError != "channel is archived" {
		t.Errorf("LastError = %q, want %q", got.LastError, "channel is archived")
	}

	// Should have received failure notification
	notified := sender.getFailedNotifIDs()
	if len(notified) != 1 || notified[0] != msg.ID {
		t.Errorf("NotifyScheduledMessageFailed called with IDs %v, want [%s]", notified, msg.ID)
	}
}

func TestWorker_ProcessDue_RetryExhausted(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "alice@example.com", "Alice")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Exhaust retries",
		ScheduledFor: time.Now().Add(-1 * time.Minute),
	}
	repo.Create(ctx, msg)

	sender := &mockSender{err: context.DeadlineExceeded}
	worker := NewWorker(repo, sender)

	// Process MaxRetries times — each time increments retry_count
	for i := 0; i < MaxRetries; i++ {
		worker.processDue(ctx)
	}

	got, _ := repo.GetByID(ctx, msg.ID)
	if got.Status != StatusFailed {
		t.Errorf("Status = %q, want %q after exhausting retries", got.Status, StatusFailed)
	}

	// Should have received failure notification
	notified := sender.getFailedNotifIDs()
	if len(notified) != 1 {
		t.Errorf("NotifyScheduledMessageFailed called %d times, want 1", len(notified))
	}
}

func TestWorker_ProcessDue_AtomicClaim(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "alice@example.com", "Alice")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Claim test",
		ScheduledFor: time.Now().Add(-1 * time.Minute),
	}
	repo.Create(ctx, msg)

	// Simulate another worker claiming the message first
	claimed, _ := repo.MarkSending(ctx, msg.ID)
	if !claimed {
		t.Fatal("initial claim should succeed")
	}

	// Now the worker shouldn't be able to claim it
	sender := &mockSender{}
	worker := NewWorker(repo, sender)
	worker.processDue(ctx) // no pending messages (it's in "sending" state)

	sent := sender.getSentIDs()
	if len(sent) != 0 {
		t.Errorf("processDue sent %d messages, want 0 (already claimed)", len(sent))
	}
}

func TestWorker_ProcessDue_NoDueMessages(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "alice@example.com", "Alice")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Future",
		ScheduledFor: time.Now().Add(1 * time.Hour),
	})

	sender := &mockSender{}
	worker := NewWorker(repo, sender)
	worker.processDue(ctx)

	sent := sender.getSentIDs()
	if len(sent) != 0 {
		t.Errorf("processDue sent %d messages, want 0", len(sent))
	}
}

func TestWorker_StartAndStop(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)

	sender := &mockSender{}
	worker := NewWorker(repo, sender)
	worker.interval = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		worker.Start(ctx)
		close(done)
	}()

	// Let it tick at least once
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Worker stopped gracefully
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop within timeout")
	}
}
