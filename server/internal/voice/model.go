package voice

import (
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

// Participant represents a user connected to a voice channel (persisted in DB).
type Participant struct {
	ID            string    `json:"id"`
	ChannelID     string    `json:"channel_id"`
	UserID        string    `json:"user_id"`
	IsMuted       bool      `json:"is_muted"`
	IsDeafened    bool      `json:"is_deafened"`
	IsServerMuted bool      `json:"is_server_muted"`
	JoinedAt      time.Time `json:"joined_at"`
	DisplayName   string    `json:"display_name,omitempty"`
	AvatarURL     *string   `json:"avatar_url,omitempty"`
}

// Peer represents an in-memory WebRTC peer within a room.
type Peer struct {
	UserID         string
	PeerConnection *webrtc.PeerConnection
	AudioTrack     *webrtc.TrackRemote
	// outputTracks are local tracks forwarding this peer's audio to others
	outputTracks map[string]*webrtc.TrackLocalStaticRTP // keyed by receiver userID
	mu           sync.Mutex
}

// Room holds the set of peers in a single voice channel.
type Room struct {
	ChannelID   string
	WorkspaceID string
	Peers       map[string]*Peer // keyed by userID
	mu          sync.RWMutex
}

func NewRoom(channelID, workspaceID string) *Room {
	return &Room{
		ChannelID:   channelID,
		WorkspaceID: workspaceID,
		Peers:       make(map[string]*Peer),
	}
}

func (r *Room) PeerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Peers)
}
