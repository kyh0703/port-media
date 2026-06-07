package session

import "time"

type RealtimeEventResult struct {
	Type string
	At   time.Time
}
