package handler

import (
	"context"

	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/version"
)

func (h *Handler) GetServerInfo(_ context.Context, _ openapi.GetServerInfoRequestObject) (openapi.GetServerInfoResponseObject, error) {
	emailEnabled := h.emailService.IsEnabled()
	return openapi.GetServerInfo200JSONResponse{
		Version:      version.Version,
		EmailEnabled: &emailEnabled,
	}, nil
}
