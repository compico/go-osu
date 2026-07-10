package model

// BeatmapSet is one mapset — a group of Beatmap difficulties sharing the
// same song. Mirrors the beatmapsets table 1:1.
type BeatmapSet struct {
	BeatmapSetID  int32  `db:"beatmap_set_id"`
	SongTitle     string `db:"song_title"`
	SongTitleUni  string `db:"song_title_uni"`
	ArtistName    string `db:"artist_name"`
	ArtistNameUni string `db:"artist_name_uni"`
	CreatorName   string `db:"creator_name"`
	SongSource    string `db:"song_source"`
	SongTags      string `db:"song_tags"`
}

// Beatmap is a single difficulty within a BeatmapSet. Mirrors the beatmaps
// table. MD5Hash is the change-detection key: any edit to the .osu file
// changes it, which is how we know a difficulty needs its Skills recomputed.
type Beatmap struct {
	BeatmapID    int32 `db:"beatmap_id"`
	BeatmapSetID int32 `db:"beatmap_set_id"`

	Difficulty       string  `db:"difficulty"`
	MD5Hash          MD5Hash `db:"md5_hash"`
	FolderName       string  `db:"folder_name"`
	NameOfTheOsuFile string  `db:"name_of_the_osu_file"`
	AudioFileName    string  `db:"audio_file_name"`
	TitleFont        string  `db:"title_font"`

	ApproachRate      float64 `db:"approach_rate"`
	CircleSize        float64 `db:"circle_size"`
	HPDrain           float64 `db:"hp_drain"`
	OverallDifficulty float64 `db:"overall_difficulty"`
	SliderVelocity    float64 `db:"slider_velocity"`
	StackLeniency     float64 `db:"stack_leniency"`

	BPM        float64 `db:"bpm"`
	StarsNoMod float64 `db:"stars_nomod"`

	DrainTime            int32 `db:"drain_time"`
	TotalTime            int32 `db:"total_time"`
	PreviewAudioTime     int32 `db:"preview_audio_time"`
	ThreadID             int32 `db:"thread_id"`
	LastModificationTime int32 `db:"last_modification_time"`
	LastCheckedOsuRepo   int64 `db:"last_checked_osu_repo"`
	LastModification     int64 `db:"last_modification"`
	LastPlay             int64 `db:"last_play"`

	NumberOfHitcircles int16 `db:"number_of_hitcircles"`
	NumberOfSliders    int16 `db:"number_of_sliders"`
	NumberOfSpinners   int16 `db:"number_of_spinners"`
	LocalOffset        int16 `db:"local_offset"`
	OnlineOffset       int16 `db:"online_offset"`

	Mode               byte `db:"mode"`
	RankedStatus       byte `db:"ranked_status"`
	GradeAchievedOsu   byte `db:"grade_achieved_osu"`
	GradeAchievedTaiko byte `db:"grade_achieved_taiko"`
	GradeAchievedCTB   byte `db:"grade_achieved_ctb"`
	GradeAchievedMania byte `db:"grade_achieved_mania"`
	ManiaScrollSpeed   byte `db:"mania_scroll_speed"`

	Unplayed bool `db:"unplayed"`
}

// SkillCache is one (Beatmap, mod combination) pair's computed Skills.
// MD5Hash is copied from the owning Beatmap at compute time so staleness
// can be checked without a join.
type SkillCache struct {
	BeatmapID int32   `db:"beatmap_id"`
	Mods      int32   `db:"mods"`
	MD5Hash   MD5Hash `db:"md5_hash"`

	Stamina   float64 `db:"stamina"`
	Tenacity  float64 `db:"tenacity"`
	Agility   float64 `db:"agility"`
	Precision float64 `db:"precision"`
	Reading   float64 `db:"reading"`
	Memory    float64 `db:"memory"`
	Accuracy  float64 `db:"accuracy"`
	Reaction  float64 `db:"reaction"`
}

// AppSettings is a fixed, small set of runtime settings.
type AppSettings struct {
	GamePath                string
	SkillCacheSchemaVersion string
	LastSyncAt              string
}
