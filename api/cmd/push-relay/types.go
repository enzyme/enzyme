package main

import (
	"errors"
	"fmt"
	"strings"
)

// NotifyRequest is the payload received from Enzyme servers.
type NotifyRequest struct {
	DeviceToken string            `json:"device_token"`
	Platform    string            `json:"platform"` // "fcm" or "apns"
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Data        map[string]string `json:"data"` // channel_id, message_id, workspace_id, server_url
}

// NotifyResponse is returned to the calling Enzyme server.
type NotifyResponse struct {
	Status string `json:"status"` // "sent", "invalid_token", "error"
	Error  string `json:"error,omitempty"`
}

// HealthResponse is returned by the GET /health endpoint.
type HealthResponse struct {
	Status string `json:"status"`
	FCM    bool   `json:"fcm"`
	APNs   bool   `json:"apns"`
}

const (
	maxRequestBodySize = 4 * 1024
	maxDeviceTokenLen  = 256
	maxTitleLen        = 200
	maxBodyLen         = 1000
	maxDataKeys        = 10
	maxDataValueLen    = 256
)

var allowedDataKeys = map[string]bool{
	"channel_id":       true,
	"message_id":       true,
	"workspace_id":     true,
	"channel_name":     true,
	"thread_parent_id": true,
	"server_url":       true,
}

func (r *NotifyRequest) Validate() error {
	r.DeviceToken = strings.TrimSpace(r.DeviceToken)
	if r.DeviceToken == "" {
		return errors.New("device_token is required")
	}
	if len(r.DeviceToken) > maxDeviceTokenLen {
		return fmt.Errorf("device_token exceeds %d characters", maxDeviceTokenLen)
	}
	if r.Platform != "fcm" && r.Platform != "apns" {
		return errors.New("platform must be \"fcm\" or \"apns\"")
	}
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		return errors.New("title is required")
	}
	if len(r.Title) > maxTitleLen {
		return fmt.Errorf("title exceeds %d characters", maxTitleLen)
	}
	if len(r.Body) > maxBodyLen {
		return fmt.Errorf("body exceeds %d characters", maxBodyLen)
	}
	if len(r.Data) > maxDataKeys {
		return fmt.Errorf("data exceeds %d keys", maxDataKeys)
	}
	for k, v := range r.Data {
		if !allowedDataKeys[k] {
			return fmt.Errorf("data key %q is not allowed", k)
		}
		if len(v) > maxDataValueLen {
			return fmt.Errorf("data value for %q exceeds %d characters", k, maxDataValueLen)
		}
	}
	return nil
}
