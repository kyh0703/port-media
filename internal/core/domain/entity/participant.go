package entity

import (
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type Participant struct {
	ID           vo.ParticipantID
	Role         vo.ParticipantRole
	State        vo.ConnectionState
	PublishAudio bool
	Tracks       map[vo.TrackID]Track
	JoinedAt     time.Time
	UpdatedAt    time.Time
}

func NewParticipant(id vo.ParticipantID, role vo.ParticipantRole, now time.Time) Participant {
	return Participant{
		ID:        id,
		Role:      role,
		State:     vo.ConnectionStateNew,
		Tracks:    make(map[vo.TrackID]Track),
		JoinedAt:  now,
		UpdatedAt: now,
	}
}

func (p Participant) Clone() Participant {
	clone := p
	clone.Tracks = make(map[vo.TrackID]Track, len(p.Tracks))
	for id, track := range p.Tracks {
		clone.Tracks[id] = track
	}
	return clone
}

func (p *Participant) SetPublishAudio(publishAudio bool, now time.Time) {
	p.PublishAudio = publishAudio
	p.UpdatedAt = now
}

func (p *Participant) AddTrack(track Track, now time.Time) {
	p.Tracks[track.ID] = track
	p.UpdatedAt = now
}

func (p *Participant) SetState(state vo.ConnectionState, now time.Time) {
	p.State = state
	p.UpdatedAt = now
}

func (p *Participant) UpdateTrackState(kind vo.TrackKind, state vo.TrackState, now time.Time) bool {
	for id, track := range p.Tracks {
		if track.Kind != kind {
			continue
		}

		track.SetState(state, now)
		p.Tracks[id] = track
		p.UpdatedAt = now
		return true
	}

	return false
}
