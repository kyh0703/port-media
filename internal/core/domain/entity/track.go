package entity

import (
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type Track struct {
	ID        vo.TrackID
	Kind      vo.TrackKind
	State     vo.TrackState
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewTrack(id vo.TrackID, kind vo.TrackKind, now time.Time) Track {
	return Track{
		ID:        id,
		Kind:      kind,
		State:     vo.TrackStatePending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (t *Track) SetState(state vo.TrackState, now time.Time) {
	t.State = state
	t.UpdatedAt = now
}
