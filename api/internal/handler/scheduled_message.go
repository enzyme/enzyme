package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/linkpreview"
	"github.com/enzyme/api/internal/message"
	"github.com/enzyme/api/internal/notification"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/scheduled"
	"github.com/enzyme/api/internal/sse"
)

// scheduledMessageToAPI converts a scheduled.ScheduledMessage to openapi.ScheduledMessage
func scheduledMessageToAPI(m *scheduled.ScheduledMessage) openapi.ScheduledMessage {
	sm := openapi.ScheduledMessage{
		Id:           m.ID,
		ChannelId:    m.ChannelID,
		UserId:       m.UserID,
		Content:      m.Content,
		ScheduledFor: m.ScheduledFor,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	if m.ThreadParentID != nil {
		sm.ThreadParentId = m.ThreadParentID
	}
	if m.AlsoSendToChannel {
		sm.AlsoSendToChannel = &m.AlsoSendToChannel
	}
	if len(m.AttachmentIDs) > 0 {
		sm.AttachmentIds = &m.AttachmentIDs
	}
	if m.Status != "" {
		status := openapi.ScheduledMessageStatus(m.Status)
		sm.Status = &status
	}
	if m.LastError != "" {
		sm.LastError = &m.LastError
	}
	return sm
}

// scheduledMessageWithChannelToAPI converts with channel info
func scheduledMessageWithChannelToAPI(m *scheduled.ScheduledMessageWithChannel) openapi.ScheduledMessage {
	sm := scheduledMessageToAPI(&m.ScheduledMessage)
	sm.ChannelName = &m.ChannelName
	sm.WorkspaceId = &m.WorkspaceID
	return sm
}

// ScheduleMessage creates a scheduled message for future delivery
func (h *Handler) ScheduleMessage(ctx context.Context, request openapi.ScheduleMessageRequestObject) (openapi.ScheduleMessageResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ScheduleMessage401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}

	if ch.ArchivedAt != nil {
		return openapi.ScheduleMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Cannot schedule message in archived channel")}, nil
	}

	// Check channel membership
	_, err = h.channelRepo.GetMembership(ctx, userID, string(request.Id))
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			return openapi.ScheduleMessage403JSONResponse{ForbiddenJSONResponse: notAMemberResponse("Not a member of this channel")}, nil
		}
		return nil, err
	}

	content := strings.TrimSpace(request.Body.Content)
	if content == "" {
		return openapi.ScheduleMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Message content is required")}, nil
	}
	if utf8.RuneCountInString(content) > maxMessageLength {
		return openapi.ScheduleMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, fmt.Sprintf("Message content exceeds maximum length of %d characters", maxMessageLength))}, nil
	}

	// Validate scheduled_for is in the future
	scheduledFor := request.Body.ScheduledFor
	if scheduledFor.Before(time.Now().Add(1 * time.Minute)) {
		return openapi.ScheduleMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Scheduled time must be at least 1 minute in the future")}, nil
	}

	var attachmentIDs []string
	if request.Body.AttachmentIds != nil {
		attachmentIDs = *request.Body.AttachmentIds
		for _, aid := range attachmentIDs {
			att, err := h.fileRepo.GetByID(ctx, aid)
			if err != nil {
				return openapi.ScheduleMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, fmt.Sprintf("Attachment %s not found", aid))}, nil
			}
			if att.ChannelID != string(request.Id) {
				return openapi.ScheduleMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, fmt.Sprintf("Attachment %s does not belong to this channel", aid))}, nil
			}
			if att.UserID == nil || *att.UserID != userID {
				return openapi.ScheduleMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, fmt.Sprintf("Attachment %s does not belong to you", aid))}, nil
			}
			if att.MessageID != nil {
				return openapi.ScheduleMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, fmt.Sprintf("Attachment %s is already linked to a message", aid))}, nil
			}
		}
	}

	msg := &scheduled.ScheduledMessage{
		ChannelID:     string(request.Id),
		UserID:        userID,
		Content:       content,
		AttachmentIDs: attachmentIDs,
		ScheduledFor:  scheduledFor,
	}
	if request.Body.ThreadParentId != nil {
		msg.ThreadParentID = request.Body.ThreadParentId
	}
	if request.Body.AlsoSendToChannel != nil && *request.Body.AlsoSendToChannel {
		msg.AlsoSendToChannel = true
	}

	if err := h.scheduledRepo.Create(ctx, msg); err != nil {
		return nil, err
	}

	apiMsg := scheduledMessageToAPI(msg)

	// Broadcast to the user
	if h.hub != nil {
		h.hub.BroadcastToUser(ch.WorkspaceID, userID, sse.Event{
			Type: sse.EventScheduledMessageCreated,
			Data: apiMsg,
		})
	}

	return openapi.ScheduleMessage200JSONResponse{
		ScheduledMessage: &apiMsg,
	}, nil
}

// ListScheduledMessages lists the user's scheduled messages in a workspace
func (h *Handler) ListScheduledMessages(ctx context.Context, request openapi.ListScheduledMessagesRequestObject) (openapi.ListScheduledMessagesResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListScheduledMessages401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	messages, err := h.scheduledRepo.ListByUser(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	apiMessages := make([]openapi.ScheduledMessage, len(messages))
	for i, m := range messages {
		apiMessages[i] = scheduledMessageWithChannelToAPI(&m)
	}

	count := len(apiMessages)
	return openapi.ListScheduledMessages200JSONResponse{
		ScheduledMessages: &apiMessages,
		Count:             &count,
	}, nil
}

// GetScheduledMessage gets a scheduled message by ID
func (h *Handler) GetScheduledMessage(ctx context.Context, request openapi.GetScheduledMessageRequestObject) (openapi.GetScheduledMessageResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.GetScheduledMessage401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	msg, err := h.scheduledRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		if errors.Is(err, scheduled.ErrScheduledMessageNotFound) {
			return openapi.GetScheduledMessage404JSONResponse{NotFoundJSONResponse: notFoundResponse("Scheduled message not found")}, nil
		}
		return nil, err
	}

	if msg.UserID != userID {
		return openapi.GetScheduledMessage404JSONResponse{NotFoundJSONResponse: notFoundResponse("Scheduled message not found")}, nil
	}

	apiMsg := scheduledMessageToAPI(msg)
	return openapi.GetScheduledMessage200JSONResponse{
		ScheduledMessage: &apiMsg,
	}, nil
}

// UpdateScheduledMessage updates a scheduled message
func (h *Handler) UpdateScheduledMessage(ctx context.Context, request openapi.UpdateScheduledMessageRequestObject) (openapi.UpdateScheduledMessageResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UpdateScheduledMessage401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	msg, err := h.scheduledRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		if errors.Is(err, scheduled.ErrScheduledMessageNotFound) {
			return openapi.UpdateScheduledMessage404JSONResponse{NotFoundJSONResponse: notFoundResponse("Scheduled message not found")}, nil
		}
		return nil, err
	}

	if msg.UserID != userID {
		return openapi.UpdateScheduledMessage403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("You can only edit your own scheduled messages")}, nil
	}

	if msg.Status == scheduled.StatusSending {
		return openapi.UpdateScheduledMessage409JSONResponse{ConflictJSONResponse: conflictResponse("Message is currently being sent")}, nil
	}

	if request.Body.Content != nil {
		content := strings.TrimSpace(*request.Body.Content)
		if content == "" {
			return openapi.UpdateScheduledMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Message content is required")}, nil
		}
		if utf8.RuneCountInString(content) > maxMessageLength {
			return openapi.UpdateScheduledMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, fmt.Sprintf("Message content exceeds maximum length of %d characters", maxMessageLength))}, nil
		}
		msg.Content = content
	}

	if request.Body.ScheduledFor != nil {
		scheduledFor := *request.Body.ScheduledFor
		if scheduledFor.Before(time.Now().Add(1 * time.Minute)) {
			return openapi.UpdateScheduledMessage400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Scheduled time must be in the future")}, nil
		}
		msg.ScheduledFor = scheduledFor
	}

	if request.Body.AttachmentIds != nil {
		msg.AttachmentIDs = *request.Body.AttachmentIds
	}

	if err := h.scheduledRepo.Update(ctx, msg); err != nil {
		return nil, err
	}

	apiMsg := scheduledMessageToAPI(msg)

	// Get workspace ID for broadcasting
	if h.hub != nil {
		if ch, err := h.channelRepo.GetByID(ctx, msg.ChannelID); err == nil {
			h.hub.BroadcastToUser(ch.WorkspaceID, userID, sse.Event{
				Type: sse.EventScheduledMessageUpdated,
				Data: apiMsg,
			})
		}
	}

	return openapi.UpdateScheduledMessage200JSONResponse{
		ScheduledMessage: &apiMsg,
	}, nil
}

// DeleteScheduledMessage deletes a scheduled message
func (h *Handler) DeleteScheduledMessage(ctx context.Context, request openapi.DeleteScheduledMessageRequestObject) (openapi.DeleteScheduledMessageResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.DeleteScheduledMessage401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	msg, err := h.scheduledRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		if errors.Is(err, scheduled.ErrScheduledMessageNotFound) {
			return openapi.DeleteScheduledMessage404JSONResponse{NotFoundJSONResponse: notFoundResponse("Scheduled message not found")}, nil
		}
		return nil, err
	}

	if msg.UserID != userID {
		return openapi.DeleteScheduledMessage403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("You can only delete your own scheduled messages")}, nil
	}

	if msg.Status == scheduled.StatusSending {
		return openapi.DeleteScheduledMessage409JSONResponse{ConflictJSONResponse: conflictResponse("Message is currently being sent")}, nil
	}

	if err := h.scheduledRepo.Delete(ctx, string(request.Id)); err != nil {
		return nil, err
	}

	// Broadcast deletion
	if h.hub != nil {
		if ch, err := h.channelRepo.GetByID(ctx, msg.ChannelID); err == nil {
			h.hub.BroadcastToUser(ch.WorkspaceID, userID, sse.Event{
				Type: sse.EventScheduledMessageDeleted,
				Data: map[string]string{"id": msg.ID},
			})
		}
	}

	return openapi.DeleteScheduledMessage200JSONResponse{
		Success: true,
	}, nil
}

// SendScheduledMessageNow sends a scheduled message immediately
func (h *Handler) SendScheduledMessageNow(ctx context.Context, request openapi.SendScheduledMessageNowRequestObject) (openapi.SendScheduledMessageNowResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.SendScheduledMessageNow401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	smsg, err := h.scheduledRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		if errors.Is(err, scheduled.ErrScheduledMessageNotFound) {
			return openapi.SendScheduledMessageNow404JSONResponse{NotFoundJSONResponse: notFoundResponse("Scheduled message not found")}, nil
		}
		return nil, err
	}

	if smsg.UserID != userID {
		return openapi.SendScheduledMessageNow403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("You can only send your own scheduled messages")}, nil
	}

	// Atomically claim the message to prevent the worker from also sending it
	claimed, err := h.scheduledRepo.MarkSending(ctx, smsg.ID)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return openapi.SendScheduledMessageNow409JSONResponse{ConflictJSONResponse: conflictResponse("Message is currently being sent")}, nil
	}

	apiMsg, err := h.executeScheduledSend(ctx, smsg)
	if err != nil {
		// Clean up: reset status so the message isn't stuck in "sending"
		var permErr *scheduled.PermanentError
		if errors.As(err, &permErr) {
			_ = h.scheduledRepo.MarkFailed(ctx, smsg.ID, permErr.Error())
			return openapi.SendScheduledMessageNow400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, permErr.Error())}, nil
		}
		// Transient error — reset to pending so the user can retry
		_ = h.scheduledRepo.IncrementRetry(ctx, smsg.ID, err.Error())
		return nil, err
	}

	return openapi.SendScheduledMessageNow200JSONResponse{
		Message: apiMsg,
	}, nil
}

// executeScheduledSend executes the actual message send for a scheduled message,
// deletes the scheduled message row, and broadcasts events.
// Used by both SendScheduledMessageNow handler and the worker.
func (h *Handler) executeScheduledSend(ctx context.Context, smsg *scheduled.ScheduledMessage) (*openapi.MessageWithUser, error) {
	ch, err := h.channelRepo.GetByID(ctx, smsg.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}

	// Check channel is not archived
	if ch.ArchivedAt != nil {
		return nil, &scheduled.PermanentError{Err: fmt.Errorf("channel is archived")}
	}

	// Check user is still a channel member
	_, err = h.channelRepo.GetMembership(ctx, smsg.UserID, smsg.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			return nil, &scheduled.PermanentError{Err: fmt.Errorf("user is no longer a channel member")}
		}
		return nil, fmt.Errorf("checking channel membership: %w", err)
	}

	// Parse mentions from content
	var mentions []string
	var originalMentions []string
	if h.notificationService != nil && smsg.Content != "" {
		mentions, _ = notification.ParseMentions(ctx, h.userRepo, ch.WorkspaceID, smsg.Content)
		originalMentions = mentions

		if h.hub != nil && slices.Contains(mentions, notification.MentionHere) {
			memberIDs, err := h.channelRepo.GetMemberUserIDs(ctx, smsg.ChannelID)
			if err != nil {
				slog.Error("failed to get channel members for @here resolution", "component", "scheduled", "error", err)
			} else {
				mentions = notification.ResolveHereMentions(mentions, memberIDs, smsg.UserID, h.hub, ch.WorkspaceID)
			}
		}
	}

	msg := &message.Message{
		ChannelID:      smsg.ChannelID,
		UserID:         &smsg.UserID,
		Content:        smsg.Content,
		Mentions:       mentions,
		ThreadParentID: smsg.ThreadParentID,
	}

	if smsg.AlsoSendToChannel && msg.ThreadParentID != nil {
		msg.AlsoSendToChannel = true
	}

	if err := h.messageRepo.Create(ctx, msg); err != nil {
		return nil, fmt.Errorf("creating message: %w", err)
	}

	// Handle thread subscription auto-subscribe
	if smsg.ThreadParentID != nil && h.threadRepo != nil {
		threadParent, tpErr := h.messageRepo.GetByID(ctx, *smsg.ThreadParentID)
		if tpErr == nil {
			_ = h.threadRepo.AutoSubscribe(ctx, threadParent.ID, smsg.UserID)
			if threadParent.ReplyCount == 0 && threadParent.UserID != nil && *threadParent.UserID != smsg.UserID {
				_ = h.threadRepo.AutoSubscribe(ctx, threadParent.ID, *threadParent.UserID)
			}
		}
	}

	// Link attachments
	for _, attachmentID := range smsg.AttachmentIDs {
		if err := h.fileRepo.UpdateMessageID(ctx, attachmentID, msg.ID); err != nil {
			slog.Error("failed to link attachment for scheduled message", "attachment_id", attachmentID, "error", err)
		}
	}

	// Fetch message with user info
	msgWithUser, err := h.messageRepo.GetByIDWithUser(ctx, msg.ID)
	if err != nil {
		msgWithUser = &message.MessageWithUser{Message: *msg}
	}

	// Load attachments
	if len(smsg.AttachmentIDs) > 0 {
		attachments, _ := h.fileRepo.ListForMessage(ctx, msg.ID)
		msgWithUser.Attachments = attachments
	}

	// Link preview
	if h.linkPreviewFetcher != nil && smsg.Content != "" {
		if firstURL := linkpreview.ExtractFirstURL(smsg.Content); firstURL != "" {
			preview := h.fetchLinkPreview(ctx, firstURL, msg.ID, msg.ChannelID, ch.WorkspaceID)
			if preview != nil {
				msgWithUser.LinkPreview = preview
			}
		}
	}

	apiMsg := messageWithUserToAPI(msgWithUser)

	// Broadcast the new message
	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, smsg.ChannelID, sse.Event{
			Type: sse.EventMessageNew,
			Data: apiMsg,
		})
	}

	// Trigger notifications
	if h.notificationService != nil {
		senderName := ""
		if sender, err := h.userRepo.GetByID(ctx, smsg.UserID); err == nil {
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
			SenderID:       smsg.UserID,
			SenderName:     senderName,
			Content:        msg.Content,
			Mentions:       originalMentions,
			ThreadParentID: msg.ThreadParentID,
		}
		go func() {
			_ = h.notificationService.Notify(context.Background(), channelInfo, msgInfo)
		}()
	}

	// Delete the scheduled message
	if err := h.scheduledRepo.Delete(ctx, smsg.ID); err != nil {
		slog.Error("failed to delete scheduled message after send", "id", smsg.ID, "error", err)
	}

	// Broadcast scheduled_message.sent event to the user
	if h.hub != nil {
		h.hub.BroadcastToUser(ch.WorkspaceID, smsg.UserID, sse.Event{
			Type: sse.EventScheduledMessageSent,
			Data: map[string]interface{}{
				"id":         smsg.ID,
				"channel_id": smsg.ChannelID,
				"message_id": msg.ID,
			},
		})
	}

	return &apiMsg, nil
}

// ExecuteScheduledSend is the exported version for the worker
func (h *Handler) ExecuteScheduledSend(ctx context.Context, smsg *scheduled.ScheduledMessage) error {
	_, err := h.executeScheduledSend(ctx, smsg)
	return err
}

// NotifyScheduledMessageFailed sends a failure notification to the user via SSE.
func (h *Handler) NotifyScheduledMessageFailed(ctx context.Context, smsg *scheduled.ScheduledMessage, reason string) {
	if h.hub == nil {
		return
	}
	ch, err := h.channelRepo.GetByID(ctx, smsg.ChannelID)
	if err != nil {
		return
	}
	h.hub.BroadcastToUser(ch.WorkspaceID, smsg.UserID, sse.Event{
		Type: sse.EventScheduledMessageFailed,
		Data: map[string]string{"id": smsg.ID, "channel_id": smsg.ChannelID, "error": reason},
	})
}
