package lifecycle

import (
	"fmt"
	"strings"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	sessionservice "github.com/kyh0703/portfoilo-media/internal/core/service/session"
)

func NewMediaServerStateReporterOptions(cfg *configs.Config) sessionservice.MediaServerStateReporterOptions {
	return sessionservice.MediaServerStateReporterOptions{
		ID:          mediaServerID(cfg),
		URL:         mediaServerURL(cfg),
		Status:      mediaServerStatus(cfg),
		MaxSessions: mediaServerMaxSessions(cfg),
	}
}

func mediaServerID(cfg *configs.Config) string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.NodeID)
}

func mediaServerURL(cfg *configs.Config) string {
	if cfg == nil {
		return "http://localhost:8080"
	}
	if url := strings.TrimSpace(cfg.MediaServer.URL); url != "" {
		return url
	}
	host := strings.TrimSpace(cfg.Server.Host)
	if host == "" || host == "0.0.0.0" {
		host = "localhost"
	}
	port := cfg.Server.Port
	if port <= 0 {
		port = 8080
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

func mediaServerStatus(cfg *configs.Config) entity.MediaServerStatus {
	if cfg == nil {
		return entity.MediaServerStatusHealthy
	}
	status := strings.TrimSpace(cfg.MediaServer.Status)
	if status == "" {
		return entity.MediaServerStatusHealthy
	}
	return entity.MediaServerStatus(status)
}

func mediaServerMaxSessions(cfg *configs.Config) int {
	if cfg == nil {
		return 0
	}
	return cfg.MediaServer.MaxSessions
}
