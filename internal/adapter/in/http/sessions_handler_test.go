package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kyh0703/portfoilo-media/internal/adapter/in/http/middleware"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	coreport "github.com/kyh0703/portfoilo-media/internal/core/port"
	"github.com/kyh0703/portfoilo-media/internal/pkg/exception"
	"go.uber.org/zap"
)

type fakeSessionUsecase struct {
	offerReq sessiondto.AcceptOfferRequest
	leaveReq sessiondto.LeaveParticipantRequest
	endReq   sessiondto.EndSessionRequest
	status   sessiondto.GetSessionStatusResponse
	found    bool
	offerErr error
}

type fakeMediaTokenVerifier struct {
	token coreport.MediaToken
	err   error
}

func (v fakeMediaTokenVerifier) Verify(ctx context.Context, raw string) (coreport.MediaToken, error) {
	_ = ctx
	_ = raw
	return v.token, v.err
}

func (f *fakeSessionUsecase) CreateSession(ctx context.Context, req sessiondto.CreateSessionRequest) (sessiondto.CreateSessionResponse, error) {
	_ = ctx
	_ = req
	return sessiondto.CreateSessionResponse{}, nil
}

func (f *fakeSessionUsecase) AcceptOffer(ctx context.Context, req sessiondto.AcceptOfferRequest) (sessiondto.AcceptOfferResponse, error) {
	_ = ctx
	f.offerReq = req
	if f.offerErr != nil {
		return sessiondto.AcceptOfferResponse{}, f.offerErr
	}
	return sessiondto.AcceptOfferResponse{SDPAnswer: "answer-sdp", RoomID: "room-1", ParticipantID: "participant-1"}, nil
}

func (f *fakeSessionUsecase) LeaveParticipant(ctx context.Context, req sessiondto.LeaveParticipantRequest) (sessiondto.LeaveParticipantResponse, error) {
	_ = ctx
	f.leaveReq = req
	return sessiondto.LeaveParticipantResponse{
		SessionID:     req.SessionID,
		RoomID:        "room-1",
		ParticipantID: req.ParticipantID,
		Status:        "active",
	}, nil
}

func (f *fakeSessionUsecase) EndSession(ctx context.Context, req sessiondto.EndSessionRequest) (sessiondto.EndSessionResponse, error) {
	_ = ctx
	f.endReq = req
	return sessiondto.EndSessionResponse{
		SessionID: req.SessionID,
		RoomID:    "room-1",
		Status:    "closed",
	}, nil
}

func (f *fakeSessionUsecase) GetSessionStatus(ctx context.Context, req sessiondto.GetSessionStatusRequest) (sessiondto.GetSessionStatusResponse, bool, error) {
	_ = ctx
	if f.status.SessionID == "" {
		f.status.SessionID = req.SessionID
	}
	return f.status, f.found, nil
}

func (f *fakeSessionUsecase) GetRuntimeStats(ctx context.Context) (sessiondto.RuntimeStatsResponse, error) {
	_ = ctx
	return sessiondto.RuntimeStatsResponse{}, nil
}

func TestSessionsHandlerGetsSessionStatusWithMediaToken(t *testing.T) {
	usecase := &fakeSessionUsecase{
		found: true,
		status: sessiondto.GetSessionStatusResponse{
			SessionID:       "session-1",
			ConversationID:  "conversation-1",
			UserID:          "user-1",
			RoomID:          "room-1",
			Status:          "active",
			ConnectionState: "connected",
			MediaState:      "active",
			Participants:    2,
		},
	}
	handler := NewSessionsHandler(usecase, fakeMediaTokenVerifier{
		token: coreport.MediaToken{
			SessionID:      "session-1",
			ConversationID: "conversation-1",
			UserID:         "user-1",
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/sessions/session-1/status", nil)
	req.Header.Set("Authorization", "Bearer media-token")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want %d body=%s", res.StatusCode, http.StatusOK, string(body))
	}
}

func (f *fakeSessionUsecase) GetHealth(ctx context.Context) error {
	_ = ctx
	return nil
}

func TestSessionsHandlerEndsSessionWithMediaToken(t *testing.T) {
	usecase := &fakeSessionUsecase{}
	handler := NewSessionsHandler(usecase, fakeMediaTokenVerifier{
		token: coreport.MediaToken{
			SessionID:      "session-1",
			ConversationID: "conversation-1",
			UserID:         "user-1",
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodPost, "/sessions/session-1/end", nil)
	req.Header.Set("Authorization", "Bearer media-token")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want %d body=%s", res.StatusCode, http.StatusOK, string(body))
	}
	if usecase.endReq.SessionID != "session-1" {
		t.Fatalf("SessionID = %q, want session-1", usecase.endReq.SessionID)
	}
	if usecase.endReq.ConversationID != "conversation-1" {
		t.Fatalf("ConversationID = %q, want conversation-1", usecase.endReq.ConversationID)
	}
	if usecase.endReq.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", usecase.endReq.UserID)
	}
}

func TestSessionsHandlerJoinsWithSDP(t *testing.T) {
	usecase := &fakeSessionUsecase{}
	handler := NewSessionsHandler(usecase, fakeMediaTokenVerifier{
		token: coreport.MediaToken{
			SessionID:      "session-1",
			ConversationID: "conversation-1",
			UserID:         "user-1",
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/sessions/session-1/join",
		strings.NewReader("offer-sdp"),
	)
	req.Header.Set("Authorization", "Bearer media-token")
	req.Header.Set("Content-Type", "application/sdp")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", res.StatusCode, http.StatusOK, string(body))
	}
	if string(body) != "answer-sdp" {
		t.Fatalf("body = %q, want answer-sdp", string(body))
	}
	if res.Header.Get("X-Room-Id") != "room-1" {
		t.Fatalf("X-Room-Id = %q, want room-1", res.Header.Get("X-Room-Id"))
	}
	if res.Header.Get("X-Participant-Id") != "participant-1" {
		t.Fatalf("X-Participant-Id = %q, want participant-1", res.Header.Get("X-Participant-Id"))
	}
	if usecase.offerReq.SessionID != "session-1" {
		t.Fatalf("SessionID = %q, want session-1", usecase.offerReq.SessionID)
	}
	if usecase.offerReq.ConversationID != "conversation-1" {
		t.Fatalf("ConversationID = %q, want conversation-1", usecase.offerReq.ConversationID)
	}
	if usecase.offerReq.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", usecase.offerReq.UserID)
	}
	if usecase.offerReq.SDP != "offer-sdp" {
		t.Fatalf("SDP = %q, want offer-sdp", usecase.offerReq.SDP)
	}
	if usecase.offerReq.AudioMode != sessiondto.AudioModePublisher {
		t.Fatalf("AudioMode = %q, want %q", usecase.offerReq.AudioMode, sessiondto.AudioModePublisher)
	}
}

func TestSessionsHandlerAcceptsListenerJoinMode(t *testing.T) {
	usecase := &fakeSessionUsecase{}
	handler := NewSessionsHandler(usecase, fakeMediaTokenVerifier{
		token: coreport.MediaToken{
			SessionID:      "session-1",
			ConversationID: "conversation-1",
			UserID:         "user-1",
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/sessions/session-1/join?mode=listener",
		strings.NewReader("offer-sdp"),
	)
	req.Header.Set("Authorization", "Bearer media-token")
	req.Header.Set("Content-Type", "application/sdp")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want %d body=%s", res.StatusCode, http.StatusOK, string(body))
	}
	if usecase.offerReq.AudioMode != sessiondto.AudioModeListener {
		t.Fatalf("AudioMode = %q, want %q", usecase.offerReq.AudioMode, sessiondto.AudioModeListener)
	}
}

func TestSessionsHandlerLeavesParticipantWithMediaToken(t *testing.T) {
	usecase := &fakeSessionUsecase{}
	handler := NewSessionsHandler(usecase, fakeMediaTokenVerifier{
		token: coreport.MediaToken{
			SessionID:      "session-1",
			ConversationID: "conversation-1",
			UserID:         "user-1",
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodPost, "/sessions/session-1/participants/participant-1/leave", nil)
	req.Header.Set("Authorization", "Bearer media-token")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want %d body=%s", res.StatusCode, http.StatusOK, string(body))
	}
	var body struct {
		StatusCode int                                 `json:"statusCode"`
		Message    string                              `json:"message"`
		Data       sessiondto.LeaveParticipantResponse `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.StatusCode != http.StatusOK {
		t.Fatalf("statusCode = %d, want %d", body.StatusCode, http.StatusOK)
	}
	if body.Message != "OK" {
		t.Fatalf("message = %q, want OK", body.Message)
	}
	if body.Data.ParticipantID != "participant-1" {
		t.Fatalf("response participant_id = %q, want participant-1", body.Data.ParticipantID)
	}
	if usecase.leaveReq.SessionID != "session-1" {
		t.Fatalf("SessionID = %q, want session-1", usecase.leaveReq.SessionID)
	}
	if usecase.leaveReq.ParticipantID != "participant-1" {
		t.Fatalf("ParticipantID = %q, want participant-1", usecase.leaveReq.ParticipantID)
	}
	if usecase.leaveReq.ConversationID != "conversation-1" {
		t.Fatalf("ConversationID = %q, want conversation-1", usecase.leaveReq.ConversationID)
	}
	if usecase.leaveReq.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", usecase.leaveReq.UserID)
	}
}

func TestSessionsHandlerRejectsJoinWithoutMediaToken(t *testing.T) {
	usecase := &fakeSessionUsecase{}
	handler := NewSessionsHandler(usecase, fakeMediaTokenVerifier{err: coreport.ErrMissingMediaToken})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/sessions/session-1/join",
		strings.NewReader("offer-sdp"),
	)
	req.Header.Set("Content-Type", "application/sdp")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
	var body struct {
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
		Error      string `json:"error"`
		Data       any    `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.StatusCode != http.StatusUnauthorized {
		t.Fatalf("statusCode = %d, want %d", body.StatusCode, http.StatusUnauthorized)
	}
	if body.Message != "invalid media token" {
		t.Fatalf("message = %q, want invalid media token", body.Message)
	}
	if body.Error != exception.CodeUnauthorized {
		t.Fatalf("error = %q, want %q", body.Error, exception.CodeUnauthorized)
	}
}

func TestSessionsHandlerRejectsSessionMismatch(t *testing.T) {
	usecase := &fakeSessionUsecase{}
	handler := NewSessionsHandler(usecase, fakeMediaTokenVerifier{
		token: coreport.MediaToken{
			SessionID:      "other-session",
			ConversationID: "conversation-1",
			UserID:         "user-1",
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/sessions/session-1/join",
		strings.NewReader("offer-sdp"),
	)
	req.Header.Set("Authorization", "Bearer media-token")
	req.Header.Set("Content-Type", "application/sdp")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
}

type testApp struct {
	mux *http.ServeMux
}

func newTestApp() *testApp {
	return &testApp{mux: http.NewServeMux()}
}

func (a *testApp) Add(method string, path string, h ErrorHandlerFunc) {
	a.mux.Handle(method+" "+path, middleware.WithErrorHandler(h, exception.NewHTTPErrorHandler(zap.NewNop())))
}

func (a *testApp) Test(req *http.Request, _ ...int) (*http.Response, error) {
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)
	return rec.Result(), nil
}
