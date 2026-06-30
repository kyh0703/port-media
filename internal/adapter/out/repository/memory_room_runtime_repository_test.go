package repository

import (
	"context"
	"testing"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

func TestMemoryRoomRuntimeRepositoryCopiesRooms(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryRoomRuntimeRepository()
	room := entity.NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now)
	client := entity.NewParticipant(vo.ParticipantID("client-1"), vo.ParticipantRoleUser, now)
	client.AddTrack(entity.NewTrack(vo.TrackID("track-1"), vo.TrackKindAudio, now), now)
	if err := room.JoinParticipant(client, now); err != nil {
		t.Fatalf("JoinParticipant() error = %v", err)
	}

	if err := repo.Save(ctx, room); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	room.RemoveParticipant(vo.ParticipantID("client-1"), now)
	found, ok, err := repo.FindBySessionID(ctx, vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !ok {
		t.Fatal("FindBySessionID() did not find room")
	}
	if found.ParticipantCount() != 1 {
		t.Fatalf("saved room participant count = %d, want 1", found.ParticipantCount())
	}

	participant, ok := found.Participant(vo.ParticipantID("client-1"))
	if !ok {
		t.Fatal("saved room participant missing")
	}
	participant.Tracks[vo.TrackID("track-2")] = entity.NewTrack(vo.TrackID("track-2"), vo.TrackKindAudio, now)

	again, ok, err := repo.FindBySessionID(ctx, vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() second error = %v", err)
	}
	if !ok {
		t.Fatal("FindBySessionID() second did not find room")
	}
	againParticipant, ok := again.Participant(vo.ParticipantID("client-1"))
	if !ok {
		t.Fatal("saved room participant missing after mutating found copy")
	}
	if _, exists := againParticipant.Tracks[vo.TrackID("track-2")]; exists {
		t.Fatal("mutating found room changed stored room")
	}

	listed, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	listedParticipant, ok := listed[0].Participant(vo.ParticipantID("client-1"))
	if !ok {
		t.Fatal("listed room participant missing")
	}
	listedParticipant.Tracks[vo.TrackID("track-3")] = entity.NewTrack(vo.TrackID("track-3"), vo.TrackKindAudio, now)

	finalRoom, ok, err := repo.FindBySessionID(ctx, vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() final error = %v", err)
	}
	if !ok {
		t.Fatal("FindBySessionID() final did not find room")
	}
	finalParticipant, ok := finalRoom.Participant(vo.ParticipantID("client-1"))
	if !ok {
		t.Fatal("saved room participant missing after mutating list copy")
	}
	if _, exists := finalParticipant.Tracks[vo.TrackID("track-3")]; exists {
		t.Fatal("mutating listed room changed stored room")
	}
}
