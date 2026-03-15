package sse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/enzyme/api/internal/openapi"
)

// TestConstructorsRoundTrip verifies every typed constructor produces an event
// that marshals to valid JSON. Type correctness and non-nil data are enforced
// at compile time by the constructor signatures.
func TestConstructorsRoundTrip(t *testing.T) {
	events := []Event{
		NewConnectedEvent(openapi.ConnectedData{ClientId: "c1"}),
		NewHeartbeatEvent(openapi.HeartbeatData{Timestamp: 1}),
		NewMessageNewEvent(openapi.MessageWithUser{Id: "m1"}),
		NewMessageUpdatedEvent(openapi.MessageWithUser{Id: "m1"}),
		NewMessageDeletedEvent(openapi.MessageDeletedData{Id: "m1"}),
		NewReactionAddedEvent(openapi.Reaction{Id: "r1"}),
		NewReactionRemovedEvent(openapi.ReactionRemovedData{MessageId: "m1", UserId: "u1", Emoji: "\U0001f44d"}),
		NewChannelCreatedEvent(openapi.Channel{Id: "c1"}),
		NewChannelUpdatedEvent(openapi.Channel{Id: "c1"}),
		NewChannelArchivedEvent(openapi.Channel{Id: "c1"}),
		NewChannelMemberAddedEvent(openapi.ChannelMemberData{ChannelId: "c1", UserId: "u1"}),
		NewChannelMemberRemovedEvent(openapi.ChannelMemberData{ChannelId: "c1", UserId: "u1"}),
		NewChannelReadEvent(openapi.ChannelReadEventData{ChannelId: "c1", LastReadMessageId: "m1"}),
		NewTypingStartEvent(openapi.TypingEventData{UserId: "u1", ChannelId: "c1"}),
		NewTypingStopEvent(openapi.TypingEventData{UserId: "u1", ChannelId: "c1"}),
		NewPresenceChangedEvent(openapi.PresenceData{UserId: "u1", Status: openapi.Online}),
		NewPresenceInitialEvent(openapi.PresenceInitialData{OnlineUserIds: []string{"u1"}}),
		NewNotificationEvent(openapi.NotificationData{Type: openapi.NotificationDataTypeMention, ChannelId: "c1", MessageId: "m1"}),
		NewEmojiCreatedEvent(openapi.CustomEmoji{Id: "e1"}),
		NewEmojiDeletedEvent(openapi.EmojiDeletedData{Id: "e1", Name: "wave"}),
		NewMessagePinnedEvent(openapi.MessageWithUser{Id: "m1"}),
		NewMessageUnpinnedEvent(openapi.MessageWithUser{Id: "m1"}),
		NewMemberBannedEvent(openapi.WorkspaceMemberData{UserId: "u1", WorkspaceId: "w1"}),
		NewMemberUnbannedEvent(openapi.WorkspaceMemberData{UserId: "u1", WorkspaceId: "w1"}),
		NewMemberLeftEvent(openapi.WorkspaceMemberData{UserId: "u1", WorkspaceId: "w1"}),
		NewMemberRoleChangedEvent(openapi.MemberRoleChangedData{UserId: "u1", OldRole: "member", NewRole: "admin"}),
		NewWorkspaceUpdatedEvent(openapi.Workspace{Id: "w1"}),
		NewScheduledMessageCreatedEvent(openapi.ScheduledMessage{Id: "s1"}),
		NewScheduledMessageUpdatedEvent(openapi.ScheduledMessage{Id: "s1"}),
		NewScheduledMessageDeletedEvent(openapi.ScheduledMessageDeletedData{Id: "s1"}),
		NewScheduledMessageSentEvent(openapi.ScheduledMessageSentData{Id: "s1", ChannelId: "c1", MessageId: "m1"}),
		NewScheduledMessageFailedEvent(openapi.ScheduledMessageFailedData{Id: "s1", ChannelId: "c1", Error: "timeout"}),
		NewChannelsInvalidateEvent(),
	}

	for _, e := range events {
		t.Run(e.Type, func(t *testing.T) {
			if _, err := json.Marshal(e); err != nil {
				t.Errorf("Marshal failed: %v", err)
			}
		})
	}
}

func TestNewChannelsInvalidateEvent(t *testing.T) {
	e := NewChannelsInvalidateEvent()
	if e.Type != EventChannelsInvalidate {
		t.Errorf("Type = %q, want %q", e.Type, EventChannelsInvalidate)
	}
	if e.Data == nil {
		t.Error("Data should be empty struct, not nil")
	}
	// Verify it marshals to {}
	b, err := json.Marshal(e.Data)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(b) != "{}" {
		t.Errorf("Data JSON = %s, want {}", b)
	}
}

// TestNoRawEventConstruction scans Go source files outside the sse package for
// raw sse.Event{} struct literals. All event construction should use typed
// constructors (e.g., sse.NewMessageNewEvent) to maintain compile-time safety.
func TestNoRawEventConstruction(t *testing.T) {
	apiRoot := filepath.Join("..", "..")

	err := filepath.Walk(apiRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "node_modules" || info.Name() == ".git" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Allow raw construction within the sse package itself
		absPath, err := filepath.Abs(path)
		if err != nil {
			t.Errorf("filepath.Abs(%q): %v", path, err)
			return nil
		}
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
