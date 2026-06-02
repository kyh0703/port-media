package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	"github.com/kyh0703/portfoilo-media/internal/pkg/auth"
	"github.com/kyh0703/portfoilo-media/internal/pkg/exception"
	"github.com/kyh0703/portfoilo-media/internal/pkg/response"
)

type SessionsHandler struct {
	session       usecase.SessionUsecase
	tokenVerifier auth.MediaTokenVerifier
}

func NewSessionsHandler(session usecase.SessionUsecase, tokenVerifier auth.MediaTokenVerifier) *SessionsHandler {
	return &SessionsHandler{session: session, tokenVerifier: tokenVerifier}
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
	var req sessiondto.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return exception.New(exception.CodeBadRequest, "invalid session request", http.StatusBadRequest)
	}

	res, err := h.session.CreateSession(r.Context(), req)
	if err != nil {
		return err
	}

	return response.WriteJSON(w, http.StatusCreated, response.Created(res))
}

func (h *SessionsHandler) GetStatus(w http.ResponseWriter, r *http.Request) error {
	claims, err := h.verifySessionToken(r)
	if err != nil {
		return err
	}

	res, found, err := h.session.GetSessionStatus(r.Context(), sessiondto.GetSessionStatusRequest{
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

	return response.WriteJSON(w, http.StatusOK, response.OK(res))
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

	res, err := h.session.AcceptOffer(r.Context(), sessiondto.AcceptOfferRequest{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
		SDP:            sdp,
		AudioMode:      audioMode,
	})
	if err != nil {
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

	res, err := h.session.LeaveParticipant(r.Context(), sessiondto.LeaveParticipantRequest{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
		ParticipantID:  r.PathValue("participantId"),
	})
	if err != nil {
		return err
	}

	return response.WriteJSON(w, http.StatusOK, response.OK(res))
}

func (h *SessionsHandler) End(w http.ResponseWriter, r *http.Request) error {
	claims, err := h.verifySessionToken(r)
	if err != nil {
		return err
	}

	res, err := h.session.EndSession(r.Context(), sessiondto.EndSessionRequest{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
	})
	if err != nil {
		return err
	}

	return response.WriteJSON(w, http.StatusOK, response.OK(res))
}

func (h *SessionsHandler) verifySessionToken(r *http.Request) (auth.MediaToken, error) {
	claims, err := h.tokenVerifier.Verify(r.Context(), readBearerToken(r.Header.Get("Authorization")))
	if err != nil {
		return auth.MediaToken{}, exception.New(exception.CodeUnauthorized, "invalid media token", http.StatusUnauthorized)
	}
	if claims.SessionID != r.PathValue("sessionId") {
		return auth.MediaToken{}, exception.New(exception.CodeUnauthorized, "media token session mismatch", http.StatusUnauthorized)
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
