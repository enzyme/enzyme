package handler

import (
	"context"

	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/version"
)

func (h *Handler) GetServerInfo(_ context.Context, _ openapi.GetServerInfoRequestObject) (openapi.GetServerInfoResponseObject, error) {
	return openapi.GetServerInfo200JSONResponse{
		Version: version.Version,
	}, nil
}
