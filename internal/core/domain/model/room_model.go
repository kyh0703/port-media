package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Room struct {
	bun.BaseModel `bun:"table:rooms,alias:r"`

	ID                    string    `bun:"id,pk,type:text"`
	SessionID             string    `bun:"session_id,notnull,type:text"`
	ConversationID        string    `bun:"conversation_id,notnull,type:text"`
	UserID                string    `bun:"user_id,type:text"`
	Status                string    `bun:"status,notnull"`
	LastRealtimeEventType string    `bun:"last_realtime_event_type,type:text"`
	LastRealtimeEventAt   time.Time `bun:"last_realtime_event_at,nullzero"`
	CreatedAt             time.Time `bun:"created_at,notnull"`
	UpdatedAt             time.Time `bun:"updated_at,notnull"`
}
