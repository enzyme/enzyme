package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/enzyme/server/internal/channel"
	"github.com/enzyme/server/internal/gravatar"
	"github.com/enzyme/server/internal/message"
	"github.com/enzyme/server/internal/moderation"
	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/sse"
	"github.com/enzyme/server/internal/workspace"
	"github.com/go-chi/chi/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/oklog/ulid/v2"
)

// CreateWorkspace creates a new workspace
func (h *Handler) CreateWorkspace(ctx context.Context, request openapi.CreateWorkspaceRequestObject) (openapi.CreateWorkspaceResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.CreateWorkspace401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	if strings.TrimSpace(request.Body.Name) == "" {
		return openapi.CreateWorkspace400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Name is required")}, nil
	}

	ws := &workspace.Workspace{
		Name:     request.Body.Name,
		Settings: "{}",
	}

	if err := h.workspaceRepo.Create(ctx, ws, userID); err != nil {
		return nil, err
	}

	// Create the default #general channel
	defaultChannel, err := h.channelRepo.CreateDefaultChannel(ctx, ws.ID, userID)
	if err != nil {
		// Log but don't fail workspace creation
		// The channel can be created later if needed
	} else if h.hub != nil {
		// Update SSE hub cache with creator as first member
		h.hub.AddChannelMember(defaultChannel.ID, userID)
	}

	apiWs := workspaceToAPI(ws)
	return openapi.CreateWorkspace200JSONResponse{
		Workspace: apiWs,
	}, nil
}

// GetWorkspace gets workspace details
func (h *Handler) GetWorkspace(ctx context.Context, request openapi.GetWorkspaceRequestObject) (openapi.GetWorkspaceResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.GetWorkspace401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			return openapi.GetWorkspace404JSONResponse{NotFoundJSONResponse: notFoundResponse("Workspace not found")}, nil
		}
		return nil, err
	}

	ws, err := h.workspaceRepo.GetByID(ctx, string(request.Wid))
	if err != nil {
		return nil, err
	}

	apiWs := workspaceToAPI(ws)
	return openapi.GetWorkspace200JSONResponse{
		Workspace: apiWs,
	}, nil
}

// UpdateWorkspace updates a workspace
func (h *Handler) UpdateWorkspace(ctx context.Context, request openapi.UpdateWorkspaceRequestObject) (openapi.UpdateWorkspaceResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UpdateWorkspace401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	if !workspace.CanManageMembers(membership.Role) {
		return openapi.UpdateWorkspace403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	ws, err := h.workspaceRepo.GetByID(ctx, string(request.Wid))
	if err != nil {
		return nil, err
	}

	if request.Body.Name != nil {
		if strings.TrimSpace(*request.Body.Name) == "" {
			return openapi.UpdateWorkspace400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Name cannot be empty")}, nil
		}
		ws.Name = *request.Body.Name
	}

	// Handle settings update
	if request.Body.Settings != nil {
		// Start with existing settings
		settings := ws.ParsedSettings()

		// Update only provided fields
		if request.Body.Settings.ShowJoinLeaveMessages != nil {
			settings.ShowJoinLeaveMessages = *request.Body.Settings.ShowJoinLeaveMessages
		}
		if request.Body.Settings.WhoCanCreateChannels != nil {
			v := workspace.PermissionLevel(*request.Body.Settings.WhoCanCreateChannels)
			if !workspace.IsValidPermissionLevel(v) {
				return openapi.UpdateWorkspace400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid value for who_can_create_channels")}, nil
			}
			settings.WhoCanCreateChannels = v
		}
		if request.Body.Settings.WhoCanCreateInvites != nil {
			v := workspace.PermissionLevel(*request.Body.Settings.WhoCanCreateInvites)
			if !workspace.IsValidPermissionLevel(v) {
				return openapi.UpdateWorkspace400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid value for who_can_create_invites")}, nil
			}
			settings.WhoCanCreateInvites = v
		}
		if request.Body.Settings.WhoCanPinMessages != nil {
			v := workspace.PermissionLevel(*request.Body.Settings.WhoCanPinMessages)
			if !workspace.IsValidPermissionLevel(v) {
				return openapi.UpdateWorkspace400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid value for who_can_pin_messages")}, nil
			}
			settings.WhoCanPinMessages = v
		}
		if request.Body.Settings.WhoCanManageCustomEmoji != nil {
			v := workspace.PermissionLevel(*request.Body.Settings.WhoCanManageCustomEmoji)
			if !workspace.IsValidPermissionLevel(v) {
				return openapi.UpdateWorkspace400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid value for who_can_manage_custom_emoji")}, nil
			}
			settings.WhoCanManageCustomEmoji = v
		}

		// Serialize back to JSON string
		ws.Settings = settings.ToJSON()
	}

	if err := h.workspaceRepo.Update(ctx, ws); err != nil {
		return nil, err
	}

	apiWs := workspaceToAPI(ws)

	// Broadcast workspace update so all connected clients refresh permission-gated UI
	if h.hub != nil {
		h.hub.BroadcastToWorkspace(string(request.Wid), sse.NewWorkspaceUpdatedEvent(apiWs))
	}

	return openapi.UpdateWorkspace200JSONResponse{
		Workspace: apiWs,
	}, nil
}

// ListWorkspaceMembers lists members of a workspace
func (h *Handler) ListWorkspaceMembers(ctx context.Context, request openapi.ListWorkspaceMembersRequestObject) (openapi.ListWorkspaceMembersResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListWorkspaceMembers401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	members, err := h.workspaceRepo.ListMembers(ctx, string(request.Wid))
	if err != nil {
		return nil, err
	}

	apiMembers := make([]openapi.WorkspaceMemberWithUser, len(members))
	for i, m := range members {
		apiMembers[i] = memberWithUserToAPI(m)
	}

	return openapi.ListWorkspaceMembers200JSONResponse{
		Members: apiMembers,
	}, nil
}

// RemoveWorkspaceMember removes a member from a workspace
func (h *Handler) RemoveWorkspaceMember(ctx context.Context, request openapi.RemoveWorkspaceMemberRequestObject) (openapi.RemoveWorkspaceMemberResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.RemoveWorkspaceMember401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	// Users can remove themselves, admins/owners can remove others
	if request.Body.UserId != userID && !workspace.CanManageMembers(membership.Role) {
		return openapi.RemoveWorkspaceMember403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	// Role hierarchy: actor can only remove users with strictly lower RoleRank
	if request.Body.UserId != userID {
		targetMembership, err := h.workspaceRepo.GetMembership(ctx, request.Body.UserId, string(request.Wid))
		if err != nil {
			return openapi.RemoveWorkspaceMember404JSONResponse{NotFoundJSONResponse: notFoundResponse("User is not a member of this workspace")}, nil
		}
		if workspace.RoleRank(membership.Role) <= workspace.RoleRank(targetMembership.Role) {
			return openapi.RemoveWorkspaceMember403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Cannot remove a user with equal or higher role")}, nil
		}
	}

	if err := h.workspaceRepo.RemoveMember(ctx, request.Body.UserId, string(request.Wid)); err != nil {
		return nil, err
	}

	// Audit log: only when an admin removes another user (not self-removal)
	if request.Body.UserId != userID {
		_ = h.moderationRepo.CreateAuditLogEntryWithMetadata(ctx, string(request.Wid), userID, "member.removed", "user", request.Body.UserId, nil)
	}

	return openapi.RemoveWorkspaceMember200JSONResponse{
		Success: true,
	}, nil
}

// LeaveWorkspace allows a user to voluntarily leave a workspace
func (h *Handler) LeaveWorkspace(ctx context.Context, request openapi.LeaveWorkspaceRequestObject) (openapi.LeaveWorkspaceResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.LeaveWorkspace401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	workspaceID := string(request.Wid)

	// Check membership
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, workspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			return openapi.LeaveWorkspace404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a member of this workspace")}, nil
		}
		return nil, err
	}

	// Begin transaction: check owner count + remove memberships atomically
	tx, err := h.workspaceRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Last owner cannot leave
	if membership.Role == workspace.RoleOwner {
		ownerCount, err := h.workspaceRepo.CountOwnersTx(ctx, tx, workspaceID)
		if err != nil {
			return nil, err
		}
		if ownerCount <= 1 {
			return openapi.LeaveWorkspace403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Workspace owners must transfer ownership before leaving")}, nil
		}
	}

	removedChannelIDs, err := h.channelRepo.RemoveAllNonDMMemberships(ctx, tx, userID, workspaceID)
	if err != nil {
		return nil, err
	}

	if err := h.workspaceRepo.RemoveMemberTx(ctx, tx, userID, workspaceID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Update SSE hub and broadcast events
	if h.hub != nil {
		for _, channelID := range removedChannelIDs {
			// Broadcast before removing from hub so the leaving user's other
			// sessions (if any) receive the removal event.
			h.hub.BroadcastToChannel(workspaceID, channelID, sse.NewChannelMemberRemovedEvent(openapi.ChannelMemberData{
				ChannelId: channelID,
				UserId:    userID,
			}))
			h.hub.RemoveChannelMember(channelID, userID)
		}

		h.hub.BroadcastToWorkspace(workspaceID, sse.NewMemberLeftEvent(openapi.WorkspaceMemberData{
			UserId:      userID,
			WorkspaceId: workspaceID,
		}))

		h.hub.DisconnectUserClients(workspaceID, userID)
	}

	return openapi.LeaveWorkspace200JSONResponse{Success: true}, nil
}

// UpdateWorkspaceMemberRole updates a member's role
func (h *Handler) UpdateWorkspaceMemberRole(ctx context.Context, request openapi.UpdateWorkspaceMemberRoleRequestObject) (openapi.UpdateWorkspaceMemberRoleResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UpdateWorkspaceMemberRole401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	if !workspace.CanChangeRole(membership.Role) {
		return openapi.UpdateWorkspaceMemberRole403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	newRole := string(request.Body.Role)
	workspaceID := string(request.Wid)
	targetUserID := request.Body.UserId
	isSelf := userID == targetUserID

	// Validate role
	if newRole != workspace.RoleOwner && newRole != workspace.RoleAdmin && newRole != workspace.RoleMember && newRole != workspace.RoleGuest {
		return openapi.UpdateWorkspaceMemberRole400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid role")}, nil
	}

	// Only owners can promote to owner
	if newRole == workspace.RoleOwner && membership.Role != workspace.RoleOwner {
		return openapi.UpdateWorkspaceMemberRole403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Only owners can promote to owner")}, nil
	}

	// Admins can't promote to admin
	if membership.Role == workspace.RoleAdmin && newRole == workspace.RoleAdmin {
		return openapi.UpdateWorkspaceMemberRole403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Admins cannot promote to admin")}, nil
	}

	targetMembership, err := h.workspaceRepo.GetMembership(ctx, targetUserID, workspaceID)
	if err != nil {
		return nil, err
	}

	// Cannot change another owner's role
	if targetMembership.Role == workspace.RoleOwner && !isSelf {
		return openapi.UpdateWorkspaceMemberRole403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Cannot change another owner's role")}, nil
	}

	// Self-demotion from owner: check + update in a transaction to prevent TOCTOU race
	if isSelf && targetMembership.Role == workspace.RoleOwner && newRole != workspace.RoleOwner {
		tx, err := h.workspaceRepo.BeginTx(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()

		ownerCount, err := h.workspaceRepo.CountOwnersTx(ctx, tx, workspaceID)
		if err != nil {
			return nil, err
		}
		if ownerCount <= 1 {
			return openapi.UpdateWorkspaceMemberRole403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Cannot demote yourself: you are the only owner")}, nil
		}
		if err := h.workspaceRepo.UpdateMemberRoleTx(ctx, tx, targetUserID, workspaceID, newRole); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	} else {
		if err := h.workspaceRepo.UpdateMemberRole(ctx, targetUserID, workspaceID, newRole); err != nil {
			return nil, err
		}
	}

	// Audit log: role change
	_ = h.moderationRepo.CreateAuditLogEntryWithMetadata(ctx, workspaceID, userID, "member.role_changed", "user", targetUserID, map[string]interface{}{
		"old_role": targetMembership.Role,
		"new_role": newRole,
	})

	// SSE broadcast: role changed
	if h.hub != nil {
		h.hub.BroadcastToWorkspace(workspaceID, sse.NewMemberRoleChangedEvent(openapi.MemberRoleChangedData{
			UserId:  targetUserID,
			OldRole: targetMembership.Role,
			NewRole: newRole,
		}))
	}

	return openapi.UpdateWorkspaceMemberRole200JSONResponse{
		Success: true,
	}, nil
}

// CreateWorkspaceInvite creates an invite to a workspace
func (h *Handler) CreateWorkspaceInvite(ctx context.Context, request openapi.CreateWorkspaceInviteRequestObject) (openapi.CreateWorkspaceInviteResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.CreateWorkspaceInvite401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	ws, err := h.workspaceRepo.GetByID(ctx, string(request.Wid))
	if err != nil {
		return nil, err
	}
	settings := ws.ParsedSettings()

	if !workspace.HasPermission(membership.Role, settings.WhoCanCreateInvites) {
		return openapi.CreateWorkspaceInvite403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	// Validate role - default to member, can't invite as owner
	role := string(request.Body.Role)
	if role != workspace.RoleAdmin && role != workspace.RoleMember && role != workspace.RoleGuest {
		role = workspace.RoleMember
	}
	if role == workspace.RoleOwner {
		role = workspace.RoleMember
	}

	// Non-admins cannot create invites with admin role
	if role == workspace.RoleAdmin && !workspace.CanManageMembers(membership.Role) {
		return openapi.CreateWorkspaceInvite403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Only admins and owners can create admin invites")}, nil
	}

	invite := &workspace.Invite{
		WorkspaceID: string(request.Wid),
		Role:        role,
		CreatedBy:   &userID,
		MaxUses:     request.Body.MaxUses,
	}

	if request.Body.InvitedEmail != nil {
		email := string(*request.Body.InvitedEmail)
		invite.InvitedEmail = &email
	}

	if request.Body.ExpiresInHours != nil && *request.Body.ExpiresInHours > 0 {
		t := time.Now().Add(time.Duration(*request.Body.ExpiresInHours) * time.Hour)
		invite.ExpiresAt = &t
	}

	if err := h.workspaceRepo.CreateInvite(ctx, invite); err != nil {
		return nil, err
	}

	apiInvite := inviteToAPI(invite)
	return openapi.CreateWorkspaceInvite200JSONResponse{
		Invite: apiInvite,
	}, nil
}

// ReorderWorkspaces reorders workspaces for the current user
func (h *Handler) ReorderWorkspaces(ctx context.Context, request openapi.ReorderWorkspacesRequestObject) (openapi.ReorderWorkspacesResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ReorderWorkspaces401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	if request.Body == nil || len(request.Body.WorkspaceIds) == 0 {
		return openapi.ReorderWorkspaces400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse(ErrCodeValidationError, "workspace_ids is required")),
		}, nil
	}

	if err := h.workspaceRepo.ReorderWorkspaces(ctx, userID, request.Body.WorkspaceIds); err != nil {
		return nil, err
	}

	return openapi.ReorderWorkspaces200JSONResponse{
		Success: true,
	}, nil
}

// AcceptInvite accepts a workspace invite
func (h *Handler) AcceptInvite(ctx context.Context, request openapi.AcceptInviteRequestObject) (openapi.AcceptInviteResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.AcceptInvite401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Look up the invite to get the workspace ID for ban check
	invite, err := h.workspaceRepo.GetInviteByCode(ctx, request.Code)
	if err == nil && invite != nil {
		// Check for active ban before allowing join
		ban, _ := h.moderationRepo.GetActiveBan(ctx, invite.WorkspaceID, userID)
		if ban != nil {
			return openapi.AcceptInvite403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("You are banned from this workspace")}, nil
		}
	}

	ws, err := h.workspaceRepo.AcceptInvite(ctx, request.Code, userID)
	if err != nil {
		return nil, err
	}

	// Add user to the default #general channel
	defaultChannel, err := h.channelRepo.GetDefaultChannel(ctx, ws.ID)
	if err == nil {
		memberRole := channel.ChannelRolePoster
		_, addErr := h.channelRepo.AddMember(ctx, userID, defaultChannel.ID, &memberRole)
		if addErr == nil && h.hub != nil {
			h.hub.AddChannelMember(defaultChannel.ID, userID)
			h.hub.BroadcastToWorkspace(ws.ID, sse.NewChannelMemberAddedEvent(openapi.ChannelMemberData{
				ChannelId: defaultChannel.ID,
				UserId:    userID,
			}))
		}
	}

	// Auto-create DMs with up to 5 existing members
	h.autoCreateDMs(ctx, ws.ID, userID)

	apiWs := workspaceToAPI(ws)
	return openapi.AcceptInvite200JSONResponse{
		Workspace: apiWs,
	}, nil
}

// autoCreateDMs creates DM channels between the joining user and up to 5
// existing workspace members (earliest first). This is best-effort — errors
// are logged but do not fail the join.
func (h *Handler) autoCreateDMs(ctx context.Context, workspaceID, joiningUserID string) {
	members, err := h.workspaceRepo.ListMembers(ctx, workspaceID)
	if err != nil {
		return
	}

	const maxDMs = 5
	created := 0
	for _, m := range members {
		if created >= maxDMs {
			break
		}
		if m.UserID == joiningUserID {
			continue
		}

		dm, err := h.channelRepo.CreateDM(ctx, workspaceID, []string{joiningUserID, m.UserID})
		if err != nil {
			continue
		}
		if h.hub != nil {
			h.hub.AddChannelMember(dm.ID, joiningUserID)
			h.hub.AddChannelMember(dm.ID, m.UserID)
		}
		created++
	}

	// Single broadcast so all connected clients refetch their channel list
	if created > 0 && h.hub != nil {
		h.hub.BroadcastToWorkspace(workspaceID, sse.NewChannelsInvalidateEvent())
	}
}

// workspaceToAPI converts a workspace.Workspace to openapi.Workspace
func workspaceToAPI(ws *workspace.Workspace) openapi.Workspace {
	apiWs := openapi.Workspace{
		Id:        ws.ID,
		Name:      ws.Name,
		IconUrl:   ws.IconURL,
		Settings:  ws.Settings,
		CreatedAt: ws.CreatedAt,
		UpdatedAt: ws.UpdatedAt,
	}

	// Add parsed settings
	settings := ws.ParsedSettings()
	whoCanCreateChannels := openapi.PermissionLevel(settings.WhoCanCreateChannels)
	whoCanCreateInvites := openapi.PermissionLevel(settings.WhoCanCreateInvites)
	whoCanPinMessages := openapi.PermissionLevel(settings.WhoCanPinMessages)
	whoCanManageCustomEmoji := openapi.PermissionLevel(settings.WhoCanManageCustomEmoji)
	apiWs.ParsedSettings = &openapi.WorkspaceSettings{
		ShowJoinLeaveMessages:   &settings.ShowJoinLeaveMessages,
		WhoCanCreateChannels:    &whoCanCreateChannels,
		WhoCanCreateInvites:     &whoCanCreateInvites,
		WhoCanPinMessages:       &whoCanPinMessages,
		WhoCanManageCustomEmoji: &whoCanManageCustomEmoji,
	}

	return apiWs
}

// memberWithUserToAPI converts a workspace.MemberWithUser to openapi.WorkspaceMemberWithUser
func memberWithUserToAPI(m workspace.MemberWithUser) openapi.WorkspaceMemberWithUser {
	member := openapi.WorkspaceMemberWithUser{
		Id:                  m.ID,
		UserId:              m.UserID,
		WorkspaceId:         m.WorkspaceID,
		Role:                openapi.WorkspaceRole(m.Role),
		DisplayNameOverride: m.DisplayNameOverride,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
		Email:               openapi_types.Email(m.Email),
		DisplayName:         m.DisplayName,
		AvatarUrl:           m.AvatarURL,
		IsBanned:            &m.IsBanned,
	}
	if g := gravatar.URL(m.Email); g != "" {
		member.GravatarUrl = &g
	}
	return member
}

// inviteToAPI converts a workspace.Invite to openapi.Invite
func inviteToAPI(invite *workspace.Invite) openapi.Invite {
	apiInvite := openapi.Invite{
		Id:          invite.ID,
		WorkspaceId: invite.WorkspaceID,
		Code:        invite.Code,
		Role:        openapi.WorkspaceRole(invite.Role),
		UseCount:    invite.UseCount,
		CreatedAt:   invite.CreatedAt,
		CreatedBy:   invite.CreatedBy,
		MaxUses:     invite.MaxUses,
		ExpiresAt:   invite.ExpiresAt,
	}
	if invite.InvitedEmail != nil {
		email := openapi_types.Email(*invite.InvitedEmail)
		apiInvite.InvitedEmail = &email
	}
	return apiInvite
}

// Allowed icon content types
var iconAllowedTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

const maxIconSize = 5 * 1024 * 1024 // 5MB

// UploadWorkspaceIcon uploads an icon image for a workspace
func (h *Handler) UploadWorkspaceIcon(ctx context.Context, request openapi.UploadWorkspaceIconRequestObject) (openapi.UploadWorkspaceIconResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UploadWorkspaceIcon401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	workspaceID := string(request.Wid)

	// Check permissions - must be owner or admin
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, workspaceID)
	if err != nil {
		return openapi.UploadWorkspaceIcon401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not a workspace member")),
		}, nil
	}

	if !workspace.CanManageMembers(membership.Role) {
		return openapi.UploadWorkspaceIcon403JSONResponse{}, nil
	}

	// Get workspace
	ws, err := h.workspaceRepo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	// Parse multipart form
	part, err := request.Body.NextPart()
	if err != nil {
		return openapi.UploadWorkspaceIcon400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse(ErrCodeValidationError, "No file provided")),
		}, nil
	}
	defer part.Close()

	// Validate content type
	contentType := part.Header.Get("Content-Type")
	ext, ok := iconAllowedTypes[contentType]
	if !ok {
		return openapi.UploadWorkspaceIcon400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse(ErrCodeValidationError, "Invalid file type. Allowed: JPEG, PNG, GIF, WebP")),
		}, nil
	}

	if h.storage == nil {
		return openapi.UploadWorkspaceIcon403JSONResponse{
			ForbiddenJSONResponse: openapi.ForbiddenJSONResponse(newErrorResponse(ErrCodeFilesDisabled, "File uploads are disabled")),
		}, nil
	}

	// Read with size limit
	data, err := io.ReadAll(io.LimitReader(part, maxIconSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxIconSize {
		return openapi.UploadWorkspaceIcon400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse(ErrCodeValidationError, "File too large. Maximum size is 5MB")),
		}, nil
	}

	// Generate storage key
	fileID := ulid.Make().String()
	filename := fileID + ext
	storageKey := "workspace-icons/" + workspaceID + "/" + filename

	if err := h.storage.Put(ctx, storageKey, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		return nil, err
	}

	// Delete old icon file if it exists and is a local icon
	if ws.IconURL != nil && strings.HasPrefix(*ws.IconURL, "/api/workspace-icons/") {
		oldPath := strings.TrimPrefix(*ws.IconURL, "/api/workspace-icons/")
		if parts := strings.SplitN(oldPath, "/", 2); len(parts) == 2 {
			_ = h.storage.Delete(ctx, "workspace-icons/"+sanitizePathSegment(parts[0])+"/"+sanitizePathSegment(parts[1]))
		}
	}

	// Update workspace's icon URL
	iconURL := "/api/workspace-icons/" + workspaceID + "/" + filename
	ws.IconURL = &iconURL
	if err := h.workspaceRepo.Update(ctx, ws); err != nil {
		_ = h.storage.Delete(ctx, storageKey)
		return nil, err
	}

	return openapi.UploadWorkspaceIcon200JSONResponse{
		IconUrl: iconURL,
	}, nil
}

// DeleteWorkspaceIcon removes a workspace's icon
func (h *Handler) DeleteWorkspaceIcon(ctx context.Context, request openapi.DeleteWorkspaceIconRequestObject) (openapi.DeleteWorkspaceIconResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.DeleteWorkspaceIcon401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	workspaceID := string(request.Wid)

	// Check permissions - must be owner or admin
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, workspaceID)
	if err != nil {
		return openapi.DeleteWorkspaceIcon401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not a workspace member")),
		}, nil
	}

	if !workspace.CanManageMembers(membership.Role) {
		return openapi.DeleteWorkspaceIcon403JSONResponse{}, nil
	}

	// Get workspace
	ws, err := h.workspaceRepo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	// Delete icon file if it's a local icon
	if h.storage != nil && ws.IconURL != nil && strings.HasPrefix(*ws.IconURL, "/api/workspace-icons/") {
		oldPath := strings.TrimPrefix(*ws.IconURL, "/api/workspace-icons/")
		if parts := strings.SplitN(oldPath, "/", 2); len(parts) == 2 {
			_ = h.storage.Delete(ctx, "workspace-icons/"+sanitizePathSegment(parts[0])+"/"+sanitizePathSegment(parts[1]))
		}
	}

	// Clear workspace's icon URL
	ws.IconURL = nil
	if err := h.workspaceRepo.Update(ctx, ws); err != nil {
		return nil, err
	}

	return openapi.DeleteWorkspaceIcon200JSONResponse{
		Success: true,
	}, nil
}

// ServeWorkspaceIcon serves workspace icon files (called manually from router, not generated)
func (h *Handler) ServeWorkspaceIcon(w http.ResponseWriter, r *http.Request) {
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
	h.storage.Serve(w, r, "workspace-icons/"+workspaceID+"/"+filename)
}

// GetWorkspaceNotifications returns aggregated notification summaries for all workspaces
func (h *Handler) GetWorkspaceNotifications(ctx context.Context, request openapi.GetWorkspaceNotificationsRequestObject) (openapi.GetWorkspaceNotificationsResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.GetWorkspaceNotifications401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	summaries, err := h.channelRepo.GetWorkspaceNotificationSummaries(ctx, userID)
	if err != nil {
		return nil, err
	}

	apiSummaries := make([]openapi.WorkspaceNotificationSummary, len(summaries))
	for i, s := range summaries {
		apiSummaries[i] = openapi.WorkspaceNotificationSummary{
			WorkspaceId:       s.WorkspaceID,
			UnreadCount:       s.UnreadCount,
			NotificationCount: s.NotificationCount,
		}
	}

	return openapi.GetWorkspaceNotifications200JSONResponse{
		Workspaces: apiSummaries,
	}, nil
}

// ListAllUnreads lists all unread messages across channels in a workspace
func (h *Handler) ListAllUnreads(ctx context.Context, request openapi.ListAllUnreadsRequestObject) (openapi.ListAllUnreadsResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListAllUnreads401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	opts := message.ListOptions{
		Limit: 50,
	}
	if request.Body != nil {
		if request.Body.Limit != nil {
			opts.Limit = *request.Body.Limit
		}
		if request.Body.Cursor != nil {
			opts.Cursor = *request.Body.Cursor
		}
	}

	filter := &moderation.FilterOptions{WorkspaceID: string(request.Wid), RequestingUserID: userID}
	result, err := h.messageRepo.ListAllUnreads(ctx, string(request.Wid), userID, opts, filter)
	if err != nil {
		return nil, err
	}

	return openapi.ListAllUnreads200JSONResponse(unreadListResultToAPI(result)), nil
}

// unreadMessageToAPI converts a message.UnreadMessage to openapi.UnreadMessage
func unreadMessageToAPI(m *message.UnreadMessage) openapi.UnreadMessage {
	apiMsg := openapi.UnreadMessage{
		Id:             m.ID,
		ChannelId:      m.ChannelID,
		UserId:         m.UserID,
		Content:        m.Content,
		ThreadParentId: m.ThreadParentID,
		ReplyCount:     m.ReplyCount,
		LastReplyAt:    m.LastReplyAt,
		EditedAt:       m.EditedAt,
		DeletedAt:      m.DeletedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		ChannelName:    m.ChannelName,
		ChannelType:    openapi.ChannelType(m.ChannelType),
	}
	if m.UserDisplayName != "" {
		apiMsg.UserDisplayName = &m.UserDisplayName
	}
	if m.UserAvatarURL != nil {
		apiMsg.UserAvatarUrl = m.UserAvatarURL
	}
	if g := gravatar.URL(m.UserEmail); g != "" {
		apiMsg.UserGravatarUrl = &g
	}
	if len(m.Reactions) > 0 {
		reactions := make([]openapi.Reaction, len(m.Reactions))
		for i, r := range m.Reactions {
			reactions[i] = openapi.Reaction{
				Id:        r.ID,
				MessageId: r.MessageID,
				UserId:    r.UserID,
				Emoji:     r.Emoji,
				CreatedAt: r.CreatedAt,
			}
		}
		apiMsg.Reactions = &reactions
	}
	return apiMsg
}

// unreadListResultToAPI converts a message.UnreadListResult to openapi.UnreadMessagesResult
func unreadListResultToAPI(result *message.UnreadListResult) openapi.UnreadMessagesResult {
	messages := make([]openapi.UnreadMessage, len(result.Messages))
	for i, m := range result.Messages {
		messages[i] = unreadMessageToAPI(&m)
	}

	apiResult := openapi.UnreadMessagesResult{
		Messages: messages,
		HasMore:  result.HasMore,
	}
	if result.NextCursor != "" {
		apiResult.NextCursor = &result.NextCursor
	}
	return apiResult
}
