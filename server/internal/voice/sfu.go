package voice

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"

	"github.com/enzyme/server/internal/config"
)

// OnICECandidateFunc is called when the server generates an ICE candidate for a peer.
type OnICECandidateFunc func(channelID, workspaceID, userID string, candidate *webrtc.ICECandidate)

// OnRenegotiateFunc is called when SDP renegotiation is needed (participant join/leave).
type OnRenegotiateFunc func(channelID, workspaceID, userID string, offer webrtc.SessionDescription)

// OnPeerDisconnectedFunc is called when a peer connection fails or disconnects unexpectedly.
type OnPeerDisconnectedFunc func(channelID, workspaceID, userID string)

// SFU manages WebRTC rooms for voice channels.
type SFU struct {
	rooms  map[string]*Room
	mu     sync.RWMutex
	config config.VoiceConfig
	api    *webrtc.API

	OnICECandidate     OnICECandidateFunc
	OnRenegotiate      OnRenegotiateFunc
	OnPeerDisconnected OnPeerDisconnectedFunc
}

func NewSFU(cfg config.VoiceConfig) (*SFU, error) {
	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return nil, fmt.Errorf("registering codecs: %w", err)
	}

	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return nil, fmt.Errorf("registering interceptors: %w", err)
	}

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(m),
		webrtc.WithInterceptorRegistry(i),
	)

	return &SFU{
		rooms:  make(map[string]*Room),
		config: cfg,
		api:    api,
	}, nil
}

// ICEServers returns the ICE server configuration for clients, including
// the embedded TURN server if configured.
func (s *SFU) ICEServers() []webrtc.ICEServer {
	servers := []webrtc.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	}
	if s.config.TURNExternalIP != "" {
		servers = append(servers, webrtc.ICEServer{
			URLs: []string{
				fmt.Sprintf("turn:%s:%d?transport=udp", s.config.TURNExternalIP, s.config.TURNPort),
			},
			Username:   s.config.TURNUsername,
			Credential: s.config.TURNPassword,
		})
	}
	return servers
}

// JoinRoom creates a PeerConnection for the user in the specified channel room
// and returns an SDP offer for the client to answer.
func (s *SFU) JoinRoom(channelID, workspaceID, userID string) (*webrtc.SessionDescription, error) {
	s.mu.Lock()
	room, ok := s.rooms[channelID]
	if !ok {
		room = NewRoom(channelID, workspaceID)
		s.rooms[channelID] = room
	}
	s.mu.Unlock()

	room.mu.Lock()
	defer room.mu.Unlock()

	// Reject duplicate joins — close any existing peer first
	if existing, ok := room.Peers[userID]; ok {
		_ = existing.PeerConnection.Close()
		delete(room.Peers, userID)
	}

	// Create peer connection
	pc, err := s.api.NewPeerConnection(webrtc.Configuration{
		ICEServers: s.ICEServers(),
	})
	if err != nil {
		s.cleanupEmptyRoom(channelID, room)
		return nil, fmt.Errorf("creating peer connection: %w", err)
	}

	peer := &Peer{
		UserID:         userID,
		PeerConnection: pc,
		outputTracks:   make(map[string]*webrtc.TrackLocalStaticRTP),
	}

	// Add a transceiver for receiving audio from this peer
	_, err = pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	})
	if err != nil {
		_ = pc.Close()
		s.cleanupEmptyRoom(channelID, room)
		return nil, fmt.Errorf("adding audio transceiver: %w", err)
	}

	// For each existing peer, add a send-only track to forward their audio to the new peer
	for existingUserID, existingPeer := range room.Peers {
		existingPeer.mu.Lock()
		audioTrack := existingPeer.AudioTrack
		existingPeer.mu.Unlock()

		if audioTrack != nil {
			track, err := webrtc.NewTrackLocalStaticRTP(
				audioTrack.Codec().RTPCodecCapability,
				fmt.Sprintf("audio-%s", existingUserID),
				fmt.Sprintf("stream-%s", existingUserID),
			)
			if err != nil {
				slog.Error("creating local track for existing peer", "error", err)
				continue
			}
			if _, err := pc.AddTrack(track); err != nil {
				slog.Error("adding track to new peer", "error", err)
				continue
			}
			existingPeer.mu.Lock()
			existingPeer.outputTracks[userID] = track
			existingPeer.mu.Unlock()
		}
	}

	// Handle incoming audio track from this peer
	pc.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		peer.mu.Lock()
		peer.AudioTrack = remoteTrack
		peer.mu.Unlock()

		// Create output tracks for all other peers in the room
		room.mu.RLock()
		otherPeers := make(map[string]*Peer)
		for uid, p := range room.Peers {
			if uid != userID {
				otherPeers[uid] = p
			}
		}
		room.mu.RUnlock()

		for otherUID, otherPeer := range otherPeers {
			track, err := webrtc.NewTrackLocalStaticRTP(
				remoteTrack.Codec().RTPCodecCapability,
				fmt.Sprintf("audio-%s", userID),
				fmt.Sprintf("stream-%s", userID),
			)
			if err != nil {
				slog.Error("creating local track", "error", err)
				continue
			}
			if _, err := otherPeer.PeerConnection.AddTrack(track); err != nil {
				slog.Error("adding track to peer", "error", err)
				continue
			}
			peer.mu.Lock()
			peer.outputTracks[otherUID] = track
			peer.mu.Unlock()

			// Trigger renegotiation for the other peer
			if s.OnRenegotiate != nil {
				go s.renegotiate(channelID, room.WorkspaceID, otherUID, otherPeer)
			}
		}

		// Forward RTP packets to all output tracks
		buf := make([]byte, 1500)
		for {
			n, _, readErr := remoteTrack.Read(buf)
			if readErr != nil {
				return
			}
			peer.mu.Lock()
			for _, outputTrack := range peer.outputTracks {
				if _, writeErr := outputTrack.Write(buf[:n]); writeErr != nil {
					slog.Debug("error writing to output track", "error", writeErr)
				}
			}
			peer.mu.Unlock()
		}
	})

	// Handle ICE candidates
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil && s.OnICECandidate != nil {
			s.OnICECandidate(channelID, room.WorkspaceID, userID, candidate)
		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		slog.Debug("peer connection state changed", "channel", channelID, "user", userID, "state", state)
		if state == webrtc.PeerConnectionStateFailed {
			go func() {
				_ = s.LeaveRoom(channelID, userID)
				if s.OnPeerDisconnected != nil {
					s.OnPeerDisconnected(channelID, room.WorkspaceID, userID)
				}
			}()
		}
	})

	room.Peers[userID] = peer

	// Create offer
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		delete(room.Peers, userID)
		_ = pc.Close()
		s.cleanupEmptyRoom(channelID, room)
		return nil, fmt.Errorf("creating offer: %w", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		delete(room.Peers, userID)
		_ = pc.Close()
		s.cleanupEmptyRoom(channelID, room)
		return nil, fmt.Errorf("setting local description: %w", err)
	}

	return pc.LocalDescription(), nil
}

// HandleAnswer processes the client's SDP answer.
func (s *SFU) HandleAnswer(channelID, userID string, answer webrtc.SessionDescription) error {
	peer := s.getPeer(channelID, userID)
	if peer == nil {
		return fmt.Errorf("peer not found: %s in %s", userID, channelID)
	}
	return peer.PeerConnection.SetRemoteDescription(answer)
}

// HandleICECandidate adds a remote ICE candidate for a peer.
func (s *SFU) HandleICECandidate(channelID, userID string, candidate webrtc.ICECandidateInit) error {
	peer := s.getPeer(channelID, userID)
	if peer == nil {
		return fmt.Errorf("peer not found: %s in %s", userID, channelID)
	}
	return peer.PeerConnection.AddICECandidate(candidate)
}

// LeaveRoom removes a peer from a room and cleans up its resources.
func (s *SFU) LeaveRoom(channelID, userID string) error {
	s.mu.Lock()
	room, ok := s.rooms[channelID]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	room.mu.Lock()
	peer, ok := room.Peers[userID]
	if !ok {
		room.mu.Unlock()
		return nil
	}
	delete(room.Peers, userID)

	// Remove output tracks from other peers that were forwarding to this peer
	// and trigger renegotiation so they drop the stale transceivers.
	peersToRenegotiate := make(map[string]*Peer)
	for otherUID, otherPeer := range room.Peers {
		otherPeer.mu.Lock()
		delete(otherPeer.outputTracks, userID)
		otherPeer.mu.Unlock()
		peersToRenegotiate[otherUID] = otherPeer
	}

	isEmpty := len(room.Peers) == 0
	room.mu.Unlock()

	// Close the peer connection (unblocks RTP read goroutines)
	if err := peer.PeerConnection.Close(); err != nil {
		slog.Error("closing peer connection", "error", err)
	}

	// Trigger renegotiation for remaining peers outside the room lock
	if s.OnRenegotiate != nil {
		for otherUID, otherPeer := range peersToRenegotiate {
			go s.renegotiate(channelID, room.WorkspaceID, otherUID, otherPeer)
		}
	}

	// Clean up empty room
	if isEmpty {
		s.mu.Lock()
		// Re-check under lock in case someone joined between releasing room.mu and acquiring s.mu
		if room.PeerCount() == 0 {
			delete(s.rooms, channelID)
		}
		s.mu.Unlock()
	}

	return nil
}

// Close shuts down all rooms and peer connections.
func (s *SFU) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for channelID, room := range s.rooms {
		room.mu.Lock()
		for _, peer := range room.Peers {
			_ = peer.PeerConnection.Close()
		}
		room.mu.Unlock()
		delete(s.rooms, channelID)
	}
	return nil
}

func (s *SFU) getPeer(channelID, userID string) *Peer {
	s.mu.RLock()
	room, ok := s.rooms[channelID]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	room.mu.RLock()
	defer room.mu.RUnlock()
	return room.Peers[userID]
}

func (s *SFU) renegotiate(channelID, workspaceID, userID string, peer *Peer) {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	offer, err := peer.PeerConnection.CreateOffer(nil)
	if err != nil {
		slog.Error("creating renegotiation offer", "error", err)
		return
	}
	if err := peer.PeerConnection.SetLocalDescription(offer); err != nil {
		slog.Error("setting local description for renegotiation", "error", err)
		return
	}
	if s.OnRenegotiate != nil {
		s.OnRenegotiate(channelID, workspaceID, userID, *peer.PeerConnection.LocalDescription())
	}
}

// cleanupEmptyRoom removes a room from the map if it has no peers.
// Must NOT be called while holding room.mu (PeerCount takes its own lock).
func (s *SFU) cleanupEmptyRoom(channelID string, room *Room) {
	if room.PeerCount() == 0 {
		s.mu.Lock()
		if room.PeerCount() == 0 {
			delete(s.rooms, channelID)
		}
		s.mu.Unlock()
	}
}
