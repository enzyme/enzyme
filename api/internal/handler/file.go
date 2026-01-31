package handler

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/feather/api/internal/api"
	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/file"
	"github.com/feather/api/internal/workspace"
	"github.com/oklog/ulid/v2"
)

// UploadFile uploads a file to a channel
func (h *Handler) UploadFile(ctx context.Context, request api.UploadFileRequestObject) (api.UploadFileResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
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
				return nil, errors.New("not a member of this channel")
			}
			// Verify workspace membership for public channels
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return nil, errors.New("not a member of this workspace")
			}
		} else {
			return nil, err
		}
	}

	// Parse multipart form
	part, err := request.Body.NextPart()
	if err != nil {
		return nil, errors.New("no file provided")
	}
	defer part.Close()

	// Validate filename
	filename := sanitizeFilename(part.FileName())
	if filename == "" {
		return nil, errors.New("invalid filename")
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
		os.Remove(storagePath)
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
		os.Remove(storagePath)
		return nil, err
	}

	sizeInt := int(size)
	return api.UploadFile200JSONResponse{
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
func (h *Handler) DownloadFile(ctx context.Context, request api.DownloadFileRequestObject) (api.DownloadFileResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
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
				return nil, errors.New("not a member of this channel")
			}
			// Verify workspace membership for public channels
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return nil, errors.New("not a member of this workspace")
			}
		} else {
			return nil, err
		}
	}

	// Open file
	f, err := os.Open(attachment.StoragePath)
	if err != nil {
		return nil, errors.New("file not found on disk")
	}

	return api.DownloadFile200ApplicationoctetStreamResponse{
		Body:          f,
		ContentLength: attachment.SizeBytes,
	}, nil
}

// DeleteFile deletes a file
func (h *Handler) DeleteFile(ctx context.Context, request api.DeleteFileRequestObject) (api.DeleteFileResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
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
		return nil, errors.New("permission denied")
	}

	// Delete file from disk
	os.Remove(attachment.StoragePath)

	// Delete from database
	if err := h.fileRepo.Delete(ctx, request.Id); err != nil {
		return nil, err
	}

	return api.DeleteFile200JSONResponse{
		Success: true,
	}, nil
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
