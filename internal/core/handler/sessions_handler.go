package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
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
		{Method: fiber.MethodPost, Path: "/sessions", Handler: []fiber.Handler{h.Create}},
		{Method: fiber.MethodGet, Path: "/sessions/:sessionId/status", Handler: []fiber.Handler{h.GetStatus}},
		{Method: fiber.MethodPost, Path: "/sessions/:sessionId/offer", Handler: []fiber.Handler{h.AcceptOffer}},
		{Method: fiber.MethodPost, Path: "/sessions/:sessionId/participants/:participantId/leave", Handler: []fiber.Handler{h.LeaveParticipant}},
		{Method: fiber.MethodPost, Path: "/sessions/:sessionId/end", Handler: []fiber.Handler{h.End}},
	}
}

func (h *SessionsHandler) Create(c *fiber.Ctx) error {
	var req sessiondto.CreateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return exception.New(exception.CodeBadRequest, "invalid session request", fiber.StatusBadRequest)
	}

	res, err := h.session.CreateSession(c.Context(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(response.Created(res))
}

func (h *SessionsHandler) GetStatus(c *fiber.Ctx) error {
	claims, err := h.verifySessionToken(c)
	if err != nil {
		return err
	}

	res, found, err := h.session.GetSessionStatus(c.Context(), sessiondto.GetSessionStatusRequest{
		SessionID: claims.SessionID,
	})
	if err != nil {
		return err
	}
	if !found {
		return exception.New(exception.CodeNotFound, "media session status not found", fiber.StatusNotFound)
	}

	return c.JSON(response.OK(res))
}

func (h *SessionsHandler) AcceptOffer(c *fiber.Ctx) error {
	claims, err := h.verifySessionToken(c)
	if err != nil {
		return err
	}

	sdp := string(c.Body())
	if sdp == "" {
		return exception.New(exception.CodeBadRequest, "empty SDP offer", fiber.StatusBadRequest)
	}
	audioMode, err := parseAudioMode(c.Query("mode", string(sessiondto.AudioModePublisher)))
	if err != nil {
		return exception.New(exception.CodeBadRequest, "invalid offer mode", fiber.StatusBadRequest)
	}

	res, err := h.session.AcceptOffer(c.Context(), sessiondto.AcceptOfferRequest{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
		SDP:            sdp,
		AudioMode:      audioMode,
	})
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, "application/sdp")
	c.Set("X-Room-Id", res.RoomID)
	c.Set("X-Participant-Id", res.ParticipantID)
	return c.SendString(res.SDPAnswer)
}

func (h *SessionsHandler) LeaveParticipant(c *fiber.Ctx) error {
	claims, err := h.verifySessionToken(c)
	if err != nil {
		return err
	}

	res, err := h.session.LeaveParticipant(c.Context(), sessiondto.LeaveParticipantRequest{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
		ParticipantID:  c.Params("participantId"),
	})
	if err != nil {
		return err
	}

	return c.JSON(response.OK(res))
}

func (h *SessionsHandler) End(c *fiber.Ctx) error {
	claims, err := h.verifySessionToken(c)
	if err != nil {
		return err
	}

	res, err := h.session.EndSession(c.Context(), sessiondto.EndSessionRequest{
		SessionID:      claims.SessionID,
		ConversationID: claims.ConversationID,
		UserID:         claims.UserID,
	})
	if err != nil {
		return err
	}

	return c.JSON(response.OK(res))
}

func (h *SessionsHandler) verifySessionToken(c *fiber.Ctx) (auth.MediaToken, error) {
	claims, err := h.tokenVerifier.Verify(c.Context(), readBearerToken(c.Get(fiber.HeaderAuthorization)))
	if err != nil {
		return auth.MediaToken{}, exception.New(exception.CodeUnauthorized, "invalid media token", fiber.StatusUnauthorized)
	}
	if claims.SessionID != c.Params("sessionId") {
		return auth.MediaToken{}, exception.New(exception.CodeUnauthorized, "media token session mismatch", fiber.StatusUnauthorized)
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
		return "", exception.New(exception.CodeBadRequest, "invalid offer mode", fiber.StatusBadRequest)
	}
}
