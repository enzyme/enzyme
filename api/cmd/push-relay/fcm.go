package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2/google"
)

var _ Dispatcher = (*FCMClient)(nil)

const fcmEndpoint = "https://fcm.googleapis.com/v1/projects/%s/messages:send"

// FCMClient sends push notifications via the FCM HTTP v1 API.
type FCMClient struct {
	httpClient *http.Client
	endpoint   string
}

// NewFCMClient initializes an FCM client from a service account credentials file.
func NewFCMClient(credentialsFile string) (*FCMClient, error) {
	keyBytes, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("reading fcm credentials: %w", err)
	}

	var sa struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(keyBytes, &sa); err != nil {
		return nil, fmt.Errorf("parsing fcm credentials: %w", err)
	}
	if sa.ProjectID == "" {
		return nil, fmt.Errorf("fcm credentials missing project_id")
	}

	cfg, err := google.JWTConfigFromJSON(keyBytes, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return nil, fmt.Errorf("creating jwt config: %w", err)
	}

	return &FCMClient{
		httpClient: cfg.Client(context.Background()),
		endpoint:   fmt.Sprintf(fcmEndpoint, sa.ProjectID),
	}, nil
}

// fcmRequest is the top-level FCM v1 API request body.
type fcmRequest struct {
	Message fcmMessage `json:"message"`
}

type fcmMessage struct {
	Token        string            `json:"token"`
	Notification *fcmNotification  `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Android      *fcmAndroid       `json:"android,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

type fcmAndroid struct {
	Priority     string              `json:"priority,omitempty"`
	Notification *fcmAndroidNotifctn `json:"notification,omitempty"`
}

type fcmAndroidNotifctn struct {
	ChannelID string `json:"channel_id,omitempty"`
}

// fcmErrorResponse is the error body returned by the FCM v1 API.
type fcmErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Type      string `json:"@type"`
			ErrorCode string `json:"errorCode"`
		} `json:"details"`
	} `json:"error"`
}

// Send dispatches a push notification via FCM. It returns "sent", "invalid_token",
// or "error" along with any error details.
func (f *FCMClient) Send(ctx context.Context, req *NotifyRequest) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	body, err := json.Marshal(fcmRequest{
		Message: fcmMessage{
			Token: req.DeviceToken,
			Notification: &fcmNotification{
				Title: req.Title,
				Body:  req.Body,
			},
			Data: req.Data,
			Android: &fcmAndroid{
				Priority: "high",
				Notification: &fcmAndroidNotifctn{
					ChannelID: "messages",
				},
			},
		},
	})
	if err != nil {
		return "error", fmt.Errorf("fcm marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, f.endpoint, bytes.NewReader(body))
	if err != nil {
		return "error", fmt.Errorf("fcm request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return "error", fmt.Errorf("fcm send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "sent", nil
	}

	// Parse error response to determine if token is invalid.
	respBody, _ := io.ReadAll(resp.Body)
	var fcmErr fcmErrorResponse
	_ = json.Unmarshal(respBody, &fcmErr)

	fcmErrorCode := ""
	for _, d := range fcmErr.Error.Details {
		if d.Type == "type.googleapis.com/google.firebase.fcm.v1.FcmError" {
			fcmErrorCode = d.ErrorCode
			break
		}
	}

	switch {
	case resp.StatusCode == 404 && fcmErrorCode == "UNREGISTERED":
		return "invalid_token", nil
	case resp.StatusCode == 400 && fcmErrorCode == "INVALID_ARGUMENT":
		return "invalid_token", nil
	case resp.StatusCode == 403 && fcmErrorCode == "SENDER_ID_MISMATCH":
		return "invalid_token", nil
	default:
		return "error", fmt.Errorf("fcm error %d/%s: %s", resp.StatusCode, fcmErrorCode, fcmErr.Error.Message)
	}
}
