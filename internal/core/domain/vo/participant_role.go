package vo

type ParticipantRole string
type TrackKind string

const (
	ParticipantRoleClient      ParticipantRole = "client"
	ParticipantRoleOpenAIAgent ParticipantRole = "openai_agent"
	ParticipantRoleMonitor     ParticipantRole = "monitor"
)

const (
	TrackKindAudio TrackKind = "audio"
	TrackKindVideo TrackKind = "video"
)
