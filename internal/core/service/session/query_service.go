package session

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	sessionio "github.com/kyh0703/portfoilo-media/internal/core/usecase/sessionio"
)

func (s *service) GetSessionStatus(ctx context.Context, req sessionio.GetSessionStatusRequest) (sessionio.GetSessionStatusResult, bool, error) {
	state, found, err := s.states.FindBySessionID(ctx, vo.SessionID(req.SessionID))
	if err != nil || !found {
		return sessionio.GetSessionStatusResult{}, found, err
	}

	return sessionio.GetSessionStatusResult{
		SessionID:             string(state.SessionID),
		ConversationID:        string(state.ConversationID),
		UserID:                state.UserID,
		RoomID:                string(state.RoomID),
		Status:                string(state.Status),
		ConnectionState:       string(state.ConnectionState),
		MediaState:            string(state.MediaState),
		Participants:          state.Participants,
		ParticipantStates:     participantStateResults(state.ParticipantStates),
		LastRealtimeEventType: state.LastRealtimeEventType,
		LastRealtimeEventAt:   state.LastRealtimeEventAt,
		RecentRealtimeEvents:  realtimeEventResults(state.RecentRealtimeEvents),
		StartedAt:             state.StartedAt,
		LastActiveAt:          state.UpdatedAt,
	}, true, nil
}

func (s *service) GetRuntimeStats(ctx context.Context) (sessionio.RuntimeStatsResponse, error) {
	rooms, err := s.runtime.List(ctx)
	if err != nil {
		return sessionio.RuntimeStatsResponse{}, err
	}

	return s.stats.Build(rooms), nil
}

func (s *service) GetHealth(ctx context.Context) error {
	_ = ctx
	return nil
}
