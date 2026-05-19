package vo

type SessionStatus string
type RoomStatus string
type ConnectionState string
type TrackState string

const (
	SessionStatusCreated SessionStatus = "created"
	SessionStatusActive  SessionStatus = "active"
	SessionStatusEnded   SessionStatus = "ended"
	SessionStatusFailed  SessionStatus = "failed"
)

const (
	RoomStatusCreated RoomStatus = "created"
	RoomStatusActive  RoomStatus = "active"
	RoomStatusClosed  RoomStatus = "closed"
	RoomStatusFailed  RoomStatus = "failed"
)

const (
	ConnectionStateNew          ConnectionState = "new"
	ConnectionStateConnecting   ConnectionState = "connecting"
	ConnectionStateConnected    ConnectionState = "connected"
	ConnectionStateDisconnected ConnectionState = "disconnected"
	ConnectionStateFailed       ConnectionState = "failed"
	ConnectionStateClosed       ConnectionState = "closed"
)

const (
	TrackStatePending TrackState = "pending"
	TrackStateActive  TrackState = "active"
	TrackStateEnded   TrackState = "ended"
	TrackStateFailed  TrackState = "failed"
)
