package vo

type ParticipantRole string
type TrackKind string

const (
	ParticipantRoleUser    ParticipantRole = "user"
	ParticipantRoleAgent   ParticipantRole = "agent"
	ParticipantRoleMonitor ParticipantRole = "monitor"
)

const (
	TrackKindAudio TrackKind = "audio"
	TrackKindVideo TrackKind = "video"
)
