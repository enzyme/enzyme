package voice

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/pion/turn/v4"

	"github.com/enzyme/server/internal/config"
)

// TURNServer wraps the embedded Pion TURN server.
type TURNServer struct {
	server *turn.Server
}

// NewTURNServer starts an embedded TURN server on the configured UDP port.
func NewTURNServer(cfg config.VoiceConfig) (*TURNServer, error) {
	if cfg.TURNExternalIP == "" {
		return nil, fmt.Errorf("voice.turn_external_ip is required when voice is enabled")
	}

	udpListener, err := net.ListenPacket("udp4", fmt.Sprintf("0.0.0.0:%d", cfg.TURNPort))
	if err != nil {
		return nil, fmt.Errorf("listening on UDP port %d: %w", cfg.TURNPort, err)
	}

	server, err := turn.NewServer(turn.ServerConfig{
		Realm: "enzyme",
		AuthHandler: func(username, realm string, srcAddr net.Addr) ([]byte, bool) {
			if username == cfg.TURNUsername {
				return turn.GenerateAuthKey(username, realm, cfg.TURNPassword), true
			}
			return nil, false
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorPortRange{
					RelayAddress: net.ParseIP(cfg.TURNExternalIP),
					Address:      "0.0.0.0",
					MinPort:      uint16(cfg.TURNRelayMin),
					MaxPort:      uint16(cfg.TURNRelayMax),
				},
			},
		},
	})
	if err != nil {
		_ = udpListener.Close()
		return nil, fmt.Errorf("creating TURN server: %w", err)
	}

	slog.Info("TURN server started", "port", cfg.TURNPort, "external_ip", cfg.TURNExternalIP)
	return &TURNServer{server: server}, nil
}

// Close shuts down the TURN server.
func (t *TURNServer) Close() error {
	if t.server != nil {
		return t.server.Close()
	}
	return nil
}
