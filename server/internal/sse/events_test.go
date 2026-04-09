package sse

import (
	"strings"
	"testing"
)

func TestSerialize_AssignsID(t *testing.T) {
	e := Event{Type: "test", Data: "hello"}
	serialized, err := e.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.ID == "" {
		t.Fatal("expected ID to be assigned via pointer receiver")
	}
	if !strings.Contains(string(serialized.Frame), "id: "+e.ID+"\n") {
		t.Errorf("frame does not contain expected id line, got: %s", serialized.Frame)
	}
}

func TestSerialize_PreservesExistingID(t *testing.T) {
	e := Event{ID: "EXISTING123", Type: "test", Data: "hello"}
	serialized, err := e.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.ID != "EXISTING123" {
		t.Fatalf("expected ID to remain EXISTING123, got %s", e.ID)
	}
	if !strings.Contains(string(serialized.Frame), "id: EXISTING123\n") {
		t.Errorf("frame does not contain expected id line, got: %s", serialized.Frame)
	}
}

func TestSerialize_FrameFormat(t *testing.T) {
	e := Event{ID: "ABC", Type: "test.event", Data: map[string]string{"key": "value"}}
	serialized, err := e.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	frame := string(serialized.Frame)

	// Must start with "id: ABC\n"
	if !strings.HasPrefix(frame, "id: ABC\n") {
		t.Errorf("frame should start with id line, got: %q", frame)
	}
	// Must have "data: " line with JSON
	if !strings.Contains(frame, "data: {") {
		t.Errorf("frame should contain data line with JSON, got: %q", frame)
	}
	// Must end with double newline
	if !strings.HasSuffix(frame, "\n\n") {
		t.Errorf("frame should end with double newline, got: %q", frame)
	}
	// JSON payload should include the type
	if !strings.Contains(frame, `"type":"test.event"`) {
		t.Errorf("frame JSON should contain event type, got: %q", frame)
	}
}

func TestSerialize_RejectsInvalidIDCharacters(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"newline", "bad\nid"},
		{"carriage return", "bad\rid"},
		{"null byte", "bad\x00id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{ID: tt.id, Type: "test"}
			_, err := e.Serialize()
			if err == nil {
				t.Fatal("expected error for invalid ID characters")
			}
			if !strings.Contains(err.Error(), "invalid characters") {
				t.Errorf("expected 'invalid characters' in error, got: %v", err)
			}
		})
	}
}

func TestSerialize_MarshalError(t *testing.T) {
	// func values cannot be marshaled to JSON
	e := Event{ID: "X", Type: "test", Data: func() {}}
	_, err := e.Serialize()
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !strings.Contains(err.Error(), "marshaling SSE event") {
		t.Errorf("expected wrapped error message, got: %v", err)
	}
}
