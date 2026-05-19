package configs

import "time"

type Config struct {
	App           AppConfig           `mapstructure:"app"`
	Server        ServerConfig        `mapstructure:"server"`
	Log           LogConfig           `mapstructure:"log"`
	OpenAI        OpenAIConfig        `mapstructure:"openai"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Realtime      RealtimeConfig      `mapstructure:"realtime"`
	Observability ObservabilityConfig `mapstructure:"observability"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Version string `mapstructure:"version"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	CORS         CORSConfig    `mapstructure:"cors"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	ExposeHeaders  []string `mapstructure:"expose_headers"`
}

type LogConfig struct {
	Level       string `mapstructure:"level"`
	Development bool   `mapstructure:"development"`
}

type OpenAIConfig struct {
	RealtimeBaseURL          string   `mapstructure:"realtime_base_url"`
	RealtimeModel            string   `mapstructure:"realtime_model"`
	RealtimeDataChannelLabel string   `mapstructure:"realtime_data_channel_label"`
	RealtimeInitialEvents    []string `mapstructure:"realtime_initial_events"`
	APIKey                   string   `mapstructure:"api_key"`
}

type DatabaseConfig struct {
	DSN          string `mapstructure:"dsn"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type RealtimeConfig struct {
	STUNURLs                  []string      `mapstructure:"stun_urls"`
	RoomIdleTimeout           time.Duration `mapstructure:"room_idle_timeout"`
	ICEGatheringTimeout       time.Duration `mapstructure:"ice_gathering_timeout"`
	RealtimeEventHistoryLimit int           `mapstructure:"realtime_event_history_limit"`
}

type ObservabilityConfig struct {
	MetricsEnabled bool `mapstructure:"metrics_enabled"`
}
