package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/enzyme/server/internal/channel"
	"github.com/enzyme/server/internal/file"
	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/signing"
	"github.com/enzyme/server/internal/sse"
	"github.com/enzyme/server/internal/workspace"
	"github.com/oklog/ulid/v2"
)

// UploadFile uploads a file to a channel
func (h *Handler) UploadFile(ctx context.Context, request openapi.UploadFileRequestObject) (openapi.UploadFileResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UploadFile401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	if h.storage == nil {
		return openapi.UploadFile403JSONResponse{ForbiddenJSONResponse: filesDisabledResponse()}, nil
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

	// Generate storage key
	fileID := ulid.Make().String()
	ext := filepath.Ext(filename)
	storageKey := ch.WorkspaceID + "/" + string(request.Id) + "/" + fileID + ext

	// Read content into buffer with size limit (one extra byte to detect oversized files)
	contentType := part.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	data, err := io.ReadAll(io.LimitReader(part, h.maxUploadSize+1))
	if err != nil {
		return nil, err
	}
	size := int64(len(data))

	// Check if file exceeded the max upload size
	if size > h.maxUploadSize {
		return openapi.UploadFile400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "File too large")}, nil
	}

	// Upload to storage with known size
	if err := h.storage.Put(ctx, storageKey, bytes.NewReader(data), size, contentType); err != nil {
		return nil, err
	}

	// Create attachment record
	attachment := &file.Attachment{
		ChannelID:   string(request.Id),
		UserID:      &userID,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   size,
		StoragePath: storageKey,
	}

	if err := h.fileRepo.Create(ctx, attachment); err != nil {
		_ = h.storage.Delete(ctx, storageKey)
		return nil, err
	}

	return openapi.UploadFile200JSONResponse{
		File: struct {
			ContentType string `json:"content_type"`
			Filename    string `json:"filename"`
			Id          string `json:"id"`
			Size        int    `json:"size"`
		}{
			Id:          attachment.ID,
			Filename:    attachment.Filename,
			Size:        int(size),
			ContentType: attachment.ContentType,
		},
	}, nil
}

// downloadFileRedirectResponse implements DownloadFileResponseObject with a 302 redirect.
type downloadFileRedirectResponse struct {
	url string
}

func (r downloadFileRedirectResponse) VisitDownloadFileResponse(w http.ResponseWriter) error {
	w.Header().Set("Location", r.url)
	w.WriteHeader(http.StatusFound)
	return nil
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

	// Use checkFileAccess for both access verification and attachment lookup
	attachment, err := h.checkFileAccess(ctx, request.Id, userID)
	if err != nil {
		if errors.Is(err, file.ErrAttachmentNotFound) {
			return openapi.DownloadFile404JSONResponse{NotFoundJSONResponse: notFoundResponse("File not found")}, nil
		}
		return openapi.DownloadFile403JSONResponse{ForbiddenJSONResponse: notAMemberResponse("Not a member of this channel")}, nil
	}

	if h.storage == nil {
		return openapi.DownloadFile404JSONResponse{NotFoundJSONResponse: notFoundResponse("File not found")}, nil
	}

	// For S3 storage, redirect to a pre-signed URL instead of proxying
	s3URL, err := h.storage.SignedURL(ctx, attachment.StoragePath, signedURLTTL)
	if err == nil && s3URL != "" {
		return downloadFileRedirectResponse{url: s3URL}, nil
	}

	// Fall back to streaming through the server (local storage)
	rc, err := h.storage.Get(ctx, attachment.StoragePath)
	if err != nil {
		return openapi.DownloadFile404JSONResponse{NotFoundJSONResponse: notFoundResponse("File not found")}, nil
	}

	return openapi.DownloadFile200ApplicationoctetStreamResponse{
		Body:          rc,
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

	// Delete file from storage
	if h.storage != nil {
		_ = h.storage.Delete(ctx, attachment.StoragePath)
	}

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
					h.hub.BroadcastToUser(sch.WorkspaceID, smsg.UserID, sse.NewScheduledMessageUpdatedEvent(apiMsg))
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
func (h *Handler) SignFileUrl(ctx context.Context, request openapi.SignFileUrlRequestObject) (openapi.SignFileUrlResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.SignFileUrl401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Verify the user has access to the file's channel
	attachment, err := h.checkFileAccess(ctx, request.Id, userID)
	if err != nil {
		if errors.Is(err, file.ErrAttachmentNotFound) {
			return openapi.SignFileUrl404JSONResponse{NotFoundJSONResponse: notFoundResponse("File not found")}, nil
		}
		return openapi.SignFileUrl403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	url, expiresAt, err := h.signFileURL(ctx, attachment, userID)
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
		// Skip files the user doesn't have access to
		attachment, err := h.checkFileAccess(ctx, fileID, userID)
		if err != nil {
			continue
		}
		url, expiresAt, err := h.signFileURL(ctx, attachment, userID)
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

// checkFileAccess verifies the user has access to the file's channel and returns the attachment.
// Returns the attachment if access is granted, or an error (including file.ErrAttachmentNotFound).
func (h *Handler) checkFileAccess(ctx context.Context, fileID, userID string) (*file.Attachment, error) {
	attachment, err := h.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	ch, err := h.channelRepo.GetByID(ctx, attachment.ChannelID)
	if err != nil {
		return nil, err
	}

	_, err = h.channelRepo.GetMembership(ctx, userID, attachment.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return nil, fmt.Errorf("not a member of this channel")
			}
			// Verify workspace membership for public channels
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return nil, fmt.Errorf("not a member of this workspace")
			}
			return attachment, nil
		}
		return nil, err
	}
	return attachment, nil
}

// signFileURL returns a signed URL for the given file. For S3 storage it
// returns an S3 pre-signed URL directly. For local storage it falls back
// to the HMAC-signed server URL.
func (h *Handler) signFileURL(ctx context.Context, attachment *file.Attachment, userID string) (string, time.Time, error) {
	// Try S3 pre-signed URL if storage supports it
	if h.storage != nil {
		s3URL, err := h.storage.SignedURL(ctx, attachment.StoragePath, signedURLTTL)
		if err != nil {
			return "", time.Time{}, err
		}
		if s3URL != "" {
			return s3URL, time.Now().Add(signedURLTTL), nil
		}
	}

	// Fall back to HMAC-signed server URL (local storage)
	baseURL := fmt.Sprintf("%s/api/files/%s/download", h.publicURL, attachment.ID)
	return h.signer.SignedURL(baseURL, attachment.ID, userID, signedURLTTL)
}

// sanitizePathSegment strips directory traversal from a single path segment.
func sanitizePathSegment(s string) string {
	// Remove slashes and path separators
	s = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '\x00' {
			return -1
		}
		return r
	}, s)
	if s == "" || s == "." || s == ".." {
		return "_"
	}
	return s
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
		if len(ext) >= 255 {
			ext = ext[:10]
		}
		base := filename[:255-len(ext)]
		filename = base + ext
	}
	return filename
}
