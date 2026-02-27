package handler

import (
	"context"
	"errors"

	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/gravatar"
	"github.com/enzyme/api/internal/message"
	"github.com/enzyme/api/internal/moderation"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/thread"
)

// GetThreadSubscription returns the user's subscription status for a thread
func (h *Handler) GetThreadSubscription(ctx context.Context, request openapi.GetThreadSubscriptionRequestObject) (openapi.GetThreadSubscriptionResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.GetThreadSubscription401JSONResponse{}, nil
	}

	// Verify the message exists and is a thread parent (not a reply itself)
	msg, err := h.messageRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		if errors.Is(err, message.ErrMessageNotFound) {
			return openapi.GetThreadSubscription404JSONResponse{}, nil
		}
		return nil, err
	}

	// Check if user has access to the channel
	ch, err := h.channelRepo.GetByID(ctx, msg.ChannelID)
	if err != nil {
		return openapi.GetThreadSubscription404JSONResponse{}, nil
	}

	_, err = h.channelRepo.GetMembership(ctx, userID, msg.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return openapi.GetThreadSubscription404JSONResponse{}, nil
			}
			// For public channels, check workspace membership
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return openapi.GetThreadSubscription404JSONResponse{}, nil
			}
		} else {
			return nil, err
		}
	}

	// Get subscription status
	sub, err := h.threadRepo.GetSubscription(ctx, string(request.Id), userID)
	if err != nil {
		return nil, err
	}

	var status openapi.ThreadSubscriptionStatus
	if sub == nil {
		status = openapi.ThreadSubscriptionStatusNone
	} else if sub.Status == thread.StatusSubscribed {
		status = openapi.ThreadSubscriptionStatusSubscribed
	} else {
		status = openapi.ThreadSubscriptionStatusUnsubscribed
	}

	return openapi.GetThreadSubscription200JSONResponse{
		Status: status,
	}, nil
}

// SubscribeToThread subscribes the user to a thread
func (h *Handler) SubscribeToThread(ctx context.Context, request openapi.SubscribeToThreadRequestObject) (openapi.SubscribeToThreadResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.SubscribeToThread401JSONResponse{}, nil
	}

	// Verify the message exists
	msg, err := h.messageRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		if errors.Is(err, message.ErrMessageNotFound) {
			return openapi.SubscribeToThread404JSONResponse{}, nil
		}
		return nil, err
	}

	// Check if user has access to the channel
	ch, err := h.channelRepo.GetByID(ctx, msg.ChannelID)
	if err != nil {
		return openapi.SubscribeToThread404JSONResponse{}, nil
	}

	_, err = h.channelRepo.GetMembership(ctx, userID, msg.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return openapi.SubscribeToThread404JSONResponse{}, nil
			}
			// For public channels, check workspace membership
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return openapi.SubscribeToThread404JSONResponse{}, nil
			}
		} else {
			return nil, err
		}
	}

	// Subscribe the user
	_, err = h.threadRepo.Subscribe(ctx, string(request.Id), userID)
	if err != nil {
		return nil, err
	}

	return openapi.SubscribeToThread200JSONResponse{
		Status: openapi.ThreadSubscriptionStatusSubscribed,
	}, nil
}

// UnsubscribeFromThread unsubscribes the user from a thread
func (h *Handler) UnsubscribeFromThread(ctx context.Context, request openapi.UnsubscribeFromThreadRequestObject) (openapi.UnsubscribeFromThreadResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UnsubscribeFromThread401JSONResponse{}, nil
	}

	// Verify the message exists
	msg, err := h.messageRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		if errors.Is(err, message.ErrMessageNotFound) {
			return openapi.UnsubscribeFromThread404JSONResponse{}, nil
		}
		return nil, err
	}

	// Check if user has access to the channel
	ch, err := h.channelRepo.GetByID(ctx, msg.ChannelID)
	if err != nil {
		return openapi.UnsubscribeFromThread404JSONResponse{}, nil
	}

	_, err = h.channelRepo.GetMembership(ctx, userID, msg.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return openapi.UnsubscribeFromThread404JSONResponse{}, nil
			}
			// For public channels, check workspace membership
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return openapi.UnsubscribeFromThread404JSONResponse{}, nil
			}
		} else {
			return nil, err
		}
	}

	// Unsubscribe the user
	_, err = h.threadRepo.Unsubscribe(ctx, string(request.Id), userID)
	if err != nil {
		return nil, err
	}

	return openapi.UnsubscribeFromThread200JSONResponse{
		Status: openapi.ThreadSubscriptionStatusUnsubscribed,
	}, nil
}

// MarkThreadRead marks a thread as read for the current user
func (h *Handler) MarkThreadRead(ctx context.Context, request openapi.MarkThreadReadRequestObject) (openapi.MarkThreadReadResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.MarkThreadRead401JSONResponse{}, nil
	}

	// Verify the message exists
	msg, err := h.messageRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		if errors.Is(err, message.ErrMessageNotFound) {
			return openapi.MarkThreadRead404JSONResponse{}, nil
		}
		return nil, err
	}

	// Check if user has access to the channel
	ch, err := h.channelRepo.GetByID(ctx, msg.ChannelID)
	if err != nil {
		return openapi.MarkThreadRead404JSONResponse{}, nil
	}

	_, err = h.channelRepo.GetMembership(ctx, userID, msg.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				return openapi.MarkThreadRead404JSONResponse{}, nil
			}
			_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
			if err != nil {
				return openapi.MarkThreadRead404JSONResponse{}, nil
			}
		} else {
			return nil, err
		}
	}

	// Determine the reply ID to mark as read
	var replyID string
	if request.Body != nil && request.Body.LastReadReplyId != nil {
		replyID = *request.Body.LastReadReplyId
	} else {
		// Default to latest reply
		replyID, err = h.threadRepo.GetLatestReplyID(ctx, string(request.Id))
		if err != nil {
			return nil, err
		}
		if replyID == "" {
			// No replies yet, nothing to mark
			return openapi.MarkThreadRead200JSONResponse{Success: true}, nil
		}
	}

	err = h.threadRepo.UpdateLastReadReplyID(ctx, string(request.Id), userID, replyID)
	if err != nil {
		return nil, err
	}

	return openapi.MarkThreadRead200JSONResponse{Success: true}, nil
}

// ListUserThreads lists all threads the user is subscribed to in a workspace
func (h *Handler) ListUserThreads(ctx context.Context, request openapi.ListUserThreadsRequestObject) (openapi.ListUserThreadsResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListUserThreads401JSONResponse{}, nil
	}

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(ctx, userID, string(request.Wid))
	if err != nil {
		return nil, err
	}

	opts := message.ListOptions{
		Limit: 20,
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
	result, err := h.messageRepo.ListUserThreads(ctx, string(request.Wid), userID, opts, filter)
	if err != nil {
		return nil, err
	}

	// Get unread thread count for badge
	unreadCount, err := h.threadRepo.CountUnreadThreads(ctx, string(request.Wid), userID)
	if err != nil {
		return nil, err
	}
	result.UnreadThreadCount = unreadCount

	return openapi.ListUserThreads200JSONResponse(threadListResultToAPI(result)), nil
}

// threadMessageToAPI converts a message.ThreadMessage to openapi.ThreadMessage
func threadMessageToAPI(m *message.ThreadMessage) openapi.ThreadMessage {
	apiMsg := openapi.ThreadMessage{
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
		HasNewReplies:  m.HasNewReplies,
	}
	if m.Type != "" {
		msgType := openapi.MessageType(m.Type)
		apiMsg.Type = &msgType
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
	if len(m.ThreadParticipants) > 0 {
		participants := make([]openapi.ThreadParticipant, len(m.ThreadParticipants))
		for i, p := range m.ThreadParticipants {
			participants[i] = openapi.ThreadParticipant{
				UserId:      p.UserID,
				DisplayName: &p.DisplayName,
				AvatarUrl:   p.AvatarURL,
			}
			if g := gravatar.URL(p.Email); g != "" {
				participants[i].GravatarUrl = &g
			}
		}
		apiMsg.ThreadParticipants = &participants
	}
	return apiMsg
}

// threadListResultToAPI converts a message.ThreadListResult to openapi.ThreadListResult
func threadListResultToAPI(result *message.ThreadListResult) openapi.ThreadListResult {
	threads := make([]openapi.ThreadMessage, len(result.Threads))
	for i, m := range result.Threads {
		threads[i] = threadMessageToAPI(&m)
	}

	apiResult := openapi.ThreadListResult{
		Threads:           threads,
		HasMore:           result.HasMore,
		UnreadThreadCount: result.UnreadThreadCount,
	}
	if result.NextCursor != "" {
		apiResult.NextCursor = &result.NextCursor
	}
	return apiResult
}
