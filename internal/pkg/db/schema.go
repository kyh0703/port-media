package db

const Schema = `
CREATE TABLE IF NOT EXISTS rooms (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  conversation_id TEXT NOT NULL,
  user_id TEXT,
  status TEXT NOT NULL,
  last_realtime_event_type TEXT,
  last_realtime_event_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rooms_session_id ON rooms(session_id);
`
