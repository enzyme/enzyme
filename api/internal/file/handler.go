package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/feather/api/internal/auth"
	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/workspace"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

type Handler struct {
	repo          *Repository
	channelRepo   *channel.Repository
	workspaceRepo *workspace.Repository
	storagePath   string
	maxUploadSize int64
}

func NewHandler(repo *Repository, channelRepo *channel.Repository, workspaceRepo *workspace.Repository, storagePath string, maxUploadSize int64) *Handler {
	return &Handler{
		repo:          repo,
		channelRepo:   channelRepo,
		workspaceRepo: workspaceRepo,
		storagePath:   storagePath,
		maxUploadSize: maxUploadSize,
	}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	// Check channel exists and user has access
	ch, err := h.channelRepo.GetByID(r.Context(), channelID)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check channel membership
	_, err = h.channelRepo.GetMembership(r.Context(), userID, channelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				writeError(w, http.StatusForbidden, "NOT_CHANNEL_MEMBER", "You are not a member of this channel")
				return
			}
			// Verify workspace membership for public channels
			_, err = h.workspaceRepo.GetMembership(r.Context(), userID, ch.WorkspaceID)
			if err != nil {
				writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
				return
			}
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
			return
		}
	}

	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "File exceeds maximum upload size")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "NO_FILE", "No file provided")
		return
	}
	defer file.Close()

	// Validate filename
	filename := sanitizeFilename(header.Filename)
	if filename == "" {
		writeError(w, http.StatusBadRequest, "INVALID_FILENAME", "Invalid filename")
		return
	}

	// Generate storage path
	fileID := ulid.Make().String()
	ext := filepath.Ext(filename)
	storageName := fileID + ext
	storagePath := filepath.Join(h.storagePath, ch.WorkspaceID, channelID, storageName)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Create file
	dst, err := os.Create(storagePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}
	defer dst.Close()

	// Copy file content
	size, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(storagePath)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Create attachment record
	attachment := &Attachment{
		ChannelID:   channelID,
		UserID:      &userID,
		Filename:    filename,
		ContentType: header.Header.Get("Content-Type"),
		SizeBytes:   size,
		StoragePath: storagePath,
	}

	if attachment.ContentType == "" {
		attachment.ContentType = "application/octet-stream"
	}

	if err := h.repo.Create(r.Context(), attachment); err != nil {
		os.Remove(storagePath)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"attachment": attachment,
	})
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	attachment, err := h.repo.GetByID(r.Context(), fileID)
	if err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check channel access
	ch, err := h.channelRepo.GetByID(r.Context(), attachment.ChannelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	_, err = h.channelRepo.GetMembership(r.Context(), userID, attachment.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				writeError(w, http.StatusForbidden, "NOT_CHANNEL_MEMBER", "You are not a member of this channel")
				return
			}
			// Verify workspace membership for public channels
			_, err = h.workspaceRepo.GetMembership(r.Context(), userID, ch.WorkspaceID)
			if err != nil {
				writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
				return
			}
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
			return
		}
	}

	// Open file
	file, err := os.Open(attachment.StoragePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "FILE_NOT_FOUND", "File not found on disk")
		return
	}
	defer file.Close()

	// Set headers
	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", attachment.SizeBytes))

	io.Copy(w, file)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	attachment, err := h.repo.GetByID(r.Context(), fileID)
	if err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check ownership or admin status
	ch, err := h.channelRepo.GetByID(r.Context(), attachment.ChannelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	canDelete := attachment.UserID != nil && *attachment.UserID == userID

	if !canDelete {
		membership, err := h.workspaceRepo.GetMembership(r.Context(), userID, ch.WorkspaceID)
		if err == nil && workspace.CanManageMembers(membership.Role) {
			canDelete = true
		}
	}

	if !canDelete {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to delete this file")
		return
	}

	// Delete file from disk
	os.Remove(attachment.StoragePath)

	// Delete from database
	if err := h.repo.Delete(r.Context(), fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
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

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
