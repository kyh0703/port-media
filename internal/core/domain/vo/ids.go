package vo

import "github.com/google/uuid"

type SessionID string
type ConversationID string
type RoomID string
type ParticipantID string
type TrackID string

func NewSessionID() SessionID {
	return SessionID(uuid.NewString())
}

func NewRoomID() RoomID {
	return RoomID(uuid.NewString())
}

func NewParticipantID() ParticipantID {
	return ParticipantID(uuid.NewString())
}

func NewTrackID() TrackID {
	return TrackID(uuid.NewString())
}
