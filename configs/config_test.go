package configs

import (
	"testing"
)

func TestValidateConfigRequiresNodeID(t *testing.T) {
	err := validateConfig(Config{})
	if err == nil {
		t.Fatal("validateConfig() error is nil, want NODE_ID required error")
	}
}

func TestValidateConfigAcceptsNodeID(t *testing.T) {
	err := validateConfig(Config{NodeID: 1})
	if err != nil {
		t.Fatalf("validateConfig() error = %v", err)
	}
}

func TestNewConfigLoadsEnvOnlyConfig(t *testing.T) {
	t.Setenv("NODE_ID", "1")
	t.Setenv("OPENAI_API_KEY", "test-api-key")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	t.Setenv("LOG_DEVELOPMENT", "true")
	t.Setenv("DATABASE_URL", "file:test.db?cache=shared")
	t.Setenv("REDIS_URL", "redis://localhost:6379/2")
	t.Setenv("MEDIA_SERVER_URL", "http://media.example.test")
	t.Setenv("REALTIME_STUN_URLS", "stun:one.example.test:19302,stun:two.example.test:19302")
	t.Setenv("REALTIME_EVENT_HISTORY_LIMIT", "25")

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() error = %v", err)
	}
	if cfg.NodeID != 1 {
		t.Fatalf("NodeID = %d, want 1", cfg.NodeID)
	}
	if cfg.OpenAI.APIKey != "test-api-key" {
		t.Fatalf("OpenAI.APIKey = %q, want test-api-key", cfg.OpenAI.APIKey)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if len(cfg.Server.CORS.AllowedOrigins) != 2 || cfg.Server.CORS.AllowedOrigins[1] != "http://localhost:5173" {
		t.Fatalf("AllowedOrigins = %#v, want local origins", cfg.Server.CORS.AllowedOrigins)
	}
	if !cfg.Log.Development {
		t.Fatal("Log.Development = false, want true")
	}
	if cfg.Database.URL != "file:test.db?cache=shared" {
		t.Fatalf("Database.URL = %q, want file:test.db?cache=shared", cfg.Database.URL)
	}
	if cfg.Redis.URL != "redis://localhost:6379/2" {
		t.Fatalf("Redis.URL = %q, want redis://localhost:6379/2", cfg.Redis.URL)
	}
	if cfg.MediaServer.URL != "http://media.example.test" {
		t.Fatalf("MediaServer.URL = %q, want http://media.example.test", cfg.MediaServer.URL)
	}
	if len(cfg.Realtime.STUNURLs) != 2 || cfg.Realtime.STUNURLs[0] != "stun:one.example.test:19302" {
		t.Fatalf("STUNURLs = %#v, want env STUN URLs", cfg.Realtime.STUNURLs)
	}
	if cfg.Realtime.RealtimeEventHistoryLimit != 25 {
		t.Fatalf("RealtimeEventHistoryLimit = %d, want 25", cfg.Realtime.RealtimeEventHistoryLimit)
	}
}

func TestNewConfigLoadsDefaults(t *testing.T) {
	t.Setenv("NODE_ID", "1")
	t.Setenv("OPENAI_API_KEY", "test-api-key")

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() error = %v", err)
	}
	if cfg.App.Name != "portfoilo-media" {
		t.Fatalf("App.Name = %q, want portfoilo-media", cfg.App.Name)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if len(cfg.Server.CORS.AllowedOrigins) != 1 || cfg.Server.CORS.AllowedOrigins[0] != "*" {
		t.Fatalf("AllowedOrigins = %#v, want wildcard default", cfg.Server.CORS.AllowedOrigins)
	}
	if cfg.Redis.URL != "redis://localhost:6379" {
		t.Fatalf("Redis.URL = %q, want redis://localhost:6379", cfg.Redis.URL)
	}
}

func TestNewConfigRejectsMissingRequiredEnv(t *testing.T) {
	t.Setenv("NODE_ID", "")
	t.Setenv("OPENAI_API_KEY", "")

	_, err := NewConfig()
	if err == nil {
		t.Fatal("NewConfig() error is nil, want required env error")
	}
}

func TestNewConfigRejectsNonNumericNodeID(t *testing.T) {
	t.Setenv("NODE_ID", "node-a")
	t.Setenv("OPENAI_API_KEY", "test-api-key")

	_, err := NewConfig()
	if err == nil {
		t.Fatal("NewConfig() error is nil, want NODE_ID parse error")
	}
}
