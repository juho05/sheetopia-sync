-- +migrate Up

-- exercise_id, routine_id and routine_entry_id have no foreign keys on purpose because a practice record must survive what it points at
CREATE TABLE practice_records (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    id TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    exercise_id TEXT NOT NULL,
    routine_id TEXT,
    routine_entry_id TEXT,
    metadata_json BLOB NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, id)
);

CREATE TABLE deleted_practice_records (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    record_id TEXT NOT NULL,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, record_id)
);

CREATE INDEX practice_records_changed_index ON practice_records (user, changed);

DROP TABLE practice_session_entries;
DROP TABLE practice_sessions;
DROP TABLE deleted_practice_sessions;

-- +migrate Down

DROP TABLE deleted_practice_records;
DROP TABLE practice_records;

-- the session data is gone, the empty tables keep the earlier down migration working
CREATE TABLE practice_sessions (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    id TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME,
    routine_id TEXT,
    metadata_json BLOB NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, id)
);

CREATE TABLE practice_session_entries (
    user TEXT NOT NULL,
    id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    exercise_id TEXT NOT NULL,
    routine_entry_id TEXT,
    metadata_json BLOB NOT NULL,
    PRIMARY KEY (user, id),
    FOREIGN KEY (user, session_id) REFERENCES practice_sessions(user, id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE deleted_practice_sessions (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, session_id)
);

CREATE INDEX practice_session_entries_session_index ON practice_session_entries (user, session_id);
