package handler

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/gravatar"
	"github.com/enzyme/api/internal/message"
	"github.com/enzyme/api/internal/notification"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/sse"
	"github.com/enzyme/api/internal/workspace"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var validChannelName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// CreateChannel creates a new channel
func (h *Handler) CreateChannel(ctx context.Context, request openapi.CreateChannelRequestObject) (openapi.CreateChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.CreateChannel401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check workspace membership and permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	if !workspace.CanCreateChannels(membership.Role) {
		return openapi.CreateChannel403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	name := strings.TrimSpace(request.Body.Name)
	if name == "" {
		return openapi.CreateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Channel name is required")}, nil
	}
	if !validChannelName.MatchString(name) {
		return openapi.CreateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Channel name must contain only lowercase letters, numbers, and dashes")}, nil
	}

	// Validate type
	channelType := string(request.Body.Type)
	if channelType != channel.TypePublic && channelType != channel.TypePrivate {
		channelType = channel.TypePublic
	}

	ch := &channel.Channel{
		WorkspaceID: string(request.Wid),
		Name:        name,
		Description: request.Body.Description,
		Type:        channelType,
	}

	if err := h.channelRepo.Create(ctx, ch, userID); err != nil {
		return nil, err
	}

	// Update SSE hub cache with creator as first member
	if h.hub != nil {
		h.hub.AddChannelMember(ch.ID, userID)
	}

	apiCh := channelToAPI(ch)
	return openapi.CreateChannel200JSONResponse{
		Channel: &apiCh,
	}, nil
}

// ListChannels lists channels in a workspace
func (h *Handler) ListChannels(ctx context.Context, request openapi.ListChannelsRequestObject) (openapi.ListChannelsResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListChannels401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
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

	apiChannels := make([]openapi.ChannelWithMembership, len(channels))
	for i, ch := range channels {
		apiChannels[i] = channelWithMembershipToAPI(ch)
	}

	return openapi.ListChannels200JSONResponse{
		Channels: &apiChannels,
	}, nil
}

// CreateDM creates or gets a DM channel
func (h *Handler) CreateDM(ctx context.Context, request openapi.CreateDMRequestObject) (openapi.CreateDMResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.CreateDM401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
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
		return openapi.CreateDM400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "DM requires at least 2 participants")}, nil
	}

	// Check for blocks between participants (workspace-scoped, batch lookup)
	if h.moderationRepo != nil {
		blockedByMe, err := h.moderationRepo.GetBlockedUserIDs(ctx, string(request.Wid), userID)
		if err != nil {
			return nil, err
		}
		blockingMe, err := h.moderationRepo.GetUsersWhoBlocked(ctx, string(request.Wid), userID)
		if err != nil {
			return nil, err
		}
		for _, otherID := range deduped {
			if otherID == userID {
				continue
			}
			if blockedByMe[otherID] || blockingMe[otherID] {
				return openapi.CreateDM400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot create DM with a blocked user")}, nil
			}
		}
	}

	ch, err := h.channelRepo.CreateDM(ctx, string(request.Wid), deduped)
	if err != nil {
		return nil, err
	}

	// Update SSE hub cache with all DM participants
	if h.hub != nil {
		for _, uid := range deduped {
			h.hub.AddChannelMember(ch.ID, uid)
		}
	}

	apiCh := channelToAPI(ch)
	return openapi.CreateDM200JSONResponse{
		Channel: &apiCh,
	}, nil
}

// UpdateChannel updates a channel
func (h *Handler) UpdateChannel(ctx context.Context, request openapi.UpdateChannelRequestObject) (openapi.UpdateChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UpdateChannel401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
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
		return openapi.UpdateChannel403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	oldName := ch.Name
	oldType := ch.Type
	oldDescription := ""
	if ch.Description != nil {
		oldDescription = *ch.Description
	}

	if request.Body.Name != nil {
		name := strings.TrimSpace(*request.Body.Name)
		if name == "" {
			return openapi.UpdateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Channel name cannot be empty")}, nil
		}
		if !validChannelName.MatchString(name) {
			return openapi.UpdateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Channel name must contain only lowercase letters, numbers, and dashes")}, nil
		}
		// Check for duplicate name if name is changing
		if name != ch.Name {
			existing, err := h.channelRepo.GetByWorkspaceAndName(ctx, ch.WorkspaceID, name)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				return openapi.UpdateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "A channel with this name already exists")}, nil
			}
		}
		ch.Name = name
	}
	if request.Body.Description != nil {
		ch.Description = request.Body.Description
	}
	if request.Body.Type != nil {
		newType := string(*request.Body.Type)
		// Only allow public or private
		if newType != channel.TypePublic && newType != channel.TypePrivate {
			return openapi.UpdateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Channel type must be public or private")}, nil
		}
		// Cannot change type on DM/group_dm channels
		if ch.Type == channel.TypeDM || ch.Type == channel.TypeGroupDM {
			return openapi.UpdateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot change visibility of DM channels")}, nil
		}
		// Cannot make the default channel private
		if ch.IsDefault && newType == channel.TypePrivate {
			return openapi.UpdateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot make the default channel private")}, nil
		}
		ch.Type = newType
	}

	if err := h.channelRepo.Update(ctx, ch); err != nil {
		if errors.Is(err, channel.ErrChannelNameTaken) {
			return openapi.UpdateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "A channel with this name already exists")}, nil
		}
		return nil, err
	}

	apiCh := channelToAPI(ch)

	// Broadcast SSE channel.updated event
	if h.hub != nil {
		h.hub.BroadcastToWorkspace(ch.WorkspaceID, sse.Event{
			Type: sse.EventChannelUpdated,
			Data: apiCh,
		})
	}

	// Create system messages for name/type changes
	if oldName != ch.Name {
		h.createChannelRenamedSystemMessage(ctx, ch, oldName, userID)
	}
	if oldType != ch.Type {
		h.createChannelVisibilityChangedSystemMessage(ctx, ch, userID)
	}
	newDescription := ""
	if ch.Description != nil {
		newDescription = *ch.Description
	}
	if oldDescription != newDescription {
		h.createChannelDescriptionUpdatedSystemMessage(ctx, ch, userID)
	}

	return openapi.UpdateChannel200JSONResponse{
		Channel: &apiCh,
	}, nil
}

// ArchiveChannel archives a channel
func (h *Handler) ArchiveChannel(ctx context.Context, request openapi.ArchiveChannelRequestObject) (openapi.ArchiveChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ArchiveChannel401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Can't archive DMs
	if ch.Type == channel.TypeDM || ch.Type == channel.TypeGroupDM {
		return openapi.ArchiveChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot archive DM channels")}, nil
	}

	// Can't archive default channel
	if ch.IsDefault {
		return openapi.ArchiveChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot archive the default channel")}, nil
	}

	// Check workspace membership
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if !workspace.CanManageMembers(membership.Role) {
		return openapi.ArchiveChannel403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	if err := h.channelRepo.Archive(ctx, string(request.Id)); err != nil {
		return nil, err
	}

	// Audit log: channel archived
	if h.moderationRepo != nil {
		_ = h.moderationRepo.CreateAuditLogEntryWithMetadata(ctx, ch.WorkspaceID, userID, "channel.archived", "channel", string(request.Id), map[string]interface{}{
			"channel_name": ch.Name,
		})
	}

	return openapi.ArchiveChannel200JSONResponse{
		Success: true,
	}, nil
}

// AddChannelMember adds a member to a channel
func (h *Handler) AddChannelMember(ctx context.Context, request openapi.AddChannelMemberRequestObject) (openapi.AddChannelMemberResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.AddChannelMember401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
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
		return openapi.AddChannelMember403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	// Verify target user is workspace member
	_, err = h.workspaceRepo.GetMembership(ctx, request.Body.UserId, ch.WorkspaceID)
	if err != nil {
		return openapi.AddChannelMember404JSONResponse{NotFoundJSONResponse: notFoundResponse("User is not a member of the workspace")}, nil
	}

	// DM-specific handling
	if ch.Type == channel.TypeDM || ch.Type == channel.TypeGroupDM {
		currentMemberIDs, err := h.channelRepo.GetMemberUserIDs(ctx, string(request.Id))
		if err != nil {
			return nil, err
		}

		// Check for blocks between new member and existing members
		if h.moderationRepo != nil {
			blockedByNew, err := h.moderationRepo.GetBlockedUserIDs(ctx, ch.WorkspaceID, request.Body.UserId)
			if err != nil {
				return nil, err
			}
			blockingNew, err := h.moderationRepo.GetUsersWhoBlocked(ctx, ch.WorkspaceID, request.Body.UserId)
			if err != nil {
				return nil, err
			}
			for _, existingID := range currentMemberIDs {
				if blockedByNew[existingID] || blockingNew[existingID] {
					return openapi.AddChannelMember403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Cannot add user due to a block relationship")}, nil
				}
			}
		}

		// Enforce soft participant limit
		if len(currentMemberIDs) >= 8 {
			return openapi.AddChannelMember400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Group DMs are limited to 8 participants. Consider creating a channel instead.")}, nil
		}

		updatedCh, err := h.channelRepo.AddMemberToDM(ctx, string(request.Id), request.Body.UserId, currentMemberIDs)
		if err != nil {
			if errors.Is(err, channel.ErrAlreadyMember) {
				return openapi.AddChannelMember200JSONResponse{Success: true}, nil
			}
			return nil, err
		}

		// Update SSE hub cache
		if h.hub != nil {
			h.hub.AddChannelMember(string(request.Id), request.Body.UserId)

			// Broadcast member added event
			h.hub.BroadcastToChannel(ch.WorkspaceID, string(request.Id), sse.Event{
				Type: sse.EventMemberAdded,
				Data: map[string]string{
					"channel_id": string(request.Id),
					"user_id":    request.Body.UserId,
				},
			})

			// Broadcast channel updated if type changed (dm -> group_dm)
			if updatedCh.Type != ch.Type {
				apiCh := channelToAPI(updatedCh)
				h.hub.BroadcastToChannel(ch.WorkspaceID, string(request.Id), sse.Event{
					Type: sse.EventChannelUpdated,
					Data: apiCh,
				})
			}
		}

		// Create system message
		h.createAddedSystemMessage(ctx, ch, request.Body.UserId, userID)

		return openapi.AddChannelMember200JSONResponse{Success: true}, nil
	}

	role := "poster"
	if request.Body.Role != nil {
		role = string(*request.Body.Role)
	}

	_, err = h.channelRepo.AddMember(ctx, request.Body.UserId, string(request.Id), &role)
	if err != nil {
		if errors.Is(err, channel.ErrAlreadyMember) {
			// User is already a member, no need to create system message
			return openapi.AddChannelMember200JSONResponse{
				Success: true,
			}, nil
		}
		return nil, err
	}

	// Update SSE hub cache for channel membership
	if h.hub != nil {
		h.hub.AddChannelMember(string(request.Id), request.Body.UserId)
	}

	// Create system message for user being added
	h.createAddedSystemMessage(ctx, ch, request.Body.UserId, userID)

	return openapi.AddChannelMember200JSONResponse{
		Success: true,
	}, nil
}

// ListChannelMembers lists members of a channel
func (h *Handler) ListChannelMembers(ctx context.Context, request openapi.ListChannelMembersRequestObject) (openapi.ListChannelMembersResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListChannelMembers401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
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
			if errors.Is(err, channel.ErrNotChannelMember) {
				return openapi.ListChannelMembers404JSONResponse{NotFoundJSONResponse: notFoundResponse("Channel not found")}, nil
			}
			return nil, err
		}
	}

	members, err := h.channelRepo.ListMembers(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	apiMembers := make([]openapi.ChannelMember, len(members))
	for i, m := range members {
		apiMembers[i] = channelMemberToAPI(m)
	}

	return openapi.ListChannelMembers200JSONResponse{
		Members: &apiMembers,
	}, nil
}

// JoinChannel joins a public channel
func (h *Handler) JoinChannel(ctx context.Context, request openapi.JoinChannelRequestObject) (openapi.JoinChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.JoinChannel401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Only public channels can be joined without invite
	if ch.Type != channel.TypePublic {
		return openapi.JoinChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot join private channels without an invite")}, nil
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return nil, err
	}

	memberRole := "poster"
	_, err = h.channelRepo.AddMember(ctx, userID, string(request.Id), &memberRole)
	wasAlreadyMember := errors.Is(err, channel.ErrAlreadyMember)
	if wasAlreadyMember {
		// Update role if already a member (in case role was NULL)
		_ = h.channelRepo.UpdateMemberRole(ctx, userID, string(request.Id), &memberRole)
	} else if err != nil {
		return nil, err
	}

	// Update SSE hub cache for channel membership
	if h.hub != nil {
		h.hub.AddChannelMember(string(request.Id), userID)
	}

	// Create system message if this is a new join (not already a member)
	if !wasAlreadyMember {
		h.createJoinSystemMessage(ctx, ch, userID)
	}

	return openapi.JoinChannel200JSONResponse{
		Success: true,
	}, nil
}

// LeaveChannel leaves a channel
func (h *Handler) LeaveChannel(ctx context.Context, request openapi.LeaveChannelRequestObject) (openapi.LeaveChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.LeaveChannel401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Get channel for system message and type comparison (before leaving)
	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	err = h.channelRepo.RemoveMember(ctx, userID, string(request.Id))
	if err != nil {
		if errors.Is(err, channel.ErrCannotLeaveChannel) {
			return openapi.LeaveChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot leave this channel")}, nil
		}
		if errors.Is(err, channel.ErrCannotLeaveDefault) {
			return openapi.LeaveChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot leave the default channel")}, nil
		}
		return nil, err
	}

	// Update SSE hub cache for channel membership
	if h.hub != nil {
		h.hub.RemoveChannelMember(string(request.Id), userID)

		// Broadcast member removed
		h.hub.BroadcastToChannel(ch.WorkspaceID, string(request.Id), sse.Event{
			Type: sse.EventMemberRemoved,
			Data: map[string]string{
				"channel_id": string(request.Id),
				"user_id":    userID,
			},
		})

		// For group DMs, broadcast channel updated (type/hash may have changed)
		if ch.Type == channel.TypeGroupDM {
			updatedCh, err := h.channelRepo.GetByID(ctx, string(request.Id))
			if err == nil {
				apiCh := channelToAPI(updatedCh)
				h.hub.BroadcastToChannel(ch.WorkspaceID, string(request.Id), sse.Event{
					Type: sse.EventChannelUpdated,
					Data: apiCh,
				})
			}
		}
	}

	// Create system message
	h.createLeaveSystemMessage(ctx, ch, userID)

	return openapi.LeaveChannel200JSONResponse{
		Success: true,
	}, nil
}

// channelToAPI converts a channel.Channel to openapi.Channel
func channelToAPI(ch *channel.Channel) openapi.Channel {
	return openapi.Channel{
		Id:                ch.ID,
		WorkspaceId:       ch.WorkspaceID,
		Name:              ch.Name,
		Description:       ch.Description,
		Type:              openapi.ChannelType(ch.Type),
		IsDefault:         ch.IsDefault,
		DmParticipantHash: ch.DMParticipantHash,
		ArchivedAt:        ch.ArchivedAt,
		CreatedBy:         ch.CreatedBy,
		CreatedAt:         ch.CreatedAt,
		UpdatedAt:         ch.UpdatedAt,
	}
}

// channelWithMembershipToAPI converts a channel.ChannelWithMembership to openapi.ChannelWithMembership
func channelWithMembershipToAPI(ch channel.ChannelWithMembership) openapi.ChannelWithMembership {
	apiCh := openapi.ChannelWithMembership{
		Id:                ch.ID,
		WorkspaceId:       ch.WorkspaceID,
		Name:              ch.Name,
		Description:       ch.Description,
		Type:              openapi.ChannelType(ch.Type),
		IsDefault:         ch.IsDefault,
		DmParticipantHash: ch.DMParticipantHash,
		ArchivedAt:        ch.ArchivedAt,
		CreatedBy:         ch.CreatedBy,
		CreatedAt:         ch.CreatedAt,
		UpdatedAt:         ch.UpdatedAt,
		LastReadMessageId: ch.LastReadMessageID,
		UnreadCount:       ch.UnreadCount,
		NotificationCount: ch.NotificationCount,
		IsStarred:         ch.IsStarred,
	}
	if ch.ChannelRole != nil {
		role := openapi.ChannelRole(*ch.ChannelRole)
		apiCh.ChannelRole = &role
	}
	if len(ch.DMParticipants) > 0 {
		participants := make([]openapi.ChannelMember, len(ch.DMParticipants))
		for i, p := range ch.DMParticipants {
			participants[i] = channelMemberToAPI(p)
		}
		apiCh.DmParticipants = &participants
	}
	return apiCh
}

// channelMemberToAPI converts a channel.MemberInfo to openapi.ChannelMember
func channelMemberToAPI(m channel.MemberInfo) openapi.ChannelMember {
	apiMember := openapi.ChannelMember{
		UserId:      m.UserID,
		Email:       openapi_types.Email(m.Email),
		DisplayName: m.DisplayName,
		AvatarUrl:   m.AvatarURL,
	}
	if m.ChannelRole != nil {
		role := openapi.ChannelRole(*m.ChannelRole)
		apiMember.ChannelRole = &role
	}
	if g := gravatar.URL(m.Email); g != "" {
		apiMember.GravatarUrl = &g
	}
	return apiMember
}

// ConvertGroupDMToChannel converts a group DM into a private channel
func (h *Handler) ConvertGroupDMToChannel(ctx context.Context, request openapi.ConvertGroupDMToChannelRequestObject) (openapi.ConvertGroupDMToChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ConvertGroupDMToChannel401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Must be a group DM
	if ch.Type != channel.TypeGroupDM {
		return openapi.ConvertGroupDMToChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Only group DMs can be converted to channels")}, nil
	}

	// Check workspace membership and permissions
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if !workspace.CanCreateChannels(membership.Role) {
		return openapi.ConvertGroupDMToChannel403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Permission denied")}, nil
	}

	// Must be a member of the group DM
	_, err = h.channelRepo.GetMembership(ctx, userID, string(request.Id))
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			return openapi.ConvertGroupDMToChannel403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("You must be a member of this conversation")}, nil
		}
		return nil, err
	}

	// Validate channel name
	name := strings.TrimSpace(request.Body.Name)
	if name == "" {
		return openapi.ConvertGroupDMToChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Channel name is required")}, nil
	}
	if !validChannelName.MatchString(name) {
		return openapi.ConvertGroupDMToChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Channel name must contain only lowercase letters, numbers, and dashes")}, nil
	}

	// Check for duplicate channel name in workspace
	existing, err := h.channelRepo.GetByWorkspaceAndName(ctx, ch.WorkspaceID, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return openapi.ConvertGroupDMToChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "A channel with this name already exists")}, nil
	}

	// Determine target channel type (default: private)
	targetType := channel.TypePrivate
	if request.Body.Type != nil && string(*request.Body.Type) == channel.TypePublic {
		targetType = channel.TypePublic
	}

	// Convert the channel
	converted, err := h.channelRepo.ConvertToChannel(ctx, string(request.Id), name, request.Body.Description, userID, targetType)
	if err != nil {
		return nil, err
	}

	// Broadcast channel updated via SSE
	if h.hub != nil {
		apiCh := channelToAPI(converted)
		h.hub.BroadcastToChannel(ch.WorkspaceID, converted.ID, sse.Event{
			Type: sse.EventChannelUpdated,
			Data: apiCh,
		})
	}

	// Create system message for the conversion
	h.createConvertSystemMessage(ctx, converted, userID, name)

	apiCh := channelToAPI(converted)
	return openapi.ConvertGroupDMToChannel200JSONResponse{
		Channel: &apiCh,
	}, nil
}

// createConvertSystemMessage creates a system message when a group DM is converted to a channel.
// Errors are logged but do not fail the conversion.
func (h *Handler) createConvertSystemMessage(ctx context.Context, ch *channel.Channel, userID, channelName string) {
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}

	event := &message.SystemEventData{
		EventType:       message.SystemEventChannelConverted,
		UserID:          userID,
		UserDisplayName: user.DisplayName,
		ChannelName:     channelName,
	}

	msg, err := h.messageRepo.CreateSystemMessage(ctx, ch.ID, event)
	if err != nil {
		return
	}

	// Broadcast via SSE
	if h.hub != nil {
		msgWithUser, _ := h.messageRepo.GetByIDWithUser(ctx, msg.ID)
		if msgWithUser != nil {
			apiMsg := messageWithUserToAPI(msgWithUser)
			h.hub.BroadcastToChannel(ch.WorkspaceID, ch.ID, sse.Event{
				Type: sse.EventMessageNew,
				Data: apiMsg,
			})
		}
	}
}

// MarkChannelRead marks a channel as read for the current user
func (h *Handler) MarkChannelRead(ctx context.Context, request openapi.MarkChannelReadRequestObject) (openapi.MarkChannelReadResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.MarkChannelRead401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Determine which message ID to use
	var messageID string
	if request.Body != nil && request.Body.MessageId != nil {
		messageID = *request.Body.MessageId
	} else {
		// Get latest message in channel
		messageID, err = h.channelRepo.GetLatestMessageID(ctx, string(request.Id))
		if err != nil {
			return nil, err
		}
	}

	// No messages to mark as read
	if messageID == "" {
		return openapi.MarkChannelRead200JSONResponse{
			LastReadMessageId: "",
		}, nil
	}

	// Update last read
	if err := h.channelRepo.UpdateLastRead(ctx, userID, string(request.Id), messageID); err != nil {
		return nil, err
	}

	// Broadcast to user's other clients
	if h.hub != nil {
		h.hub.BroadcastToUser(ch.WorkspaceID, userID, sse.Event{
			Type: sse.EventChannelRead,
			Data: map[string]string{
				"channel_id":           string(request.Id),
				"last_read_message_id": messageID,
			},
		})
	}

	return openapi.MarkChannelRead200JSONResponse{
		LastReadMessageId: messageID,
	}, nil
}

// MarkAllChannelsRead marks all channels in a workspace as read
func (h *Handler) MarkAllChannelsRead(ctx context.Context, request openapi.MarkAllChannelsReadRequestObject) (openapi.MarkAllChannelsReadResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.MarkAllChannelsRead401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	// Get all channels user is a member of
	channelIDs, err := h.channelRepo.ListMemberChannelIDs(ctx, string(request.Wid), userID)
	if err != nil {
		return nil, err
	}

	// Mark each channel as read
	for _, channelID := range channelIDs {
		messageID, err := h.channelRepo.GetLatestMessageID(ctx, channelID)
		if err != nil {
			continue
		}
		if messageID == "" {
			continue
		}

		if err := h.channelRepo.UpdateLastRead(ctx, userID, channelID, messageID); err != nil {
			continue
		}

		// Broadcast to user's other clients
		if h.hub != nil {
			h.hub.BroadcastToUser(string(request.Wid), userID, sse.Event{
				Type: sse.EventChannelRead,
				Data: map[string]string{
					"channel_id":           channelID,
					"last_read_message_id": messageID,
				},
			})
		}
	}

	return openapi.MarkAllChannelsRead200JSONResponse{
		Success: true,
	}, nil
}

// GetChannelNotifications returns notification preferences for a channel
func (h *Handler) GetChannelNotifications(ctx context.Context, request openapi.GetChannelNotificationsRequestObject) (openapi.GetChannelNotificationsResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.GetChannelNotifications401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
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

	// Get preferences (will return defaults if not set)
	pref, err := h.notificationService.GetPreferences(ctx, userID, string(request.Id), ch.Type)
	if err != nil {
		return nil, err
	}

	apiPrefs := notificationPreferencesToAPI(pref)
	return openapi.GetChannelNotifications200JSONResponse{
		Preferences: &apiPrefs,
	}, nil
}

// UpdateChannelNotifications updates notification preferences for a channel
func (h *Handler) UpdateChannelNotifications(ctx context.Context, request openapi.UpdateChannelNotificationsRequestObject) (openapi.UpdateChannelNotificationsResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UpdateChannelNotifications401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
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

	// Validate notify level
	notifyLevel := string(request.Body.NotifyLevel)
	if notifyLevel != notification.NotifyAll && notifyLevel != notification.NotifyMentions && notifyLevel != notification.NotifyNone {
		return openapi.UpdateChannelNotifications400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid notify_level")}, nil
	}

	pref := &notification.NotificationPreference{
		UserID:       userID,
		ChannelID:    string(request.Id),
		NotifyLevel:  notifyLevel,
		EmailEnabled: request.Body.EmailEnabled,
	}

	if err := h.notificationService.SetPreferences(ctx, pref); err != nil {
		return nil, err
	}

	apiPrefs := notificationPreferencesToAPI(pref)
	return openapi.UpdateChannelNotifications200JSONResponse{
		Preferences: &apiPrefs,
	}, nil
}

// notificationPreferencesToAPI converts notification preferences to API type
func notificationPreferencesToAPI(pref *notification.NotificationPreference) openapi.NotificationPreferences {
	return openapi.NotificationPreferences{
		NotifyLevel:  openapi.NotifyLevel(pref.NotifyLevel),
		EmailEnabled: pref.EmailEnabled,
	}
}

// StarChannel stars a channel for the current user
func (h *Handler) StarChannel(ctx context.Context, request openapi.StarChannelRequestObject) (openapi.StarChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.StarChannel401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	if err := h.channelRepo.StarChannel(ctx, userID, string(request.Id)); err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			return openapi.StarChannel404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a member of this channel")}, nil
		}
		return nil, err
	}

	return openapi.StarChannel200JSONResponse{
		Success: true,
	}, nil
}

// UnstarChannel unstars a channel for the current user
func (h *Handler) UnstarChannel(ctx context.Context, request openapi.UnstarChannelRequestObject) (openapi.UnstarChannelResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UnstarChannel401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	if err := h.channelRepo.UnstarChannel(ctx, userID, string(request.Id)); err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			return openapi.UnstarChannel404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a member of this channel")}, nil
		}
		return nil, err
	}

	return openapi.UnstarChannel200JSONResponse{
		Success: true,
	}, nil
}

// createJoinSystemMessage creates a system message when a user joins a channel
func (h *Handler) createJoinSystemMessage(ctx context.Context, ch *channel.Channel, userID string) {
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}
	h.createChannelSystemMessage(ctx, ch, &message.SystemEventData{
		EventType:       message.SystemEventUserJoined,
		UserID:          userID,
		UserDisplayName: user.DisplayName,
		ChannelName:     ch.Name,
	})
}

// createLeaveSystemMessage creates a system message when a user leaves a channel
func (h *Handler) createLeaveSystemMessage(ctx context.Context, ch *channel.Channel, userID string) {
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}
	h.createChannelSystemMessage(ctx, ch, &message.SystemEventData{
		EventType:       message.SystemEventUserLeft,
		UserID:          userID,
		UserDisplayName: user.DisplayName,
		ChannelName:     ch.Name,
	})
}

// createChannelSystemMessage creates and broadcasts a system message for a channel event.
// Checks workspace settings and silently returns on any error.
func (h *Handler) createChannelSystemMessage(ctx context.Context, ch *channel.Channel, event *message.SystemEventData) {
	ws, err := h.workspaceRepo.GetByID(ctx, ch.WorkspaceID)
	if err != nil {
		return
	}
	settings := ws.ParsedSettings()
	if !settings.ShowJoinLeaveMessages {
		return
	}

	msg, err := h.messageRepo.CreateSystemMessage(ctx, ch.ID, event)
	if err != nil {
		return
	}

	if h.hub != nil {
		msgWithUser, _ := h.messageRepo.GetByIDWithUser(ctx, msg.ID)
		if msgWithUser != nil {
			apiMsg := messageWithUserToAPI(msgWithUser)
			h.hub.BroadcastToChannel(ch.WorkspaceID, ch.ID, sse.Event{
				Type: sse.EventMessageNew,
				Data: apiMsg,
			})
		}
	}
}

// createChannelRenamedSystemMessage creates a system message when a channel is renamed
func (h *Handler) createChannelRenamedSystemMessage(ctx context.Context, ch *channel.Channel, oldName string, userID string) {
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}
	h.createChannelSystemMessage(ctx, ch, &message.SystemEventData{
		EventType:       message.SystemEventChannelRenamed,
		UserID:          userID,
		UserDisplayName: user.DisplayName,
		ChannelName:     ch.Name,
		OldChannelName:  &oldName,
	})
}

// createChannelVisibilityChangedSystemMessage creates a system message when channel visibility changes
func (h *Handler) createChannelVisibilityChangedSystemMessage(ctx context.Context, ch *channel.Channel, userID string) {
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}
	h.createChannelSystemMessage(ctx, ch, &message.SystemEventData{
		EventType:       message.SystemEventChannelVisibilityChanged,
		UserID:          userID,
		UserDisplayName: user.DisplayName,
		ChannelName:     ch.Name,
		ChannelType:     &ch.Type,
	})
}

// createChannelDescriptionUpdatedSystemMessage creates a system message when the channel description changes
func (h *Handler) createChannelDescriptionUpdatedSystemMessage(ctx context.Context, ch *channel.Channel, userID string) {
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}
	h.createChannelSystemMessage(ctx, ch, &message.SystemEventData{
		EventType:       message.SystemEventChannelDescriptionUpdated,
		UserID:          userID,
		UserDisplayName: user.DisplayName,
		ChannelName:     ch.Name,
	})
}

// createAddedSystemMessage creates a system message when a user is added to a channel
func (h *Handler) createAddedSystemMessage(ctx context.Context, ch *channel.Channel, addedUserID, actorID string) {
	// Check workspace settings
	ws, err := h.workspaceRepo.GetByID(ctx, ch.WorkspaceID)
	if err != nil {
		return
	}
	settings := ws.ParsedSettings()
	if !settings.ShowJoinLeaveMessages {
		return
	}

	// Get added user's display name
	addedUser, err := h.userRepo.GetByID(ctx, addedUserID)
	if err != nil {
		return
	}

	event := &message.SystemEventData{
		EventType:       message.SystemEventUserAdded,
		UserID:          addedUserID,
		UserDisplayName: addedUser.DisplayName,
		ChannelName:     ch.Name,
	}

	// Get actor's display name if different from added user
	if actorID != addedUserID {
		actor, err := h.userRepo.GetByID(ctx, actorID)
		if err == nil {
			event.ActorID = &actorID
			event.ActorDisplayName = &actor.DisplayName
		}
	}

	msg, err := h.messageRepo.CreateSystemMessage(ctx, ch.ID, event)
	if err != nil {
		return
	}

	// Broadcast via SSE
	if h.hub != nil {
		msgWithUser, _ := h.messageRepo.GetByIDWithUser(ctx, msg.ID)
		if msgWithUser != nil {
			apiMsg := messageWithUserToAPI(msgWithUser)
			h.hub.BroadcastToChannel(ch.WorkspaceID, ch.ID, sse.Event{
				Type: sse.EventMessageNew,
				Data: apiMsg,
			})
		}
	}
}
