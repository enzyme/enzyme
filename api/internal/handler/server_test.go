package handler

import (
	"context"
	"testing"

	"github.com/enzyme/api/internal/email"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/version"
)

func TestGetServerInfo(t *testing.T) {
	h := &Handler{emailService: email.NewTestService(false, "")}

	resp, err := h.GetServerInfo(context.Background(), openapi.GetServerInfoRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jsonResp, ok := resp.(openapi.GetServerInfo200JSONResponse)
	if !ok {
		t.Fatalf("expected GetServerInfo200JSONResponse, got %T", resp)
	}

	if jsonResp.Version != version.Version {
		t.Errorf("expected version %q, got %q", version.Version, jsonResp.Version)
	}

	if jsonResp.EmailEnabled == nil || *jsonResp.EmailEnabled != false {
		t.Error("expected email_enabled to be false")
	}
}

func TestGetServerInfo_EmailEnabled(t *testing.T) {
	h := &Handler{emailService: email.NewTestService(true, "")}

	resp, err := h.GetServerInfo(context.Background(), openapi.GetServerInfoRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jsonResp := resp.(openapi.GetServerInfo200JSONResponse)
	if jsonResp.EmailEnabled == nil || *jsonResp.EmailEnabled != true {
		t.Error("expected email_enabled to be true")
	}
}
