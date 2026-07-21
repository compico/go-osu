package response

import (
	"fmt"
	"strconv"

	"github.com/compico/go-osu/internal/model"
)

// OsuQueueAdjacentResponse is the JSON shape the frontend's Track interface expects
// (frontend/src/types/player.ts). GroupSetID is included alongside the
// string GroupKey so the store can compare groups directly by id instead
// of parsing it back out of the composite key string.
type OsuQueueAdjacentResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	Duration   int32  `json:"duration"`
	URL        string `json:"url"`
	CoverURL   string `json:"coverUrl"`
	GroupKey   string `json:"groupKey"`
	GroupSetID int32  `json:"groupSetId"`
}

func NewOsuQueueAdjacentResponse(set *model.BeatmapSet, diff *model.Beatmap) OsuQueueAdjacentResponse {
	return OsuQueueAdjacentResponse{
		ID:         strconv.Itoa(int(diff.BeatmapID)),
		Title:      set.SongTitle,
		Artist:     set.ArtistName,
		Album:      diff.Difficulty,
		Duration:   diff.DrainTime,
		URL:        fmt.Sprintf("/api/osu/songs/%d/track", diff.BeatmapID),
		CoverURL:   fmt.Sprintf("/api/osu/bg/%d.jpg", set.BeatmapSetID),
		GroupKey:   fmt.Sprintf("%d:%s", set.BeatmapSetID, diff.AudioFileName),
		GroupSetID: set.BeatmapSetID,
	}
}
