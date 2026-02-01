package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/file"
	"github.com/feather/api/internal/message"
	"github.com/feather/api/internal/notification"
	"github.com/feather/api/internal/openapi"
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
				memberRole := "poster"
				_, _ = h.channelRepo.AddMember(ctx, userID, string(request.Id), &memberRole)
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

	// Content is required unless attachments are provided
	content := ""
	if request.Body.Content != nil {
		content = strings.TrimSpace(*request.Body.Content)
	}
	hasContent := content != ""
	hasAttachments := request.Body.AttachmentIds != nil && len(*request.Body.AttachmentIds) > 0

	if !hasContent && !hasAttachments {
		return nil, errors.New("message content or attachments required")
	}

	// Validate attachments if provided
	var attachmentIDs []string
	if hasAttachments {
		attachmentIDs = *request.Body.AttachmentIds
		// Validate each attachment
		for _, attachmentID := range attachmentIDs {
			attachment, err := h.fileRepo.GetByID(ctx, attachmentID)
			if err != nil {
				return nil, fmt.Errorf("attachment %s not found", attachmentID)
			}
			// Verify attachment belongs to this channel
			if attachment.ChannelID != string(request.Id) {
				return nil, fmt.Errorf("attachment %s does not belong to this channel", attachmentID)
			}
			// Verify user owns the attachment
			if attachment.UserID == nil || *attachment.UserID != userID {
				return nil, fmt.Errorf("attachment %s does not belong to you", attachmentID)
			}
			// Verify attachment not already linked to a message
			if attachment.MessageID != nil {
				return nil, fmt.Errorf("attachment %s is already linked to a message", attachmentID)
			}
		}
	}

	// Validate thread parent if provided
	var threadParent *message.Message
	if request.Body.ThreadParentId != nil {
		var err error
		threadParent, err = h.messageRepo.GetByID(ctx, *request.Body.ThreadParentId)
		if err != nil {
			if errors.Is(err, message.ErrMessageNotFound) {
				return nil, errors.New("thread parent message not found")
			}
			return nil, err
		}
		if threadParent.ChannelID != string(request.Id) {
			return nil, errors.New("thread parent must be in the same channel")
		}
		// Can't reply to a reply
		if threadParent.ThreadParentID != nil {
			return nil, errors.New("cannot reply to a thread reply")
		}
	}

	// Parse mentions from content
	var mentions []string
	if h.notificationService != nil && content != "" {
		mentions, _ = notification.ParseMentions(ctx, h.userRepo, ch.WorkspaceID, content)
	}

	msg := &message.Message{
		ChannelID:      string(request.Id),
		UserID:         &userID,
		Content:        content,
		Mentions:       mentions,
		ThreadParentID: request.Body.ThreadParentId,
	}

	if err := h.messageRepo.Create(ctx, msg); err != nil {
		return nil, err
	}

	// Handle thread subscription auto-subscribe
	if threadParent != nil && h.threadRepo != nil {
		// Auto-subscribe the sender to the thread (respects explicit unsubscribe)
		_ = h.threadRepo.AutoSubscribe(ctx, threadParent.ID, userID)

		// If this is the first reply, auto-subscribe the thread author
		if threadParent.ReplyCount == 0 && threadParent.UserID != nil && *threadParent.UserID != userID {
			_ = h.threadRepo.AutoSubscribe(ctx, threadParent.ID, *threadParent.UserID)
		}
	}

	// Link attachments to the message
	if len(attachmentIDs) > 0 {
		for _, attachmentID := range attachmentIDs {
			if err := h.fileRepo.UpdateMessageID(ctx, attachmentID, msg.ID); err != nil {
				return nil, err
			}
		}
	}

	// Fetch message with user info for response and broadcast
	msgWithUser, err := h.messageRepo.GetByIDWithUser(ctx, msg.ID)
	if err != nil {
		// Fall back to basic message if user fetch fails
		msgWithUser = &message.MessageWithUser{Message: *msg}
	}

	// Load attachments for the message
	if len(attachmentIDs) > 0 {
		attachments, _ := h.fileRepo.ListForMessage(ctx, msg.ID)
		msgWithUser.Attachments = attachments
	}

	apiMsg := messageWithUserToAPI(msgWithUser)

	// Broadcast message via SSE (use API type to include attachment URLs)
	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, string(request.Id), sse.Event{
			Type: sse.EventMessageNew,
			Data: apiMsg,
		})
	}

	// Trigger notifications
	if h.notificationService != nil {
		// Get sender's display name
		senderName := ""
		if sender, err := h.userRepo.GetByID(ctx, userID); err == nil {
			senderName = sender.DisplayName
		}

		channelInfo := &notification.ChannelInfo{
			ID:          ch.ID,
			WorkspaceID: ch.WorkspaceID,
			Name:        ch.Name,
			Type:        ch.Type,
		}
		msgInfo := &notification.MessageInfo{
			ID:             msg.ID,
			ChannelID:      msg.ChannelID,
			SenderID:       userID,
			SenderName:     senderName,
			Content:        msg.Content,
			Mentions:       mentions,
			ThreadParentID: msg.ThreadParentID,
		}
		// Send notifications asynchronously
		go func() {
			_ = h.notificationService.Notify(context.Background(), channelInfo, msgInfo)
		}()
	}

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

	// Load attachments for all messages
	h.loadAttachmentsForMessages(ctx, result.Messages)

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

	// Load attachments for the message
	if msgWithUser != nil {
		attachments, _ := h.fileRepo.ListForMessage(ctx, msg.ID)
		msgWithUser.Attachments = attachments
	}

	apiMsg := messageWithUserToAPI(msgWithUser)

	// Broadcast update via SSE (use API type to include attachment URLs)
	if h.hub != nil && ch != nil && msgWithUser != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, msg.ChannelID, sse.Event{
			Type: sse.EventMessageUpdated,
			Data: apiMsg,
		})
	}

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

	// Load attachments for all messages
	h.loadAttachmentsForMessages(ctx, result.Messages)

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
	if len(m.Attachments) > 0 {
		attachments := make([]openapi.Attachment, len(m.Attachments))
		for i, a := range m.Attachments {
			attachments[i] = attachmentToAPI(&a)
		}
		apiMsg.Attachments = &attachments
	}
	return apiMsg
}

// attachmentToAPI converts a file.Attachment to openapi.Attachment
func attachmentToAPI(a *file.Attachment) openapi.Attachment {
	url := fmt.Sprintf("/api/files/%s/download", a.ID)
	return openapi.Attachment{
		Id:          a.ID,
		Filename:    a.Filename,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		Url:         url,
		CreatedAt:   a.CreatedAt,
	}
}

// loadAttachmentsForMessages loads attachments for a slice of messages in batch
func (h *Handler) loadAttachmentsForMessages(ctx context.Context, messages []message.MessageWithUser) {
	if len(messages) == 0 {
		return
	}

	messageIDs := make([]string, len(messages))
	for i, m := range messages {
		messageIDs[i] = m.ID
	}

	attachmentsMap, err := h.fileRepo.ListForMessages(ctx, messageIDs)
	if err != nil {
		return
	}

	for i := range messages {
		if attachments, ok := attachmentsMap[messages[i].ID]; ok {
			messages[i].Attachments = attachments
		}
	}
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

// GetMessage retrieves a single message by ID
func (h *Handler) GetMessage(ctx context.Context, request openapi.GetMessageRequestObject) (openapi.GetMessageResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.GetMessage401JSONResponse{}, nil
	}

	// Get the message with user info
	msgWithUser, err := h.messageRepo.GetByIDWithUser(ctx, string(request.Id))
	if err != nil {
		if errors.Is(err, message.ErrMessageNotFound) {
			return openapi.GetMessage404JSONResponse{}, nil
		}
		return nil, err
	}

	// Check channel access
	ch, err := h.channelRepo.GetByID(ctx, msgWithUser.ChannelID)
	if err != nil {
		return openapi.GetMessage404JSONResponse{}, nil
	}

	_, err = h.channelRepo.GetMembership(ctx, userID, msgWithUser.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return openapi.GetMessage404JSONResponse{}, nil
			}
			// Verify workspace membership for public channels
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return openapi.GetMessage404JSONResponse{}, nil
			}
		} else {
			return nil, err
		}
	}

	// Load reactions for the message
	reactions, err := h.messageRepo.GetReactionsForMessage(ctx, msgWithUser.ID)
	if err == nil {
		msgWithUser.Reactions = reactions
	}

	// Load attachments for the message
	attachments, _ := h.fileRepo.ListForMessage(ctx, msgWithUser.ID)
	msgWithUser.Attachments = attachments

	// Load thread participants if this is a parent message with replies
	if msgWithUser.ReplyCount > 0 {
		participants, err := h.messageRepo.GetThreadParticipants(ctx, msgWithUser.ID)
		if err == nil {
			msgWithUser.ThreadParticipants = participants
		}
	}

	apiMsg := messageWithUserToAPI(msgWithUser)
	return openapi.GetMessage200JSONResponse{
		Message: &apiMsg,
	}, nil
}

// MarkMessageUnread marks a message as unread by setting last_read to the previous message
func (h *Handler) MarkMessageUnread(ctx context.Context, request openapi.MarkMessageUnreadRequestObject) (openapi.MarkMessageUnreadResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}

	// Get the message
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

	// Get the message before this one
	prevMessageID, err := h.channelRepo.GetPreviousMessageID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	// Update last read to the previous message (or empty if no previous)
	if prevMessageID != "" {
		if err := h.channelRepo.UpdateLastRead(ctx, userID, msg.ChannelID, prevMessageID); err != nil {
			return nil, err
		}
	} else {
		// No previous message - clear last_read_message_id to mark all as unread
		if err := h.channelRepo.UpdateLastRead(ctx, userID, msg.ChannelID, ""); err != nil {
			return nil, err
		}
	}

	// Broadcast to user's other clients
	if h.hub != nil {
		h.hub.BroadcastToUser(ch.WorkspaceID, userID, sse.Event{
			Type: sse.EventChannelRead,
			Data: map[string]string{
				"channel_id":           msg.ChannelID,
				"last_read_message_id": prevMessageID,
			},
		})
	}

	return openapi.MarkMessageUnread200JSONResponse{
		Success: true,
	}, nil
}
