package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/feather/api/internal/api"
	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/workspace"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// CreateChannel creates a new channel
func (h *Handler) CreateChannel(ctx context.Context, request api.CreateChannelRequestObject) (api.CreateChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	// Check workspace membership and permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	if !workspace.CanCreateChannels(membership.Role) {
		return nil, errors.New("permission denied")
	}

	if strings.TrimSpace(request.Body.Name) == "" {
		return nil, errors.New("channel name is required")
	}

	// Validate type
	channelType := string(request.Body.Type)
	if channelType != channel.TypePublic && channelType != channel.TypePrivate {
		channelType = channel.TypePublic
	}

	ch := &channel.Channel{
		WorkspaceID: string(request.Wid),
		Name:        request.Body.Name,
		Description: request.Body.Description,
		Type:        channelType,
	}

	if err := h.channelRepo.Create(ctx, ch, userID); err != nil {
		return nil, err
	}

	apiCh := channelToAPI(ch)
	return api.CreateChannel200JSONResponse{
		Channel: &apiCh,
	}, nil
}

// ListChannels lists channels in a workspace
func (h *Handler) ListChannels(ctx context.Context, request api.ListChannelsRequestObject) (api.ListChannelsResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	channels, err := h.channelRepo.ListForWorkspace(ctx, string(request.Wid), userID)
	if err != nil {
		return nil, err
	}

	apiChannels := make([]api.ChannelWithMembership, len(channels))
	for i, ch := range channels {
		apiChannels[i] = channelWithMembershipToAPI(ch)
	}

	return api.ListChannels200JSONResponse{
		Channels: &apiChannels,
	}, nil
}

// CreateDM creates or gets a DM channel
func (h *Handler) CreateDM(ctx context.Context, request api.CreateDMRequestObject) (api.CreateDMResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	// Always include current user and dedupe
	userIDs := append(request.Body.UserIds, userID)
	uniqueIDs := make(map[string]bool)
	var deduped []string
	for _, id := range userIDs {
		if !uniqueIDs[id] {
			uniqueIDs[id] = true
			deduped = append(deduped, id)
		}
	}

	if len(deduped) < 2 {
		return nil, errors.New("DM requires at least 2 participants")
	}

	ch, err := h.channelRepo.CreateDM(ctx, string(request.Wid), deduped)
	if err != nil {
		return nil, err
	}

	apiCh := channelToAPI(ch)
	return api.CreateDM200JSONResponse{
		Channel: &apiCh,
	}, nil
}

// UpdateChannel updates a channel
func (h *Handler) UpdateChannel(ctx context.Context, request api.UpdateChannelRequestObject) (api.UpdateChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Check workspace membership
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return nil, err
	}

	// Check channel membership and role
	channelMembership, err := h.channelRepo.GetMembership(ctx, userID, string(request.Id))
	if err != nil && !errors.Is(err, channel.ErrNotChannelMember) {
		return nil, err
	}

	// Workspace admins or channel admins can update
	canUpdate := workspace.CanManageMembers(membership.Role) || (channelMembership != nil && channel.CanManageChannel(channelMembership.ChannelRole))
	if !canUpdate {
		return nil, errors.New("permission denied")
	}

	if request.Body.Name != nil {
		if strings.TrimSpace(*request.Body.Name) == "" {
			return nil, errors.New("channel name cannot be empty")
		}
		ch.Name = *request.Body.Name
	}
	if request.Body.Description != nil {
		ch.Description = request.Body.Description
	}

	if err := h.channelRepo.Update(ctx, ch); err != nil {
		return nil, err
	}

	apiCh := channelToAPI(ch)
	return api.UpdateChannel200JSONResponse{
		Channel: &apiCh,
	}, nil
}

// ArchiveChannel archives a channel
func (h *Handler) ArchiveChannel(ctx context.Context, request api.ArchiveChannelRequestObject) (api.ArchiveChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Can't archive DMs
	if ch.Type == channel.TypeDM || ch.Type == channel.TypeGroupDM {
		return nil, errors.New("cannot archive DM channels")
	}

	// Check workspace membership
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if !workspace.CanManageMembers(membership.Role) {
		return nil, errors.New("permission denied")
	}

	if err := h.channelRepo.Archive(ctx, string(request.Id)); err != nil {
		return nil, err
	}

	return api.ArchiveChannel200JSONResponse{
		Success: true,
	}, nil
}

// AddChannelMember adds a member to a channel
func (h *Handler) AddChannelMember(ctx context.Context, request api.AddChannelMemberRequestObject) (api.AddChannelMemberResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Check workspace membership
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return nil, err
	}

	// Check permissions - workspace admins or channel members can add
	channelMembership, _ := h.channelRepo.GetMembership(ctx, userID, string(request.Id))
	canAdd := workspace.CanManageMembers(membership.Role) || channelMembership != nil
	if !canAdd {
		return nil, errors.New("permission denied")
	}

	// Verify target user is workspace member
	_, err = h.workspaceRepo.GetMembership(ctx, request.Body.UserId, ch.WorkspaceID)
	if err != nil {
		return nil, errors.New("user is not a member of the workspace")
	}

	var rolePtr *string
	if request.Body.Role != nil {
		role := string(*request.Body.Role)
		rolePtr = &role
	}

	_, err = h.channelRepo.AddMember(ctx, request.Body.UserId, string(request.Id), rolePtr)
	if err != nil {
		return nil, err
	}

	return api.AddChannelMember200JSONResponse{
		Success: true,
	}, nil
}

// ListChannelMembers lists members of a channel
func (h *Handler) ListChannelMembers(ctx context.Context, request api.ListChannelMembersRequestObject) (api.ListChannelMembersResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return nil, err
	}

	// For private channels, must be a member to see members
	if ch.Type == channel.TypePrivate {
		_, err = h.channelRepo.GetMembership(ctx, userID, string(request.Id))
		if err != nil {
			return nil, err
		}
	}

	members, err := h.channelRepo.ListMembers(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	apiMembers := make([]api.ChannelMember, len(members))
	for i, m := range members {
		apiMembers[i] = channelMemberToAPI(m)
	}

	return api.ListChannelMembers200JSONResponse{
		Members: &apiMembers,
	}, nil
}

// JoinChannel joins a public channel
func (h *Handler) JoinChannel(ctx context.Context, request api.JoinChannelRequestObject) (api.JoinChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Only public channels can be joined without invite
	if ch.Type != channel.TypePublic {
		return nil, errors.New("cannot join private channels without an invite")
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return nil, err
	}

	_, err = h.channelRepo.AddMember(ctx, userID, string(request.Id), nil)
	if err != nil && !errors.Is(err, channel.ErrAlreadyMember) {
		return nil, err
	}

	return api.JoinChannel200JSONResponse{
		Success: true,
	}, nil
}

// LeaveChannel leaves a channel
func (h *Handler) LeaveChannel(ctx context.Context, request api.LeaveChannelRequestObject) (api.LeaveChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	err := h.channelRepo.RemoveMember(ctx, userID, string(request.Id))
	if err != nil {
		return nil, err
	}

	return api.LeaveChannel200JSONResponse{
		Success: true,
	}, nil
}

// channelToAPI converts a channel.Channel to api.Channel
func channelToAPI(ch *channel.Channel) api.Channel {
	return api.Channel{
		Id:                ch.ID,
		WorkspaceId:       ch.WorkspaceID,
		Name:              ch.Name,
		Description:       ch.Description,
		Type:              api.ChannelType(ch.Type),
		DmParticipantHash: ch.DMParticipantHash,
		ArchivedAt:        ch.ArchivedAt,
		CreatedBy:         ch.CreatedBy,
		CreatedAt:         ch.CreatedAt,
		UpdatedAt:         ch.UpdatedAt,
	}
}

// channelWithMembershipToAPI converts a channel.ChannelWithMembership to api.ChannelWithMembership
func channelWithMembershipToAPI(ch channel.ChannelWithMembership) api.ChannelWithMembership {
	apiCh := api.ChannelWithMembership{
		Id:                ch.ID,
		WorkspaceId:       ch.WorkspaceID,
		Name:              ch.Name,
		Description:       ch.Description,
		Type:              api.ChannelType(ch.Type),
		DmParticipantHash: ch.DMParticipantHash,
		ArchivedAt:        ch.ArchivedAt,
		CreatedBy:         ch.CreatedBy,
		CreatedAt:         ch.CreatedAt,
		UpdatedAt:         ch.UpdatedAt,
		LastReadMessageId: ch.LastReadMessageID,
		UnreadCount:       ch.UnreadCount,
	}
	if ch.ChannelRole != nil {
		role := api.ChannelRole(*ch.ChannelRole)
		apiCh.ChannelRole = &role
	}
	return apiCh
}

// channelMemberToAPI converts a channel.MemberInfo to api.ChannelMember
func channelMemberToAPI(m channel.MemberInfo) api.ChannelMember {
	apiMember := api.ChannelMember{
		UserId:      m.UserID,
		Email:       openapi_types.Email(m.Email),
		DisplayName: m.DisplayName,
		AvatarUrl:   m.AvatarURL,
	}
	if m.ChannelRole != nil {
		role := api.ChannelRole(*m.ChannelRole)
		apiMember.ChannelRole = &role
	}
	return apiMember
}
