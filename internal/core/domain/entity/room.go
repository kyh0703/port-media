package entity

import (
	"errors"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

var (
	ErrRoomNotJoinable          = errors.New("room is not joinable")
	ErrParticipantRoleMismatch  = errors.New("participant role mismatch")
	ErrParticipantAlreadyJoined = errors.New("participant already joined")
	ErrOpenAIAgentAlreadyJoined = errors.New("openai agent already joined")
)

type Room struct {
	ID                    vo.RoomID
	SessionID             vo.SessionID
	ConversationID        vo.ConversationID
	UserID                string
	Status                vo.RoomStatus
	Participants          map[vo.ParticipantID]Participant
	LastRealtimeEventType string
	LastRealtimeEventAt   time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func NewRoom(id vo.RoomID, sessionID vo.SessionID, conversationID vo.ConversationID, now time.Time) Room {
	return Room{
		ID:             id,
		SessionID:      sessionID,
		ConversationID: conversationID,
		Status:         vo.RoomStatusCreated,
		Participants:   make(map[vo.ParticipantID]Participant),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (r *Room) SetUserID(userID string, now time.Time) {
	r.UserID = userID
	r.UpdatedAt = now
}

func (r Room) CanJoinParticipants() bool {
	return r.Status == vo.RoomStatusCreated || r.Status == vo.RoomStatusActive
}

func (r *Room) JoinClient(participant Participant, now time.Time) error {
	if participant.Role != vo.ParticipantRoleClient {
		return ErrParticipantRoleMismatch
	}
	return r.addParticipant(participant, now)
}

func (r *Room) AttachOpenAIAgent(participant Participant, now time.Time) error {
	if participant.Role != vo.ParticipantRoleOpenAIAgent {
		return ErrParticipantRoleMismatch
	}
	if r.HasOpenAIAgent() {
		return ErrOpenAIAgentAlreadyJoined
	}
	return r.addParticipant(participant, now)
}

func (r *Room) addParticipant(participant Participant, now time.Time) error {
	if !r.CanJoinParticipants() {
		return ErrRoomNotJoinable
	}
	if _, exists := r.Participants[participant.ID]; exists {
		return ErrParticipantAlreadyJoined
	}
	r.Participants[participant.ID] = participant
	r.UpdatedAt = now
	if r.Status == vo.RoomStatusCreated {
		r.Status = vo.RoomStatusActive
	}
	return nil
}

func (r Room) Participant(participantID vo.ParticipantID) (Participant, bool) {
	participant, ok := r.Participants[participantID]
	return participant, ok
}

func (r *Room) RemoveParticipant(participantID vo.ParticipantID, now time.Time) (Participant, bool) {
	participant, ok := r.Participants[participantID]
	if !ok {
		return Participant{}, false
	}
	delete(r.Participants, participantID)
	r.UpdatedAt = now
	return participant, true
}

func (r *Room) Touch(now time.Time) {
	r.UpdatedAt = now
}

func (r *Room) RecordRealtimeEvent(eventType string, now time.Time) {
	r.LastRealtimeEventType = eventType
	r.LastRealtimeEventAt = now
	r.UpdatedAt = now
}

func (r Room) HasOpenAIAgent() bool {
	return r.hasParticipantRole(vo.ParticipantRoleOpenAIAgent)
}

func (r Room) hasParticipantRole(role vo.ParticipantRole) bool {
	for _, participant := range r.Participants {
		if participant.Role == role {
			return true
		}
	}
	return false
}

func (r *Room) UpdateParticipantState(participantID vo.ParticipantID, state vo.ConnectionState, now time.Time) bool {
	participant, ok := r.Participants[participantID]
	if !ok {
		return false
	}

	participant.SetState(state, now)
	r.Participants[participantID] = participant
	r.UpdatedAt = now
	return true
}

func (r *Room) UpdateParticipantTrackState(participantID vo.ParticipantID, kind vo.TrackKind, state vo.TrackState, now time.Time) bool {
	participant, ok := r.Participants[participantID]
	if !ok {
		return false
	}
	if !participant.UpdateTrackState(kind, state, now) {
		return false
	}

	r.Participants[participantID] = participant
	r.UpdatedAt = now
	return true
}

func (r *Room) Close(now time.Time) {
	r.Status = vo.RoomStatusClosed
	r.UpdatedAt = now
	for id, participant := range r.Participants {
		participant.SetState(vo.ConnectionStateClosed, now)
		for trackID, track := range participant.Tracks {
			track.SetState(vo.TrackStateEnded, now)
			participant.Tracks[trackID] = track
		}
		r.Participants[id] = participant
	}
}

func (r *Room) Fail(now time.Time) {
	r.Status = vo.RoomStatusFailed
	r.UpdatedAt = now
	for id, participant := range r.Participants {
		participant.SetState(vo.ConnectionStateFailed, now)
		for trackID, track := range participant.Tracks {
			track.SetState(vo.TrackStateFailed, now)
			participant.Tracks[trackID] = track
		}
		r.Participants[id] = participant
	}
}
