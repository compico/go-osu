-- +goose Up
-- Значения по умолчанию; game_path пустой означает "определить из реестра"
-- (см. initGamePath/getPathFromRegistry).
INSERT OR IGNORE INTO app_settings (key, value) VALUES
    ('game_path', ''),
    ('skill_cache_schema_version', '1'),
    ('last_sync_at', '');

-- +goose Down
DELETE FROM app_settings WHERE key IN ('game_path', 'skill_cache_schema_version', 'last_sync_at');