package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/enzyme/server/internal/gravatar"
	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/user"
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

	profile := openapi.UserProfile{
		Id:          u.ID,
		DisplayName: u.DisplayName,
		AvatarUrl:   u.AvatarURL,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt,
	}
	if g := gravatar.URL(u.Email); g != "" {
		profile.GravatarUrl = &g
	}
	return openapi.GetUser200JSONResponse{
		User: profile,
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
		User: userToAPI(u),
	}, nil
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

	if h.storage == nil {
		return openapi.UploadAvatar403JSONResponse{
			ForbiddenJSONResponse: openapi.ForbiddenJSONResponse(newErrorResponse(ErrCodeFilesDisabled, "File uploads are disabled")),
		}, nil
	}

	// Read with size limit
	data, err := io.ReadAll(io.LimitReader(part, maxAvatarSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxAvatarSize {
		return openapi.UploadAvatar400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse(ErrCodeValidationError, "File too large. Maximum size is 5MB")),
		}, nil
	}

	// Generate storage key
	fileID := ulid.Make().String()
	filename := fileID + ext
	storageKey := "avatars/" + filename

	if err := h.storage.Put(ctx, storageKey, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		return nil, err
	}

	// Delete old avatar file if it exists and is a local avatar
	if u.AvatarURL != nil && strings.HasPrefix(*u.AvatarURL, "/api/avatars/") {
		oldFilename := sanitizePathSegment(strings.TrimPrefix(*u.AvatarURL, "/api/avatars/"))
		_ = h.storage.Delete(ctx, "avatars/"+oldFilename)
	}

	// Update user's avatar URL
	avatarURL := "/api/avatars/" + filename
	u.AvatarURL = &avatarURL
	if err := h.userRepo.Update(ctx, u); err != nil {
		_ = h.storage.Delete(ctx, storageKey)
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
	if h.storage != nil && u.AvatarURL != nil && strings.HasPrefix(*u.AvatarURL, "/api/avatars/") {
		filename := sanitizePathSegment(strings.TrimPrefix(*u.AvatarURL, "/api/avatars/"))
		_ = h.storage.Delete(ctx, "avatars/"+filename)
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
	filename = sanitizePathSegment(filename)
	if h.storage == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	h.storage.Serve(w, r, "avatars/"+filename)
}
