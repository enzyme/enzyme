package sse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/enzyme/api/internal/openapi"
	"go.yaml.in/yaml/v3"
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
		{"channel.created", NewChannelCreatedEvent(openapi.Channel{Id: "c1"}), EventChannelCreated},
		{"channel.updated", NewChannelUpdatedEvent(openapi.Channel{Id: "c1"}), EventChannelUpdated},
		{"channel.archived", NewChannelArchivedEvent(openapi.Channel{Id: "c1"}), EventChannelArchived},
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

func TestNewChannelCreatedSignal(t *testing.T) {
	e := NewChannelCreatedSignal()
	if e.Type != EventChannelCreated {
		t.Errorf("Type = %q, want %q", e.Type, EventChannelCreated)
	}
	// Data is intentionally nil for signal events
	if e.Data != nil {
		t.Error("signal event should have nil Data")
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
			if info.Name() == "node_modules" || info.Name() == ".git" {
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
			t.Fatalf("filepath.Abs(%q): %v", path, err)
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

// TestSSEDataTypeConformance verifies that hand-written Go SSE data structs
// have the same JSON field names as their corresponding OpenAPI schemas.
// This catches drift between Go types and the spec at test time.
func TestSSEDataTypeConformance(t *testing.T) {
	specPath := filepath.Join("..", "..", "openapi.yaml")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read openapi.yaml: %v", err)
	}

	var spec map[string]interface{}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("failed to parse openapi.yaml: %v", err)
	}

	schemas := yamlNav(spec, "components", "schemas")
	if schemas == nil {
		t.Fatal("components.schemas not found in spec")
	}

	// Each entry maps a Go struct to the OpenAPI schema that defines its fields.
	// "schemaName" is the top-level schema under components.schemas.
	// "inline" means the data fields are defined inline under .properties.data.properties;
	// otherwise the data property is a $ref to a named component whose .properties we check.
	tests := []struct {
		name       string
		goStruct   interface{}
		schemaName string
		inline     bool // true = fields are inline under .properties.data
	}{
		{"ConnectedData", ConnectedData{}, "SSEEventConnected", true},
		{"HeartbeatData", HeartbeatData{}, "SSEEventHeartbeat", true},
		{"MessageDeletedData", MessageDeletedData{}, "SSEEventMessageDeleted", true},
		{"ReactionRemovedData", ReactionRemovedData{}, "SSEEventReactionRemoved", true},
		{"ChannelMemberData/Added", ChannelMemberData{}, "SSEEventChannelMemberAdded", true},
		{"MemberBannedData", MemberBannedData{}, "SSEEventMemberBanned", true},
		{"MemberUnbannedData", MemberUnbannedData{}, "SSEEventMemberUnbanned", true},
		{"MemberLeftData", MemberLeftData{}, "SSEEventMemberLeft", true},
		{"MemberRoleChangedData", MemberRoleChangedData{}, "SSEEventMemberRoleChanged", true},
		{"EmojiDeletedData", EmojiDeletedData{}, "SSEEventEmojiDeleted", true},
		{"ScheduledMessageDeletedData", ScheduledMessageDeletedData{}, "SSEEventScheduledMessageDeleted", true},
		{"ScheduledMessageSentData", ScheduledMessageSentData{}, "SSEEventScheduledMessageSent", true},
		{"ScheduledMessageFailedData", ScheduledMessageFailedData{}, "SSEEventScheduledMessageFailed", true},
		{"PresenceData", PresenceData{}, "PresenceData", false},
		{"PresenceInitialData", PresenceInitialData{}, "PresenceInitialData", false},
		{"TypingEventData", TypingEventData{}, "TypingEventData", false},
		{"ChannelReadEventData", ChannelReadEventData{}, "ChannelReadEventData", false},
		{"NotificationData", NotificationData{}, "NotificationData", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goFields := jsonFieldNames(tt.goStruct)

			var specFields []string
			if tt.inline {
				// Inline: schema.properties.data.properties
				dataNode := yamlNav(schemas, tt.schemaName, "properties", "data")
				if dataNode == nil {
					t.Fatalf("schema %s.properties.data not found", tt.schemaName)
				}
				// Resolve $ref if present
				dataNode = resolveRef(schemas, dataNode)
				props := yamlNav(dataNode, "properties")
				if props == nil {
					t.Fatalf("schema %s.properties.data.properties not found", tt.schemaName)
				}
				specFields = mapKeys(props)
			} else {
				// Named component: schema.properties
				schemaNode := yamlNav(schemas, tt.schemaName)
				if schemaNode == nil {
					t.Fatalf("schema %s not found", tt.schemaName)
				}
				schemaNode = resolveRef(schemas, schemaNode)
				props := yamlNav(schemaNode, "properties")
				if props == nil {
					t.Fatalf("schema %s.properties not found", tt.schemaName)
				}
				specFields = mapKeys(props)
			}

			sort.Strings(goFields)
			sort.Strings(specFields)

			if !reflect.DeepEqual(goFields, specFields) {
				t.Errorf("field mismatch\n  Go struct: %v\n  OpenAPI:   %v", goFields, specFields)
			}
		})
	}
}

// jsonFieldNames extracts JSON tag names from a struct, stripping ",omitempty".
func jsonFieldNames(v interface{}) []string {
	t := reflect.TypeOf(v)
	var names []string
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		names = append(names, name)
	}
	return names
}

// yamlNav navigates nested map[string]interface{} by key path.
func yamlNav(m interface{}, keys ...string) map[string]interface{} {
	cur, ok := m.(map[string]interface{})
	if !ok {
		return nil
	}
	for _, k := range keys {
		val, exists := cur[k]
		if !exists {
			return nil
		}
		cur, ok = val.(map[string]interface{})
		if !ok {
			return nil
		}
	}
	return cur
}

// resolveRef follows a $ref if present (local refs only: #/components/schemas/X).
func resolveRef(schemas map[string]interface{}, node map[string]interface{}) map[string]interface{} {
	ref, ok := node["$ref"].(string)
	if !ok {
		return node
	}
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return node
	}
	name := strings.TrimPrefix(ref, prefix)
	resolved := yamlNav(schemas, name)
	if resolved == nil {
		return node
	}
	return resolved
}

// mapKeys returns the keys of a map[string]interface{}.
func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
