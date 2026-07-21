package dslquery

import "github.com/compico/go-osu/pkg/osu"

type FieldKind int

const (
	FieldBeatmap FieldKind = iota // numeric column on beatmaps (b.*)
	FieldSkill                    // numeric column on skill_cache (sc.*), requires join
	FieldText                     // text column, matched via LIKE — artist=/title=/etc
	FieldMode                     // special: mode=/mods= parses into osu.Mod, not a WHERE column
)

type FieldDef struct {
	Kind   FieldKind
	Column string
}

// fieldRegistry maps DSL field names (lowercase) to where they live.
// Multiple keys can point at the same FieldDef to support aliases (e.g.
// "star" -> "stars").
var fieldRegistry = map[string]FieldDef{
	"stars":  {FieldBeatmap, "stars_nomod"},
	"star":   {FieldBeatmap, "stars_nomod"}, // alias
	"bpm":    {FieldBeatmap, "bpm"},
	"ar":     {FieldBeatmap, "approach_rate"},
	"cs":     {FieldBeatmap, "circle_size"},
	"hp":     {FieldBeatmap, "hp_drain"},
	"od":     {FieldBeatmap, "overall_difficulty"},
	"sv":     {FieldBeatmap, "slider_velocity"},
	"drain":  {FieldBeatmap, "drain_time"},
	"length": {FieldBeatmap, "total_time"},

	"stamina":   {FieldSkill, "stamina"},
	"tenacity":  {FieldSkill, "tenacity"},
	"agility":   {FieldSkill, "agility"},
	"precision": {FieldSkill, "precision"},
	"reading":   {FieldSkill, "reading"},
	"memory":    {FieldSkill, "memory"},
	"accuracy":  {FieldSkill, "accuracy"},
	"reaction":  {FieldSkill, "reaction"},

	// Text fields — column lives on beatmapsets (bs.*) except "difficulty",
	// which is per-diff and lives on beatmaps (b.*).
	"artist":     {FieldText, "bs.artist_name"},
	"creator":    {FieldText, "bs.creator_name"},
	"title":      {FieldText, "bs.song_title"},
	"difficulty": {FieldText, "b.difficulty"},

	"mode": {Kind: FieldMode},
	"mods": {Kind: FieldMode}, // alias
}

// freeTextColumns lists the (aliased) columns a bare-value token searches
// across, mirroring osu!'s own "search everything" behavior for unquoted
// terms.
var freeTextColumns = []string{
	"bs.artist_name",
	"bs.artist_name_uni",
	"bs.song_title",
	"bs.song_title_uni",
	"bs.creator_name",
	"bs.song_tags",
	"b.difficulty",
}

var _ = osu.NF
