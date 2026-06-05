package lifecycle

import (
	"testing"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
)

func TestMediaServerStateReporterOptionsDefaultsURLFromServerConfig(t *testing.T) {
	options := NewMediaServerStateReporterOptions(&configs.Config{
		Server: configs.ServerConfig{
			Host: "0.0.0.0",
			Port: 9090,
		},
	})

	if options.URL != "http://localhost:9090" {
		t.Fatalf("options.URL = %q, want http://localhost:9090", options.URL)
	}
}

func TestMediaServerStateReporterOptionsUsesConfigFields(t *testing.T) {
	options := NewMediaServerStateReporterOptions(&configs.Config{
		NodeID: "node-a",
		MediaServer: configs.MediaServerConfig{
			URL:         "http://media-a.internal:8080",
			Status:      "healthy",
			MaxSessions: 10,
		},
	})

	if options.ID != "node-a" {
		t.Fatalf("options.ID = %q, want node-a", options.ID)
	}
	if options.URL != "http://media-a.internal:8080" {
		t.Fatalf("options.URL = %q", options.URL)
	}
	if options.Status != entity.MediaServerStatusHealthy {
		t.Fatalf("options.Status = %q, want healthy", options.Status)
	}
	if options.MaxSessions != 10 {
		t.Fatalf("options.MaxSessions = %d, want 10", options.MaxSessions)
	}
}
