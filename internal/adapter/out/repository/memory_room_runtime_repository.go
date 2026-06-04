package repository

import (
	"context"
	"sync"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	domainrepo "github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type MemoryRoomRuntimeRepository struct {
	mu    sync.RWMutex
	rooms map[vo.RoomID]entity.Room
}

func NewMemoryRoomRuntimeRepository() domainrepo.RoomRuntimeRepository {
	return &MemoryRoomRuntimeRepository{rooms: make(map[vo.RoomID]entity.Room)}
}

func (r *MemoryRoomRuntimeRepository) Save(ctx context.Context, room entity.Room) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rooms[room.ID] = room
	return nil
}

func (r *MemoryRoomRuntimeRepository) FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.Room, bool, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, room := range r.rooms {
		if room.SessionID == sessionID {
			return room, true, nil
		}
	}

	return entity.Room{}, false, nil
}

func (r *MemoryRoomRuntimeRepository) List(ctx context.Context) ([]entity.Room, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()

	rooms := make([]entity.Room, 0, len(r.rooms))
	for _, room := range r.rooms {
		rooms = append(rooms, room)
	}

	return rooms, nil
}

func (r *MemoryRoomRuntimeRepository) Delete(ctx context.Context, roomID vo.RoomID) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rooms, roomID)
	return nil
}
