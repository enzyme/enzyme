package sse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/enzyme/api/internal/openapi"
)

func TestTypedConstructors(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		wantType string
	}{
		{"connected", NewConnectedEvent(ConnectedData{ClientID: "c1"}), EventConnected},
		{"heartbeat", NewHeartbeatEvent(HeartbeatData{Timestamp: 1}), EventHeartbeat},
		{"message.new", NewMessageNewEvent(openapi.MessageWithUser{Id: "m1"}), EventMessageNew},
		{"message.updated", NewMessageUpdatedEvent(openapi.MessageWithUser{Id: "m1"}), EventMessageUpdated},
		{"message.deleted", NewMessageDeletedEvent(MessageDeletedData{ID: "m1"}), EventMessageDeleted},
		{"reaction.added", NewReactionAddedEvent(openapi.Reaction{Id: "r1"}), EventReactionAdded},
		{"reaction.removed", NewReactionRemovedEvent(ReactionRemovedData{MessageID: "m1", UserID: "u1", Emoji: "\U0001f44d"}), EventReactionRemoved},
		{"channel.updated", NewChannelUpdatedEvent(openapi.Channel{Id: "c1"}), EventChannelUpdated},
		{"channel.member_added", NewChannelMemberAddedEvent(ChannelMemberData{ChannelID: "c1", UserID: "u1"}), EventMemberAdded},
		{"channel.member_removed", NewChannelMemberRemovedEvent(ChannelMemberData{ChannelID: "c1", UserID: "u1"}), EventMemberRemoved},
		{"channel.read", NewChannelReadEvent(ChannelReadEventData{ChannelID: "c1", LastReadMessageID: "m1"}), EventChannelRead},
		{"typing.start", NewTypingStartEvent(TypingEventData{UserID: "u1", ChannelID: "c1"}), EventTypingStart},
		{"typing.stop", NewTypingStopEvent(TypingEventData{UserID: "u1", ChannelID: "c1"}), EventTypingStop},
		{"presence.changed", NewPresenceChangedEvent(PresenceData{UserID: "u1", Status: Online}), EventPresenceChanged},
		{"presence.initial", NewPresenceInitialEvent(PresenceInitialData{OnlineUserIDs: []string{"u1"}}), EventPresenceInitial},
		{"notification", NewNotificationEvent(NotificationData{Type: "mention", ChannelID: "c1", MessageID: "m1"}), EventNotification},
		{"emoji.created", NewEmojiCreatedEvent(openapi.CustomEmoji{Id: "e1"}), EventEmojiCreated},
		{"emoji.deleted", NewEmojiDeletedEvent(EmojiDeletedData{ID: "e1", Name: "wave"}), EventEmojiDeleted},
		{"message.pinned", NewMessagePinnedEvent(openapi.MessageWithUser{Id: "m1"}), EventMessagePinned},
		{"message.unpinned", NewMessageUnpinnedEvent(openapi.MessageWithUser{Id: "m1"}), EventMessageUnpinned},
		{"member.banned", NewMemberBannedEvent(MemberBannedData{UserID: "u1", WorkspaceID: "w1"}), EventMemberBanned},
		{"member.unbanned", NewMemberUnbannedEvent(MemberUnbannedData{UserID: "u1", WorkspaceID: "w1"}), EventMemberUnbanned},
		{"member.left", NewMemberLeftEvent(MemberLeftData{UserID: "u1", WorkspaceID: "w1"}), EventMemberLeft},
		{"member.role_changed", NewMemberRoleChangedEvent(MemberRoleChangedData{UserID: "u1", OldRole: "member", NewRole: "admin"}), EventMemberRoleChanged},
		{"workspace.updated", NewWorkspaceUpdatedEvent(openapi.Workspace{Id: "w1"}), EventWorkspaceUpdated},
		{"scheduled_message.created", NewScheduledMessageCreatedEvent(openapi.ScheduledMessage{Id: "s1"}), EventScheduledMessageCreated},
		{"scheduled_message.updated", NewScheduledMessageUpdatedEvent(openapi.ScheduledMessage{Id: "s1"}), EventScheduledMessageUpdated},
		{"scheduled_message.deleted", NewScheduledMessageDeletedEvent(ScheduledMessageDeletedData{ID: "s1"}), EventScheduledMessageDeleted},
		{"scheduled_message.sent", NewScheduledMessageSentEvent(ScheduledMessageSentData{ID: "s1", ChannelID: "c1", MessageID: "m1"}), EventScheduledMessageSent},
		{"scheduled_message.failed", NewScheduledMessageFailedEvent(ScheduledMessageFailedData{ID: "s1", ChannelID: "c1", Error: "timeout"}), EventScheduledMessageFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", tt.event.Type, tt.wantType)
			}
			if tt.event.Data == nil {
				t.Error("Data is nil")
			}
			// Verify data marshals to valid JSON
			if _, err := json.Marshal(tt.event.Data); err != nil {
				t.Errorf("Data failed to marshal: %v", err)
			}
		})
	}
}

// TestNoRawEventConstruction scans Go source files outside the sse package for
// raw sse.Event{} struct literals. All event construction should use typed
// constructors (e.g., sse.NewMessageNewEvent) to maintain compile-time safety.
func TestNoRawEventConstruction(t *testing.T) {
	apiRoot := filepath.Join("..", "..", "..")

	err := filepath.Walk(apiRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Allow raw construction within the sse package itself
		absPath, _ := filepath.Abs(path)
		if strings.Contains(absPath, filepath.Join("internal", "sse")) {
			return nil
		}
		// Skip generated and test files
		if strings.HasSuffix(path, ".gen.go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.Contains(string(data), "sse.Event{") {
			t.Errorf("%s: raw sse.Event{} construction found; use a typed constructor (e.g., sse.NewMessageNewEvent) instead", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk source tree: %v", err)
	}
}
