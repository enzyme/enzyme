package notification

import (
	"context"
	"time"

	"github.com/feather/api/internal/sse"
)

// Default email delay before sending batched notifications
const DefaultEmailDelay = 5 * time.Minute

// ChannelInfo contains channel information needed for notifications
type ChannelInfo struct {
	ID          string
	WorkspaceID string
	Name        string
	Type        string
}

// MessageInfo contains message information needed for notifications
type MessageInfo struct {
	ID             string
	ChannelID      string
	SenderID       string
	SenderName     string
	Content        string
	Mentions       []string
	ThreadParentID *string // If set, this is a thread reply
}

// ChannelMemberProvider provides channel membership information
type ChannelMemberProvider interface {
	GetMemberUserIDs(ctx context.Context, channelID string) ([]string, error)
}

// WorkspaceMemberProvider provides workspace membership information
type WorkspaceMemberProvider interface {
	GetMemberUserIDs(ctx context.Context, workspaceID string) ([]string, error)
}

// OnlineChecker checks if a user is currently online
type OnlineChecker interface {
	IsUserOnline(workspaceID, userID string) bool
}

// ThreadSubscriptionProvider provides thread subscription information
type ThreadSubscriptionProvider interface {
	GetSubscribedUserIDs(ctx context.Context, threadParentID string) ([]string, error)
}

// Service handles notification logic
type Service struct {
	prefsRepo          *PreferencesRepository
	pendingRepo        *PendingRepository
	channelProvider    ChannelMemberProvider
	threadSubProvider  ThreadSubscriptionProvider
	hub                *sse.Hub
	emailDelay         time.Duration
}

// NewService creates a new notification service
func NewService(
	prefsRepo *PreferencesRepository,
	pendingRepo *PendingRepository,
	channelProvider ChannelMemberProvider,
	hub *sse.Hub,
) *Service {
	return &Service{
		prefsRepo:         prefsRepo,
		pendingRepo:       pendingRepo,
		channelProvider:   channelProvider,
		threadSubProvider: nil, // Set via SetThreadSubscriptionProvider
		hub:               hub,
		emailDelay:        DefaultEmailDelay,
	}
}

// SetThreadSubscriptionProvider sets the thread subscription provider
// This is done separately to avoid circular dependencies
func (s *Service) SetThreadSubscriptionProvider(provider ThreadSubscriptionProvider) {
	s.threadSubProvider = provider
}

// NotificationEvent is the SSE event data for notifications
type NotificationEvent struct {
	Type           string  `json:"type"`
	ChannelID      string  `json:"channel_id"`
	MessageID      string  `json:"message_id"`
	ChannelName    string  `json:"channel_name"`
	SenderName     string  `json:"sender_name"`
	Preview        string  `json:"preview"`
	ThreadParentID *string `json:"thread_parent_id,omitempty"`
}

// Notify processes a message and sends notifications to appropriate recipients
func (s *Service) Notify(ctx context.Context, channel *ChannelInfo, msg *MessageInfo) error {
	_, notificationTypes := s.determineRecipients(ctx, channel, msg)

	for userID, notifType := range notificationTypes {
		// Skip the sender
		if userID == msg.SenderID {
			continue
		}

		// Check if user is online in this workspace
		isOnline := s.hub.IsUserOnline(channel.WorkspaceID, userID)

		// Build notification event
		event := NotificationEvent{
			Type:           notifType,
			ChannelID:      channel.ID,
			MessageID:      msg.ID,
			ChannelName:    channel.Name,
			SenderName:     msg.SenderName,
			Preview:        truncatePreview(msg.Content, 100),
			ThreadParentID: msg.ThreadParentID,
		}

		if isOnline {
			// Send real-time SSE notification
			s.hub.BroadcastToUser(channel.WorkspaceID, userID, sse.Event{
				Type: sse.EventNotification,
				Data: event,
			})
		} else {
			// Queue for email notification
			if s.shouldSendEmail(ctx, userID, channel.ID, channel.Type) {
				pending := &PendingNotification{
					UserID:           userID,
					WorkspaceID:      channel.WorkspaceID,
					ChannelID:        channel.ID,
					MessageID:        msg.ID,
					NotificationType: notifType,
					SendAfter:        time.Now().UTC().Add(s.emailDelay),
				}
				// Ignore error - email is best effort
				_ = s.pendingRepo.Create(ctx, pending)
			}
		}
	}

	return nil
}

// determineRecipients determines who should receive notifications and why
func (s *Service) determineRecipients(ctx context.Context, channel *ChannelInfo, msg *MessageInfo) ([]string, map[string]string) {
	notificationTypes := make(map[string]string) // userID -> notification type

	// Handle thread replies - notify subscribers regardless of channel notification preferences
	// Thread subscriptions override channel mute (like Slack behavior)
	if msg.ThreadParentID != nil && s.threadSubProvider != nil {
		subscriberIDs, err := s.threadSubProvider.GetSubscribedUserIDs(ctx, *msg.ThreadParentID)
		if err == nil {
			for _, userID := range subscriberIDs {
				if userID != msg.SenderID {
					notificationTypes[userID] = TypeThreadReply
				}
			}
		}
	}

	// Check for special mentions
	hasChannelMention := false
	hasHereMention := false
	hasEveryoneMention := false

	for _, mention := range msg.Mentions {
		switch mention {
		case MentionChannel:
			hasChannelMention = true
		case MentionHere:
			hasHereMention = true
		case MentionEveryone:
			hasEveryoneMention = true
		}
	}

	// Get channel members
	memberIDs, err := s.channelProvider.GetMemberUserIDs(ctx, channel.ID)
	if err != nil {
		memberIDs = []string{}
	}

	// DM channels: notify all participants
	if channel.Type == "dm" || channel.Type == "group_dm" {
		for _, userID := range memberIDs {
			if userID != msg.SenderID {
				if s.shouldNotify(ctx, userID, channel.ID, channel.Type, false) {
					notificationTypes[userID] = TypeDM
				}
			}
		}
	}

	// @channel: notify all channel members
	if hasChannelMention {
		for _, userID := range memberIDs {
			if userID != msg.SenderID && notificationTypes[userID] == "" {
				if s.shouldNotify(ctx, userID, channel.ID, channel.Type, true) {
					notificationTypes[userID] = TypeChannel
				}
			}
		}
	}

	// @here: notify all online channel members
	if hasHereMention {
		for _, userID := range memberIDs {
			if userID != msg.SenderID && notificationTypes[userID] == "" {
				if s.hub.IsUserOnline(channel.WorkspaceID, userID) {
					if s.shouldNotify(ctx, userID, channel.ID, channel.Type, true) {
						notificationTypes[userID] = TypeHere
					}
				}
			}
		}
	}

	// @everyone: notify all workspace members (would need workspace member provider)
	// For now, treat same as @channel for simplicity
	if hasEveryoneMention {
		for _, userID := range memberIDs {
			if userID != msg.SenderID && notificationTypes[userID] == "" {
				if s.shouldNotify(ctx, userID, channel.ID, channel.Type, true) {
					notificationTypes[userID] = TypeEveryone
				}
			}
		}
	}

	// Individual @mentions - these should notify even if they're not thread subscribers
	for _, mention := range msg.Mentions {
		if IsSpecialMention(mention) {
			continue
		}
		// mention is a user ID
		userID := mention
		if userID != msg.SenderID && notificationTypes[userID] == "" {
			if s.shouldNotify(ctx, userID, channel.ID, channel.Type, true) {
				notificationTypes[userID] = TypeMention
			}
		}
	}

	// Build recipient list
	recipients := make([]string, 0, len(notificationTypes))
	for userID := range notificationTypes {
		recipients = append(recipients, userID)
	}

	return recipients, notificationTypes
}

// shouldNotify checks if a user should receive notifications based on preferences
func (s *Service) shouldNotify(ctx context.Context, userID, channelID, channelType string, isMention bool) bool {
	pref, err := s.prefsRepo.GetOrDefault(ctx, userID, channelID, channelType)
	if err != nil {
		// Default to notify on mentions
		return isMention
	}

	switch pref.NotifyLevel {
	case NotifyAll:
		return true
	case NotifyMentions:
		return isMention
	case NotifyNone:
		return false
	default:
		return isMention
	}
}

// shouldSendEmail checks if a user has email notifications enabled
func (s *Service) shouldSendEmail(ctx context.Context, userID, channelID, channelType string) bool {
	pref, err := s.prefsRepo.GetOrDefault(ctx, userID, channelID, channelType)
	if err != nil {
		return true // Default to email enabled
	}
	return pref.EmailEnabled
}

// CancelPendingForUser cancels all pending email notifications for a user
// Called when user comes online
func (s *Service) CancelPendingForUser(ctx context.Context, userID string) error {
	return s.pendingRepo.DeleteForUser(ctx, userID)
}

// CancelPendingForUserInWorkspace cancels pending notifications for a user in a workspace
func (s *Service) CancelPendingForUserInWorkspace(ctx context.Context, userID, workspaceID string) error {
	return s.pendingRepo.DeleteForUserInWorkspace(ctx, userID, workspaceID)
}

// GetPreferences returns notification preferences for a channel
func (s *Service) GetPreferences(ctx context.Context, userID, channelID, channelType string) (*NotificationPreference, error) {
	return s.prefsRepo.GetOrDefault(ctx, userID, channelID, channelType)
}

// SetPreferences updates notification preferences for a channel
func (s *Service) SetPreferences(ctx context.Context, pref *NotificationPreference) error {
	return s.prefsRepo.Upsert(ctx, pref)
}

// truncatePreview truncates content for notification preview
func truncatePreview(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}
