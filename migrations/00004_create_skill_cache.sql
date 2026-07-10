-- +goose Up
CREATE TABLE IF NOT EXISTS skill_cache (
    beatmap_id INTEGER NOT NULL REFERENCES beatmaps(beatmap_id) ON DELETE CASCADE,
    mods       INTEGER NOT NULL,
    md5_hash   BLOB NOT NULL, -- 16 raw bytes, copied from beatmaps.md5_hash at compute time

    stamina   REAL NOT NULL DEFAULT 0,
    tenacity  REAL NOT NULL DEFAULT 0,
    agility   REAL NOT NULL DEFAULT 0,
    precision REAL NOT NULL DEFAULT 0,
    reading   REAL NOT NULL DEFAULT 0,
    memory    REAL NOT NULL DEFAULT 0,
    accuracy  REAL NOT NULL DEFAULT 0,
    reaction  REAL NOT NULL DEFAULT 0,

    PRIMARY KEY (beatmap_id, mods)
);

CREATE INDEX IF NOT EXISTS idx_skill_cache_mods ON skill_cache(mods);

-- +goose Down
DROP INDEX IF EXISTS idx_skill_cache_mods;
DROP TABLE IF EXISTS skill_cache;