package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/skills"
)

// strainCurvesTimeout bounds one curve calculation. Calculating the curves of
// an ordinary map takes about a millisecond; the limit is for pathological
// maps, so that a request cannot hang.
const strainCurvesTimeout = 10 * time.Second

// ErrStrainCurvesTimeout is returned when the calculation took longer than
// strainCurvesTimeout. The calculation itself cannot be stopped and finishes in
// the background.
var ErrStrainCurvesTimeout = errors.New("strain curves calculation timed out")

// StrainCurves calculates the difficulty curves of a beatmap from the library
// for the given mods. Nothing is stored: the .osu file is read and calculated
// on every call.
//
// The error wraps sql.ErrNoRows when the beatmap is not in the library and
// fs.ErrNotExist when its .osu file is gone from the Songs folder.
func (o *Osu) StrainCurves(ctx context.Context, beatmapID int32, mods osu.Mod) (*skills.Curves, error) {
	bm, err := o.repos.Beatmaps.Get(ctx, beatmapID)
	if err != nil {
		return nil, err
	}

	full, err := o.ParseBeatmapFile(osu.DatabaseBeatmap{
		FolderName:       bm.FolderName,
		NameOfTheOsuFile: bm.NameOfTheOsuFile,
	})
	if err != nil {
		return nil, err
	}

	done := make(chan *skills.Curves, 1) // buffered: an abandoned calculation does not block
	go func() { done <- skills.ProcessCurves(full, mods, skills.NewVars()) }()

	timer := time.NewTimer(strainCurvesTimeout)
	defer timer.Stop()

	select {
	case curves := <-done:
		return curves, nil
	case <-timer.C:
		return nil, fmt.Errorf("beatmap %d: %w", beatmapID, ErrStrainCurvesTimeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
