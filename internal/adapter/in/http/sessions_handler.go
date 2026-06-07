package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	coreport "github.com/kyh0703/portfoilo-media/internal/core/port"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	"github.com/kyh0703/portfoilo-media/internal/pkg/exception"
	"github.com/kyh0703/portfoilo-media/internal/pkg/response"
	"go.uber.org/fx"
)

type SessionsHandler struct {
	createSession usecase.CreateSessionUsecase
	joinSession   usecase.JoinSessionUsecase
	leave         usecase.LeaveParticipantUsecase
	end           usecase.EndSessionUsecase
	status        usecase.GetSessionStatusQuery
	tokenVerifier coreport.MediaTokenVerifier
}

type SessionsHandlerParams struct {
	fx.In

	CreateSession usecase.CreateSessionUsecase
	JoinSession   usecase.JoinSessionUsecase
	Leave         usecase.LeaveParticipantUsecase
	End           usecase.EndSessionUsecase
	Status        usecase.GetSessionStatusQuery
	TokenVerifier coreport.MediaTokenVerifier
}

func NewSessionsHandler(params SessionsHandlerParams) *SessionsHandler {
	return &SessionsHandler{
		createSession: params.CreateSession,
		joinSession:   params.JoinSession,
		leave:         params.Leave,
		end:           params.End,
		status:        params.Status,
		tokenVerifier: params.TokenVerifier,
	}
}

func (h *SessionsHandler) Table() []Mapper {
	return []Mapper{
		{Method: http.MethodPost, Path: "/sessions", Handler: h.Create},
		{Method: http.MethodGet, Path: "/sessions/{sessionId}/status", Handler: h.GetStatus},
		{Method: http.MethodPost, Path: "/sessions/{sessionId}/join", Handler: h.Join},
		{Method: http.MethodPost, Path: "/sessions/{sessionId}/participants/{participantId}/leave", Handler: h.LeaveParticipant},
		{Method: http.MethodPost, Path: "/sessions/{sessionId}/end", Handler: h.End},
	}
}

func (h *SessionsHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return exception.New(exception.CodeBadRequest, "invalid session request", http.StatusBadRequest)
	}

	res, err := h.createSession.CreateSession(r.Context(), sessiondto.CreateSessionRequest{
		SessionID:      req.SessionID,
		ConversationID: req.ConversationID,
	})
	if err != nil {
		return err
	}

	return response.WriteJSON(w, http.StatusCreated, response.Created(toCreateSessionResponse(res)))
}

func (h *SessionsHandler) GetStatus(w http.ResponseWriter, r *http.Request) error {
	claims, err := h.verifySessionToken(r)
	if err != nil {
		return err
	}

	res, found, err := h.status.GetSessionStatus(r.Context(), sessiondto.GetSessionStatusRequest{
		SessionID: claims.SessionID,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrSessionNotFound) {
			return exception.New(exception.CodeNotFound, "media session not found", http.StatusNotFound)
		}
		return err
	}
	if !found {
		return exception.New(exception.CodeNotFound, "media session status not found", http.StatusNotFound)
	}

	return response.WriteJSON(w, http.StatusOK, response.OK(toGetSessionStatusResponse(res)))
}

func (h *SessionsHandler) Join(w http.ResponseWriter, r *http.Request) error {
	claims, err := h.verifySessionToken(r)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	sdp := string(body)
	if sdp == "" {
		return exception.New(exception.CodeBadRequest, "empty join SDP", http.StatusBadRequest)
	}
	audioMode, err := parseAudioMode(r.URL.Query().Get("mode"))
	if err != nil {
		return exception.New(exception.CodeBadRequest, "invalid join mode", http.StatusBadRequest)
	}

	res, err := h.joinSession.JoinSession(r.Context(), sessiondto.JoinSessionCommand{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
		SDP:            sdp,
		AudioMode:      audioMode,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrSessionNotJoinable) {
			return exception.New(exception.CodeConflict, "media session is not joinable", http.StatusConflict)
		}
		return err
	}

	w.Header().Set("Content-Type", "application/sdp")
	w.Header().Set("X-Room-Id", res.RoomID)
	w.Header().Set("X-Participant-Id", res.ParticipantID)
	_, err = io.WriteString(w, res.SDPAnswer)
	return err
}

func (h *SessionsHandler) LeaveParticipant(w http.ResponseWriter, r *http.Request) error {
	claims, err := h.verifySessionToken(r)
	if err != nil {
		return err
	}

	res, err := h.leave.LeaveParticipant(r.Context(), sessiondto.LeaveParticipantRequest{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
		ParticipantID:  r.PathValue("participantId"),
	})
	if err != nil {
		return err
	}

	return response.WriteJSON(w, http.StatusOK, response.OK(toLeaveParticipantResponse(res)))
}

func (h *SessionsHandler) End(w http.ResponseWriter, r *http.Request) error {
	claims, err := h.verifySessionToken(r)
	if err != nil {
		return err
	}

	res, err := h.end.EndSession(r.Context(), sessiondto.EndSessionRequest{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
	})
	if err != nil {
		return err
	}

	return response.WriteJSON(w, http.StatusOK, response.OK(toEndSessionResponse(res)))
}

func (h *SessionsHandler) verifySessionToken(r *http.Request) (coreport.MediaToken, error) {
	claims, err := h.tokenVerifier.Verify(r.Context(), readBearerToken(r.Header.Get("Authorization")))
	if err != nil {
		return coreport.MediaToken{}, exception.New(exception.CodeUnauthorized, "invalid media token", http.StatusUnauthorized)
	}
	if claims.SessionID != r.PathValue("sessionId") {
		return coreport.MediaToken{}, exception.New(exception.CodeUnauthorized, "media token session mismatch", http.StatusUnauthorized)
	}

	return claims, nil
}

func readBearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func parseAudioMode(mode string) (sessiondto.AudioMode, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "publisher", "speaker":
		return sessiondto.AudioModePublisher, nil
	case "listener", "listen_only", "listen-only":
		return sessiondto.AudioModeListener, nil
	default:
		return "", exception.New(exception.CodeBadRequest, "invalid join mode", http.StatusBadRequest)
	}
}

type getSessionStatusResponse struct {
	SessionID             string                     `json:"session_id"`
	ConversationID        string                     `json:"conversation_id"`
	UserID                string                     `json:"user_id"`
	RoomID                string                     `json:"room_id"`
	Status                string                     `json:"status"`
	ConnectionState       string                     `json:"connection_state"`
	MediaState            string                     `json:"media_state"`
	OpenAIProviderCallID  string                     `json:"openai_provider_call_id"`
	Participants          int                        `json:"participants"`
	ParticipantStates     []participantStateResponse `json:"participant_states"`
	LastRealtimeEventType string                     `json:"last_realtime_event_type"`
	LastRealtimeEventAt   string                     `json:"last_realtime_event_at"`
	RecentRealtimeEvents  []realtimeEventResponse    `json:"recent_realtime_events"`
	StartedAt             string                     `json:"started_at"`
	LastActiveAt          string                     `json:"last_active_at"`
}

type participantStateResponse struct {
	ID              string `json:"id"`
	Role            string `json:"role"`
	AudioMode       string `json:"audio_mode"`
	ConnectionState string `json:"connection_state"`
	Tracks          int    `json:"tracks"`
}

type realtimeEventResponse struct {
	Type string `json:"type"`
	At   string `json:"at"`
}

type createSessionRequest struct {
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
}

type createSessionResponse struct {
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
	RoomID         string `json:"room_id"`
	Status         string `json:"status"`
}

type leaveParticipantResponse struct {
	SessionID     string `json:"session_id"`
	RoomID        string `json:"room_id"`
	ParticipantID string `json:"participant_id"`
	Status        string `json:"status"`
}

type endSessionResponse struct {
	SessionID string `json:"session_id"`
	RoomID    string `json:"room_id"`
	Status    string `json:"status"`
}

func toCreateSessionResponse(result sessiondto.CreateSessionResponse) createSessionResponse {
	return createSessionResponse{
		SessionID:      result.SessionID,
		ConversationID: result.ConversationID,
		RoomID:         result.RoomID,
		Status:         result.Status,
	}
}

func toLeaveParticipantResponse(result sessiondto.LeaveParticipantResponse) leaveParticipantResponse {
	return leaveParticipantResponse{
		SessionID:     result.SessionID,
		RoomID:        result.RoomID,
		ParticipantID: result.ParticipantID,
		Status:        result.Status,
	}
}

func toEndSessionResponse(result sessiondto.EndSessionResponse) endSessionResponse {
	return endSessionResponse{
		SessionID: result.SessionID,
		RoomID:    result.RoomID,
		Status:    result.Status,
	}
}

func toGetSessionStatusResponse(result sessiondto.GetSessionStatusResult) getSessionStatusResponse {
	return getSessionStatusResponse{
		SessionID:             result.SessionID,
		ConversationID:        result.ConversationID,
		UserID:                result.UserID,
		RoomID:                result.RoomID,
		Status:                result.Status,
		ConnectionState:       result.ConnectionState,
		MediaState:            result.MediaState,
		OpenAIProviderCallID:  result.OpenAIProviderCallID,
		Participants:          result.Participants,
		ParticipantStates:     toParticipantStateResponses(result.ParticipantStates),
		LastRealtimeEventType: result.LastRealtimeEventType,
		LastRealtimeEventAt:   formatOptionalTime(result.LastRealtimeEventAt),
		RecentRealtimeEvents:  toRealtimeEventResponses(result.RecentRealtimeEvents),
		StartedAt:             formatOptionalTime(result.StartedAt),
		LastActiveAt:          formatOptionalTime(result.LastActiveAt),
	}
}

func toParticipantStateResponses(states []sessiondto.ParticipantStateResult) []participantStateResponse {
	responses := make([]participantStateResponse, 0, len(states))
	for _, state := range states {
		responses = append(responses, participantStateResponse{
			ID:              state.ID,
			Role:            state.Role,
			AudioMode:       state.AudioMode,
			ConnectionState: state.ConnectionState,
			Tracks:          state.Tracks,
		})
	}
	return responses
}

func toRealtimeEventResponses(events []sessiondto.RealtimeEventResult) []realtimeEventResponse {
	responses := make([]realtimeEventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, realtimeEventResponse{
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
