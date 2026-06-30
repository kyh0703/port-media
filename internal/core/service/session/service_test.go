package session

import (
	"context"
	"testing"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
	sessionquery "github.com/kyh0703/portfoilo-media/internal/core/query/session"
	sessionio "github.com/kyh0703/portfoilo-media/internal/core/usecase/sessionio"
)

func TestServiceJoinsTokenParticipantWithoutCreatingProviderCall(t *testing.T) {
	records := newTestRecordRepository()
	runtime := newTestRoomRuntimeRepository()
	states := &testStateRepository{}
	media := &testMediaGateway{}
	svc := NewService(records, runtime, states, media)

	created, err := svc.CreateSession(context.Background(), sessionio.CreateSessionRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	joined, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:       created.SessionID,
		ConversationID:  "conversation-1",
		ParticipantID:   "agent-1",
		ParticipantRole: "agent",
		UserID:          "user-1",
		SDP:             "offer-sdp",
		AudioMode:       sessionio.AudioModePublisher,
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	if joined.ParticipantID != "agent-1" {
		t.Fatalf("ParticipantID = %q, want agent-1", joined.ParticipantID)
	}
	if media.accepted.Role != vo.ParticipantRoleAgent {
		t.Fatalf("media role = %q, want agent", media.accepted.Role)
	}
	if media.accepted.ParticipantID != vo.ParticipantID("agent-1") {
		t.Fatalf("media participant id = %q, want agent-1", media.accepted.ParticipantID)
	}
	if media.createOfferCalls != 0 {
		t.Fatalf("CreateOffer calls = %d, want 0", media.createOfferCalls)
	}
}

type testRecordRepository struct {
	records map[vo.SessionID]entity.MediaSessionRecord
}

func newTestRecordRepository() *testRecordRepository {
	return &testRecordRepository{records: map[vo.SessionID]entity.MediaSessionRecord{}}
}

func (r *testRecordRepository) Save(_ context.Context, record entity.MediaSessionRecord) error {
	r.records[record.SessionID] = record
	return nil
}

func (r *testRecordRepository) FindBySessionID(_ context.Context, sessionID vo.SessionID) (entity.MediaSessionRecord, bool, error) {
	record, ok := r.records[sessionID]
	return record, ok, nil
}

func (r *testRecordRepository) Delete(_ context.Context, roomID vo.RoomID) error {
	for sessionID, record := range r.records {
		if record.ID == roomID {
			delete(r.records, sessionID)
		}
	}
	return nil
}

type testRoomRuntimeRepository struct {
	rooms map[vo.SessionID]entity.Room
}

func newTestRoomRuntimeRepository() *testRoomRuntimeRepository {
	return &testRoomRuntimeRepository{rooms: map[vo.SessionID]entity.Room{}}
}

func (r *testRoomRuntimeRepository) Save(_ context.Context, room entity.Room) error {
	r.rooms[room.SessionID] = room.Clone()
	return nil
}

func (r *testRoomRuntimeRepository) FindBySessionID(_ context.Context, sessionID vo.SessionID) (entity.Room, bool, error) {
	room, ok := r.rooms[sessionID]
	return room.Clone(), ok, nil
}

func (r *testRoomRuntimeRepository) List(_ context.Context) ([]entity.Room, error) {
	rooms := make([]entity.Room, 0, len(r.rooms))
	for _, room := range r.rooms {
		rooms = append(rooms, room.Clone())
	}
	return rooms, nil
}

func (r *testRoomRuntimeRepository) Delete(_ context.Context, roomID vo.RoomID) error {
	for sessionID, room := range r.rooms {
		if room.ID == roomID {
			delete(r.rooms, sessionID)
		}
	}
	return nil
}

type testStateRepository struct {
	state sessionquery.MediaSessionState
	found bool
}

func (r *testStateRepository) Save(_ context.Context, state sessionquery.MediaSessionState) error {
	r.state = state
	r.found = true
	return nil
}

func (r *testStateRepository) FindBySessionID(_ context.Context, _ vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
	return r.state, r.found, nil
}

func (r *testStateRepository) Delete(_ context.Context, _ vo.SessionID) error {
	r.found = false
	return nil
}

type testMediaGateway struct {
	accepted         port.OfferInput
	createOfferCalls int
}

func (g *testMediaGateway) AcceptOffer(_ context.Context, input port.OfferInput) (*port.Peer, error) {
	g.accepted = input
	return &port.Peer{
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
		Role:          input.Role,
		AnswerSDP:     "answer-sdp",
	}, nil
}

func (g *testMediaGateway) CreateOffer(_ context.Context, input port.CreateOfferInput) (*port.PeerOffer, error) {
	g.createOfferCalls++
	return &port.PeerOffer{
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
		Role:          input.Role,
		SDPOffer:      "offer-sdp",
	}, nil
}

func (g *testMediaGateway) ApplyAnswer(_ context.Context, offer *port.PeerOffer, _ string) (*port.Peer, error) {
	return &port.Peer{
		SessionID:     offer.SessionID,
		ParticipantID: offer.ParticipantID,
		Role:          offer.Role,
	}, nil
}

func (g *testMediaGateway) CloseSession(_ context.Context, _ vo.SessionID) error {
	return nil
}

func (g *testMediaGateway) CloseParticipant(_ context.Context, _ vo.SessionID, _ vo.ParticipantID) error {
	return nil
}
