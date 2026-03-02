package notification

import (
	"context"
	"log/slog"

	"github.com/enzyme/api/internal/email"
	"github.com/enzyme/api/internal/sse"
	"github.com/enzyme/api/internal/user"
)

// EmailWorker processes pending notifications and sends digest emails
type EmailWorker struct {
	pendingRepo  *PendingRepository
	userRepo     *user.Repository
	emailService *email.Service
	hub          *sse.Hub
}

// NewEmailWorker creates a new email notification worker
func NewEmailWorker(
	pendingRepo *PendingRepository,
	userRepo *user.Repository,
	emailService *email.Service,
	hub *sse.Hub,
) *EmailWorker {
	return &EmailWorker{
		pendingRepo:  pendingRepo,
		userRepo:     userRepo,
		emailService: emailService,
		hub:          hub,
	}
}

// ProcessPending processes all pending notifications ready to be sent.
func (w *EmailWorker) ProcessPending(ctx context.Context) error {
	// Get notifications grouped by user
	grouped, err := w.pendingRepo.GetGroupedByUser(ctx)
	if err != nil {
		return err
	}

	if len(grouped) == 0 {
		return nil
	}

	for userID, notifications := range grouped {
		// Check if user is now online in any of the workspaces
		allOnline := true
		for _, n := range notifications {
			if !w.hub.IsUserOnline(n.WorkspaceID, userID) {
				allOnline = false
				break
			}
		}

		// If user is online everywhere, cancel pending notifications
		if allOnline {
			ids := make([]string, len(notifications))
			for i, n := range notifications {
				ids[i] = n.ID
			}
			_ = w.pendingRepo.DeleteByIDs(ctx, ids)
			continue
		}

		// Get user email
		usr, err := w.userRepo.GetByID(ctx, userID)
		if err != nil {
			slog.Error("error getting user for notification", "component", "notification", "user_id", userID, "error", err)
			continue
		}

		// Skip unverified users — digest emails aren't deliverable
		if usr.EmailVerifiedAt == nil {
			slog.Warn("skipping notification digest for unverified email", "component", "notification", "user_id", userID)
			ids := make([]string, len(notifications))
			for i, n := range notifications {
				ids[i] = n.ID
			}
			if err := w.pendingRepo.DeleteByIDs(ctx, ids); err != nil {
				slog.Error("error deleting pending notifications for unverified user", "component", "notification", "user_id", userID, "error", err)
			}
			continue
		}

		// Group notifications by workspace for better email structure
		byWorkspace := make(map[string][]PendingNotification)
		for _, n := range notifications {
			byWorkspace[n.WorkspaceID] = append(byWorkspace[n.WorkspaceID], n)
		}

		// Send digest email for each workspace
		for workspaceID, wsNotifications := range byWorkspace {
			// Skip if user came online in this workspace
			if w.hub.IsUserOnline(workspaceID, userID) {
				ids := make([]string, len(wsNotifications))
				for i, n := range wsNotifications {
					ids[i] = n.ID
				}
				_ = w.pendingRepo.DeleteByIDs(ctx, ids)
				continue
			}

			// Build digest data
			items := make([]email.NotificationDigestItem, len(wsNotifications))
			for i, n := range wsNotifications {
				items[i] = email.NotificationDigestItem{
					ChannelName: "", // Would need channel name from DB
					SenderName:  "", // Would need sender name from DB
					Preview:     "", // Would need message content from DB
					Type:        n.NotificationType,
				}
			}

			// Send email
			err := w.emailService.SendNotificationDigest(ctx, usr.Email, email.NotificationDigestData{
				WorkspaceName: "Enzyme", // Would need workspace name
				Items:         items,
				WorkspaceURL:  w.emailService.GetPublicURL(),
			})
			if err != nil {
				slog.Error("error sending notification digest", "component", "notification", "to", usr.Email, "error", err)
				continue
			}

			// Delete processed notifications
			ids := make([]string, len(wsNotifications))
			for i, n := range wsNotifications {
				ids[i] = n.ID
			}
			_ = w.pendingRepo.DeleteByIDs(ctx, ids)
		}
	}
	return nil
}

// CancelForUser cancels all pending notifications for a user
// Called when user connects to SSE
func (w *EmailWorker) CancelForUser(ctx context.Context, userID string) {
	_ = w.pendingRepo.DeleteForUser(ctx, userID)
}
