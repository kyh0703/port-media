package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	domainrepo "github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	corerepo "github.com/kyh0703/portfoilo-media/internal/core/repository"
	sessionservice "github.com/kyh0703/portfoilo-media/internal/core/service/session"
	"github.com/kyh0703/portfoilo-media/internal/pkg/auth"
	"github.com/kyh0703/portfoilo-media/internal/pkg/openai"
	rtc "github.com/kyh0703/portfoilo-media/internal/pkg/webrtc"
	pionwebrtc "github.com/pion/webrtc/v4"
)

func TestOfferEndpointSmokeWithPionClient(t *testing.T) {
	cfg := &configs.Config{
		Realtime: configs.RealtimeConfig{
			ICEGatheringTimeout: 500 * time.Millisecond,
		},
	}
	media, err := rtc.NewEngine(cfg)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	rooms := newSmokeRoomRepository()
	runtime := corerepo.NewMemoryRoomRuntimeRepository()
	states := newSmokeMediaSessionStateRepository()
	provider := newSmokeRealtimeProvider(t)
	svc := sessionservice.NewService(rooms, runtime, states, media, provider)
	handler := NewSessionsHandler(svc, fakeMediaTokenVerifier{
		token: auth.MediaToken{
			SessionID:      "session-1",
			ConversationID: "conversation-1",
			UserID:         "user-1",
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler...)
	}

	clientPeer, offerSDP := newSmokeClientOffer(t)
	defer func() {
		_ = clientPeer.Close()
	}()

	req := httptest.NewRequest(
		http.MethodPost,
		"/sessions/session-1/offer",
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

func (p *smokeRealtimeProvider) CreateCall(ctx context.Context, input openai.CreateCallInput) (openai.CreateCallResult, error) {
	_ = ctx
	peer, err := pionwebrtc.NewPeerConnection(pionwebrtc.Configuration{})
	if err != nil {
		return openai.CreateCallResult{}, err
	}
	if err := peer.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeOffer,
		SDP:  input.SDPOffer,
	}); err != nil {
		_ = peer.Close()
		return openai.CreateCallResult{}, err
	}
	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		_ = peer.Close()
		return openai.CreateCallResult{}, err
	}
	gatherComplete := pionwebrtc.GatheringCompletePromise(peer)
	if err := peer.SetLocalDescription(answer); err != nil {
		_ = peer.Close()
		return openai.CreateCallResult{}, err
	}
	select {
	case <-gatherComplete:
	case <-time.After(500 * time.Millisecond):
		_ = peer.Close()
		return openai.CreateCallResult{}, context.DeadlineExceeded
	}
	p.peers["rtc_smoke"] = peer

	localDescription := peer.LocalDescription()
	if localDescription == nil {
		_ = peer.Close()
		return openai.CreateCallResult{}, context.Canceled
	}
	return openai.CreateCallResult{
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

type smokeRoomRepository struct {
	rooms map[vo.RoomID]entity.Room
}

func newSmokeRoomRepository() domainrepo.RoomRepository {
	return &smokeRoomRepository{rooms: make(map[vo.RoomID]entity.Room)}
}

func (r *smokeRoomRepository) Save(ctx context.Context, room entity.Room) error {
	_ = ctx
	r.rooms[room.ID] = room
	return nil
}

func (r *smokeRoomRepository) FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.Room, bool, error) {
	_ = ctx
	for _, room := range r.rooms {
		if room.SessionID == sessionID {
			return room, true, nil
		}
	}
	return entity.Room{}, false, nil
}

func (r *smokeRoomRepository) Delete(ctx context.Context, roomID vo.RoomID) error {
	_ = ctx
	delete(r.rooms, roomID)
	return nil
}

type smokeMediaSessionStateRepository struct {
	states map[vo.SessionID]entity.MediaSessionState
}

func newSmokeMediaSessionStateRepository() domainrepo.MediaSessionStateRepository {
	return &smokeMediaSessionStateRepository{states: make(map[vo.SessionID]entity.MediaSessionState)}
}

func (r *smokeMediaSessionStateRepository) Save(ctx context.Context, state entity.MediaSessionState) error {
	_ = ctx
	r.states[state.SessionID] = state
	return nil
}

func (r *smokeMediaSessionStateRepository) FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.MediaSessionState, bool, error) {
	_ = ctx
	state, found := r.states[sessionID]
	return state, found, nil
}

func (r *smokeMediaSessionStateRepository) Delete(ctx context.Context, sessionID vo.SessionID) error {
	_ = ctx
	delete(r.states, sessionID)
	return nil
}
