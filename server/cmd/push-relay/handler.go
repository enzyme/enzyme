package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Dispatcher abstracts push notification delivery for FCM and APNs.
type Dispatcher interface {
	Send(ctx context.Context, req *NotifyRequest) (string, error)
}

type notifyHandler struct {
	fcm  Dispatcher
	apns Dispatcher
}

func (h *notifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Limit request body size.
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req NotifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var status string
	var sendErr error

	switch req.Platform {
	case "fcm":
		if h.fcm == nil {
			writeJSON(w, http.StatusServiceUnavailable, NotifyResponse{
				Status: "error",
				Error:  "FCM client not configured",
			})
			return
		}
		status, sendErr = h.fcm.Send(r.Context(), &req)
	case "apns":
		if h.apns == nil {
			writeJSON(w, http.StatusServiceUnavailable, NotifyResponse{
				Status: "error",
				Error:  "APNs client not configured",
			})
			return
		}
		status, sendErr = h.apns.Send(r.Context(), &req)
	}

	if sendErr != nil {
		slog.Error("push dispatch failed",
			"platform", req.Platform,
			"error", sendErr.Error(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
		writeJSON(w, http.StatusBadGateway, NotifyResponse{Status: "error", Error: "dispatch failed"})
		return
	}

	slog.Info("push notification dispatched",
		"platform", req.Platform,
		"status", status,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	writeJSON(w, http.StatusOK, NotifyResponse{Status: status})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
