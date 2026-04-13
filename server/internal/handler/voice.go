package handler

import (
	"context"
	"errors"
	"log/slog"

	"github.com/enzyme/server/internal/channel"
	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/sse"
	"github.com/enzyme/server/internal/voice"
	"github.com/enzyme/server/internal/workspace"
	"github.com/pion/webrtc/v4"
)

// voiceDisabledResponse returns a 400 response indicating voice is not enabled.
func voiceDisabledResponse() openapi.BadRequestJSONResponse {
	return badRequestResponse(ErrCodeValidationError, "Voice channels are not enabled")
}

func (h *Handler) JoinVoiceChannel(ctx context.Context, request openapi.JoinVoiceChannelRequestObject) (openapi.JoinVoiceChannelResponseObject, error) {
	if h.voiceSFU == nil {
		return openapi.JoinVoiceChannel400JSONResponse{BadRequestJSONResponse: voiceDisabledResponse()}, nil
	}

	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.JoinVoiceChannel401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return openapi.JoinVoiceChannel404JSONResponse{NotFoundJSONResponse: notFoundResponse("Channel not found")}, nil
	}

	if ch.Type != channel.TypeVoice {
		return openapi.JoinVoiceChannel400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Not a voice channel")}, nil
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return openapi.JoinVoiceChannel404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a workspace member")}, nil
	}

	// Check if already a participant
	isParticipant, err := h.voiceRepo.IsParticipant(ctx, ch.ID, userID)
	if err != nil {
		return nil, err
	}
	if isParticipant {
		return openapi.JoinVoiceChannel409JSONResponse{ConflictJSONResponse: conflictResponse("Already in this voice channel")}, nil
	}

	// Atomically check max participants and insert (prevents TOCTOU race)
	participant, err := h.voiceRepo.AddParticipantIfUnderLimit(ctx, ch.ID, userID, h.voiceMaxPerChannel)
	if err != nil {
		if errors.Is(err, voice.ErrChannelFull) {
			return openapi.JoinVoiceChannel409JSONResponse{ConflictJSONResponse: conflictResponse("Voice channel is full")}, nil
		}
		return nil, err
	}

	// Join the SFU room
	offer, err := h.voiceSFU.JoinRoom(ch.ID, ch.WorkspaceID, userID)
	if err != nil {
		slog.Error("joining voice room", "error", err, "channel", ch.ID, "user", userID)
		// Roll back DB participant on SFU failure
		_ = h.voiceRepo.RemoveParticipant(ctx, ch.ID, userID)
		return nil, err
	}
	_ = participant // used for the insert

	// Broadcast voice.joined to channel members
	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, ch.ID, sse.NewVoiceJoinedEvent(openapi.VoiceParticipantEvent{
			UserId:    userID,
			ChannelId: ch.ID,
		}))
	}

	// Build ICE server list for the client
	servers := h.voiceSFU.ICEServers()
	iceServers := make([]openapi.ICEServer, len(servers))
	for i, s := range servers {
		iceServers[i] = openapi.ICEServer{
			Urls: s.URLs,
		}
		if s.Username != "" {
			iceServers[i].Username = &s.Username
		}
		cred, _ := s.Credential.(string)
		if cred != "" {
			iceServers[i].Credential = &cred
		}
	}

	return openapi.JoinVoiceChannel200JSONResponse{
		Offer: openapi.SDPDescription{
			Sdp:  offer.SDP,
			Type: openapi.SDPDescriptionType(offer.Type.String()),
		},
		IceServers: iceServers,
	}, nil
}

func (h *Handler) LeaveVoiceChannel(ctx context.Context, request openapi.LeaveVoiceChannelRequestObject) (openapi.LeaveVoiceChannelResponseObject, error) {
	if h.voiceSFU == nil {
		return openapi.LeaveVoiceChannel404JSONResponse{NotFoundJSONResponse: notFoundResponse("Voice channels are not enabled")}, nil
	}

	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.LeaveVoiceChannel401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return openapi.LeaveVoiceChannel404JSONResponse{NotFoundJSONResponse: notFoundResponse("Channel not found")}, nil
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return openapi.LeaveVoiceChannel404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a workspace member")}, nil
	}

	// Leave SFU room
	if err := h.voiceSFU.LeaveRoom(ch.ID, userID); err != nil {
		slog.Error("leaving voice room", "error", err, "channel", ch.ID, "user", userID)
	}

	// Remove DB participant
	if err := h.voiceRepo.RemoveParticipant(ctx, ch.ID, userID); err != nil {
		return nil, err
	}

	// Broadcast voice.left
	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, ch.ID, sse.NewVoiceLeftEvent(openapi.VoiceParticipantEvent{
			UserId:    userID,
			ChannelId: ch.ID,
		}))
	}

	return openapi.LeaveVoiceChannel200JSONResponse{Success: true}, nil
}

func (h *Handler) VoiceAnswer(ctx context.Context, request openapi.VoiceAnswerRequestObject) (openapi.VoiceAnswerResponseObject, error) {
	if h.voiceSFU == nil {
		return openapi.VoiceAnswer400JSONResponse{BadRequestJSONResponse: voiceDisabledResponse()}, nil
	}

	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.VoiceAnswer401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return openapi.VoiceAnswer404JSONResponse{NotFoundJSONResponse: notFoundResponse("Channel not found")}, nil
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return openapi.VoiceAnswer404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a workspace member")}, nil
	}

	sdpType := webrtc.SDPTypeAnswer
	if err := h.voiceSFU.HandleAnswer(ch.ID, userID, webrtc.SessionDescription{
		Type: sdpType,
		SDP:  request.Body.Answer.Sdp,
	}); err != nil {
		return openapi.VoiceAnswer400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid SDP answer")}, nil
	}

	return openapi.VoiceAnswer200JSONResponse{Success: true}, nil
}

func (h *Handler) VoiceICECandidate(ctx context.Context, request openapi.VoiceICECandidateRequestObject) (openapi.VoiceICECandidateResponseObject, error) {
	if h.voiceSFU == nil {
		return openapi.VoiceICECandidate400JSONResponse{BadRequestJSONResponse: voiceDisabledResponse()}, nil
	}

	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.VoiceICECandidate401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return openapi.VoiceICECandidate404JSONResponse{NotFoundJSONResponse: notFoundResponse("Channel not found")}, nil
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return openapi.VoiceICECandidate404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a workspace member")}, nil
	}

	candidate := webrtc.ICECandidateInit{
		Candidate: request.Body.Candidate,
	}
	if request.Body.SdpMid != nil {
		candidate.SDPMid = request.Body.SdpMid
	}
	if request.Body.SdpMlineIndex != nil {
		idx := uint16(*request.Body.SdpMlineIndex)
		candidate.SDPMLineIndex = &idx
	}

	if err := h.voiceSFU.HandleICECandidate(ch.ID, userID, candidate); err != nil {
		return openapi.VoiceICECandidate400JSONResponse{BadRequestJSONResponse: badRequestResponse(ErrCodeValidationError, "Invalid ICE candidate")}, nil
	}

	return openapi.VoiceICECandidate200JSONResponse{Success: true}, nil
}

func (h *Handler) MuteVoice(ctx context.Context, request openapi.MuteVoiceRequestObject) (openapi.MuteVoiceResponseObject, error) {
	if h.voiceSFU == nil {
		return openapi.MuteVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Voice channels are not enabled")}, nil
	}

	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.MuteVoice401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return openapi.MuteVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Channel not found")}, nil
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return openapi.MuteVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a workspace member")}, nil
	}

	participant, err := h.voiceRepo.GetParticipant(ctx, ch.ID, userID)
	if err != nil {
		return openapi.MuteVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not in voice channel")}, nil
	}

	muted := request.Body.Muted
	if err := h.voiceRepo.UpdateMuteState(ctx, ch.ID, userID, muted, participant.IsDeafened, participant.IsServerMuted); err != nil {
		return nil, err
	}

	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, ch.ID, sse.NewVoiceMutedEvent(openapi.VoiceMutedEvent{
			UserId:      userID,
			ChannelId:   ch.ID,
			Muted:       muted,
			Deafened:    participant.IsDeafened,
			ServerMuted: participant.IsServerMuted,
		}))
	}

	return openapi.MuteVoice200JSONResponse{Success: true}, nil
}

func (h *Handler) DeafenVoice(ctx context.Context, request openapi.DeafenVoiceRequestObject) (openapi.DeafenVoiceResponseObject, error) {
	if h.voiceSFU == nil {
		return openapi.DeafenVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Voice channels are not enabled")}, nil
	}

	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.DeafenVoice401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return openapi.DeafenVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Channel not found")}, nil
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return openapi.DeafenVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a workspace member")}, nil
	}

	participant, err := h.voiceRepo.GetParticipant(ctx, ch.ID, userID)
	if err != nil {
		return openapi.DeafenVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not in voice channel")}, nil
	}

	deafened := request.Body.Deafened
	muted := participant.IsMuted
	// When deafened, also mute
	if deafened {
		muted = true
	}

	if err := h.voiceRepo.UpdateMuteState(ctx, ch.ID, userID, muted, deafened, participant.IsServerMuted); err != nil {
		return nil, err
	}

	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, ch.ID, sse.NewVoiceMutedEvent(openapi.VoiceMutedEvent{
			UserId:      userID,
			ChannelId:   ch.ID,
			Muted:       muted,
			Deafened:    deafened,
			ServerMuted: participant.IsServerMuted,
		}))
	}

	return openapi.DeafenVoice200JSONResponse{Success: true}, nil
}

func (h *Handler) ServerMuteVoice(ctx context.Context, request openapi.ServerMuteVoiceRequestObject) (openapi.ServerMuteVoiceResponseObject, error) {
	if h.voiceSFU == nil {
		return openapi.ServerMuteVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Voice channels are not enabled")}, nil
	}

	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ServerMuteVoice401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return openapi.ServerMuteVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Channel not found")}, nil
	}

	// Check admin permission
	membership, err := h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return openapi.ServerMuteVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a workspace member")}, nil
	}
	if !workspace.CanManageMembers(membership.Role) {
		return openapi.ServerMuteVoice403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Admin permission required")}, nil
	}

	targetUserID := request.Body.UserId

	// Prevent server-muting users with equal or higher role
	targetMembership, err := h.workspaceRepo.GetMembership(ctx, targetUserID, ch.WorkspaceID)
	if err != nil {
		return openapi.ServerMuteVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("Target user not a workspace member")}, nil
	}
	if workspace.RoleRank(membership.Role) <= workspace.RoleRank(targetMembership.Role) {
		return openapi.ServerMuteVoice403JSONResponse{ForbiddenJSONResponse: forbiddenResponse("Cannot server-mute a user with equal or higher role")}, nil
	}

	participant, err := h.voiceRepo.GetParticipant(ctx, ch.ID, targetUserID)
	if err != nil {
		return openapi.ServerMuteVoice404JSONResponse{NotFoundJSONResponse: notFoundResponse("User not in voice channel")}, nil
	}

	serverMuted := request.Body.Muted
	if err := h.voiceRepo.UpdateMuteState(ctx, ch.ID, targetUserID, participant.IsMuted, participant.IsDeafened, serverMuted); err != nil {
		return nil, err
	}

	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, ch.ID, sse.NewVoiceMutedEvent(openapi.VoiceMutedEvent{
			UserId:      targetUserID,
			ChannelId:   ch.ID,
			Muted:       participant.IsMuted,
			Deafened:    participant.IsDeafened,
			ServerMuted: serverMuted,
		}))
	}

	return openapi.ServerMuteVoice200JSONResponse{Success: true}, nil
}

func (h *Handler) ListVoiceParticipants(ctx context.Context, request openapi.ListVoiceParticipantsRequestObject) (openapi.ListVoiceParticipantsResponseObject, error) {
	if h.voiceSFU == nil {
		return openapi.ListVoiceParticipants404JSONResponse{NotFoundJSONResponse: notFoundResponse("Voice channels are not enabled")}, nil
	}

	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ListVoiceParticipants401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil
	}

	ch, err := h.channelRepo.GetByID(ctx, string(request.Id))
	if err != nil {
		return openapi.ListVoiceParticipants404JSONResponse{NotFoundJSONResponse: notFoundResponse("Channel not found")}, nil
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(ctx, userID, ch.WorkspaceID)
	if err != nil {
		return openapi.ListVoiceParticipants404JSONResponse{NotFoundJSONResponse: notFoundResponse("Not a workspace member")}, nil
	}

	participants, err := h.voiceRepo.GetParticipants(ctx, ch.ID)
	if err != nil {
		return nil, err
	}

	apiParticipants := make([]openapi.VoiceParticipant, len(participants))
	for i, p := range participants {
		apiParticipants[i] = voiceParticipantToAPI(p)
	}

	return openapi.ListVoiceParticipants200JSONResponse{
		Participants: apiParticipants,
	}, nil
}

func voiceParticipantToAPI(p voice.Participant) openapi.VoiceParticipant {
	vp := openapi.VoiceParticipant{
		Id:            p.ID,
		ChannelId:     p.ChannelID,
		UserId:        p.UserID,
		IsMuted:       p.IsMuted,
		IsDeafened:    p.IsDeafened,
		IsServerMuted: p.IsServerMuted,
		JoinedAt:      p.JoinedAt,
	}
	if p.DisplayName != "" {
		vp.DisplayName = &p.DisplayName
	}
	if p.AvatarURL != nil {
		vp.AvatarUrl = p.AvatarURL
	}
	return vp
}
