package configs

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestValidateConfigRequiresNodeID(t *testing.T) {
	err := validateConfig(Config{})
	if err == nil {
		t.Fatal("validateConfig() error is nil, want NODE_ID required error")
	}
}

func TestValidateConfigAcceptsNodeID(t *testing.T) {
	err := validateConfig(Config{NodeID: "node-a"})
	if err != nil {
		t.Fatalf("validateConfig() error = %v", err)
	}
}

func TestBindEnvLoadsNodeID(t *testing.T) {
	t.Setenv("NODE_ID", "node-a")

	v := viper.New()
	if err := bindEnv(v); err != nil {
		t.Fatalf("bindEnv() error = %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if cfg.NodeID != "node-a" {
		t.Fatalf("NodeID = %q, want node-a", cfg.NodeID)
	}
}

func TestBindEnvLoadsRuntimeOverrides(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-api-key")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("LOG_DEVELOPMENT", "true")
	t.Setenv("DATABASE_BUSY_TIMEOUT", "7s")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("MEDIA_SERVER_URL", "http://media.example.test")
	t.Setenv("REALTIME_EVENT_HISTORY_LIMIT", "25")

	v := viper.New()
	if err := bindEnv(v); err != nil {
		t.Fatalf("bindEnv() error = %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if cfg.OpenAI.APIKey != "test-api-key" {
		t.Fatalf("OpenAI.APIKey = %q, want test-api-key", cfg.OpenAI.APIKey)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if !cfg.Log.Development {
		t.Fatal("Log.Development = false, want true")
	}
	if cfg.Database.BusyTimeout != 7*time.Second {
		t.Fatalf("Database.BusyTimeout = %v, want 7s", cfg.Database.BusyTimeout)
	}
	if cfg.Redis.DB != 2 {
		t.Fatalf("Redis.DB = %d, want 2", cfg.Redis.DB)
	}
	if cfg.MediaServer.URL != "http://media.example.test" {
		t.Fatalf("MediaServer.URL = %q, want http://media.example.test", cfg.MediaServer.URL)
	}
	if cfg.Realtime.RealtimeEventHistoryLimit != 25 {
		t.Fatalf("RealtimeEventHistoryLimit = %d, want 25", cfg.Realtime.RealtimeEventHistoryLimit)
	}
}

func TestNewVarsReadsAPPProfile(t *testing.T) {
	t.Setenv("APP_PROFILE", "prod")

	vars := NewVars()
	if vars.Profile != "prod" {
		t.Fatalf("Profile = %q, want prod", vars.Profile)
	}
}
