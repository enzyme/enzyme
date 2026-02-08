package handler

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/message"
	"github.com/feather/api/internal/notification"
	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/sse"
	"github.com/feather/api/internal/workspace"
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

	if request.Body.Name != nil {
		name := strings.TrimSpace(*request.Body.Name)
		if name == "" {
			return openapi.UpdateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Channel name cannot be empty")}, nil
		}
		if !validChannelName.MatchString(name) {
			return openapi.UpdateChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Channel name must contain only lowercase letters, numbers, and dashes")}, nil
		}
		ch.Name = name
	}
	if request.Body.Description != nil {
		ch.Description = request.Body.Description
	}

	if err := h.channelRepo.Update(ctx, ch); err != nil {
		return nil, err
	}

	apiCh := channelToAPI(ch)
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

	// Get channel for system message (before leaving)
	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	err = h.channelRepo.RemoveMember(ctx, userID, string(request.Id))
	if err != nil {
		if errors.Is(err, channel.ErrCannotLeaveDefault) {
			return openapi.LeaveChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot leave the default channel")}, nil
		}
		return nil, err
	}

	// Update SSE hub cache for channel membership
	if h.hub != nil {
		h.hub.RemoveChannelMember(string(request.Id), userID)
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
	return apiMember
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

// GetDMSuggestions returns recent DMs and suggested users for starting conversations
func (h *Handler) GetDMSuggestions(ctx context.Context, request openapi.GetDMSuggestionsRequestObject) (openapi.GetDMSuggestionsResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.GetDMSuggestions401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	// Get DMs with messages in the last 30 days
	since := time.Now().AddDate(0, 0, -30)
	recentDMs, err := h.channelRepo.ListRecentDMs(ctx, string(request.Wid), userID, since, 10)
	if err != nil {
		return nil, err
	}

	// Convert to API types
	apiRecentDMs := make([]openapi.ChannelWithMembership, len(recentDMs))
	recentDMUserIDs := make(map[string]bool)
	for i, dm := range recentDMs {
		apiRecentDMs[i] = channelWithMembershipToAPI(dm)
		// Track users in recent DMs
		for _, p := range dm.DMParticipants {
			recentDMUserIDs[p.UserID] = true
		}
	}

	// Get suggested users if we have fewer than 5 recent DMs
	var apiSuggestedUsers []openapi.SuggestedUser
	if len(recentDMs) < 5 {
		// Get workspace members, excluding those already in recent DMs
		members, err := h.workspaceRepo.ListMembers(ctx, string(request.Wid))
		if err != nil {
			return nil, err
		}

		// Filter and build suggested users
		for _, m := range members {
			// Skip current user and users already in recent DMs
			if m.UserID == userID || recentDMUserIDs[m.UserID] {
				continue
			}

			suggestedUser := openapi.SuggestedUser{
				Id:          m.UserID,
				DisplayName: m.DisplayName,
				AvatarUrl:   m.AvatarURL,
			}
			email := openapi_types.Email(m.Email)
			suggestedUser.Email = &email

			apiSuggestedUsers = append(apiSuggestedUsers, suggestedUser)

			// Limit to 10 suggestions
			if len(apiSuggestedUsers) >= 10 {
				break
			}
		}
	}

	if apiSuggestedUsers == nil {
		apiSuggestedUsers = []openapi.SuggestedUser{}
	}

	return openapi.GetDMSuggestions200JSONResponse{
		RecentDms:      apiRecentDMs,
		SuggestedUsers: apiSuggestedUsers,
	}, nil
}

// createJoinSystemMessage creates a system message when a user joins a channel
func (h *Handler) createJoinSystemMessage(ctx context.Context, ch *channel.Channel, userID string) {
	// Check workspace settings
	ws, err := h.workspaceRepo.GetByID(ctx, ch.WorkspaceID)
	if err != nil {
		return
	}
	settings := ws.ParsedSettings()
	if !settings.ShowJoinLeaveMessages {
		return
	}

	// Get user's display name
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}

	event := &message.SystemEventData{
		EventType:       message.SystemEventUserJoined,
		UserID:          userID,
		UserDisplayName: user.DisplayName,
		ChannelName:     ch.Name,
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

// createLeaveSystemMessage creates a system message when a user leaves a channel
func (h *Handler) createLeaveSystemMessage(ctx context.Context, ch *channel.Channel, userID string) {
	// Check workspace settings
	ws, err := h.workspaceRepo.GetByID(ctx, ch.WorkspaceID)
	if err != nil {
		return
	}
	settings := ws.ParsedSettings()
	if !settings.ShowJoinLeaveMessages {
		return
	}

	// Get user's display name
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}

	event := &message.SystemEventData{
		EventType:       message.SystemEventUserLeft,
		UserID:          userID,
		UserDisplayName: user.DisplayName,
		ChannelName:     ch.Name,
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
