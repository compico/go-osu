package repository

import (
	"context"
	"fmt"

	"github.com/compico/go-osu/internal/database"
	"github.com/compico/go-osu/internal/model"
	"github.com/jmoiron/sqlx"
)

type BeatmapRepo struct {
	db *database.DB
}

func NewBeatmapRepo(db *database.DB) *BeatmapRepo {
	return &BeatmapRepo{db: db}
}

func (r *BeatmapRepo) UpsertBatch(ctx context.Context, beatmaps []model.Beatmap) error {
	return r.db.UpsertBatch(ctx, "beatmaps", []string{"beatmap_id"}, beatmaps)
}

func (r *BeatmapRepo) List(ctx context.Context) ([]model.Beatmap, error) {
	var beatmaps []model.Beatmap
	if err := r.db.SelectContext(ctx, &beatmaps, `SELECT * FROM beatmaps`); err != nil {
		return nil, fmt.Errorf("list beatmaps: %w", err)
	}
	return beatmaps, nil
}

// Hashes returns beatmap_id -> md5_hash for every beatmap currently stored.
// This is the "pluck" you're after: call it before writing a fresh parse of
// osu!.db to know what was there previously, then diff against the newly
// parsed hashes to find changed/new difficulties.
func (r *BeatmapRepo) Hashes(ctx context.Context) (map[int32]model.MD5Hash, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT beatmap_id, md5_hash FROM beatmaps`)
	if err != nil {
		return nil, fmt.Errorf("get beatmap hashes: %w", err)
	}
	defer rows.Close()

	out := make(map[int32]model.MD5Hash)
	for rows.Next() {
		var id int32
		var hash model.MD5Hash
		if err := rows.Scan(&id, &hash); err != nil {
			return nil, fmt.Errorf("scan beatmap hash row: %w", err)
		}
		out[id] = hash
	}

	return out, rows.Err()
}

// DeleteMissing removes beatmaps whose id is not in keepIDs — used after a
// sync pass to drop rows for maps the player deleted from the game since
// the last run. ON DELETE CASCADE on skill_cache handles cleanup there.
func (r *BeatmapRepo) DeleteMissing(ctx context.Context, keepIDs []int32) error {
	if len(keepIDs) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`DELETE FROM beatmaps WHERE beatmap_id NOT IN (?)`, keepIDs)
	if err != nil {
		return fmt.Errorf("build delete-missing query: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, r.db.Rebind(query), args...); err != nil {
		return fmt.Errorf("delete missing beatmaps: %w", err)
	}
	return nil
}
