package session

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type memoryRoomRuntimeRepositoryForTest struct {
	rooms map[vo.RoomID]entity.Room
}

func newMemoryRoomRuntimeRepositoryForTest() *memoryRoomRuntimeRepositoryForTest {
	return &memoryRoomRuntimeRepositoryForTest{rooms: make(map[vo.RoomID]entity.Room)}
}

func (r *memoryRoomRuntimeRepositoryForTest) Save(ctx context.Context, room entity.Room) error {
	_ = ctx
	r.rooms[room.ID] = room
	return nil
}

func (r *memoryRoomRuntimeRepositoryForTest) FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.Room, bool, error) {
	_ = ctx
	for _, room := range r.rooms {
		if room.SessionID == sessionID {
			return room, true, nil
		}
	}
	return entity.Room{}, false, nil
}

func (r *memoryRoomRuntimeRepositoryForTest) List(ctx context.Context) ([]entity.Room, error) {
	_ = ctx
	rooms := make([]entity.Room, 0, len(r.rooms))
	for _, room := range r.rooms {
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (r *memoryRoomRuntimeRepositoryForTest) Delete(ctx context.Context, roomID vo.RoomID) error {
	_ = ctx
	delete(r.rooms, roomID)
	return nil
}

type memoryMediaSessionRecordRepositoryForTest struct {
	records map[vo.RoomID]entity.MediaSessionRecord
}

func newMemoryMediaSessionRecordRepositoryForTest() *memoryMediaSessionRecordRepositoryForTest {
	return &memoryMediaSessionRecordRepositoryForTest{records: make(map[vo.RoomID]entity.MediaSessionRecord)}
}

func (r *memoryMediaSessionRecordRepositoryForTest) Save(ctx context.Context, record entity.MediaSessionRecord) error {
	_ = ctx
	r.records[record.ID] = record
	return nil
}

func (r *memoryMediaSessionRecordRepositoryForTest) FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.MediaSessionRecord, bool, error) {
	_ = ctx
	for _, record := range r.records {
		if record.SessionID == sessionID {
			return record, true, nil
		}
	}
	return entity.MediaSessionRecord{}, false, nil
}

func (r *memoryMediaSessionRecordRepositoryForTest) Delete(ctx context.Context, roomID vo.RoomID) error {
	_ = ctx
	delete(r.records, roomID)
	return nil
}
