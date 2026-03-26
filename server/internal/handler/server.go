package handler

import (
	"context"

	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/version"
)

func (h *Handler) GetServerInfo(_ context.Context, _ openapi.GetServerInfoRequestObject) (openapi.GetServerInfoResponseObject, error) {
	emailEnabled := h.emailService.IsEnabled()
	filesEnabled := h.storage != nil
	return openapi.GetServerInfo200JSONResponse{
		Version:      version.Version,
		EmailEnabled: &emailEnabled,
		FilesEnabled: &filesEnabled,
	}, nil
}
