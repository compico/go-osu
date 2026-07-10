-- +goose Up
CREATE TABLE IF NOT EXISTS beatmaps (
    beatmap_id      INTEGER PRIMARY KEY,
    beatmap_set_id  INTEGER NOT NULL REFERENCES beatmapsets(beatmap_set_id) ON DELETE CASCADE,

    difficulty                TEXT NOT NULL DEFAULT '',
    md5_hash                  BLOB NOT NULL, -- 16 raw bytes, not hex text — see model.MD5Hash
    folder_name                TEXT NOT NULL DEFAULT '',
    name_of_the_osu_file        TEXT NOT NULL DEFAULT '',
    audio_file_name             TEXT NOT NULL DEFAULT '',
    title_font                 TEXT NOT NULL DEFAULT '',

    approach_rate              REAL NOT NULL DEFAULT 0,
    circle_size                 REAL NOT NULL DEFAULT 0,
    hp_drain                    REAL NOT NULL DEFAULT 0,
    overall_difficulty          REAL NOT NULL DEFAULT 0,
    slider_velocity             REAL NOT NULL DEFAULT 0,
    stack_leniency               REAL NOT NULL DEFAULT 0,

    bpm                         REAL NOT NULL DEFAULT 0,
    stars_nomod                 REAL NOT NULL DEFAULT 0,

    drain_time                  INTEGER NOT NULL DEFAULT 0,
    total_time                  INTEGER NOT NULL DEFAULT 0,
    preview_audio_time           INTEGER NOT NULL DEFAULT 0,
    thread_id                   INTEGER NOT NULL DEFAULT 0,
    last_modification_time       INTEGER NOT NULL DEFAULT 0,
    last_checked_osu_repo         INTEGER NOT NULL DEFAULT 0,
    last_modification            INTEGER NOT NULL DEFAULT 0,
    last_play                   INTEGER NOT NULL DEFAULT 0,

    number_of_hitcircles          INTEGER NOT NULL DEFAULT 0,
    number_of_sliders            INTEGER NOT NULL DEFAULT 0,
    number_of_spinners            INTEGER NOT NULL DEFAULT 0,
    local_offset                INTEGER NOT NULL DEFAULT 0,
    online_offset                INTEGER NOT NULL DEFAULT 0,

    mode                        INTEGER NOT NULL DEFAULT 0,
    ranked_status                INTEGER NOT NULL DEFAULT 0,
    grade_achieved_osu            INTEGER NOT NULL DEFAULT 0,
    grade_achieved_taiko          INTEGER NOT NULL DEFAULT 0,
    grade_achieved_ctb           INTEGER NOT NULL DEFAULT 0,
    grade_achieved_mania          INTEGER NOT NULL DEFAULT 0,
    mania_scroll_speed            INTEGER NOT NULL DEFAULT 0,

    unplayed                    INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_beatmaps_md5_hash   ON beatmaps(md5_hash);
CREATE INDEX IF NOT EXISTS idx_beatmaps_beatmap_set_id ON beatmaps(beatmap_set_id);
CREATE INDEX IF NOT EXISTS idx_beatmaps_stars_nomod    ON beatmaps(stars_nomod);
CREATE INDEX IF NOT EXISTS idx_beatmaps_bpm            ON beatmaps(bpm);
CREATE INDEX IF NOT EXISTS idx_beatmaps_ar             ON beatmaps(approach_rate);
CREATE INDEX IF NOT EXISTS idx_beatmaps_mode           ON beatmaps(mode);

-- +goose Down
DROP INDEX IF EXISTS idx_beatmaps_mode;
DROP INDEX IF EXISTS idx_beatmaps_ar;
DROP INDEX IF EXISTS idx_beatmaps_bpm;
DROP INDEX IF EXISTS idx_beatmaps_stars_nomod;
DROP INDEX IF EXISTS idx_beatmaps_beatmap_set_id;
DROP INDEX IF EXISTS idx_beatmaps_md5_hash;
DROP TABLE IF EXISTS beatmaps;