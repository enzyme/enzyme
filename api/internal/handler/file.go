package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/file"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/signing"
	"github.com/enzyme/api/internal/sse"
	"github.com/enzyme/api/internal/workspace"
	"github.com/oklog/ulid/v2"
)

// UploadFile uploads a file to a channel
func (h *Handler) UploadFile(ctx context.Context, request openapi.UploadFileRequestObject) (openapi.UploadFileResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UploadFile401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check channel exists and user has access
	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Check channel membership
	_, err = h.channelRepo.GetMembership(ctx, userID, string(request.Id))
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return openapi.UploadFile403JSONResponse{ForbiddenJSONResponse: notAMemberResponse("Not a member of this channel")}, nil
			}
			// Verify workspace membership for public channels
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return openapi.UploadFile403JSONResponse{ForbiddenJSONResponse: notAMemberResponse("Not a member of this workspace")}, nil
			}
		} else {
			return nil, err
		}
	}

	// Parse multipart form
	part, err := request.Body.NextPart()
	if err != nil {
		return openapi.UploadFile400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "No file provided")}, nil
	}
	defer part.Close()

	// Validate filename
	filename := sanitizeFilename(part.FileName())
	if filename == "" {
		return openapi.UploadFile400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid filename")}, nil
	}

	// Generate storage path
	fileID := ulid.Make().String()
	ext := filepath.Ext(filename)
	storageName := fileID + ext
	storagePath := filepath.Join(h.storagePath, ch.WorkspaceID, string(request.Id), storageName)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
		return nil, err
	}

	// Create file
	dst, err := os.Create(storagePath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	// Copy file content with size limit
	size, err := io.Copy(dst, io.LimitReader(part, h.maxUploadSize))
	if err != nil {
		_ = os.Remove(storagePath)
		return nil, err
	}

	// Create attachment record
	attachment := &file.Attachment{
		ChannelID:   string(request.Id),
		UserID:      &userID,
		Filename:    filename,
		ContentType: part.Header.Get("Content-Type"),
		SizeBytes:   size,
		StoragePath: storagePath,
	}

	if attachment.ContentType == "" {
		attachment.ContentType = "application/octet-stream"
	}

	if err := h.fileRepo.Create(ctx, attachment); err != nil {
		_ = os.Remove(storagePath)
		return nil, err
	}

	sizeInt := int(size)
	return openapi.UploadFile200JSONResponse{
		File: &struct {
			ContentType *string `json:"content_type,omitempty"`
			Filename    *string `json:"filename,omitempty"`
			Id          *string `json:"id,omitempty"`
			Size        *int    `json:"size,omitempty"`
		}{
			Id:          &attachment.ID,
			Filename:    &attachment.Filename,
			Size:        &sizeInt,
			ContentType: &attachment.ContentType,
		},
	}, nil
}

// DownloadFile downloads a file
func (h *Handler) DownloadFile(ctx context.Context, request openapi.DownloadFileRequestObject) (openapi.DownloadFileResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		// Fall back to signed URL verification
		if request.Params.Expires != nil && request.Params.Uid != nil && request.Params.Sig != nil {
			err := h.signer.Verify(request.Id, *request.Params.Uid, *request.Params.Expires, *request.Params.Sig)
			if err != nil {
				if errors.Is(err, signing.ErrExpired) {
					return openapi.DownloadFile403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Signed URL has expired")}, nil
				}
				return openapi.DownloadFile403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Invalid signature")}, nil
			}
			userID = *request.Params.Uid
		} else {
			return openapi.DownloadFile401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
		}
	}

	attachment, err := h.fileRepo.GetByID(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	// Check channel access
	ch, err := h.channelRepo.GetByID(ctx, attachment.ChannelID)
	if err != nil {
		return nil, err
	}

	_, err = h.channelRepo.GetMembership(ctx, userID, attachment.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return openapi.DownloadFile403JSONResponse{ForbiddenJSONResponse: notAMemberResponse("Not a member of this channel")}, nil
			}
			// Verify workspace membership for public channels
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return openapi.DownloadFile403JSONResponse{ForbiddenJSONResponse: notAMemberResponse("Not a member of this workspace")}, nil
			}
		} else {
			return nil, err
		}
	}

	// Open file
	f, err := os.Open(attachment.StoragePath)
	if err != nil {
		return openapi.DownloadFile404JSONResponse{NotFoundJSONResponse: notFoundResponse("File not found on disk")}, nil
	}

	return openapi.DownloadFile200ApplicationoctetStreamResponse{
		Body:          f,
		ContentLength: attachment.SizeBytes,
	}, nil
}

// DeleteFile deletes a file
func (h *Handler) DeleteFile(ctx context.Context, request openapi.DeleteFileRequestObject) (openapi.DeleteFileResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.DeleteFile401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	attachment, err := h.fileRepo.GetByID(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	// Check ownership or admin status
	ch, err := h.channelRepo.GetByID(ctx, attachment.ChannelID)
	if err != nil {
		return nil, err
	}

	canDelete := attachment.UserID != nil && *attachment.UserID == userID

	if !canDelete {
		membership, err := h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
		if err == nil && workspace.CanManageMembers(membership.Role) {
			canDelete = true
		}
	}

	if !canDelete {
		return openapi.DeleteFile403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	// Delete file from disk
	_ = os.Remove(attachment.StoragePath)

	// Delete from database
	if err := h.fileRepo.Delete(ctx, request.Id); err != nil {
		return nil, err
	}

	// Remove attachment reference from any scheduled messages and notify affected users
	if h.scheduledRepo != nil {
		affected, err := h.scheduledRepo.RemoveAttachmentID(ctx, request.Id)
		if err != nil {
			slog.Error("failed to remove attachment from scheduled messages", "error", err)
		}
		for _, smsg := range affected {
			if h.hub != nil {
				if sch, err := h.channelRepo.GetByID(ctx, smsg.ChannelID); err == nil {
					apiMsg := scheduledMessageToAPI(&smsg)
					h.hub.BroadcastToUser(sch.WorkspaceID, smsg.UserID, sse.Event{
						Type: sse.EventScheduledMessageUpdated,
						Data: apiMsg,
					})
				}
			}
		}
	}

	return openapi.DeleteFile200JSONResponse{
		Success: true,
	}, nil
}

const signedURLTTL = time.Hour

// SignFileUrl generates a signed download URL for a single file.
// Authorization is enforced at download time, not at signing time.
func (h *Handler) SignFileUrl(ctx context.Context, request openapi.SignFileUrlRequestObject) (openapi.SignFileUrlResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.SignFileUrl401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	baseURL := fmt.Sprintf("%s/api/files/%s/download", h.publicURL, request.Id)
	url, expiresAt, err := h.signer.SignedURL(baseURL, request.Id, userID, signedURLTTL)
	if err != nil {
		return nil, err
	}

	return openapi.SignFileUrl200JSONResponse{
		FileId:    request.Id,
		Url:       url,
		ExpiresAt: expiresAt,
	}, nil
}

// SignFileUrls generates signed download URLs for multiple files.
func (h *Handler) SignFileUrls(ctx context.Context, request openapi.SignFileUrlsRequestObject) (openapi.SignFileUrlsResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.SignFileUrls401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	if len(request.Body.FileIds) > 100 {
		return openapi.SignFileUrls400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Maximum 100 file IDs per request")}, nil
	}

	urls := make([]openapi.SignedUrl, 0, len(request.Body.FileIds))
	for _, fileID := range request.Body.FileIds {
		baseURL := fmt.Sprintf("%s/api/files/%s/download", h.publicURL, fileID)
		url, expiresAt, err := h.signer.SignedURL(baseURL, fileID, userID, signedURLTTL)
		if err != nil {
			return nil, err
		}
		urls = append(urls, openapi.SignedUrl{
			FileId:    fileID,
			Url:       url,
			ExpiresAt: expiresAt,
		})
	}

	return openapi.SignFileUrls200JSONResponse{Urls: urls}, nil
}

func sanitizeFilename(filename string) string {
	// Remove path separators
	filename = filepath.Base(filename)
	// Remove any remaining unsafe characters
	filename = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '\x00' {
			return -1
		}
		return r
	}, filename)
	// Limit length
	if len(filename) > 255 {
		ext := filepath.Ext(filename)
		base := filename[:255-len(ext)]
		filename = base + ext
	}
	return filename
}
