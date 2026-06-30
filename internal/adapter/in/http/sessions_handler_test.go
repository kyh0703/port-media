package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	httpdto "github.com/kyh0703/portfoilo-media/internal/adapter/in/http/dto"
	"github.com/kyh0703/portfoilo-media/internal/adapter/in/http/middleware"
	coreport "github.com/kyh0703/portfoilo-media/internal/core/port"
	sessionio "github.com/kyh0703/portfoilo-media/internal/core/usecase/sessionio"
	"github.com/kyh0703/portfoilo-media/internal/pkg/exception"
	"go.uber.org/zap"
)

type fakeSessionUsecases struct {
	offerReq sessionio.JoinSessionCommand
	leaveReq sessionio.LeaveParticipantRequest
	endReq   sessionio.EndSessionRequest
	status   sessionio.GetSessionStatusResult
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

func (f *fakeSessionUsecases) CreateSession(ctx context.Context, req sessionio.CreateSessionRequest) (sessionio.CreateSessionResponse, error) {
	_ = ctx
	_ = req
	return sessionio.CreateSessionResponse{}, nil
}

func (f *fakeSessionUsecases) JoinSession(ctx context.Context, req sessionio.JoinSessionCommand) (sessionio.JoinSessionResult, error) {
	_ = ctx
	f.offerReq = req
	if f.offerErr != nil {
		return sessionio.JoinSessionResult{}, f.offerErr
	}
	return sessionio.JoinSessionResult{SDPAnswer: "answer-sdp", RoomID: "room-1", ParticipantID: "participant-1"}, nil
}

func (f *fakeSessionUsecases) LeaveParticipant(ctx context.Context, req sessionio.LeaveParticipantRequest) (sessionio.LeaveParticipantResponse, error) {
	_ = ctx
	f.leaveReq = req
	return sessionio.LeaveParticipantResponse{
		SessionID:     req.SessionID,
		RoomID:        "room-1",
		ParticipantID: req.ParticipantID,
		Status:        "active",
	}, nil
}

func (f *fakeSessionUsecases) EndSession(ctx context.Context, req sessionio.EndSessionRequest) (sessionio.EndSessionResponse, error) {
	_ = ctx
	f.endReq = req
	return sessionio.EndSessionResponse{
		SessionID: req.SessionID,
		RoomID:    "room-1",
		Status:    "closed",
	}, nil
}

func (f *fakeSessionUsecases) GetSessionStatus(ctx context.Context, req sessionio.GetSessionStatusRequest) (sessionio.GetSessionStatusResult, bool, error) {
	_ = ctx
	if f.status.SessionID == "" {
		f.status.SessionID = req.SessionID
	}
	return f.status, f.found, nil
}

func (f *fakeSessionUsecases) GetRuntimeStats(ctx context.Context) (sessionio.RuntimeStatsResponse, error) {
	_ = ctx
	return sessionio.RuntimeStatsResponse{}, nil
}

func newFakeSessionsHandler(session *fakeSessionUsecases, tokenVerifier coreport.MediaTokenVerifier) *SessionsHandler {
	return NewSessionsHandler(SessionsHandlerParams{
		CreateSession: session,
		JoinSession:   session,
		Leave:         session,
		End:           session,
		Status:        session,
		TokenVerifier: tokenVerifier,
	})
}

func TestSessionsHandlerGetsSessionStatusWithMediaToken(t *testing.T) {
	usecase := &fakeSessionUsecases{
		found: true,
		status: sessionio.GetSessionStatusResult{
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
	handler := newFakeSessionsHandler(usecase, fakeMediaTokenVerifier{
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

func (f *fakeSessionUsecases) GetHealth(ctx context.Context) error {
	_ = ctx
	return nil
}

func TestSessionsHandlerEndsSessionWithMediaToken(t *testing.T) {
	usecase := &fakeSessionUsecases{}
	handler := newFakeSessionsHandler(usecase, fakeMediaTokenVerifier{
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
	usecase := &fakeSessionUsecases{}
	handler := newFakeSessionsHandler(usecase, fakeMediaTokenVerifier{
		token: coreport.MediaToken{
			SessionID:       "session-1",
			ConversationID:  "conversation-1",
			ParticipantID:   "participant-1",
			ParticipantRole: "user",
			UserID:          "user-1",
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	server := httptest.NewServer(app.mux)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL, "/sessions/session-1/join"), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()
	if err := conn.WriteJSON(map[string]string{
		"type":             "offer",
		"participantToken": "media-token",
		"offerSdp":         "offer-sdp",
	}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	var answer struct {
		Type string `json:"type"`
		SDP  string `json:"sdp"`
	}
	if err := conn.ReadJSON(&answer); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if answer.Type != "answer" || answer.SDP != "answer-sdp" {
		t.Fatalf("answer = %#v, want answer-sdp", answer)
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
	if usecase.offerReq.ParticipantID != "participant-1" {
		t.Fatalf("ParticipantID = %q, want participant-1", usecase.offerReq.ParticipantID)
	}
	if usecase.offerReq.ParticipantRole != "user" {
		t.Fatalf("ParticipantRole = %q, want user", usecase.offerReq.ParticipantRole)
	}
	if usecase.offerReq.SDP != "offer-sdp" {
		t.Fatalf("SDP = %q, want offer-sdp", usecase.offerReq.SDP)
	}
	if usecase.offerReq.AudioMode != sessionio.AudioModePublisher {
		t.Fatalf("AudioMode = %q, want %q", usecase.offerReq.AudioMode, sessionio.AudioModePublisher)
	}
}

func TestSessionsHandlerAcceptsListenerJoinMode(t *testing.T) {
	usecase := &fakeSessionUsecases{}
	handler := newFakeSessionsHandler(usecase, fakeMediaTokenVerifier{
		token: coreport.MediaToken{
			SessionID:       "session-1",
			ConversationID:  "conversation-1",
			ParticipantID:   "participant-1",
			ParticipantRole: "user",
			UserID:          "user-1",
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	server := httptest.NewServer(app.mux)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL, "/sessions/session-1/join?mode=listener"), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()
	if err := conn.WriteJSON(map[string]string{
		"type":             "offer",
		"participantToken": "media-token",
		"offerSdp":         "offer-sdp",
	}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	var answer map[string]string
	if err := conn.ReadJSON(&answer); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if usecase.offerReq.AudioMode != sessionio.AudioModeListener {
		t.Fatalf("AudioMode = %q, want %q", usecase.offerReq.AudioMode, sessionio.AudioModeListener)
	}
}

func TestSessionsHandlerLeavesParticipantWithMediaToken(t *testing.T) {
	usecase := &fakeSessionUsecases{}
	handler := newFakeSessionsHandler(usecase, fakeMediaTokenVerifier{
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
		StatusCode int                              `json:"statusCode"`
		Message    string                           `json:"message"`
		Data       httpdto.LeaveParticipantResponse `json:"data"`
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
	usecase := &fakeSessionUsecases{}
	handler := newFakeSessionsHandler(usecase, fakeMediaTokenVerifier{err: coreport.ErrMissingMediaToken})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	server := httptest.NewServer(app.mux)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL, "/sessions/session-1/join"), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()
	if err := conn.WriteJSON(map[string]string{
		"type":             "offer",
		"participantToken": "missing-token",
		"offerSdp":         "offer-sdp",
	}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	var body struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	if err := conn.ReadJSON(&body); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if body.Type != "error" || body.Message != "invalid participant token" {
		t.Fatalf("body = %#v, want invalid participant token error", body)
	}
}

func TestSessionsHandlerRejectsSessionMismatch(t *testing.T) {
	usecase := &fakeSessionUsecases{}
	handler := newFakeSessionsHandler(usecase, fakeMediaTokenVerifier{
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

	server := httptest.NewServer(app.mux)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL, "/sessions/session-1/join"), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()
	if err := conn.WriteJSON(map[string]string{
		"type":             "offer",
		"participantToken": "media-token",
		"offerSdp":         "offer-sdp",
	}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	var body struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	if err := conn.ReadJSON(&body); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if body.Type != "error" || body.Message != "session token mismatch" {
		t.Fatalf("body = %#v, want session token mismatch error", body)
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

func wsURL(serverURL string, path string) string {
	return "ws" + serverURL[len("http"):] + path
}
