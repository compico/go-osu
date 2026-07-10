-- +goose Up
-- Группа (mapset) — соответствует Directory на бэкенде.
-- Одному BeatmapSetID соответствует несколько строк в beatmaps.
CREATE TABLE IF NOT EXISTS beatmapsets (
    beatmap_set_id  INTEGER PRIMARY KEY,
    song_title      TEXT NOT NULL DEFAULT '',
    song_title_uni  TEXT NOT NULL DEFAULT '',
    artist_name     TEXT NOT NULL DEFAULT '',
    artist_name_uni TEXT NOT NULL DEFAULT '',
    creator_name    TEXT NOT NULL DEFAULT '',
    song_source     TEXT NOT NULL DEFAULT '',
    song_tags       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_beatmapsets_artist ON beatmapsets(artist_name);
CREATE INDEX IF NOT EXISTS idx_beatmapsets_title  ON beatmapsets(song_title);

-- +goose Down
DROP INDEX IF EXISTS idx_beatmapsets_title;
DROP INDEX IF EXISTS idx_beatmapsets_artist;
DROP TABLE IF EXISTS beatmapsets;