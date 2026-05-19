package webrtc

import (
	"context"
	"testing"
	"time"
)

func TestWaitForICEGatheringCompleteWaitsUntilDone(t *testing.T) {
	done := make(chan struct{})
	waited := make(chan error, 1)

	go func() {
		waited <- waitForICEGatheringComplete(context.Background(), done)
	}()

	select {
	case err := <-waited:
		t.Fatalf("waitForICEGatheringComplete() returned early: %v", err)
	case <-time.After(10 * time.Millisecond):
	}

	close(done)

	select {
	case err := <-waited:
		if err != nil {
			t.Fatalf("waitForICEGatheringComplete() error = %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("waitForICEGatheringComplete() did not return after done")
	}
}

func TestWaitForICEGatheringCompleteReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForICEGatheringComplete(ctx, make(chan struct{}))
	if err == nil {
		t.Fatal("waitForICEGatheringComplete() error = nil, want context error")
	}
}
