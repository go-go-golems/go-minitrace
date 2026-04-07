-- 05-sqlite-schema.sql
-- Raw SQLite schema for the annotations.db file.
-- Run with: sqlite3 annotations.db < 05-sqlite-schema.sql

CREATE TABLE IF NOT EXISTS annotations (
    id              TEXT PRIMARY KEY,
    session_id      TEXT NOT NULL,
    annotator       TEXT NOT NULL,
    scope_type      TEXT NOT NULL,
    target_id       TEXT NOT NULL,
    category        TEXT NOT NULL,
    title           TEXT NOT NULL,
    detail          TEXT NOT NULL DEFAULT '',
    tags            TEXT NOT NULL DEFAULT '[]',
    taxonomy_m      TEXT NOT NULL DEFAULT '[]',
    taxonomy_mast   TEXT NOT NULL DEFAULT '[]',
    taxonomy_tm     TEXT NOT NULL DEFAULT '[]',
    classification  TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_anno_session   ON annotations(session_id);
CREATE INDEX IF NOT EXISTS idx_anno_scope    ON annotations(scope_type, target_id);
CREATE INDEX IF NOT EXISTS idx_anno_category ON annotations(category);
CREATE INDEX IF NOT EXISTS idx_anno_annot    ON annotations(annotator);

CREATE TABLE IF NOT EXISTS sync_state (
    session_id       TEXT PRIMARY KEY,
    last_synced_at   TEXT,
    annotation_count  INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sessions (
    session_id    TEXT PRIMARY KEY,
    file_path     TEXT NOT NULL,
    title         TEXT,
    framework     TEXT,
    model         TEXT,
    loaded_at     TEXT
);
