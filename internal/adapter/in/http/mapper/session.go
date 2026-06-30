package mapper

import (
	"time"

	httpdto "github.com/kyh0703/portfoilo-media/internal/adapter/in/http/dto"
	sessionio "github.com/kyh0703/portfoilo-media/internal/core/usecase/sessionio"
)

func ToCreateSessionResponse(result sessionio.CreateSessionResponse) httpdto.CreateSessionResponse {
	return httpdto.CreateSessionResponse{
		SessionID:      result.SessionID,
		ConversationID: result.ConversationID,
		RoomID:         result.RoomID,
		Status:         result.Status,
	}
}

func ToLeaveParticipantResponse(result sessionio.LeaveParticipantResponse) httpdto.LeaveParticipantResponse {
	return httpdto.LeaveParticipantResponse{
		SessionID:     result.SessionID,
		RoomID:        result.RoomID,
		ParticipantID: result.ParticipantID,
		Status:        result.Status,
	}
}

func ToEndSessionResponse(result sessionio.EndSessionResponse) httpdto.EndSessionResponse {
	return httpdto.EndSessionResponse{
		SessionID: result.SessionID,
		RoomID:    result.RoomID,
		Status:    result.Status,
	}
}

func ToGetSessionStatusResponse(result sessionio.GetSessionStatusResult) httpdto.GetSessionStatusResponse {
	return httpdto.GetSessionStatusResponse{
		SessionID:            result.SessionID,
		ConversationID:       result.ConversationID,
		UserID:               result.UserID,
		RoomID:               result.RoomID,
		Status:               result.Status,
		ConnectionState:      result.ConnectionState,
		MediaState:           result.MediaState,
		Participants:         result.Participants,
		ParticipantStates:    toParticipantStateResponses(result.ParticipantStates),
		LastRuntimeEventType: result.LastRuntimeEventType,
		LastRuntimeEventAt:   formatOptionalTime(result.LastRuntimeEventAt),
		RecentRuntimeEvents:  toRuntimeEventResponses(result.RecentRuntimeEvents),
		StartedAt:            formatOptionalTime(result.StartedAt),
		LastActiveAt:         formatOptionalTime(result.LastActiveAt),
	}
}

func toParticipantStateResponses(states []sessionio.ParticipantStateResult) []httpdto.ParticipantStateResponse {
	responses := make([]httpdto.ParticipantStateResponse, 0, len(states))
	for _, state := range states {
		responses = append(responses, httpdto.ParticipantStateResponse{
			ID:              state.ID,
			Role:            state.Role,
			AudioMode:       state.AudioMode,
			ConnectionState: state.ConnectionState,
			Tracks:          state.Tracks,
		})
	}
	return responses
}

func toRuntimeEventResponses(events []sessionio.RuntimeEventResult) []httpdto.RuntimeEventResponse {
	responses := make([]httpdto.RuntimeEventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, httpdto.RuntimeEventResponse{
			Type: event.Type,
			At:   formatOptionalTime(event.At),
		})
	}
	return responses
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
