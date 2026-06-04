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

type BunRoomRepository struct {
	db *bun.DB
}

func NewBunRoomRepository(db *bun.DB) domainrepo.RoomRepository {
	return &BunRoomRepository{db: db}
}

func (r *BunRoomRepository) Save(ctx context.Context, room entity.Room) error {
	row := toRoomModel(room)
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

func (r *BunRoomRepository) FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.Room, bool, error) {
	var row model.Room
	err := r.db.NewSelect().
		Model(&row).
		Where("session_id = ?", string(sessionID)).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Room{}, false, nil
		}
		return entity.Room{}, false, err
	}

	return toRoomEntity(row), true, nil
}

func (r *BunRoomRepository) Delete(ctx context.Context, roomID vo.RoomID) error {
	_, err := r.db.NewDelete().
		Model((*model.Room)(nil)).
		Where("id = ?", string(roomID)).
		Exec(ctx)
	return err
}

func toRoomModel(room entity.Room) model.Room {
	return model.Room{
		ID:                    string(room.ID),
		SessionID:             string(room.SessionID),
		ConversationID:        string(room.ConversationID),
		UserID:                room.UserID,
		Status:                string(room.Status),
		LastRealtimeEventType: room.LastRealtimeEventType,
		LastRealtimeEventAt:   room.LastRealtimeEventAt,
		CreatedAt:             room.CreatedAt,
		UpdatedAt:             room.UpdatedAt,
	}
}

func toRoomEntity(row model.Room) entity.Room {
	return entity.Room{
		ID:                    vo.RoomID(row.ID),
		SessionID:             vo.SessionID(row.SessionID),
		ConversationID:        vo.ConversationID(row.ConversationID),
		UserID:                row.UserID,
		Status:                vo.RoomStatus(row.Status),
		Participants:          make(map[vo.ParticipantID]entity.Participant),
		LastRealtimeEventType: row.LastRealtimeEventType,
		LastRealtimeEventAt:   row.LastRealtimeEventAt,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}
