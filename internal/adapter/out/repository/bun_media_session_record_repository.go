package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kyh0703/portfoilo-media/internal/adapter/out/repository/model"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	domainrepo "github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/uptrace/bun"
)

type BunMediaSessionRecordRepository struct {
	db *bun.DB
}

func NewBunMediaSessionRecordRepository(db *bun.DB) domainrepo.MediaSessionRecordRepository {
	return &BunMediaSessionRecordRepository{db: db}
}

func (r *BunMediaSessionRecordRepository) Save(ctx context.Context, record entity.MediaSessionRecord) error {
	row := toMediaSessionRecordModel(record)
	_, err := r.db.NewInsert().
		Model(&row).
		On("CONFLICT (id) DO UPDATE").
		Set("session_id = EXCLUDED.session_id").
		Set("conversation_id = EXCLUDED.conversation_id").
		Set("user_id = EXCLUDED.user_id").
		Set("status = EXCLUDED.status").
		Set("last_realtime_event_type = EXCLUDED.last_realtime_event_type").
		Set("last_realtime_event_at = EXCLUDED.last_realtime_event_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (r *BunMediaSessionRecordRepository) FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.MediaSessionRecord, bool, error) {
	var row model.Room
	err := r.db.NewSelect().
		Model(&row).
		Where("session_id = ?", string(sessionID)).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.MediaSessionRecord{}, false, nil
		}
		return entity.MediaSessionRecord{}, false, err
	}

	return toMediaSessionRecord(row), true, nil
}

func (r *BunMediaSessionRecordRepository) Delete(ctx context.Context, roomID vo.RoomID) error {
	_, err := r.db.NewDelete().
		Model((*model.Room)(nil)).
		Where("id = ?", string(roomID)).
		Exec(ctx)
	return err
}

func toMediaSessionRecordModel(record entity.MediaSessionRecord) model.Room {
	return model.Room{
		ID:                    string(record.ID),
		SessionID:             string(record.SessionID),
		ConversationID:        string(record.ConversationID),
		UserID:                record.UserID,
		Status:                string(record.Status),
		LastRealtimeEventType: record.LastRealtimeEventType,
		LastRealtimeEventAt:   record.LastRealtimeEventAt,
		CreatedAt:             record.CreatedAt,
		UpdatedAt:             record.UpdatedAt,
	}
}

func toMediaSessionRecord(row model.Room) entity.MediaSessionRecord {
	return entity.MediaSessionRecord{
		ID:                    vo.RoomID(row.ID),
		SessionID:             vo.SessionID(row.SessionID),
		ConversationID:        vo.ConversationID(row.ConversationID),
		UserID:                row.UserID,
		Status:                vo.RoomStatus(row.Status),
		LastRealtimeEventType: row.LastRealtimeEventType,
		LastRealtimeEventAt:   row.LastRealtimeEventAt,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}
