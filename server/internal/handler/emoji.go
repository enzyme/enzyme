package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/enzyme/server/internal/emoji"
	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/sse"
	"github.com/enzyme/server/internal/workspace"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

var (
	emojiNameRegexp   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)
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

	if h.storage == nil {
		return openapi.UploadCustomEmoji403JSONResponse{ForbiddenJSONResponse: filesDisabledResponse()}, nil
	}

	workspaceID := request.Wid

	// Check workspace membership
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, workspaceID)
	if err != nil {
		return openapi.UploadCustomEmoji403JSONResponse{ForbiddenJSONResponse: notAMemberResponse("Not a member of this workspace")}, nil
	}

	// Check workspace-level emoji permission
	ws, err := h.workspaceRepo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	wsSettings := ws.ParsedSettings()
	if !workspace.HasPermission(membership.Role, wsSettings.WhoCanManageCustomEmoji) {
		return openapi.UploadCustomEmoji403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
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

	// Pre-generate ID and storage key so StoragePath is persisted with the DB record
	emojiID := ulid.Make().String()
	storageKey := "emojis/" + workspaceID + "/" + emojiID + ext

	e := &emoji.CustomEmoji{
		ID:          emojiID,
		WorkspaceID: workspaceID,
		Name:        name,
		CreatedBy:   userID,
		ContentType: contentType,
		SizeBytes:   int64(len(fileData)),
		StoragePath: storageKey,
	}

	if err := h.emojiRepo.Create(ctx, e); err != nil {
		if errors.Is(err, emoji.ErrEmojiNameTaken) {
			return openapi.UploadCustomEmoji400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeConflict, "Emoji name already taken")}, nil
		}
		return nil, err
	}

	// Write file to storage
	if err := h.storage.Put(ctx, storageKey, bytes.NewReader(fileData), int64(len(fileData)), contentType); err != nil {
		_ = h.emojiRepo.Delete(ctx, e.ID)
		return nil, err
	}

	apiEmoji := toOpenAPIEmoji(e)

	// Broadcast SSE event
	if h.hub != nil {
		h.hub.BroadcastToWorkspace(workspaceID, sse.NewEmojiCreatedEvent(apiEmoji))
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

	// Check permission: creator can always delete their own, otherwise admin/owner only
	canDelete := e.CreatedBy == userID

	if !canDelete {
		membership, err := h.workspaceRepo.GetMembership(ctx, userID, e.WorkspaceID)
		if err == nil {
			canDelete = workspace.CanManageMembers(membership.Role)
		}
	}

	if !canDelete {
		return openapi.DeleteCustomEmoji403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	// Delete file from storage
	if h.storage != nil {
		if e.StoragePath != "" {
			_ = h.storage.Delete(ctx, e.StoragePath)
		} else {
			// Fallback for emojis created before StoragePath was persisted
			ext := ".png"
			if e.ContentType == "image/gif" {
				ext = ".gif"
			}
			_ = h.storage.Delete(ctx, "emojis/"+e.WorkspaceID+"/"+e.ID+ext)
		}
	}

	// Delete from database
	if err := h.emojiRepo.Delete(ctx, request.Id); err != nil {
		return nil, err
	}

	// Broadcast SSE event
	if h.hub != nil {
		h.hub.BroadcastToWorkspace(e.WorkspaceID, sse.NewEmojiDeletedEvent(openapi.EmojiDeletedData{
			Id:   e.ID,
			Name: e.Name,
		}))
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
	workspaceID = sanitizePathSegment(workspaceID)
	filename = sanitizePathSegment(filename)

	if h.storage == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	h.storage.Serve(w, r, "emojis/"+workspaceID+"/"+filename)
}
