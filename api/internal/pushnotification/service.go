package pushnotification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Service handles sending push notifications via the relay.
type Service struct {
	repo     *Repository
	relayURL string
	client   *http.Client
}

// NewService creates a new push notification service.
func NewService(repo *Repository, relayURL string) *Service {
	return &Service{
		repo:     repo,
		relayURL: relayURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Send dispatches push notifications for a user. Returns true if at least one
// notification was successfully dispatched (meaning we should suppress email fallback).
func (s *Service) Send(ctx context.Context, userID string, data NotificationData) bool {
	tokens, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		slog.Error("push: failed to list device tokens", "user_id", userID, "error", err)
		return false
	}
	if len(tokens) == 0 {
		return false
	}

	relayData := RelayRequestData{
		ChannelID:      data.ChannelID,
		MessageID:      data.MessageID,
		WorkspaceID:    data.WorkspaceID,
		ChannelName:    data.ChannelName,
		ThreadParentID: data.ThreadParentID,
		ServerURL:      data.ServerURL,
	}

	var (
		mu         sync.Mutex
		dispatched bool
	)

	g, gCtx := errgroup.WithContext(ctx)
	for _, t := range tokens {
		t := t // capture loop var
		g.Go(func() error {
			req := RelayRequest{
				DeviceToken: t.Token,
				Platform:    t.Platform,
				Title:       data.Title,
				Body:        data.Body,
				Data:        relayData,
			}

			resp, err := s.sendToRelay(gCtx, req)
			if err != nil {
				slog.Error("push: relay request failed", "token_id", t.ID, "error", err)
				return nil // don't abort other sends
			}

			switch resp.Status {
			case "sent":
				mu.Lock()
				dispatched = true
				mu.Unlock()
			case "invalid_token":
				slog.Info("push: removing invalid token", "token_id", t.ID)
				if err := s.repo.Delete(ctx, userID, t.Token); err != nil {
					slog.Error("push: failed to delete invalid token", "token_id", t.ID, "error", err)
				}
			default:
				slog.Error("push: relay returned error", "token_id", t.ID, "status", resp.Status, "error", resp.Error)
			}
			return nil
		})
	}
	_ = g.Wait() // errors are handled per-goroutine above

	return dispatched
}

func (s *Service) sendToRelay(ctx context.Context, payload RelayRequest) (*RelayResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.relayURL+"/notify", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("relay returned HTTP %d", resp.StatusCode)
	}

	var relayResp RelayResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&relayResp); err != nil {
		return nil, fmt.Errorf("decoding relay response: %w", err)
	}
	return &relayResp, nil
}
