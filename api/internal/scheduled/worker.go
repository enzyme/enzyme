package scheduled

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// MessageSender is the interface the worker uses to send messages.
// Implemented by handler.Handler via ExecuteScheduledSend.
type MessageSender interface {
	ExecuteScheduledSend(ctx context.Context, msg *ScheduledMessage) error
	NotifyScheduledMessageFailed(ctx context.Context, msg *ScheduledMessage, reason string)
}

// Worker processes due scheduled messages on a ticker.
type Worker struct {
	repo     *Repository
	sender   MessageSender
	interval time.Duration
}

// NewWorker creates a new scheduled message worker.
func NewWorker(repo *Repository, sender MessageSender) *Worker {
	return &Worker{
		repo:     repo,
		sender:   sender,
		interval: 30 * time.Second,
	}
}

// Start begins the worker loop.
func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("scheduled message worker started", "component", "scheduled")

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduled message worker stopped", "component", "scheduled")
			return
		case <-ticker.C:
			w.processDue(ctx)
		}
	}
}

func (w *Worker) processDue(ctx context.Context) {
	// Recover any messages stuck in "sending" state from a previous crash
	reset, err := w.repo.ResetStuckSending(ctx, 5*time.Minute)
	if err != nil {
		slog.Error("failed to reset stuck sending messages", "component", "scheduled", "error", err)
	} else if reset > 0 {
		slog.Warn("reset stuck sending messages", "component", "scheduled", "count", reset)
	}

	messages, err := w.repo.ListDue(ctx)
	if err != nil {
		slog.Error("failed to list due scheduled messages", "component", "scheduled", "error", err)
		return
	}

	for _, msg := range messages {
		// Atomically claim this message
		claimed, err := w.repo.MarkSending(ctx, msg.ID)
		if err != nil {
			slog.Error("failed to mark message as sending", "component", "scheduled", "id", msg.ID, "error", err)
			continue
		}
		if !claimed {
			continue // Another worker got it
		}

		if err := w.sender.ExecuteScheduledSend(ctx, &msg); err != nil {
			slog.Error("failed to send scheduled message",
				"component", "scheduled",
				"id", msg.ID,
				"channel_id", msg.ChannelID,
				"error", err,
			)

			var permErr *PermanentError
			if errors.As(err, &permErr) {
				// Permanent failure — no point retrying
				if markErr := w.repo.MarkFailed(ctx, msg.ID, err.Error()); markErr != nil {
					slog.Error("failed to mark message as failed", "component", "scheduled", "id", msg.ID, "error", markErr)
				}
				w.sender.NotifyScheduledMessageFailed(ctx, &msg, err.Error())
			} else if msg.RetryCount+1 >= MaxRetries {
				// Exhausted retries
				if markErr := w.repo.MarkFailed(ctx, msg.ID, err.Error()); markErr != nil {
					slog.Error("failed to mark message as failed", "component", "scheduled", "id", msg.ID, "error", markErr)
				}
				w.sender.NotifyScheduledMessageFailed(ctx, &msg, err.Error())
			} else {
				// Transient failure — retry later
				if retryErr := w.repo.IncrementRetry(ctx, msg.ID, err.Error()); retryErr != nil {
					slog.Error("failed to increment retry", "component", "scheduled", "id", msg.ID, "error", retryErr)
				}
			}
			continue
		}

		slog.Info("scheduled message sent",
			"component", "scheduled",
			"id", msg.ID,
			"channel_id", msg.ChannelID,
		)
	}
}
