package webrtc

import (
	"testing"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	pionwebrtc "github.com/pion/webrtc/v4"
)

func TestMapPeerConnectionState(t *testing.T) {
	tests := []struct {
		name  string
		input pionwebrtc.PeerConnectionState
		want  vo.ConnectionState
	}{
		{name: "new", input: pionwebrtc.PeerConnectionStateNew, want: vo.ConnectionStateNew},
		{name: "connecting", input: pionwebrtc.PeerConnectionStateConnecting, want: vo.ConnectionStateConnecting},
		{name: "connected", input: pionwebrtc.PeerConnectionStateConnected, want: vo.ConnectionStateConnected},
		{name: "disconnected", input: pionwebrtc.PeerConnectionStateDisconnected, want: vo.ConnectionStateDisconnected},
		{name: "failed", input: pionwebrtc.PeerConnectionStateFailed, want: vo.ConnectionStateFailed},
		{name: "closed", input: pionwebrtc.PeerConnectionStateClosed, want: vo.ConnectionStateClosed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapPeerConnectionState(tt.input); got != tt.want {
				t.Fatalf("mapPeerConnectionState(%s) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
