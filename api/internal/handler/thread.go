package handler

import (
	"context"
	"errors"

	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/message"
	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/thread"
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
