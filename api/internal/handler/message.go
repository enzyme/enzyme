package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/message"
	"github.com/feather/api/internal/sse"
	"github.com/feather/api/internal/workspace"
)

// SendMessage sends a message to a channel
func (h *Handler) SendMessage(ctx context.Context, request openapi.SendMessageRequestObject) (openapi.SendMessageResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Check channel is not archived
	if ch.ArchivedAt != nil {
		return nil, errors.New("cannot post to archived channel")
	}

	// Check channel membership
	membership, err := h.channelRepo.GetMembership(ctx, userID, string(request.Id))
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			// For public channels, check if user is workspace member
			if ch.Type == channel.TypePublic {
				_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
				if err != nil {
					return nil, errors.New("not a member of this workspace")
				}
				// Auto-join public channel
				_, _ = h.channelRepo.AddMember(ctx, userID, string(request.Id), nil)
				// Update SSE hub cache
				if h.hub != nil {
					h.hub.AddChannelMember(string(request.Id), userID)
				}
			} else {
				return nil, errors.New("not a member of this channel")
			}
		} else {
			return nil, err
		}
	} else if !channel.CanPost(membership.ChannelRole) {
		return nil, errors.New("permission denied")
	}

	if strings.TrimSpace(request.Body.Content) == "" {
		return nil, errors.New("message content is required")
	}

	// Validate thread parent if provided
	if request.Body.ThreadParentId != nil {
		parent, err := h.messageRepo.GetByID(ctx, *request.Body.ThreadParentId)
		if err != nil {
			if errors.Is(err, message.ErrMessageNotFound) {
				return nil, errors.New("thread parent message not found")
			}
			return nil, err
		}
		if parent.ChannelID != string(request.Id) {
			return nil, errors.New("thread parent must be in the same channel")
		}
		// Can't reply to a reply
		if parent.ThreadParentID != nil {
			return nil, errors.New("cannot reply to a thread reply")
		}
	}

	msg := &message.Message{
		ChannelID:      string(request.Id),
		UserID:         &userID,
		Content:        request.Body.Content,
		ThreadParentID: request.Body.ThreadParentId,
	}

	if err := h.messageRepo.Create(ctx, msg); err != nil {
		return nil, err
	}

	// Fetch message with user info for response and broadcast
	msgWithUser, err := h.messageRepo.GetByIDWithUser(ctx, msg.ID)
	if err != nil {
		// Fall back to basic message if user fetch fails
		msgWithUser = &message.MessageWithUser{Message: *msg}
	}

	// Broadcast message via SSE
	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, string(request.Id), sse.Event{
			Type: sse.EventMessageNew,
			Data: msgWithUser,
		})
	}

	apiMsg := messageWithUserToAPI(msgWithUser)
	return openapi.SendMessage200JSONResponse{
		Message: &apiMsg,
	}, nil
}

// ListMessages lists messages in a channel
func (h *Handler) ListMessages(ctx context.Context, request openapi.ListMessagesRequestObject) (openapi.ListMessagesResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Check access
	_, err = h.channelRepo.GetMembership(ctx, userID, string(request.Id))
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return nil, errors.New("not a member of this channel")
			}
			// Public channels: verify workspace membership
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return nil, errors.New("not a member of this workspace")
			}
		} else {
			return nil, err
		}
	}

	opts := message.ListOptions{}
	if request.Body != nil {
		if request.Body.Cursor != nil {
			opts.Cursor = *request.Body.Cursor
		}
		if request.Body.Limit != nil {
			opts.Limit = *request.Body.Limit
		}
		if request.Body.Direction != nil {
			opts.Direction = string(*request.Body.Direction)
		}
	}

	result, err := h.messageRepo.List(ctx, string(request.Id), opts)
	if err != nil {
		return nil, err
	}

	return openapi.ListMessages200JSONResponse(messageListResultToAPI(result)), nil
}

// UpdateMessage updates a message
func (h *Handler) UpdateMessage(ctx context.Context, request openapi.UpdateMessageRequestObject) (openapi.UpdateMessageResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	msg, err := h.messageRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Only message author can edit
	if msg.UserID == nil || *msg.UserID != userID {
		return nil, errors.New("you can only edit your own messages")
	}

	// Can't edit deleted messages
	if msg.DeletedAt != nil {
		return nil, errors.New("cannot edit deleted message")
	}

	if strings.TrimSpace(request.Body.Content) == "" {
		return nil, errors.New("message content is required")
	}

	if err := h.messageRepo.Update(ctx, string(request.Id), request.Body.Content); err != nil {
		return nil, err
	}

	// Get updated message with user info
	msgWithUser, _ := h.messageRepo.GetByIDWithUser(ctx, string(request.Id))

	// Get channel for workspace ID
	ch, _ := h.channelRepo.GetByID(ctx, msg.ChannelID)

	// Broadcast update via SSE
	if h.hub != nil && ch != nil && msgWithUser != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, msg.ChannelID, sse.Event{
			Type: sse.EventMessageUpdated,
			Data: msgWithUser,
		})
	}

	apiMsg := messageWithUserToAPI(msgWithUser)
	return openapi.UpdateMessage200JSONResponse{
		Message: &apiMsg,
	}, nil
}

// DeleteMessage deletes a message
func (h *Handler) DeleteMessage(ctx context.Context, request openapi.DeleteMessageRequestObject) (openapi.DeleteMessageResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	msg, err := h.messageRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Check permissions: author or workspace admin
	ch, err := h.channelRepo.GetByID(ctx, msg.ChannelID)
	if err != nil {
		return nil, err
	}

	canDelete := msg.UserID != nil && *msg.UserID == userID

	if !canDelete {
		membership, err := h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
		if err == nil && workspace.CanManageMembers(membership.Role) {
			canDelete = true
		}
	}

	if !canDelete {
		return nil, errors.New("permission denied")
	}

	if err := h.messageRepo.Delete(ctx, string(request.Id)); err != nil {
		return nil, err
	}

	// Broadcast delete via SSE
	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, msg.ChannelID, sse.Event{
			Type: sse.EventMessageDeleted,
			Data: map[string]string{"id": string(request.Id)},
		})
	}

	return openapi.DeleteMessage200JSONResponse{
		Success: true,
	}, nil
}

// AddReaction adds a reaction to a message
func (h *Handler) AddReaction(ctx context.Context, request openapi.AddReactionRequestObject) (openapi.AddReactionResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	msg, err := h.messageRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Check channel membership
	ch, err := h.channelRepo.GetByID(ctx, msg.ChannelID)
	if err != nil {
		return nil, err
	}

	_, err = h.channelRepo.GetMembership(ctx, userID, msg.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return nil, errors.New("not a member of this channel")
			}
		} else {
			return nil, err
		}
	}

	if strings.TrimSpace(request.Body.Emoji) == "" {
		return nil, errors.New("emoji is required")
	}

	reaction, err := h.messageRepo.AddReaction(ctx, string(request.Id), userID, request.Body.Emoji)
	if err != nil {
		return nil, err
	}

	// Broadcast reaction via SSE
	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, msg.ChannelID, sse.Event{
			Type: sse.EventReactionAdded,
			Data: reaction,
		})
	}

	apiReaction := reactionToAPI(reaction)
	return openapi.AddReaction200JSONResponse{
		Reaction: &apiReaction,
	}, nil
}

// RemoveReaction removes a reaction from a message
func (h *Handler) RemoveReaction(ctx context.Context, request openapi.RemoveReactionRequestObject) (openapi.RemoveReactionResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	msg, err := h.messageRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	err = h.messageRepo.RemoveReaction(ctx, string(request.Id), userID, request.Body.Emoji)
	if err != nil {
		return nil, err
	}

	// Get channel for broadcasting
	ch, _ := h.channelRepo.GetByID(ctx, msg.ChannelID)

	// Broadcast removal via SSE
	if h.hub != nil && ch != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, msg.ChannelID, sse.Event{
			Type: sse.EventReactionRemoved,
			Data: map[string]string{
				"message_id": string(request.Id),
				"user_id":    userID,
				"emoji":      request.Body.Emoji,
			},
		})
	}

	return openapi.RemoveReaction200JSONResponse{
		Success: true,
	}, nil
}

// ListThread lists thread replies
func (h *Handler) ListThread(ctx context.Context, request openapi.ListThreadRequestObject) (openapi.ListThreadResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	msg, err := h.messageRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Check channel access
	ch, err := h.channelRepo.GetByID(ctx, msg.ChannelID)
	if err != nil {
		return nil, err
	}

	_, err = h.channelRepo.GetMembership(ctx, userID, msg.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return nil, errors.New("not a member of this channel")
			}
			// Verify workspace membership for public channels
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return nil, errors.New("not a member of this workspace")
			}
		} else {
			return nil, err
		}
	}

	opts := message.ListOptions{}
	if request.Body != nil {
		if request.Body.Cursor != nil {
			opts.Cursor = *request.Body.Cursor
		}
		if request.Body.Limit != nil {
			opts.Limit = *request.Body.Limit
		}
	}

	result, err := h.messageRepo.ListThread(ctx, string(request.Id), opts)
	if err != nil {
		return nil, err
	}

	return openapi.ListThread200JSONResponse(messageListResultToAPI(result)), nil
}

// messageWithUserToAPI converts a message.MessageWithUser to openapi.MessageWithUser
func messageWithUserToAPI(m *message.MessageWithUser) openapi.MessageWithUser {
	apiMsg := openapi.MessageWithUser{
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
			reactions[i] = reactionToAPI(&r)
		}
		apiMsg.Reactions = &reactions
	}
	if len(m.ThreadParticipants) > 0 {
		participants := make([]openapi.ThreadParticipant, len(m.ThreadParticipants))
		for i, p := range m.ThreadParticipants {
			participants[i] = threadParticipantToAPI(&p)
		}
		apiMsg.ThreadParticipants = &participants
	}
	return apiMsg
}

// threadParticipantToAPI converts a message.ThreadParticipant to openapi.ThreadParticipant
func threadParticipantToAPI(p *message.ThreadParticipant) openapi.ThreadParticipant {
	participant := openapi.ThreadParticipant{
		UserId: p.UserID,
	}
	if p.DisplayName != "" {
		participant.DisplayName = &p.DisplayName
	}
	if p.AvatarURL != nil {
		participant.AvatarUrl = p.AvatarURL
	}
	return participant
}

// reactionToAPI converts a message.Reaction to openapi.Reaction
func reactionToAPI(r *message.Reaction) openapi.Reaction {
	return openapi.Reaction{
		Id:        r.ID,
		MessageId: r.MessageID,
		UserId:    r.UserID,
		Emoji:     r.Emoji,
		CreatedAt: r.CreatedAt,
	}
}

// messageListResultToAPI converts a message.ListResult to openapi.MessageListResult
func messageListResultToAPI(result *message.ListResult) openapi.MessageListResult {
	messages := make([]openapi.MessageWithUser, len(result.Messages))
	for i, m := range result.Messages {
		messages[i] = messageWithUserToAPI(&m)
	}

	apiResult := openapi.MessageListResult{
		Messages: messages,
		HasMore:  result.HasMore,
	}
	if result.NextCursor != "" {
		apiResult.NextCursor = &result.NextCursor
	}
	return apiResult
}
