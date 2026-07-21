package repository

import (
	"context"
	"fmt"

	"github.com/compico/go-osu/internal/database"
	"github.com/compico/go-osu/internal/model"
)

type BeatmapSetRepo struct {
	db *database.DB
}

func NewBeatmapSetRepo(db *database.DB) *BeatmapSetRepo {
	return &BeatmapSetRepo{db: db}
}

func (r *BeatmapSetRepo) UpsertBatch(ctx context.Context, sets []model.BeatmapSet) error {
	return r.db.UpsertBatch(ctx, "beatmapsets", []string{"beatmap_set_id"}, sets)
}

func (r *BeatmapSetRepo) List(ctx context.Context) ([]model.BeatmapSet, error) {
	var sets []model.BeatmapSet
	if err := r.db.SelectContext(ctx, &sets, `SELECT * FROM beatmapsets`); err != nil {
		return nil, fmt.Errorf("list beatmapsets: %w", err)
	}
	return sets, nil
}

func (r *BeatmapSetRepo) Get(ctx context.Context, beatmapSetID int32) (*model.BeatmapSet, error) {
	var set model.BeatmapSet
	err := r.db.GetContext(ctx, &set, `SELECT * FROM beatmapsets WHERE beatmap_set_id = ?`, beatmapSetID)
	if err != nil {
		return nil, fmt.Errorf("get beatmapset %d: %w", beatmapSetID, err)
	}
	return &set, nil
}
