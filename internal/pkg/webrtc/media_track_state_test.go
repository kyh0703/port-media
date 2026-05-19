package webrtc

import (
	"errors"
	"io"
	"testing"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

func TestTrackStateForRelayError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want vo.TrackState
	}{
		{name: "eof", err: io.EOF, want: vo.TrackStateEnded},
		{name: "other error", err: errors.New("write failed"), want: vo.TrackStateFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trackStateForRelayError(tt.err); got != tt.want {
				t.Fatalf("trackStateForRelayError(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}
