package voice_test

import (
	"testing"

	"github.com/pion/webrtc/v4"

	"github.com/enzyme/server/internal/config"
	"github.com/enzyme/server/internal/voice"
)

func newTestSFU(t *testing.T) *voice.SFU {
	t.Helper()
	sfu, err := voice.NewSFU(config.VoiceConfig{
		Enabled:       true,
		MaxPerChannel: 15,
	})
	if err != nil {
		t.Fatalf("NewSFU: %v", err)
	}
	t.Cleanup(func() { _ = sfu.Close() })
	return sfu
}

func TestSFU_JoinCreatesRoom(t *testing.T) {
	sfu := newTestSFU(t)

	offer, err := sfu.JoinRoom("channel-1", "ws-1", "user-a")
	if err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}
	if offer == nil {
		t.Fatal("expected non-nil offer")
	}
	if offer.SDP == "" {
		t.Fatal("expected non-empty SDP in offer")
	}
}

func TestSFU_LeaveRoom(t *testing.T) {
	sfu := newTestSFU(t)

	if _, err := sfu.JoinRoom("channel-1", "ws-1", "user-a"); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}

	if err := sfu.LeaveRoom("channel-1", "user-a"); err != nil {
		t.Fatalf("LeaveRoom: %v", err)
	}

	// Leaving again should be a no-op (no error)
	if err := sfu.LeaveRoom("channel-1", "user-a"); err != nil {
		t.Fatalf("second LeaveRoom: %v", err)
	}
}

func TestSFU_RoomCleanupWhenLastLeaves(t *testing.T) {
	sfu := newTestSFU(t)

	if _, err := sfu.JoinRoom("channel-1", "ws-1", "user-a"); err != nil {
		t.Fatalf("JoinRoom user-a: %v", err)
	}
	if _, err := sfu.JoinRoom("channel-1", "ws-1", "user-b"); err != nil {
		t.Fatalf("JoinRoom user-b: %v", err)
	}

	// First leave — room still exists
	if err := sfu.LeaveRoom("channel-1", "user-a"); err != nil {
		t.Fatalf("LeaveRoom user-a: %v", err)
	}

	// Second leave — room should be cleaned up
	if err := sfu.LeaveRoom("channel-1", "user-b"); err != nil {
		t.Fatalf("LeaveRoom user-b: %v", err)
	}

	// Joining again should work (room is recreated)
	offer, err := sfu.JoinRoom("channel-1", "ws-1", "user-c")
	if err != nil {
		t.Fatalf("JoinRoom after cleanup: %v", err)
	}
	if offer == nil || offer.SDP == "" {
		t.Fatal("expected valid offer after room cleanup")
	}
}

func TestSFU_MultipleRooms(t *testing.T) {
	sfu := newTestSFU(t)

	offer1, err := sfu.JoinRoom("channel-1", "ws-1", "user-a")
	if err != nil {
		t.Fatalf("JoinRoom channel-1: %v", err)
	}
	offer2, err := sfu.JoinRoom("channel-2", "ws-1", "user-b")
	if err != nil {
		t.Fatalf("JoinRoom channel-2: %v", err)
	}

	if offer1.SDP == "" || offer2.SDP == "" {
		t.Fatal("expected valid offers for both rooms")
	}

	// Leaving one room doesn't affect the other
	if err := sfu.LeaveRoom("channel-1", "user-a"); err != nil {
		t.Fatalf("LeaveRoom channel-1: %v", err)
	}

	// Channel-2 peer should still be able to leave cleanly
	if err := sfu.LeaveRoom("channel-2", "user-b"); err != nil {
		t.Fatalf("LeaveRoom channel-2: %v", err)
	}
}

func TestSFU_RenegotiateCallback(t *testing.T) {
	sfu := newTestSFU(t)

	renegotiateCalled := make(chan struct{}, 10)
	sfu.OnRenegotiate = func(channelID, workspaceID, userID string, offer webrtc.SessionDescription) {
		renegotiateCalled <- struct{}{}
	}

	// First user joins — no renegotiation needed
	if _, err := sfu.JoinRoom("channel-1", "ws-1", "user-a"); err != nil {
		t.Fatalf("JoinRoom user-a: %v", err)
	}

	// Second user joins — renegotiation may fire when tracks are exchanged,
	// but that happens asynchronously via OnTrack, so we just verify no error
	if _, err := sfu.JoinRoom("channel-1", "ws-1", "user-b"); err != nil {
		t.Fatalf("JoinRoom user-b: %v", err)
	}
}

func TestSFU_Close(t *testing.T) {
	sfu, err := voice.NewSFU(config.VoiceConfig{
		Enabled:       true,
		MaxPerChannel: 15,
	})
	if err != nil {
		t.Fatalf("NewSFU: %v", err)
	}

	// Join some rooms
	if _, err := sfu.JoinRoom("channel-1", "ws-1", "user-a"); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}
	if _, err := sfu.JoinRoom("channel-2", "ws-1", "user-b"); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}

	// Close should clean up everything
	if err := sfu.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Leaving after close should not panic or error
	if err := sfu.LeaveRoom("channel-1", "user-a"); err != nil {
		t.Fatalf("LeaveRoom after Close: %v", err)
	}
}

func TestSFU_ICEServers(t *testing.T) {
	sfu := newTestSFU(t)

	servers := sfu.ICEServers()
	if len(servers) == 0 {
		t.Fatal("expected at least one ICE server")
	}
	// Default config should include STUN
	if servers[0].URLs[0] != "stun:stun.l.google.com:19302" {
		t.Errorf("expected Google STUN server, got %v", servers[0].URLs)
	}
}

func TestSFU_ICEServersWithTURN(t *testing.T) {
	sfu, err := voice.NewSFU(config.VoiceConfig{
		Enabled:        true,
		TURNPort:       3478,
		TURNExternalIP: "203.0.113.1",
	})
	if err != nil {
		t.Fatalf("NewSFU: %v", err)
	}
	t.Cleanup(func() { _ = sfu.Close() })

	servers := sfu.ICEServers()
	if len(servers) < 2 {
		t.Fatalf("expected STUN + TURN, got %d servers", len(servers))
	}
	turnURL := servers[1].URLs[0]
	if turnURL != "turn:203.0.113.1:3478?transport=udp" {
		t.Errorf("unexpected TURN URL: %s", turnURL)
	}
}

func TestSFU_HandleAnswer_NoPeer(t *testing.T) {
	sfu := newTestSFU(t)

	// Calling HandleAnswer without joining should return an error
	err := sfu.HandleAnswer("nonexistent", "no-user", webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  "v=0\r\n",
	})
	if err == nil {
		t.Fatal("expected error for HandleAnswer on nonexistent peer")
	}
}
