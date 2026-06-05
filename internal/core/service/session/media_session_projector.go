package session

import (
	"sort"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	sessionreadmodel "github.com/kyh0703/portfoilo-media/internal/core/readmodel/session"
)

type mediaSessionProjector struct {
	realtimeEventHistoryLimit int
}

func (p mediaSessionProjector) Project(room entity.Room, userID string, now time.Time) sessionreadmodel.MediaSessionState {
	return sessionreadmodel.MediaSessionState{
		SessionID:            room.SessionID,
		ConversationID:       room.ConversationID,
		UserID:               coalesceUserID(userID, room.UserID),
		RoomID:               room.ID,
		Status:               room.Status,
		ConnectionState:      roomConnectionState(room),
		MediaState:           roomMediaState(room),
		OpenAIProviderCallID: openAIProviderCallID(room),
		Participants:         len(room.Participants),
		ParticipantStates:    mediaSessionParticipantStates(room),
		StartedAt:            room.CreatedAt,
		UpdatedAt:            now,
	}
}

func (p mediaSessionProjector) ProjectWithRealtimeEvent(
	room entity.Room,
	userID string,
	now time.Time,
	eventType string,
	recentEvents []sessionreadmodel.RealtimeEvent,
) sessionreadmodel.MediaSessionState {
	state := p.Project(room, userID, now)
	state.LastRealtimeEventType = eventType
	state.LastRealtimeEventAt = now
	state.RecentRealtimeEvents = appendRealtimeEvent(recentEvents, sessionreadmodel.RealtimeEvent{
		Type: eventType,
		At:   now,
	}, p.realtimeEventHistoryLimit)
	return state
}

func coalesceUserID(userID string, fallback string) string {
	if userID != "" {
		return userID
	}
	return fallback
}

func roomConnectionState(room entity.Room) vo.ConnectionState {
	if room.Status == vo.RoomStatusFailed {
		return vo.ConnectionStateFailed
	}
	if room.Status == vo.RoomStatusClosed {
		return vo.ConnectionStateClosed
	}
	if len(room.Participants) == 0 {
		return vo.ConnectionStateNew
	}

	hasConnected := false
	hasConnecting := false
	hasDisconnected := false
	for _, participant := range room.Participants {
		switch participant.State {
		case vo.ConnectionStateFailed:
			if isCriticalParticipant(participant.Role) {
				return vo.ConnectionStateFailed
			}
			hasDisconnected = true
		case vo.ConnectionStateDisconnected:
			hasDisconnected = true
		case vo.ConnectionStateConnected:
			hasConnected = true
		case vo.ConnectionStateConnecting, vo.ConnectionStateNew:
			hasConnecting = true
		default:
			hasConnecting = true
		}
	}
	if hasConnected {
		return vo.ConnectionStateConnected
	}
	if hasConnecting {
		return vo.ConnectionStateConnecting
	}
	if hasDisconnected {
		return vo.ConnectionStateDisconnected
	}
	return vo.ConnectionStateNew
}

func roomMediaState(room entity.Room) vo.TrackState {
	if room.Status == vo.RoomStatusFailed {
		return vo.TrackStateFailed
	}
	if room.Status == vo.RoomStatusClosed {
		return vo.TrackStateEnded
	}

	hasTrack := false
	hasPending := false
	hasFailed := false
	hasEnded := false
	for _, participant := range room.Participants {
		for _, track := range participant.Tracks {
			if track.Kind != vo.TrackKindAudio {
				continue
			}
			hasTrack = true
			switch track.State {
			case vo.TrackStateFailed:
				if isCriticalParticipant(participant.Role) {
					return vo.TrackStateFailed
				}
				hasFailed = true
			case vo.TrackStateActive:
				return vo.TrackStateActive
			case vo.TrackStatePending:
				hasPending = true
			case vo.TrackStateEnded:
				hasEnded = true
			default:
				hasPending = true
			}
		}
	}
	if !hasTrack || hasPending {
		return vo.TrackStatePending
	}
	if hasFailed {
		return vo.TrackStateFailed
	}
	if hasEnded {
		return vo.TrackStateEnded
	}
	return vo.TrackStatePending
}

func countTracks(room entity.Room) int {
	var count int
	for _, participant := range room.Participants {
		count += len(participant.Tracks)
	}
	return count
}

func mediaSessionParticipantStates(room entity.Room) []sessionreadmodel.MediaSessionParticipantState {
	states := make([]sessionreadmodel.MediaSessionParticipantState, 0, len(room.Participants))
	for _, participant := range room.Participants {
		states = append(states, sessionreadmodel.MediaSessionParticipantState{
			ID:              participant.ID,
			Role:            participant.Role,
			AudioMode:       participantAudioMode(participant),
			ConnectionState: participant.State,
			Tracks:          len(participant.Tracks),
		})
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].ID < states[j].ID
	})
	return states
}

func participantStateResponses(states []sessionreadmodel.MediaSessionParticipantState) []sessiondto.ParticipantStateResponse {
	responses := make([]sessiondto.ParticipantStateResponse, 0, len(states))
	for _, state := range states {
		responses = append(responses, sessiondto.ParticipantStateResponse{
			ID:              string(state.ID),
			Role:            string(state.Role),
			AudioMode:       state.AudioMode,
			ConnectionState: string(state.ConnectionState),
			Tracks:          state.Tracks,
		})
	}
	return responses
}

func realtimeEventResponses(events []sessionreadmodel.RealtimeEvent) []sessiondto.RealtimeEventResponse {
	responses := make([]sessiondto.RealtimeEventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, sessiondto.RealtimeEventResponse{
			Type: event.Type,
			At:   formatOptionalTime(event.At),
		})
	}
	return responses
}

func countClientAudioModes(room entity.Room) (publishers int, listeners int) {
	for _, participant := range room.Participants {
		if participant.Role != vo.ParticipantRoleClient {
			continue
		}
		if participant.PublishAudio {
			publishers++
			continue
		}
		listeners++
	}
	return publishers, listeners
}

func participantAudioMode(participant entity.Participant) string {
	if participant.Role != vo.ParticipantRoleClient {
		return ""
	}
	if participant.PublishAudio {
		return string(sessiondto.AudioModePublisher)
	}
	return string(sessiondto.AudioModeListener)
}
