package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	domainrepo "github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/redis/go-redis/v9"
)

const mediaSessionStateKeyPrefix = "media:session:"

type RedisMediaSessionStateRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisMediaSessionStateRepository(client *redis.Client, cfg *configs.Config) domainrepo.MediaSessionStateRepository {
	return &RedisMediaSessionStateRepository{
		client: client,
		ttl:    cfg.Realtime.RoomIdleTimeout,
	}
}

func (r *RedisMediaSessionStateRepository) Save(ctx context.Context, state entity.MediaSessionState) error {
	body, err := json.Marshal(mediaSessionStatePayload{
		SessionID:             string(state.SessionID),
		ConversationID:        string(state.ConversationID),
		UserID:                state.UserID,
		RoomID:                string(state.RoomID),
		Status:                string(state.Status),
		ConnectionState:       string(state.ConnectionState),
		MediaState:            string(state.MediaState),
		OpenAIProviderCallID:  state.OpenAIProviderCallID,
		Participants:          state.Participants,
		ParticipantStates:     toMediaSessionParticipantPayloads(state.ParticipantStates),
		LastRealtimeEventType: state.LastRealtimeEventType,
		LastRealtimeEventAt:   state.LastRealtimeEventAt,
		RecentRealtimeEvents:  toRealtimeEventPayloads(state.RecentRealtimeEvents),
		StartedAt:             state.StartedAt,
		LastActiveAt:          state.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal media session state: %w", err)
	}

	if err := r.client.Set(ctx, mediaSessionStateKey(state.SessionID), body, r.ttl).Err(); err != nil {
		return fmt.Errorf("save media session state %s: %w", state.SessionID, err)
	}
	return nil
}

func (r *RedisMediaSessionStateRepository) FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.MediaSessionState, bool, error) {
	raw, err := r.client.Get(ctx, mediaSessionStateKey(sessionID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return entity.MediaSessionState{}, false, nil
		}
		return entity.MediaSessionState{}, false, fmt.Errorf("find media session state %s: %w", sessionID, err)
	}

	var payload mediaSessionStatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return entity.MediaSessionState{}, false, fmt.Errorf("decode media session state %s: %w", sessionID, err)
	}

	return entity.MediaSessionState{
		SessionID:             vo.SessionID(payload.SessionID),
		ConversationID:        vo.ConversationID(payload.ConversationID),
		UserID:                payload.UserID,
		RoomID:                vo.RoomID(payload.RoomID),
		Status:                vo.RoomStatus(payload.Status),
		ConnectionState:       vo.ConnectionState(payload.ConnectionState),
		MediaState:            vo.TrackState(payload.MediaState),
		OpenAIProviderCallID:  payload.OpenAIProviderCallID,
		Participants:          payload.Participants,
		ParticipantStates:     toMediaSessionParticipantStates(payload.ParticipantStates),
		LastRealtimeEventType: payload.LastRealtimeEventType,
		LastRealtimeEventAt:   payload.LastRealtimeEventAt,
		RecentRealtimeEvents:  toRealtimeEvents(payload.RecentRealtimeEvents),
		StartedAt:             payload.StartedAt,
		UpdatedAt:             payload.LastActiveAt,
	}, true, nil
}

func (r *RedisMediaSessionStateRepository) Delete(ctx context.Context, sessionID vo.SessionID) error {
	if err := r.client.Del(ctx, mediaSessionStateKey(sessionID)).Err(); err != nil {
		return fmt.Errorf("delete media session state %s: %w", sessionID, err)
	}
	return nil
}

func mediaSessionStateKey(sessionID vo.SessionID) string {
	return mediaSessionStateKeyPrefix + string(sessionID)
}

type mediaSessionStatePayload struct {
	SessionID             string                         `json:"session_id"`
	ConversationID        string                         `json:"conversation_id"`
	UserID                string                         `json:"user_id"`
	RoomID                string                         `json:"room_id"`
	Status                string                         `json:"status"`
	ConnectionState       string                         `json:"connection_state"`
	MediaState            string                         `json:"media_state"`
	OpenAIProviderCallID  string                         `json:"openai_provider_call_id"`
	Participants          int                            `json:"participants"`
	ParticipantStates     []mediaSessionParticipantState `json:"participant_states"`
	LastRealtimeEventType string                         `json:"last_realtime_event_type,omitempty"`
	LastRealtimeEventAt   time.Time                      `json:"last_realtime_event_at,omitempty"`
	RecentRealtimeEvents  []realtimeEventPayload         `json:"recent_realtime_events,omitempty"`
	StartedAt             time.Time                      `json:"started_at"`
	LastActiveAt          time.Time                      `json:"last_active_at"`
}

type realtimeEventPayload struct {
	Type string    `json:"type"`
	At   time.Time `json:"at"`
}

type mediaSessionParticipantState struct {
	ID              string `json:"id"`
	Role            string `json:"role"`
	AudioMode       string `json:"audio_mode"`
	ConnectionState string `json:"connection_state"`
	Tracks          int    `json:"tracks"`
}

func toMediaSessionParticipantPayloads(states []entity.MediaSessionParticipantState) []mediaSessionParticipantState {
	payloads := make([]mediaSessionParticipantState, 0, len(states))
	for _, state := range states {
		payloads = append(payloads, mediaSessionParticipantState{
			ID:              string(state.ID),
			Role:            string(state.Role),
			AudioMode:       state.AudioMode,
			ConnectionState: string(state.ConnectionState),
			Tracks:          state.Tracks,
		})
	}
	return payloads
}

func toMediaSessionParticipantStates(payloads []mediaSessionParticipantState) []entity.MediaSessionParticipantState {
	states := make([]entity.MediaSessionParticipantState, 0, len(payloads))
	for _, payload := range payloads {
		states = append(states, entity.MediaSessionParticipantState{
			ID:              vo.ParticipantID(payload.ID),
			Role:            vo.ParticipantRole(payload.Role),
			AudioMode:       payload.AudioMode,
			ConnectionState: vo.ConnectionState(payload.ConnectionState),
			Tracks:          payload.Tracks,
		})
	}
	return states
}

func toRealtimeEventPayloads(events []entity.RealtimeEvent) []realtimeEventPayload {
	payloads := make([]realtimeEventPayload, 0, len(events))
	for _, event := range events {
		payloads = append(payloads, realtimeEventPayload{
			Type: event.Type,
			At:   event.At,
		})
	}
	return payloads
}

func toRealtimeEvents(payloads []realtimeEventPayload) []entity.RealtimeEvent {
	events := make([]entity.RealtimeEvent, 0, len(payloads))
	for _, payload := range payloads {
		events = append(events, entity.RealtimeEvent{
			Type: payload.Type,
			At:   payload.At,
		})
	}
	return events
}
