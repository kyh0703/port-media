package entity

import (
	"errors"
	"testing"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

func TestRoomJoinParticipantActivatesJoinableRoom(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	room := NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now)
	client := NewParticipant(vo.ParticipantID("client-1"), vo.ParticipantRoleUser, now)

	if err := room.JoinParticipant(client, now); err != nil {
		t.Fatalf("JoinParticipant() error = %v", err)
	}

	if room.Status != vo.RoomStatusActive {
		t.Fatalf("room status = %q, want active", room.Status)
	}
	if _, found := room.Participant(vo.ParticipantID("client-1")); !found {
		t.Fatal("client participant not found")
	}
}

func TestRoomRejectsJoinWhenClosed(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	room := NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now)
	room.Close(now)
	client := NewParticipant(vo.ParticipantID("client-1"), vo.ParticipantRoleUser, now)

	err := room.JoinParticipant(client, now)
	if !errors.Is(err, ErrRoomNotJoinable) {
		t.Fatalf("JoinParticipant() error = %v, want %v", err, ErrRoomNotJoinable)
	}
}

func TestRoomAllowsUserAndAgentParticipants(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	room := NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now)
	user := NewParticipant(vo.ParticipantID("user-1"), vo.ParticipantRoleUser, now)
	agent := NewParticipant(vo.ParticipantID("agent-1"), vo.ParticipantRoleAgent, now)
	if err := room.JoinParticipant(user, now); err != nil {
		t.Fatalf("JoinParticipant(user) error = %v", err)
	}
	if err := room.JoinParticipant(agent, now); err != nil {
		t.Fatalf("JoinParticipant(agent) error = %v", err)
	}
	if room.ParticipantCount() != 2 {
		t.Fatalf("ParticipantCount() = %d, want 2", room.ParticipantCount())
	}
}

func TestRoomParticipantsReturnsSnapshot(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	room := NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now)
	client := NewParticipant(vo.ParticipantID("client-1"), vo.ParticipantRoleUser, now)
	client.AddTrack(NewTrack(vo.TrackID("track-1"), vo.TrackKindAudio, now), now)
	if err := room.JoinParticipant(client, now); err != nil {
		t.Fatalf("JoinParticipant() error = %v", err)
	}

	participants := room.Participants()
	clientSnapshot := participants[vo.ParticipantID("client-1")]
	clientSnapshot.Tracks[vo.TrackID("track-2")] = NewTrack(vo.TrackID("track-2"), vo.TrackKindAudio, now)
	participants[vo.ParticipantID("client-1")] = clientSnapshot
	delete(participants, vo.ParticipantID("client-1"))

	participant, found := room.Participant(vo.ParticipantID("client-1"))
	if !found {
		t.Fatal("participant missing after mutating snapshot")
	}
	if _, found := participant.Tracks[vo.TrackID("track-2")]; found {
		t.Fatal("snapshot track mutation changed room participant")
	}
}
