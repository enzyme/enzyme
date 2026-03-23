package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"
)

var _ Dispatcher = (*APNsClient)(nil)

// APNsClient wraps the Apple Push Notification service client.
type APNsClient struct {
	client   *apns2.Client
	bundleID string
}

// NewAPNsClient initializes an APNs client from a .p8 auth key file.
func NewAPNsClient(keyFile, keyID, teamID, bundleID string, production bool) (*APNsClient, error) {
	authKey, err := token.AuthKeyFromFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("loading apns auth key: %w", err)
	}

	tok := &token.Token{
		AuthKey: authKey,
		KeyID:   keyID,
		TeamID:  teamID,
	}

	client := apns2.NewTokenClient(tok)
	if production {
		client = client.Production()
	} else {
		client = client.Development()
	}

	return &APNsClient{client: client, bundleID: bundleID}, nil
}

// Send dispatches a push notification via APNs. It returns "sent", "invalid_token",
// or "error" along with any error details.
func (a *APNsClient) Send(ctx context.Context, req *NotifyRequest) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	p := payload.NewPayload().
		AlertTitle(req.Title).
		AlertBody(req.Body).
		Sound("default").
		MutableContent()

	for k, v := range req.Data {
		if k == "aps" {
			continue
		}
		p.Custom(k, v)
	}

	notification := &apns2.Notification{
		DeviceToken: req.DeviceToken,
		Topic:       a.bundleID,
		Payload:     p,
		Priority:    apns2.PriorityHigh,
		PushType:    apns2.PushTypeAlert,
	}

	resp, err := a.client.PushWithContext(ctx, notification)
	if err != nil {
		return "error", fmt.Errorf("apns send: %w", err)
	}

	if resp.StatusCode == 410 || resp.Reason == apns2.ReasonBadDeviceToken || resp.Reason == apns2.ReasonUnregistered {
		return "invalid_token", nil
	}

	if resp.StatusCode != 200 {
		return "error", fmt.Errorf("apns error: %d %s", resp.StatusCode, resp.Reason)
	}

	return "sent", nil
}
