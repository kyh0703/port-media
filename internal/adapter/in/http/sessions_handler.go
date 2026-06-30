package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	httpdto "github.com/kyh0703/portfoilo-media/internal/adapter/in/http/dto"
	httpmapper "github.com/kyh0703/portfoilo-media/internal/adapter/in/http/mapper"
	coreport "github.com/kyh0703/portfoilo-media/internal/core/port"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	sessionio "github.com/kyh0703/portfoilo-media/internal/core/usecase/sessionio"
	"github.com/kyh0703/portfoilo-media/internal/pkg/bind"
	"github.com/kyh0703/portfoilo-media/internal/pkg/exception"
	"github.com/kyh0703/portfoilo-media/internal/pkg/response"
	"go.uber.org/fx"
)

var signalingUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type mediaOfferMessage struct {
	Type             string `json:"type"`
	ParticipantToken string `json:"participantToken"`
	OfferSDP         string `json:"offerSdp"`
}

type mediaAnswerMessage struct {
	Type string `json:"type"`
	SDP  string `json:"sdp"`
}

type mediaErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

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
		{Method: http.MethodGet, Path: "/sessions/{sessionId}/join", Handler: h.Join},
		{Method: http.MethodPost, Path: "/sessions/{sessionId}/participants/{participantId}/leave", Handler: h.LeaveParticipant},
		{Method: http.MethodPost, Path: "/sessions/{sessionId}/end", Handler: h.End},
	}
}

func (h *SessionsHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req httpdto.CreateSessionRequest
	if err := bind.JSON(r, &req); err != nil {
		return exception.New(exception.CodeBadRequest, "invalid session request", http.StatusBadRequest)
	}

	res, err := h.createSession.CreateSession(r.Context(), sessionio.CreateSessionRequest{
		SessionID:      req.SessionID,
		ConversationID: req.ConversationID,
	})
	if err != nil {
		return err
	}

	return response.WriteJSON(w, http.StatusCreated, response.Created(httpmapper.ToCreateSessionResponse(res)))
}

func (h *SessionsHandler) GetStatus(w http.ResponseWriter, r *http.Request) error {
	claims, err := h.verifySessionToken(r)
	if err != nil {
		return err
	}

	res, found, err := h.status.GetSessionStatus(r.Context(), sessionio.GetSessionStatusRequest{
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

	return response.WriteJSON(w, http.StatusOK, response.OK(httpmapper.ToGetSessionStatusResponse(res)))
}

func (h *SessionsHandler) Join(w http.ResponseWriter, r *http.Request) error {
	conn, err := signalingUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	var offer mediaOfferMessage
	if err := conn.ReadJSON(&offer); err != nil {
		return h.writeSignalingError(conn, "invalid signaling offer")
	}
	if offer.Type != "offer" || strings.TrimSpace(offer.OfferSDP) == "" {
		return h.writeSignalingError(conn, "invalid signaling offer")
	}

	claims, err := h.tokenVerifier.Verify(r.Context(), offer.ParticipantToken)
	if err != nil {
		return h.writeSignalingError(conn, "invalid participant token")
	}
	if claims.SessionID != r.PathValue("sessionId") {
		return h.writeSignalingError(conn, "session token mismatch")
	}

	audioMode, err := parseAudioMode(r.URL.Query().Get("mode"))
	if err != nil {
		return h.writeSignalingError(conn, "invalid join mode")
	}

	res, err := h.joinSession.JoinSession(r.Context(), sessionio.JoinSessionCommand{
		SessionID:       claims.SessionID,
		ConversationID:  claims.ConversationID,
		ParticipantID:   claims.ParticipantID,
		ParticipantRole: claims.ParticipantRole,
		UserID:          claims.UserID,
		SDP:             offer.OfferSDP,
		AudioMode:       audioMode,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrSessionNotJoinable) {
			return h.writeSignalingError(conn, "media session is not joinable")
		}
		return h.writeSignalingError(conn, "media join failed")
	}

	return conn.WriteJSON(mediaAnswerMessage{
		Type: "answer",
		SDP:  res.SDPAnswer,
	})
}

func (h *SessionsHandler) writeSignalingError(conn *websocket.Conn, message string) error {
	return conn.WriteJSON(mediaErrorMessage{Type: "error", Message: message})
}

func (h *SessionsHandler) LeaveParticipant(w http.ResponseWriter, r *http.Request) error {
	claims, err := h.verifySessionToken(r)
	if err != nil {
		return err
	}

	res, err := h.leave.LeaveParticipant(r.Context(), sessionio.LeaveParticipantRequest{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
		ParticipantID:  r.PathValue("participantId"),
	})
	if err != nil {
		return err
	}

	return response.WriteJSON(w, http.StatusOK, response.OK(httpmapper.ToLeaveParticipantResponse(res)))
}

func (h *SessionsHandler) End(w http.ResponseWriter, r *http.Request) error {
	claims, err := h.verifySessionToken(r)
	if err != nil {
		return err
	}

	res, err := h.end.EndSession(r.Context(), sessionio.EndSessionRequest{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
	})
	if err != nil {
		return err
	}

	return response.WriteJSON(w, http.StatusOK, response.OK(httpmapper.ToEndSessionResponse(res)))
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

func parseAudioMode(mode string) (sessionio.AudioMode, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "publisher", "speaker":
		return sessionio.AudioModePublisher, nil
	case "listener", "listen_only", "listen-only":
		return sessionio.AudioModeListener, nil
	default:
		return "", exception.New(exception.CodeBadRequest, "invalid join mode", http.StatusBadRequest)
	}
}
