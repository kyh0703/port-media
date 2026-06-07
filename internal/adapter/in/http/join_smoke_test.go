package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	corerepo "github.com/kyh0703/portfoilo-media/internal/adapter/out/repository"
	rtc "github.com/kyh0703/portfoilo-media/internal/adapter/out/webrtc"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/repository/repositoryfakes"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
	sessionquery "github.com/kyh0703/portfoilo-media/internal/core/query/session"
	"github.com/kyh0703/portfoilo-media/internal/core/query/session/sessionfakes"
	sessionservice "github.com/kyh0703/portfoilo-media/internal/core/service/session"
	pionwebrtc "github.com/pion/webrtc/v4"
)

func TestJoinEndpointSmokeWithPionClient(t *testing.T) {
	cfg := &configs.Config{
		Realtime: configs.RealtimeConfig{
			ICEGatheringTimeout: 500 * time.Millisecond,
		},
	}
	media, err := rtc.NewEngine(cfg)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	records := newSmokeMediaSessionRecordRepository()
	runtime := corerepo.NewMemoryRoomRuntimeRepository()
	states := newSmokeMediaSessionStateRepository()
	provider := newSmokeRealtimeProvider(t)
	svc := sessionservice.NewService(records, runtime, states, rtc.NewGateway(media), provider)
	if _, err := svc.CreateSession(context.Background(), sessiondto.CreateSessionRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
	}); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	handler := NewSessionsHandler(SessionsHandlerParams{
		CreateSession: svc,
		JoinSession:   svc,
		Leave:         svc,
		End:           svc,
		Status:        svc,
		TokenVerifier: fakeMediaTokenVerifier{
			token: port.MediaToken{
				SessionID:      "session-1",
				ConversationID: "conversation-1",
				UserID:         "user-1",
			},
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	clientPeer, offerSDP := newSmokeClientOffer(t)
	defer func() {
		_ = clientPeer.Close()
	}()

	req := httptest.NewRequest(
		http.MethodPost,
		"/sessions/session-1/join",
		strings.NewReader(offerSDP),
	)
	req.Header.Set("Authorization", "Bearer media-token")
	req.Header.Set("Content-Type", "application/sdp")

	res, err := app.Test(req, 2_000)
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
	if !strings.Contains(string(body), "m=audio") {
		t.Fatalf("answer SDP missing audio media section: %s", string(body))
	}
	if err := clientPeer.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeAnswer,
		SDP:  string(body),
	}); err != nil {
		t.Fatalf("client SetRemoteDescription(answer) error = %v", err)
	}

	state, found, err := states.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("media session state not found")
	}
	if state.Status != vo.RoomStatusActive {
		t.Fatalf("state status = %q, want %q", state.Status, vo.RoomStatusActive)
	}
}

func newSmokeClientOffer(t *testing.T) (*pionwebrtc.PeerConnection, string) {
	t.Helper()

	peer, err := pionwebrtc.NewPeerConnection(pionwebrtc.Configuration{})
	if err != nil {
		t.Fatalf("NewPeerConnection() error = %v", err)
	}
	if _, err := peer.AddTransceiverFromKind(pionwebrtc.RTPCodecTypeAudio); err != nil {
		t.Fatalf("AddTransceiverFromKind(audio) error = %v", err)
	}

	offer, err := peer.CreateOffer(nil)
	if err != nil {
		t.Fatalf("CreateOffer() error = %v", err)
	}
	gatherComplete := pionwebrtc.GatheringCompletePromise(peer)
	if err := peer.SetLocalDescription(offer); err != nil {
		t.Fatalf("SetLocalDescription(offer) error = %v", err)
	}
	select {
	case <-gatherComplete:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("client ICE gathering timed out")
	}

	localDescription := peer.LocalDescription()
	if localDescription == nil {
		t.Fatal("client local description is nil")
	}
	return peer, localDescription.SDP
}

type smokeRealtimeProvider struct {
	t     *testing.T
	peers map[string]*pionwebrtc.PeerConnection
}

func newSmokeRealtimeProvider(t *testing.T) *smokeRealtimeProvider {
	t.Helper()
	return &smokeRealtimeProvider{t: t, peers: make(map[string]*pionwebrtc.PeerConnection)}
}

func (p *smokeRealtimeProvider) CreateCall(ctx context.Context, input port.CreateCallInput) (port.CreateCallResult, error) {
	_ = ctx
	peer, err := pionwebrtc.NewPeerConnection(pionwebrtc.Configuration{})
	if err != nil {
		return port.CreateCallResult{}, err
	}
	if err := peer.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeOffer,
		SDP:  input.SDPOffer,
	}); err != nil {
		_ = peer.Close()
		return port.CreateCallResult{}, err
	}
	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		_ = peer.Close()
		return port.CreateCallResult{}, err
	}
	gatherComplete := pionwebrtc.GatheringCompletePromise(peer)
	if err := peer.SetLocalDescription(answer); err != nil {
		_ = peer.Close()
		return port.CreateCallResult{}, err
	}
	select {
	case <-gatherComplete:
	case <-time.After(500 * time.Millisecond):
		_ = peer.Close()
		return port.CreateCallResult{}, context.DeadlineExceeded
	}
	p.peers["rtc_smoke"] = peer

	localDescription := peer.LocalDescription()
	if localDescription == nil {
		_ = peer.Close()
		return port.CreateCallResult{}, context.Canceled
	}
	return port.CreateCallResult{
		SDPAnswer:      localDescription.SDP,
		ProviderCallID: "rtc_smoke",
	}, nil
}

func (p *smokeRealtimeProvider) HangupCall(ctx context.Context, providerCallID string) error {
	_ = ctx
	if peer := p.peers[providerCallID]; peer != nil {
		return peer.Close()
	}
	return nil
}

func newSmokeMediaSessionRecordRepository() *repositoryfakes.FakeMediaSessionRecordRepository {
	var mu sync.RWMutex
	records := make(map[vo.RoomID]entity.MediaSessionRecord)
	repo := &repositoryfakes.FakeMediaSessionRecordRepository{}

	repo.SaveCalls(func(ctx context.Context, record entity.MediaSessionRecord) error {
		_ = ctx
		mu.Lock()
		defer mu.Unlock()

		records[record.ID] = record
		return nil
	})
	repo.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (entity.MediaSessionRecord, bool, error) {
		_ = ctx
		mu.RLock()
		defer mu.RUnlock()

		for _, record := range records {
			if record.SessionID == sessionID {
				return record, true, nil
			}
		}
		return entity.MediaSessionRecord{}, false, nil
	})
	repo.DeleteCalls(func(ctx context.Context, roomID vo.RoomID) error {
		_ = ctx
		mu.Lock()
		defer mu.Unlock()

		delete(records, roomID)
		return nil
	})

	return repo
}

func newSmokeMediaSessionStateRepository() *sessionfakes.FakeMediaSessionStateRepository {
	var mu sync.RWMutex
	states := make(map[vo.SessionID]sessionquery.MediaSessionState)
	repo := &sessionfakes.FakeMediaSessionStateRepository{}

	repo.SaveCalls(func(ctx context.Context, state sessionquery.MediaSessionState) error {
		_ = ctx
		mu.Lock()
		defer mu.Unlock()

		states[state.SessionID] = state
		return nil
	})
	repo.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		mu.RLock()
		defer mu.RUnlock()

		state, found := states[sessionID]
		return state, found, nil
	})
	repo.DeleteCalls(func(ctx context.Context, sessionID vo.SessionID) error {
		_ = ctx
		mu.Lock()
		defer mu.Unlock()

		delete(states, sessionID)
		return nil
	})

	return repo
}
