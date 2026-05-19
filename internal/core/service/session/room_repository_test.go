package session

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type memoryRoomRepositoryForTest struct {
	rooms map[vo.RoomID]entity.Room
}

func newMemoryRoomRepositoryForTest() *memoryRoomRepositoryForTest {
	return &memoryRoomRepositoryForTest{rooms: make(map[vo.RoomID]entity.Room)}
}

func (r *memoryRoomRepositoryForTest) Save(ctx context.Context, room entity.Room) error {
	_ = ctx
	r.rooms[room.ID] = room
	return nil
}

func (r *memoryRoomRepositoryForTest) FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.Room, bool, error) {
	_ = ctx
	for _, room := range r.rooms {
		if room.SessionID == sessionID {
			return room, true, nil
		}
	}
	return entity.Room{}, false, nil
}

func (r *memoryRoomRepositoryForTest) List(ctx context.Context) ([]entity.Room, error) {
	_ = ctx
	rooms := make([]entity.Room, 0, len(r.rooms))
	for _, room := range r.rooms {
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (r *memoryRoomRepositoryForTest) Delete(ctx context.Context, roomID vo.RoomID) error {
	_ = ctx
	delete(r.rooms, roomID)
	return nil
}
