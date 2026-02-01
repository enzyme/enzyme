package handler

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/feather/api/internal/message"
	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/workspace"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

func validateSlug(slug string) error {
	if !slugRegex.MatchString(slug) {
		return errors.New("slug must be 3-50 characters, lowercase letters, numbers, and hyphens only")
	}
	return nil
}

// CreateWorkspace creates a new workspace
func (h *Handler) CreateWorkspace(ctx context.Context, request openapi.CreateWorkspaceRequestObject) (openapi.CreateWorkspaceResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	if err := validateSlug(request.Body.Slug); err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.Body.Name) == "" {
		return nil, errors.New("name is required")
	}

	ws := &workspace.Workspace{
		Slug:     request.Body.Slug,
		Name:     request.Body.Name,
		Settings: "{}",
	}

	if err := h.workspaceRepo.Create(ctx, ws, userID); err != nil {
		if errors.Is(err, workspace.ErrSlugAlreadyInUse) {
			return nil, err
		}
		return nil, err
	}

	apiWs := workspaceToAPI(ws)
	return openapi.CreateWorkspace200JSONResponse{
		Workspace: &apiWs,
	}, nil
}

// GetWorkspace gets workspace details
func (h *Handler) GetWorkspace(ctx context.Context, request openapi.GetWorkspaceRequestObject) (openapi.GetWorkspaceResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	// Check membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	ws, err := h.workspaceRepo.GetByID(ctx, string(request.Wid))
	if err != nil {
		return nil, err
	}

	apiWs := workspaceToAPI(ws)
	return openapi.GetWorkspace200JSONResponse{
		Workspace: &apiWs,
	}, nil
}

// UpdateWorkspace updates a workspace
func (h *Handler) UpdateWorkspace(ctx context.Context, request openapi.UpdateWorkspaceRequestObject) (openapi.UpdateWorkspaceResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	// Check permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	if !workspace.CanManageMembers(membership.Role) {
		return nil, errors.New("permission denied")
	}

	ws, err := h.workspaceRepo.GetByID(ctx, string(request.Wid))
	if err != nil {
		return nil, err
	}

	if request.Body.Slug != nil {
		if err := validateSlug(*request.Body.Slug); err != nil {
			return nil, err
		}
		ws.Slug = *request.Body.Slug
	}
	if request.Body.Name != nil {
		if strings.TrimSpace(*request.Body.Name) == "" {
			return nil, errors.New("name cannot be empty")
		}
		ws.Name = *request.Body.Name
	}

	if err := h.workspaceRepo.Update(ctx, ws); err != nil {
		return nil, err
	}

	apiWs := workspaceToAPI(ws)
	return openapi.UpdateWorkspace200JSONResponse{
		Workspace: &apiWs,
	}, nil
}

// ListWorkspaceMembers lists members of a workspace
func (h *Handler) ListWorkspaceMembers(ctx context.Context, request openapi.ListWorkspaceMembersRequestObject) (openapi.ListWorkspaceMembersResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
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
		Members: &apiMembers,
	}, nil
}

// RemoveWorkspaceMember removes a member from a workspace
func (h *Handler) RemoveWorkspaceMember(ctx context.Context, request openapi.RemoveWorkspaceMemberRequestObject) (openapi.RemoveWorkspaceMemberResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	// Check permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	// Users can remove themselves, admins/owners can remove others
	if request.Body.UserId != userID && !workspace.CanManageMembers(membership.Role) {
		return nil, errors.New("permission denied")
	}

	if err := h.workspaceRepo.RemoveMember(ctx, request.Body.UserId, string(request.Wid)); err != nil {
		return nil, err
	}

	return openapi.RemoveWorkspaceMember200JSONResponse{
		Success: true,
	}, nil
}

// UpdateWorkspaceMemberRole updates a member's role
func (h *Handler) UpdateWorkspaceMemberRole(ctx context.Context, request openapi.UpdateWorkspaceMemberRoleRequestObject) (openapi.UpdateWorkspaceMemberRoleResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	// Check permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	if !workspace.CanChangeRole(membership.Role) {
		return nil, errors.New("permission denied")
	}

	// Validate role
	newRole := string(request.Body.Role)
	if newRole != workspace.RoleAdmin && newRole != workspace.RoleMember && newRole != workspace.RoleGuest {
		return nil, errors.New("invalid role")
	}

	// Can't change owner role
	targetMembership, err := h.workspaceRepo.GetMembership(ctx, request.Body.UserId, string(request.Wid))
	if err != nil {
		return nil, err
	}

	if targetMembership.Role == workspace.RoleOwner {
		return nil, errors.New("cannot change owner's role")
	}

	// Admins can't promote to admin
	if membership.Role == workspace.RoleAdmin && newRole == workspace.RoleAdmin {
		return nil, errors.New("admins cannot promote to admin")
	}

	if err := h.workspaceRepo.UpdateMemberRole(ctx, request.Body.UserId, string(request.Wid), newRole); err != nil {
		return nil, err
	}

	return openapi.UpdateWorkspaceMemberRole200JSONResponse{
		Success: true,
	}, nil
}

// CreateWorkspaceInvite creates an invite to a workspace
func (h *Handler) CreateWorkspaceInvite(ctx context.Context, request openapi.CreateWorkspaceInviteRequestObject) (openapi.CreateWorkspaceInviteResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	// Check permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	if !workspace.CanManageMembers(membership.Role) {
		return nil, errors.New("permission denied")
	}

	// Validate role - default to member, can't invite as owner
	role := string(request.Body.Role)
	if role != workspace.RoleAdmin && role != workspace.RoleMember && role != workspace.RoleGuest {
		role = workspace.RoleMember
	}
	if role == workspace.RoleOwner {
		role = workspace.RoleMember
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
		Invite: &apiInvite,
	}, nil
}

// AcceptInvite accepts a workspace invite
func (h *Handler) AcceptInvite(ctx context.Context, request openapi.AcceptInviteRequestObject) (openapi.AcceptInviteResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	ws, err := h.workspaceRepo.AcceptInvite(ctx, request.Code, userID)
	if err != nil {
		return nil, err
	}

	apiWs := workspaceToAPI(ws)
	return openapi.AcceptInvite200JSONResponse{
		Workspace: &apiWs,
	}, nil
}

// workspaceToAPI converts a workspace.Workspace to openapi.Workspace
func workspaceToAPI(ws *workspace.Workspace) openapi.Workspace {
	return openapi.Workspace{
		Id:        ws.ID,
		Slug:      ws.Slug,
		Name:      ws.Name,
		IconUrl:   ws.IconURL,
		Settings:  ws.Settings,
		CreatedAt: ws.CreatedAt,
		UpdatedAt: ws.UpdatedAt,
	}
}

// memberWithUserToAPI converts a workspace.MemberWithUser to openapi.WorkspaceMemberWithUser
func memberWithUserToAPI(m workspace.MemberWithUser) openapi.WorkspaceMemberWithUser {
	return openapi.WorkspaceMemberWithUser{
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
	}
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

// ListAllUnreads lists all unread messages across channels in a workspace
func (h *Handler) ListAllUnreads(ctx context.Context, request openapi.ListAllUnreadsRequestObject) (openapi.ListAllUnreadsResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
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

	result, err := h.messageRepo.ListAllUnreads(ctx, string(request.Wid), userID, opts)
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
