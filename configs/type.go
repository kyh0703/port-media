package configs

import "time"

type Config struct {
	NodeID      int               `env:"NODE_ID,required" validate:"gt=0"`
	App         AppConfig         `envPrefix:"APP_"`
	Server      ServerConfig      `envPrefix:"SERVER_"`
	Log         LogConfig         `envPrefix:"LOG_"`
	OpenAI      OpenAIConfig      `envPrefix:"OPENAI_"`
	Database    DatabaseConfig    `envPrefix:"DATABASE_"`
	Redis       RedisConfig       `envPrefix:"REDIS_"`
	MediaServer MediaServerConfig `envPrefix:"MEDIA_SERVER_"`
	Events      EventsConfig      `envPrefix:"EVENTS_"`
	Realtime    RealtimeConfig    `envPrefix:"REALTIME_"`
}

type AppConfig struct {
	Name    string `env:"NAME" envDefault:"portfoilo-media"`
	Env     string `env:"ENV" envDefault:"dev"`
	Version string `env:"VERSION" envDefault:"0.1.0"`
}

type ServerConfig struct {
	Host         string        `env:"HOST" envDefault:"0.0.0.0"`
	Port         int           `env:"-"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	CORS         CORSConfig    `envPrefix:"CORS_"`
}

type CORSConfig struct {
	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envDefault:"*"`
	AllowedMethods []string `env:"ALLOWED_METHODS" envDefault:"GET,POST,OPTIONS"`
	AllowedHeaders []string `env:"ALLOWED_HEADERS" envDefault:"Authorization,Content-Type,X-Request-Id"`
	ExposeHeaders  []string `env:"EXPOSE_HEADERS" envDefault:"X-Room-Id,X-Participant-Id,X-Request-Id"`
}

type LogConfig struct {
	Level       string `env:"LEVEL" envDefault:"info"`
	Development bool   `env:"DEVELOPMENT" envDefault:"false"`
}

type OpenAIConfig struct {
	RealtimeBaseURL          string   `env:"REALTIME_BASE_URL" envDefault:"https://api.openai.com"`
	RealtimeModel            string   `env:"REALTIME_MODEL" envDefault:"gpt-realtime-2"`
	RealtimeDataChannelLabel string   `env:"REALTIME_DATA_CHANNEL_LABEL" envDefault:"oai-events"`
	RealtimeInitialEvents    []string `env:"REALTIME_INITIAL_EVENTS"`
	APIKey                   string   `env:"API_KEY,notEmpty"`
}

type DatabaseConfig struct {
	URL string `env:"URL" envDefault:"postgres://postgres:postgres@localhost:5432/portfoilo_media?sslmode=disable"`
}

type RedisConfig struct {
	URL string `env:"URL" envDefault:"redis://localhost:6379"`
}

type MediaServerConfig struct {
	URL               string        `env:"URL"`
	Status            string        `env:"STATUS" envDefault:"healthy"`
	HeartbeatEnabled  bool          `env:"HEARTBEAT_ENABLED" envDefault:"true"`
	HeartbeatInterval time.Duration `env:"HEARTBEAT_INTERVAL" envDefault:"10s"`
	HeartbeatTTL      time.Duration `env:"HEARTBEAT_TTL" envDefault:"30s"`
	MaxSessions       int           `env:"MAX_SESSIONS" envDefault:"0"`
}

type EventsConfig struct {
	ConversationStreamEnabled bool   `env:"CONVERSATION_STREAM_ENABLED" envDefault:"true"`
	ConversationStreamName    string `env:"CONVERSATION_STREAM_NAME" envDefault:"media:conversation-events:v1"`
	ConversationStreamMaxLen  int64  `env:"CONVERSATION_STREAM_MAX_LEN" envDefault:"0"`
}

type RealtimeConfig struct {
	STUNURLs                  []string      `env:"STUN_URLS" envDefault:"stun:stun.l.google.com:19302"`
	RoomIdleTimeout           time.Duration `env:"ROOM_IDLE_TIMEOUT" envDefault:"2m"`
	ICEGatheringTimeout       time.Duration `env:"ICE_GATHERING_TIMEOUT" envDefault:"5s"`
	RealtimeEventHistoryLimit int           `env:"EVENT_HISTORY_LIMIT" envDefault:"10"`
}
