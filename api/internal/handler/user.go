package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/user"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

// GetUser returns a user's public profile
func (h *Handler) GetUser(ctx context.Context, request openapi.GetUserRequestObject) (openapi.GetUserResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.GetUser401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	u, err := h.userRepo.GetByID(ctx, request.Id)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return openapi.GetUser404JSONResponse{
				NotFoundJSONResponse: openapi.NotFoundJSONResponse(newErrorResponse(ErrCodeNotFound, "User not found")),
			}, nil
		}
		return nil, err
	}

	return openapi.GetUser200JSONResponse{
		User: &openapi.UserProfile{
			Id:          u.ID,
			DisplayName: u.DisplayName,
			AvatarUrl:   u.AvatarURL,
			Status:      u.Status,
			CreatedAt:   u.CreatedAt,
		},
	}, nil
}

// UpdateProfile updates the current user's profile
func (h *Handler) UpdateProfile(ctx context.Context, request openapi.UpdateProfileRequestObject) (openapi.UpdateProfileResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UpdateProfile401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	u, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return openapi.UpdateProfile401JSONResponse{
				UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
			}, nil
		}
		return nil, err
	}

	// Update fields if provided
	if request.Body.DisplayName != nil {
		displayName := strings.TrimSpace(*request.Body.DisplayName)
		if displayName == "" {
			return openapi.UpdateProfile400JSONResponse{
				BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse(ErrCodeValidationError, "Display name cannot be empty")),
			}, nil
		}
		u.DisplayName = displayName
	}

	if err := h.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}

	return openapi.UpdateProfile200JSONResponse{
		User: ptr(userToAPI(u)),
	}, nil
}

// ptr returns a pointer to the given value
func ptr[T any](v T) *T {
	return &v
}

// Allowed avatar content types
var avatarAllowedTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

const maxAvatarSize = 5 * 1024 * 1024 // 5MB

// UploadAvatar uploads an avatar image for the current user
func (h *Handler) UploadAvatar(ctx context.Context, request openapi.UploadAvatarRequestObject) (openapi.UploadAvatarResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UploadAvatar401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	// Get current user
	u, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return openapi.UploadAvatar401JSONResponse{
				UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
			}, nil
		}
		return nil, err
	}

	// Parse multipart form
	part, err := request.Body.NextPart()
	if err != nil {
		return openapi.UploadAvatar400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse(ErrCodeValidationError, "No file provided")),
		}, nil
	}
	defer part.Close()

	// Validate content type
	contentType := part.Header.Get("Content-Type")
	ext, ok := avatarAllowedTypes[contentType]
	if !ok {
		return openapi.UploadAvatar400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse(ErrCodeValidationError, "Invalid file type. Allowed: JPEG, PNG, GIF, WebP")),
		}, nil
	}

	// Generate filename and storage path
	fileID := ulid.Make().String()
	filename := fileID + ext
	avatarDir := filepath.Join(h.storagePath, "avatars")
	storagePath := filepath.Join(avatarDir, filename)

	// Ensure directory exists
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		return nil, err
	}

	// Create file
	dst, err := os.Create(storagePath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	// Copy file content with size limit
	size, err := io.Copy(dst, io.LimitReader(part, maxAvatarSize+1))
	if err != nil {
		_ = os.Remove(storagePath)
		return nil, err
	}

	// Check if file exceeds max size
	if size > maxAvatarSize {
		_ = os.Remove(storagePath)
		return openapi.UploadAvatar400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse(ErrCodeValidationError, "File too large. Maximum size is 5MB")),
		}, nil
	}

	// Delete old avatar file if it exists and is a local avatar
	if u.AvatarURL != nil && strings.HasPrefix(*u.AvatarURL, "/api/avatars/") {
		oldFilename := strings.TrimPrefix(*u.AvatarURL, "/api/avatars/")
		oldPath := filepath.Join(avatarDir, oldFilename)
		_ = os.Remove(oldPath) // Ignore errors - file may not exist
	}

	// Update user's avatar URL
	avatarURL := "/api/avatars/" + filename
	u.AvatarURL = &avatarURL
	if err := h.userRepo.Update(ctx, u); err != nil {
		_ = os.Remove(storagePath)
		return nil, err
	}

	return openapi.UploadAvatar200JSONResponse{
		AvatarUrl: avatarURL,
	}, nil
}

// DeleteAvatar removes the current user's avatar
func (h *Handler) DeleteAvatar(ctx context.Context, request openapi.DeleteAvatarRequestObject) (openapi.DeleteAvatarResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.DeleteAvatar401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	// Get current user
	u, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return openapi.DeleteAvatar401JSONResponse{
				UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
			}, nil
		}
		return nil, err
	}

	// Delete avatar file if it's a local avatar
	if u.AvatarURL != nil && strings.HasPrefix(*u.AvatarURL, "/api/avatars/") {
		filename := strings.TrimPrefix(*u.AvatarURL, "/api/avatars/")
		avatarPath := filepath.Join(h.storagePath, "avatars", filename)
		_ = os.Remove(avatarPath) // Ignore errors - file may not exist
	}

	// Clear user's avatar URL
	u.AvatarURL = nil
	if err := h.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}

	return openapi.DeleteAvatar200JSONResponse{
		Success: true,
	}, nil
}

// ServeAvatar serves avatar files (called manually from router, not generated)
func (h *Handler) ServeAvatar(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Sanitize filename to prevent directory traversal
	filename = filepath.Base(filename)
	avatarPath := filepath.Join(h.storagePath, "avatars", filename)

	// Check if file exists
	if _, err := os.Stat(avatarPath); os.IsNotExist(err) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, avatarPath)
}
