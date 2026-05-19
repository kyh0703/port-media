package configs

import (
	"fmt"
	"strings"

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

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
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
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("realtime.stun_urls", []string{"stun:stun.l.google.com:19302"})
	v.SetDefault("realtime.room_idle_timeout", "2m")
	v.SetDefault("realtime.ice_gathering_timeout", "5s")
	v.SetDefault("realtime.realtime_event_history_limit", 10)
	v.SetDefault("observability.metrics_enabled", true)
}
