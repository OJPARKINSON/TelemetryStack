 CREATE TABLE sessions (
     session_id   TEXT PRIMARY KEY,
     track_name   TEXT NOT NULL,
     track_id     TEXT NOT NULL,
     session_name TEXT NOT NULL,
     session_num  TEXT NOT NULL,
     car_id       TEXT NOT NULL,
     max_lap_id   INTEGER NOT NULL DEFAULT 0,
     last_updated TEXT NOT NULL,
     synced_at    TEXT NOT NULL
 );

 CREATE TABLE laps (
     id         INTEGER PRIMARY KEY AUTOINCREMENT,
     session_id TEXT    NOT NULL REFERENCES sessions(session_id),
     lap_id     INTEGER NOT NULL,
     lap_time   REAL,
     tick_count INTEGER NOT NULL DEFAULT 0,
     r2_key     TEXT    NOT NULL,
     synced_at  TEXT    NOT NULL,
     UNIQUE(session_id, lap_id)
 );

 CREATE INDEX idx_laps_session ON laps(session_id);