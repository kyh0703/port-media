package configs

import (
	"os"
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
	t.Setenv("DB_PATH", "./data/test.sqlite")
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
	if cfg.DBPath != "./data/test.sqlite" {
		t.Fatalf("DBPath = %q, want ./data/test.sqlite", cfg.DBPath)
	}
	if cfg.Database.URL != "file:./data/test.sqlite?cache=shared" {
		t.Fatalf("Database.URL = %q, want file:./data/test.sqlite?cache=shared", cfg.Database.URL)
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

func TestNewConfigLoadsDotEnvFile(t *testing.T) {
	restoreEnv(t, "NODE_ID")
	restoreEnv(t, "OPENAI_API_KEY")
	restoreEnv(t, "SERVER_PORT")
	if err := os.Unsetenv("NODE_ID"); err != nil {
		t.Fatalf("Unsetenv(NODE_ID) error = %v", err)
	}
	if err := os.Unsetenv("OPENAI_API_KEY"); err != nil {
		t.Fatalf("Unsetenv(OPENAI_API_KEY) error = %v", err)
	}
	if err := os.Unsetenv("SERVER_PORT"); err != nil {
		t.Fatalf("Unsetenv(SERVER_PORT) error = %v", err)
	}

	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(".env", []byte("NODE_ID=7\nOPENAI_API_KEY=dot-env-key\nSERVER_PORT=7070\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.env) error = %v", err)
	}

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() error = %v", err)
	}
	if cfg.NodeID != 7 {
		t.Fatalf("NodeID = %d, want 7", cfg.NodeID)
	}
	if cfg.OpenAI.APIKey != "dot-env-key" {
		t.Fatalf("OpenAI.APIKey = %q, want dot-env-key", cfg.OpenAI.APIKey)
	}
	if cfg.Server.Port != 7070 {
		t.Fatalf("Server.Port = %d, want 7070", cfg.Server.Port)
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
	if !containsString(cfg.Server.CORS.AllowedHeaders, "X-Request-Id") {
		t.Fatalf("AllowedHeaders = %#v, want X-Request-Id", cfg.Server.CORS.AllowedHeaders)
	}
	if !containsString(cfg.Server.CORS.ExposeHeaders, "X-Request-Id") {
		t.Fatalf("ExposeHeaders = %#v, want X-Request-Id", cfg.Server.CORS.ExposeHeaders)
	}
	if cfg.Redis.URL != "redis://localhost:6379" {
		t.Fatalf("Redis.URL = %q, want redis://localhost:6379", cfg.Redis.URL)
	}
	if cfg.DBPath != "./data/portfoilo_media.sqlite" {
		t.Fatalf("DBPath = %q, want ./data/portfoilo_media.sqlite", cfg.DBPath)
	}
	if cfg.Database.URL != "file:./data/portfoilo_media.sqlite?cache=shared" {
		t.Fatalf("Database.URL = %q, want file:./data/portfoilo_media.sqlite?cache=shared", cfg.Database.URL)
	}
}

func TestNewConfigKeepsExplicitDatabaseURL(t *testing.T) {
	t.Setenv("NODE_ID", "1")
	t.Setenv("OPENAI_API_KEY", "test-api-key")
	t.Setenv("DB_PATH", "./data/ignored.sqlite")
	t.Setenv("DATABASE_URL", "file:custom.db?cache=shared")

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() error = %v", err)
	}
	if cfg.Database.URL != "file:custom.db?cache=shared" {
		t.Fatalf("Database.URL = %q, want file:custom.db?cache=shared", cfg.Database.URL)
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

func restoreEnv(t *testing.T, key string) {
	t.Helper()

	value, ok := os.LookupEnv(key)
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
