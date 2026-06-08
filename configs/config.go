package configs

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

func NewConfig(vars Vars) (*Config, error) {
	v := viper.New()
	v.SetConfigName(vars.Profile)
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := bindEnv(v); err != nil {
		return nil, err
	}

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func bindEnv(v *viper.Viper) error {
	envBindings := map[string]string{
		"node_id":                               "NODE_ID",
		"app.name":                              "APP_NAME",
		"app.env":                               "APP_ENV",
		"app.version":                           "APP_VERSION",
		"server.host":                           "SERVER_HOST",
		"server.port":                           "SERVER_PORT",
		"server.read_timeout":                   "SERVER_READ_TIMEOUT",
		"server.write_timeout":                  "SERVER_WRITE_TIMEOUT",
		"log.level":                             "LOG_LEVEL",
		"log.development":                       "LOG_DEVELOPMENT",
		"openai.realtime_base_url":              "OPENAI_REALTIME_BASE_URL",
		"openai.realtime_model":                 "OPENAI_REALTIME_MODEL",
		"openai.realtime_data_channel_label":    "OPENAI_REALTIME_DATA_CHANNEL_LABEL",
		"openai.api_key":                        "OPENAI_API_KEY",
		"database.dsn":                          "DATABASE_DSN",
		"database.busy_timeout":                 "DATABASE_BUSY_TIMEOUT",
		"database.max_open_conns":               "DATABASE_MAX_OPEN_CONNS",
		"database.max_idle_conns":               "DATABASE_MAX_IDLE_CONNS",
		"redis.url":                             "REDIS_URL",
		"media_server.url":                      "MEDIA_SERVER_URL",
		"media_server.status":                   "MEDIA_SERVER_STATUS",
		"media_server.heartbeat_enabled":        "MEDIA_SERVER_HEARTBEAT_ENABLED",
		"media_server.heartbeat_interval":       "MEDIA_SERVER_HEARTBEAT_INTERVAL",
		"media_server.heartbeat_ttl":            "MEDIA_SERVER_HEARTBEAT_TTL",
		"media_server.max_sessions":             "MEDIA_SERVER_MAX_SESSIONS",
		"events.conversation_stream_enabled":    "EVENTS_CONVERSATION_STREAM_ENABLED",
		"events.conversation_stream_name":       "EVENTS_CONVERSATION_STREAM_NAME",
		"events.conversation_stream_max_len":    "EVENTS_CONVERSATION_STREAM_MAX_LEN",
		"realtime.room_idle_timeout":            "REALTIME_ROOM_IDLE_TIMEOUT",
		"realtime.ice_gathering_timeout":        "REALTIME_ICE_GATHERING_TIMEOUT",
		"realtime.realtime_event_history_limit": "REALTIME_EVENT_HISTORY_LIMIT",
	}

	for key, envName := range envBindings {
		if err := v.BindEnv(key, envName); err != nil {
			return fmt.Errorf("bind %s: %w", envName, err)
		}
	}
	return nil
}

func validateConfig(cfg Config) error {
	cfg.NodeID = strings.TrimSpace(cfg.NodeID)
	if err := validator.New().Struct(cfg); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "portfoilo-media")
	v.SetDefault("app.env", "dev")
	v.SetDefault("app.version", "0.1.0")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.cors.allowed_origins", []string{"*"})
	v.SetDefault("server.cors.allowed_methods", []string{"GET", "POST", "OPTIONS"})
	v.SetDefault("server.cors.allowed_headers", []string{"Authorization", "Content-Type"})
	v.SetDefault("server.cors.expose_headers", []string{"X-Room-Id", "X-Participant-Id"})
	v.SetDefault("log.level", "info")
	v.SetDefault("log.development", false)
	v.SetDefault("openai.realtime_base_url", "https://api.openai.com")
	v.SetDefault("openai.realtime_model", "gpt-realtime-2")
	v.SetDefault("openai.realtime_data_channel_label", "oai-events")
	v.SetDefault("openai.realtime_initial_events", []string{})
	v.SetDefault("database.dsn", "file:portfoilo_media.db?cache=shared")
	v.SetDefault("database.busy_timeout", "5s")
	v.SetDefault("database.max_open_conns", 10)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("redis.url", "redis://localhost:6379")
	v.SetDefault("media_server.url", "")
	v.SetDefault("media_server.status", "healthy")
	v.SetDefault("media_server.heartbeat_enabled", true)
	v.SetDefault("media_server.heartbeat_interval", "10s")
	v.SetDefault("media_server.heartbeat_ttl", "30s")
	v.SetDefault("media_server.max_sessions", 0)
	v.SetDefault("events.conversation_stream_enabled", true)
	v.SetDefault("events.conversation_stream_name", "media:conversation-events:v1")
	v.SetDefault("events.conversation_stream_max_len", 0)
	v.SetDefault("realtime.stun_urls", []string{"stun:stun.l.google.com:19302"})
	v.SetDefault("realtime.room_idle_timeout", "2m")
	v.SetDefault("realtime.ice_gathering_timeout", "5s")
	v.SetDefault("realtime.realtime_event_history_limit", 10)
}
