package repository

import (
	"context"
	"fmt"

	"github.com/compico/go-osu/internal/database"
	"github.com/compico/go-osu/internal/model"
	"github.com/jmoiron/sqlx"
)

type SkillRepo struct {
	db *database.DB
}

func NewSkillRepo(db *database.DB) *SkillRepo {
	return &SkillRepo{db: db}
}

func (r *SkillRepo) UpsertBatch(ctx context.Context, rows []model.SkillCache) error {
	return r.db.UpsertBatch(ctx, "skill_cache", []string{"beatmap_id", "mods"}, rows)
}

// DeleteForBeatmaps removes every mod-combination row for the given
// beatmap ids — used right before recomputing a changed beatmap's skills,
// so a mod combination that no longer applies (rare, but e.g. after
// changing DefaultModCombinations) doesn't linger as stale data.
func (r *SkillRepo) DeleteForBeatmaps(ctx context.Context, beatmapIDs []int32) error {
	if len(beatmapIDs) == 0 {
		return nil
	}
	query, args, err := sqlx.In(`DELETE FROM skill_cache WHERE beatmap_id IN (?)`, beatmapIDs)
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, r.db.Rebind(query), args...); err != nil {
		return fmt.Errorf("delete skill_cache for %d beatmaps: %w", len(beatmapIDs), err)
	}
	return nil
}

func (r *SkillRepo) ForMods(ctx context.Context, mods int32) (map[int32]model.SkillCache, error) {
	var rows []model.SkillCache
	if err := r.db.SelectContext(ctx, &rows, `SELECT * FROM skill_cache WHERE mods = ?`, mods); err != nil {
		return nil, fmt.Errorf("get skill_cache for mods %d: %w", mods, err)
	}
	out := make(map[int32]model.SkillCache, len(rows))
	for _, row := range rows {
		out[row.BeatmapID] = row
	}
	return out, nil
}
