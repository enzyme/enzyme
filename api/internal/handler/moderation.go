package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/enzyme/api/internal/moderation"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/sse"
	"github.com/enzyme/api/internal/workspace"
)

// BanUser bans a user from a workspace
func (h *Handler) BanUser(ctx context.Context, request openapi.BanUserRequestObject) (openapi.BanUserResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.BanUser401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check actor is admin+
	actorMembership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return openapi.BanUser403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Not a workspace member")}, nil
	}
	if !workspace.CanManageMembers(actorMembership.Role) {
		return openapi.BanUser403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Only admins can ban users")}, nil
	}

	targetUserID := request.Body.UserId

	// Prevent self-ban
	if targetUserID == userID {
		return openapi.BanUser400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot ban yourself")}, nil
	}

	// Check target is a workspace member and enforce role hierarchy
	targetMembership, err := h.workspaceRepo.GetMembership(ctx, targetUserID, string(request.Wid))
	if err != nil {
		return openapi.BanUser404JSONResponse{NotFoundJSONResponse: notFoundResponse("User is not a member of this workspace")}, nil
	}

	// Role hierarchy: actor can only ban users with strictly lower RoleRank
	if workspace.RoleRank(actorMembership.Role) <= workspace.RoleRank(targetMembership.Role) {
		return openapi.BanUser403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Cannot ban a user with equal or higher role")}, nil
	}

	// Build ban record
	ban := &moderation.Ban{
		WorkspaceID: string(request.Wid),
		UserID:      targetUserID,
		BannedBy:    userID,
		Reason:      request.Body.Reason,
	}
	if request.Body.HideMessages != nil && *request.Body.HideMessages {
		ban.HideMessages = true
	}
	if request.Body.DurationHours != nil && *request.Body.DurationHours > 0 {
		expiresAt := time.Now().UTC().Add(time.Duration(*request.Body.DurationHours) * time.Hour)
		ban.ExpiresAt = &expiresAt
	}

	// Use a transaction for ban + membership removal
	tx, err := h.workspaceRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create ban record
	if err := h.moderationRepo.CreateBan(ctx, tx, ban); err != nil {
		if errors.Is(err, moderation.ErrAlreadyBanned) {
			return openapi.BanUser409JSONResponse(newErrorResponse(ErrCodeConflict, "User is already banned")), nil
		}
		return nil, err
	}

	// Remove workspace membership
	if err := h.workspaceRepo.RemoveMemberTx(ctx, tx, targetUserID, string(request.Wid)); err != nil {
		slog.Error("failed to remove workspace membership during ban", "error", err)
		return nil, err
	}

	// Remove channel memberships (except DM channels)
	channelIDs, err := h.channelRepo.RemoveAllNonDMMemberships(ctx, tx, targetUserID, string(request.Wid))
	if err != nil {
		slog.Error("failed to remove channel memberships during ban", "error", err)
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Update SSE hub: remove from all channel member caches
	if h.hub != nil {
		for _, chID := range channelIDs {
			h.hub.RemoveChannelMember(chID, targetUserID)
		}
	}

	// Disconnect user's SSE clients
	if h.hub != nil {
		h.hub.DisconnectUserClients(string(request.Wid), targetUserID)
	}

	// Create audit log entry (non-critical)
	metadata := map[string]interface{}{}
	if ban.Reason != nil {
		metadata["reason"] = *ban.Reason
	}
	if request.Body.DurationHours != nil {
		metadata["duration_hours"] = *request.Body.DurationHours
	}
	metadata["hide_messages"] = ban.HideMessages
	if err := h.moderationRepo.CreateAuditLogEntryWithMetadata(ctx, string(request.Wid), userID, moderation.ActionUserBanned, moderation.TargetTypeUser, targetUserID, metadata); err != nil {
		slog.Error("failed to create audit log entry for ban", "error", err)
	}

	// Broadcast SSE event
	if h.hub != nil {
		h.hub.BroadcastToWorkspace(string(request.Wid), sse.Event{
			Type: sse.EventMemberBanned,
			Data: map[string]interface{}{
				"user_id":      targetUserID,
				"workspace_id": string(request.Wid),
				"banned_by":    userID,
			},
		})
	}

	// Convert to API response
	apiBan := openapi.Ban{
		Id:           ban.ID,
		WorkspaceId:  ban.WorkspaceID,
		UserId:       ban.UserID,
		BannedBy:     ban.BannedBy,
		Reason:       ban.Reason,
		HideMessages: ban.HideMessages,
		ExpiresAt:    ban.ExpiresAt,
		CreatedAt:    ban.CreatedAt,
	}

	return openapi.BanUser200JSONResponse{Ban: &apiBan}, nil
}

// UnbanUser removes a ban from a user
func (h *Handler) UnbanUser(ctx context.Context, request openapi.UnbanUserRequestObject) (openapi.UnbanUserResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UnbanUser401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check actor is admin+
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return openapi.UnbanUser403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Not a workspace member")}, nil
	}
	if !workspace.CanManageMembers(membership.Role) {
		return openapi.UnbanUser403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Only admins can unban users")}, nil
	}

	if err := h.moderationRepo.DeleteBan(ctx, string(request.Wid), request.Body.UserId); err != nil {
		if errors.Is(err, moderation.ErrBanNotFound) {
			return openapi.UnbanUser404JSONResponse{NotFoundJSONResponse: notFoundResponse("Ban not found")}, nil
		}
		return nil, err
	}

	// Create audit log entry
	if err := h.moderationRepo.CreateAuditLogEntryWithMetadata(ctx, string(request.Wid), userID, moderation.ActionUserUnbanned, moderation.TargetTypeUser, request.Body.UserId, nil); err != nil {
		slog.Error("failed to create audit log entry for unban", "error", err)
	}

	// Broadcast SSE event
	if h.hub != nil {
		h.hub.BroadcastToWorkspace(string(request.Wid), sse.Event{
			Type: sse.EventMemberUnbanned,
			Data: map[string]interface{}{
				"user_id":      request.Body.UserId,
				"workspace_id": string(request.Wid),
			},
		})
	}

	return openapi.UnbanUser200JSONResponse{Success: true}, nil
}

// ListBans lists active bans in a workspace
func (h *Handler) ListBans(ctx context.Context, request openapi.ListBansRequestObject) (openapi.ListBansResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListBans401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check actor is admin+
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return openapi.ListBans403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Not a workspace member")}, nil
	}
	if !workspace.CanManageMembers(membership.Role) {
		return openapi.ListBans403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Only admins can view bans")}, nil
	}

	cursor := ""
	limit := 50
	if request.Body != nil {
		if request.Body.Cursor != nil {
			cursor = *request.Body.Cursor
		}
		if request.Body.Limit != nil {
			limit = *request.Body.Limit
		}
	}

	bans, hasMore, nextCursor, err := h.moderationRepo.ListActiveBans(ctx, string(request.Wid), cursor, limit)
	if err != nil {
		return nil, err
	}

	apiBans := make([]openapi.BanWithUser, len(bans))
	for i, b := range bans {
		apiBans[i] = openapi.BanWithUser{
			Id:              b.ID,
			WorkspaceId:     b.WorkspaceID,
			UserId:          b.UserID,
			BannedBy:        b.BannedBy,
			Reason:          b.Reason,
			HideMessages:    b.HideMessages,
			ExpiresAt:       b.ExpiresAt,
			CreatedAt:       b.CreatedAt,
			UserDisplayName: &b.UserDisplayName,
			UserEmail:       &b.UserEmail,
			UserAvatarUrl:   b.UserAvatarURL,
			BannedByName:    &b.BannedByName,
		}
	}

	return openapi.ListBans200JSONResponse{
		Bans:       &apiBans,
		HasMore:    &hasMore,
		NextCursor: &nextCursor,
	}, nil
}

// BlockUser blocks another user
func (h *Handler) BlockUser(ctx context.Context, request openapi.BlockUserRequestObject) (openapi.BlockUserResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.BlockUser401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	targetUserID := request.Body.UserId

	// Prevent self-block
	if targetUserID == userID {
		return openapi.BlockUser400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot block yourself")}, nil
	}

	// Verify target exists
	_, err := h.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return openapi.BlockUser404JSONResponse{NotFoundJSONResponse: notFoundResponse("User not found")}, nil
	}

	if err := h.moderationRepo.CreateBlock(ctx, userID, targetUserID); err != nil {
		return nil, err
	}

	return openapi.BlockUser200JSONResponse{Success: true}, nil
}

// UnblockUser unblocks a user
func (h *Handler) UnblockUser(ctx context.Context, request openapi.UnblockUserRequestObject) (openapi.UnblockUserResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UnblockUser401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	if err := h.moderationRepo.DeleteBlock(ctx, userID, request.Body.UserId); err != nil {
		return nil, err
	}

	return openapi.UnblockUser200JSONResponse{Success: true}, nil
}

// ListBlocks lists users blocked by the current user
func (h *Handler) ListBlocks(ctx context.Context, request openapi.ListBlocksRequestObject) (openapi.ListBlocksResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListBlocks401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	blocks, err := h.moderationRepo.ListBlocks(ctx, userID)
	if err != nil {
		return nil, err
	}

	apiBlocks := make([]openapi.BlockWithUser, len(blocks))
	for i, b := range blocks {
		apiBlocks[i] = openapi.BlockWithUser{
			BlockerId:   b.BlockerID,
			BlockedId:   b.BlockedID,
			CreatedAt:   b.CreatedAt,
			DisplayName: &b.DisplayName,
			Email:       &b.Email,
			AvatarUrl:   b.AvatarURL,
		}
	}

	return openapi.ListBlocks200JSONResponse{
		Blocks: &apiBlocks,
	}, nil
}

// ListModerationLog lists moderation audit log entries for a workspace
func (h *Handler) ListModerationLog(ctx context.Context, request openapi.ListModerationLogRequestObject) (openapi.ListModerationLogResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListModerationLog401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check actor is admin+
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return openapi.ListModerationLog403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Not a workspace member")}, nil
	}
	if !workspace.CanManageMembers(membership.Role) {
		return openapi.ListModerationLog403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Only admins can view the moderation log")}, nil
	}

	cursor := ""
	limit := 50
	if request.Body != nil {
		if request.Body.Cursor != nil {
			cursor = *request.Body.Cursor
		}
		if request.Body.Limit != nil {
			limit = *request.Body.Limit
		}
	}

	entries, hasMore, nextCursor, err := h.moderationRepo.ListAuditLog(ctx, string(request.Wid), cursor, limit)
	if err != nil {
		return nil, err
	}

	apiEntries := make([]openapi.ModerationLogEntryWithActor, len(entries))
	for i, e := range entries {
		apiEntries[i] = openapi.ModerationLogEntryWithActor{
			Id:               e.ID,
			WorkspaceId:      e.WorkspaceID,
			ActorId:          e.ActorID,
			Action:           e.Action,
			TargetType:       e.TargetType,
			TargetId:         e.TargetID,
			Metadata:         e.Metadata,
			CreatedAt:        e.CreatedAt,
			ActorDisplayName: &e.ActorDisplayName,
			ActorAvatarUrl:   e.ActorAvatarURL,
		}
	}

	return openapi.ListModerationLog200JSONResponse{
		Entries:    &apiEntries,
		HasMore:    &hasMore,
		NextCursor: &nextCursor,
	}, nil
}
