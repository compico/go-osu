package repository

// SortValueType tells the cursor codec how to bind/decode a tuple element —
// SQLite's parameter binding needs the right Go type (string vs float64)
// to compare correctly against a column's declared affinity.
type SortValueType string

const (
	SortText SortValueType = "text"
	SortNum  SortValueType = "num"
)

// SortKey defines an orderable dimension for beatmap groups. OrderExprs and
// Types are parallel slices — same length, same order — used both for
// ORDER BY and for keyset row-value comparisons in SearchGroups/
// AdjacentGroup (SQLite's native `(a,b,c) > (x,y,z)` row-value syntax).
//
// Diff-level (per-beatmap) expressions MUST be wrapped in an aggregate
// (MIN/MAX/etc.) since a group can have many diffs and callers only want
// one row per group. Group-level (beatmapset) expressions don't need
// aggregation under GROUP BY bs.beatmap_set_id.
type SortKey struct {
	Name       string
	OrderExprs []string
	Types      []SortValueType
}

var sortRegistry = map[string]SortKey{
	"artist_name": {
		Name:       "artist_name",
		OrderExprs: []string{"bs.artist_name", "bs.song_title", "bs.beatmap_set_id"},
		Types:      []SortValueType{SortText, SortText, SortNum},
	},
	"stars": {
		Name:       "stars",
		OrderExprs: []string{"MIN(b.stars_nomod)", "bs.beatmap_set_id"},
		Types:      []SortValueType{SortNum, SortNum},
	},
}

const defaultSortKey = "artist_name"

func resolveSort(name string) SortKey {
	if sk, ok := sortRegistry[name]; ok {
		return sk
	}
	return sortRegistry[defaultSortKey]
}
