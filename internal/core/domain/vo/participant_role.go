package vo

type ParticipantRole string
type TrackKind string

const (
	ParticipantRoleUser     ParticipantRole = "user"
	ParticipantRoleAgent    ParticipantRole = "agent"
	ParticipantRoleRecorder ParticipantRole = "recorder"
	ParticipantRoleSIP      ParticipantRole = "sip"
	ParticipantRoleMonitor  ParticipantRole = "monitor"
	ParticipantRoleService  ParticipantRole = "service"
)

const (
	TrackKindAudio TrackKind = "audio"
	TrackKindVideo TrackKind = "video"
)
