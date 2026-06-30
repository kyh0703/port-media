package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Room struct {
	bun.BaseModel `bun:"table:rooms,alias:r"`

	ID                   string    `bun:"id,pk,type:text"`
	SessionID            string    `bun:"session_id,notnull,type:text"`
	ConversationID       string    `bun:"conversation_id,notnull,type:text"`
	UserID               string    `bun:"user_id,type:text"`
	Status               string    `bun:"status,notnull"`
	LastRuntimeEventType string    `bun:"last_runtime_event_type,type:text"`
	LastRuntimeEventAt   time.Time `bun:"last_runtime_event_at,nullzero"`
	CreatedAt            time.Time `bun:"created_at,notnull"`
	UpdatedAt            time.Time `bun:"updated_at,notnull"`
}
