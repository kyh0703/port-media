package session

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
)

func (s *service) GetSessionStatus(ctx context.Context, req sessiondto.GetSessionStatusRequest) (sessiondto.GetSessionStatusResult, bool, error) {
	state, found, err := s.states.FindBySessionID(ctx, vo.SessionID(req.SessionID))
	if err != nil || !found {
		return sessiondto.GetSessionStatusResult{}, found, err
	}

	return sessiondto.GetSessionStatusResult{
		SessionID:             string(state.SessionID),
		ConversationID:        string(state.ConversationID),
		UserID:                state.UserID,
		RoomID:                string(state.RoomID),
		Status:                string(state.Status),
		ConnectionState:       string(state.ConnectionState),
		MediaState:            string(state.MediaState),
		OpenAIProviderCallID:  state.OpenAIProviderCallID,
		Participants:          state.Participants,
		ParticipantStates:     participantStateResults(state.ParticipantStates),
		LastRealtimeEventType: state.LastRealtimeEventType,
		LastRealtimeEventAt:   state.LastRealtimeEventAt,
		RecentRealtimeEvents:  realtimeEventResults(state.RecentRealtimeEvents),
		StartedAt:             state.StartedAt,
		LastActiveAt:          state.UpdatedAt,
	}, true, nil
}

func (s *service) GetRuntimeStats(ctx context.Context) (sessiondto.RuntimeStatsResponse, error) {
	rooms, err := s.runtime.List(ctx)
	if err != nil {
		return sessiondto.RuntimeStatsResponse{}, err
	}

	return s.stats.Build(rooms), nil
}

func (s *service) GetHealth(ctx context.Context) error {
	_ = ctx
	return nil
}
