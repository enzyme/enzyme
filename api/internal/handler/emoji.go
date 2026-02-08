package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/feather/api/internal/emoji"
	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/sse"
	"github.com/feather/api/internal/workspace"
	"github.com/go-chi/chi/v5"
)

var (
	emojiNameRegexp  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)
	allowedEmojiTypes = map[string]string{
		"image/png": ".png",
		"image/gif": ".gif",
	}
	maxEmojiSize int64 = 256 * 1024 // 256KB
)

func emojiURL(workspaceID, emojiID, ext string) string {
	return fmt.Sprintf("/api/emojis/%s/%s%s", workspaceID, emojiID, ext)
}

func toOpenAPIEmoji(e *emoji.CustomEmoji) openapi.CustomEmoji {
	ext := ".png"
	if e.ContentType == "image/gif" {
		ext = ".gif"
	}
	return openapi.CustomEmoji{
		Id:          e.ID,
		WorkspaceId: e.WorkspaceID,
		Name:        e.Name,
		CreatedBy:   e.CreatedBy,
		ContentType: e.ContentType,
		SizeBytes:   e.SizeBytes,
		Url:         emojiURL(e.WorkspaceID, e.ID, ext),
		CreatedAt:   e.CreatedAt,
	}
}

// UploadCustomEmoji uploads a custom emoji image
func (h *Handler) UploadCustomEmoji(ctx context.Context, request openapi.UploadCustomEmojiRequestObject) (openapi.UploadCustomEmojiResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UploadCustomEmoji401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	workspaceID := request.Wid

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, workspaceID)
	if err != nil {
		return openapi.UploadCustomEmoji403JSONResponse{ForbiddenJSONResponse: notAMemberResponse("Not a member of this workspace")}, nil
	}

	// Parse multipart: read "name" field and "file" field
	var name string
	var fileData []byte
	var contentType string

	for {
		part, err := request.Body.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return openapi.UploadCustomEmoji400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid multipart data")}, nil
		}

		switch part.FormName() {
		case "name":
			data, err := io.ReadAll(io.LimitReader(part, 128))
			if err != nil {
				_ = part.Close()
				return openapi.UploadCustomEmoji400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Failed to read name field")}, nil
			}
			name = string(data)
		case "file":
			ct := part.Header.Get("Content-Type")
			if ct != "" {
				contentType = ct
			}
			data, err := io.ReadAll(io.LimitReader(part, maxEmojiSize+1))
			if err != nil {
				_ = part.Close()
				return openapi.UploadCustomEmoji400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Failed to read file")}, nil
			}
			fileData = data
		}
		_ = part.Close()
	}

	if name == "" {
		return openapi.UploadCustomEmoji400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Name is required")}, nil
	}
	if len(fileData) == 0 {
		return openapi.UploadCustomEmoji400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "File is required")}, nil
	}

	// Validate name
	name = strings.ToLower(name)
	if !emojiNameRegexp.MatchString(name) {
		return openapi.UploadCustomEmoji400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid emoji name: must be alphanumeric with hyphens/underscores, 1-63 characters")}, nil
	}

	// Validate content type
	ext, ok := allowedEmojiTypes[contentType]
	if !ok {
		return openapi.UploadCustomEmoji400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid file type: only PNG and GIF are allowed")}, nil
	}

	// Validate size
	if int64(len(fileData)) > maxEmojiSize {
		return openapi.UploadCustomEmoji400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "File too large: maximum size is 256KB")}, nil
	}

	// Create emoji record first to get ID
	e := &emoji.CustomEmoji{
		WorkspaceID: workspaceID,
		Name:        name,
		CreatedBy:   userID,
		ContentType: contentType,
		SizeBytes:   int64(len(fileData)),
	}

	// Build storage path
	storagePath := filepath.Join(h.storagePath, "emojis", workspaceID)
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, err
	}

	// We need the ID before writing to disk, so create DB record first
	if err := h.emojiRepo.Create(ctx, e); err != nil {
		if errors.Is(err, emoji.ErrEmojiNameTaken) {
			return openapi.UploadCustomEmoji400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeConflict, "Emoji name already taken")}, nil
		}
		return nil, err
	}

	// Write file to disk
	filePath := filepath.Join(storagePath, e.ID+ext)
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		// Clean up DB record on file write failure
		_ = h.emojiRepo.Delete(ctx, e.ID)
		return nil, err
	}

	// Update storage path in the struct (not stored in DB response but used internally)
	e.StoragePath = filePath

	apiEmoji := toOpenAPIEmoji(e)

	// Broadcast SSE event
	if h.hub != nil {
		h.hub.BroadcastToWorkspace(workspaceID, sse.Event{
			Type: sse.EventEmojiCreated,
			Data: apiEmoji,
		})
	}

	return openapi.UploadCustomEmoji200JSONResponse{
		Emoji: apiEmoji,
	}, nil
}

// ListCustomEmojis lists all custom emojis for a workspace
func (h *Handler) ListCustomEmojis(ctx context.Context, request openapi.ListCustomEmojisRequestObject) (openapi.ListCustomEmojisResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListCustomEmojis401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	workspaceID := request.Wid

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, workspaceID)
	if err != nil {
		return openapi.ListCustomEmojis403JSONResponse{ForbiddenJSONResponse: notAMemberResponse("Not a member of this workspace")}, nil
	}

	emojis, err := h.emojiRepo.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	apiEmojis := make([]openapi.CustomEmoji, len(emojis))
	for i := range emojis {
		apiEmojis[i] = toOpenAPIEmoji(&emojis[i])
	}

	return openapi.ListCustomEmojis200JSONResponse{
		Emojis: apiEmojis,
	}, nil
}

// DeleteCustomEmoji deletes a custom emoji
func (h *Handler) DeleteCustomEmoji(ctx context.Context, request openapi.DeleteCustomEmojiRequestObject) (openapi.DeleteCustomEmojiResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.DeleteCustomEmoji401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	e, err := h.emojiRepo.GetByID(ctx, request.Id)
	if err != nil {
		if errors.Is(err, emoji.ErrEmojiNotFound) {
			return openapi.DeleteCustomEmoji404JSONResponse{NotFoundJSONResponse: notFoundResponse("Emoji not found")}, nil
		}
		return nil, err
	}

	// Check permission: creator or admin
	canDelete := e.CreatedBy == userID

	if !canDelete {
		membership, err := h.workspaceRepo.GetMembership(ctx, userID, e.WorkspaceID)
		if err == nil && workspace.CanManageMembers(membership.Role) {
			canDelete = true
		}
	}

	if !canDelete {
		return openapi.DeleteCustomEmoji403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	// Delete file from disk
	ext := ".png"
	if e.ContentType == "image/gif" {
		ext = ".gif"
	}
	filePath := filepath.Join(h.storagePath, "emojis", e.WorkspaceID, e.ID+ext)
	_ = os.Remove(filePath)

	// Delete from database
	if err := h.emojiRepo.Delete(ctx, request.Id); err != nil {
		return nil, err
	}

	// Broadcast SSE event
	if h.hub != nil {
		h.hub.BroadcastToWorkspace(e.WorkspaceID, sse.Event{
			Type: sse.EventEmojiDeleted,
			Data: map[string]string{
				"id":   e.ID,
				"name": e.Name,
			},
		})
	}

	return openapi.DeleteCustomEmoji200JSONResponse{
		Success: true,
	}, nil
}

// ServeEmoji serves a custom emoji file
func (h *Handler) ServeEmoji(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspaceId")
	filename := chi.URLParam(r, "filename")
	if workspaceID == "" || filename == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Sanitize to prevent directory traversal
	workspaceID = filepath.Base(workspaceID)
	filename = filepath.Base(filename)
	emojiPath := filepath.Join(h.storagePath, "emojis", workspaceID, filename)

	// Check if file exists
	if _, err := os.Stat(emojiPath); os.IsNotExist(err) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, emojiPath)
}
