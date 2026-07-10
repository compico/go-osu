package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/compico/go-osu/internal/database"
	"github.com/compico/go-osu/internal/model"
	"github.com/jmoiron/sqlx"
)

const (
	keyGamePath                = "game_path"
	keySkillCacheSchemaVersion = "skill_cache_schema_version"
	keyLastSyncAt              = "last_sync_at"
)

type SettingsRepo struct {
	db *database.DB
}

func NewSettingsRepo(db *database.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Load(ctx context.Context) (*model.AppSettings, error) {
	rows := make(map[string]string)
	rs, err := r.db.QueryxContext(ctx, `SELECT key, value FROM app_settings`)
	if err != nil {
		return nil, fmt.Errorf("load app_settings: %w", err)
	}
	defer rs.Close()

	for rs.Next() {
		var k, v string
		if err := rs.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan app_settings row: %w", err)
		}
		rows[k] = v
	}
	if err := rs.Err(); err != nil {
		return nil, err
	}

	return &model.AppSettings{
		GamePath:                rows[keyGamePath],
		SkillCacheSchemaVersion: rows[keySkillCacheSchemaVersion],
		LastSyncAt:              rows[keyLastSyncAt],
	}, nil
}

func (r *SettingsRepo) Save(ctx context.Context, s *model.AppSettings) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func(tx *sqlx.Tx) {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			r.db.Logger.Error(err.Error())
		}
	}(tx)

	set := func(key, value string) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO app_settings (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value
		`, key, value)
		return err
	}

	if err := set(keyGamePath, s.GamePath); err != nil {
		return fmt.Errorf("save game_path: %w", err)
	}
	if err := set(keySkillCacheSchemaVersion, s.SkillCacheSchemaVersion); err != nil {
		return fmt.Errorf("save skill_cache_schema_version: %w", err)
	}
	if err := set(keyLastSyncAt, s.LastSyncAt); err != nil {
		return fmt.Errorf("save last_sync_at: %w", err)
	}

	return tx.Commit()
}
