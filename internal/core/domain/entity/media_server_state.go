package entity

import "time"

type MediaServerStatus string

const (
	MediaServerStatusHealthy MediaServerStatus = "healthy"
	MediaServerStatusOffline MediaServerStatus = "offline"
)

type MediaServerState struct {
	ID                 string
	URL                string
	Status             MediaServerStatus
	ActiveRooms        int
	ActiveSessions     int
	ActiveParticipants int
	ActiveTracks       int
	MaxSessions        int
	UpdatedAt          time.Time
}
