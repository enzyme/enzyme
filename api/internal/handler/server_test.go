package handler

import (
	"context"
	"testing"

	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/version"
)

func TestGetServerInfo(t *testing.T) {
	h := &Handler{}

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
}
