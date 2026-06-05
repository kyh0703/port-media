package entity

import (
	"errors"
	"testing"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

func TestRoomJoinClientActivatesJoinableRoom(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	room := NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now)
	client := NewParticipant(vo.ParticipantID("client-1"), vo.ParticipantRoleClient, now)

	if err := room.JoinClient(client, now); err != nil {
		t.Fatalf("JoinClient() error = %v", err)
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
	client := NewParticipant(vo.ParticipantID("client-1"), vo.ParticipantRoleClient, now)

	err := room.JoinClient(client, now)
	if !errors.Is(err, ErrRoomNotJoinable) {
		t.Fatalf("JoinClient() error = %v, want %v", err, ErrRoomNotJoinable)
	}
}

func TestRoomRejectsDuplicateOpenAIAgent(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	room := NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now)
	agent := NewParticipant(vo.ParticipantID("agent-1"), vo.ParticipantRoleOpenAIAgent, now)
	if err := room.AttachOpenAIAgent(agent, now); err != nil {
		t.Fatalf("AttachOpenAIAgent() first error = %v", err)
	}

	otherAgent := NewParticipant(vo.ParticipantID("agent-2"), vo.ParticipantRoleOpenAIAgent, now)
	err := room.AttachOpenAIAgent(otherAgent, now)
	if !errors.Is(err, ErrOpenAIAgentAlreadyJoined) {
		t.Fatalf("AttachOpenAIAgent() error = %v, want %v", err, ErrOpenAIAgentAlreadyJoined)
	}
}
