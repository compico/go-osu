package osu

import (
	"github.com/compico/go-osu/pkg/vector2d"
)

type PairIntFloat struct {
	Int   int32   `json:"int"`
	Float float32 `json:"float"`
}

type SampleSet int

const (
	DefaultSampleSet SampleSet = iota
	NormalSampleSet
	SoftSampleSet
	DrumSampleSet
)

type GameMode int

const (
	// ModeOsu osu!standard/std
	ModeOsu GameMode = iota
	// ModeTaiko osu!taiko
	ModeTaiko
	// ModeCtb osu!catch
	ModeCtb
	// ModeMania osu!mania
	ModeMania
)

type DatabaseTimingPoint struct {
	BeatLength float64 `json:"beat_length"`
	TimeOffset float64 `json:"time_offset"`
	Inherited  bool    `json:"inherited"`
}

type DateTime float64

type Database struct {
	Beatmaps         []DatabaseBeatmap
	PlayerName       string
	DateUnlocked     float64
	Version          int32
	FolderCount      int32
	NumberOfBeatmaps int32
	Permissions      int32
	AccountUnlocked  bool
}

type DatabaseBeatmap struct {
	TimingPoints         []DatabaseTimingPoint `json:"timing_points"`
	OsuModeStars         []PairIntFloat        `json:"osu_mode_stars"`
	TaikoModeStars       []PairIntFloat        `json:"taiko_mode_stars"`
	CTBModeStars         []PairIntFloat        `json:"ctb_mode_stars"`
	ManiaModeStars       []PairIntFloat        `json:"mania_mode_stars"`
	ArtistName           string                `json:"artist_name"`
	ArtistNameUni        string                `json:"artist_name_uni"`
	SongTitle            string                `json:"song_title"`
	SongTitleUni         string                `json:"song_title_uni"`
	CreatorName          string                `json:"creator_name"`
	Difficulty           string                `json:"difficulty"`
	AudioFileName        string                `json:"audio_file_name"`
	MD5Hash              string                `json:"md5_hash"`
	NameOfTheOsuFile     string                `json:"name_of_the_osu_file"`
	SongSource           string                `json:"song_source"`
	SongTags             string                `json:"song_tags"`
	TitleFont            string                `json:"title_font"`
	FolderName           string                `json:"folder_name"`
	SliderVelocity       float64               `json:"slider_velocity"`
	LastCheckedOsuRepo   int64                 `json:"last_checked_osu_repo"`
	LastModification     int64                 `json:"last_modification"`
	LastPlay             int64                 `json:"last_play"`
	ApproachRate         float32               `json:"approach_rate"`
	CircleSize           float32               `json:"circle_size"`
	HPDrain              float32               `json:"hp_drain"`
	OverallDifficulty    float32               `json:"overall_difficulty"`
	StackLeniency        float32               `json:"stack_leniency"`
	DrainTime            int32                 `json:"drain_time"`
	TotalTime            int32                 `json:"total_time"`
	PreviewAudioTime     int32                 `json:"preview_audio_time"`
	BeatmapID            int32                 `json:"beatmap_id"`
	BeatmapSetID         int32                 `json:"beatmap_set_id"`
	ThreadID             int32                 `json:"thread_id"`
	LastModificationTime int32                 `json:"last_modification_time"`
	NumberOfHitcircles   int16                 `json:"number_of_hitcircles"`
	NumberOfSliders      int16                 `json:"number_of_sliders"`
	NumberOfSpinners     int16                 `json:"number_of_spinners"`
	LocalOffset          int16                 `json:"local_offset"`
	OnlineOffset         int16                 `json:"online_offset"`
	Unknown              int16                 `json:"unknown"`
	RankedStatus         byte                  `json:"ranked_status"`
	GradeAchievedOsu     byte                  `json:"grade_achieved_osu"`
	GradeAchievedTaiko   byte                  `json:"grade_achieved_taiko"`
	GradeAchievedCTB     byte                  `json:"grade_achieved_ctb"`
	GradeAchievedMania   byte                  `json:"grade_achieved_mania"`
	//0x00 = osu, 0x01 = taiko, 0x02 ctb, 0x03 = mania
	Mode              byte `json:"mode"`
	ManiaScrollSpeed  byte `json:"mania_scroll_speed"`
	Unplayed          bool `json:"unplayed"`
	IsOsz2            bool `json:"is_osz2"`
	IgnoreSound       bool `json:"ignore_sound"`
	IgnoreSkin        bool `json:"ignore_skin"`
	DisableStoryboard bool `json:"disable_storyboard"`
	DisableVideo      bool `json:"disable_video"`
	VisualOverride    bool `json:"visual_override"`
}

type HitObject struct {
	Pos          vector2d.Vector2dd
	Time         int
	Type         HitObjectType
	CurveType    CurveType
	Curves       []vector2d.Vector2dd
	LerpPoints   []vector2d.Vector2dd
	Ncurve       int
	Repeat       int
	RepeatTimes  []int
	PixelLength  float64
	EndTime      int
	ToRepeatTime int
	EndPoint     vector2d.Vector2dd
	Ticks        []int
}

type Countdown int

const (
	NoCountdown Countdown = iota
	NormalCountdown
	HalfCountdown
	DoubleCountdown
)

type Beatmap struct {
	// Format file specifies the file format version
	Format int

	// [General]
	// AudioFilename Location of the audio file relative to the current folder
	AudioFilename string
	// AudioLeadIn Milliseconds of silence before the audio starts playing
	AudioLeadIn int
	// AudioHash Deprecated
	AudioHash string
	// PreviewTime Time in milliseconds when the audio preview should start
	PreviewTime int
	// Countdown Speed of the countdown before the first hit object
	Countdown Countdown
	// SampleSet Sample set that will be used if timing points do not override it (NormalSampleSet, SoftSampleSet, DrumSampleSet)
	SampleSet string
	// StackLeniency Multiplier for the threshold in time where hit objects placed close together stack (0–1)
	StackLeniency float64
	// Mode Game mode see GameMode
	Mode GameMode
	// LetterboxInBreaks Whether breaks have a letterboxing effect
	LetterboxInBreaks bool
	// StoryFireInFront Deprecated
	StoryFireInFront bool
	// UseSkinSprites Whether the storyboard can use the user's skin images
	UseSkinSprites bool
	// AlwaysShowPlayfield Deprecated
	AlwaysShowPlayfield bool
	// OverlayPosition	Draw order of hit circle overlays compared to hit numbers (NoChange = use skin setting, Below = draw overlays under numbers, Above = draw overlays on top of numbers) 	NoChange
	OverlayPosition string
	// SkinPreference 	Preferred skin to use during gameplay
	SkinPreference bool
	// EpilepsyWarning	Whether a warning about flashing colors should be shown at the beginning of the map 	0
	EpilepsyWarning bool
	// CountdownOffset	Time in beats that the countdown starts before the first hit object 	0
	CountdownOffset int
	// SpecialStyle	Whether the "N+1" style key layout is used for osu!mania 	0
	SpecialStyle bool
	// WidescreenStoryboard	Whether the storyboard allows widescreen viewing 	0
	WidescreenStoryboard bool
	// SamplesMatchPlaybackRate	Whether sound samples will change rate when playing with speed-changing mods 	0
	SamplesMatchPlaybackRate bool

	// [Editor]
	// Bookmarks Time in milliseconds of bookmarks
	Bookmarks []int
	// DistanceSpacing Distance snap multiplier
	DistanceSpacing float64
	// BeatDivisor Beat snap divisor
	BeatDivisor int
	// GridSize Grid size
	GridSize int
	// TimelineZoom Scale factor for the object timeline
	TimelineZoom float64

	// [Metadata]
	// Title Romanised song title
	Title string
	// TitleUnicode Song title
	TitleUnicode string
	// Artist Romanised song artist
	Artist string
	// ArtistUnicode Song artist
	ArtistUnicode string
	// Creator Beatmap creator
	Creator string
	// Version Difficulty name
	Version string
	// Source Original media the song was produced for
	Source string
	// Tags Search terms
	Tags []string
	// BeatmapID Difficulty ID
	BeatmapID int
	// BeatmapSetID Beatmap ID
	BeatmapSetID int

	// [Difficulty]
	HPDrainRate       float64
	CircleSize        float64
	OverallDifficulty float64
	ApproachRate      float64
	SliderMultiplier  float64
	SliderTickRate    float64

	// [TimingPoints]
	TimingPoints []TimingPoint

	// [HitObjects]
	HitObjects []HitObject
}

// Mod Mods flags
type Mod int

const (
	NM Mod = 0
	NF Mod = 1
	EZ Mod = 2
	HD Mod = 8
	HR Mod = 16
	SD Mod = 32
	DT Mod = 64
	RL Mod = 128
	HT Mod = 256
	FL Mod = 1024
	AU Mod = 2048
	SO Mod = 4096
	AP Mod = 8192
)

// HitObjectType flags
type HitObjectType int

const (
	HitNormal         HitObjectType = 1
	HitSlider         HitObjectType = 2
	HitNewCombo       HitObjectType = 4
	HitNormalNewCombo HitObjectType = 5
	HitSliderNewCombo HitObjectType = 6
	HitSpinner        HitObjectType = 8
	HitColourHax      HitObjectType = 112
	HitHold           HitObjectType = 128
	HitManiaLong      HitObjectType = 128
)

func (hot HitObjectType) IsHitNormal() bool {
	return hot&HitNormal != 0
}

func (hot HitObjectType) IsHitSlider() bool {
	return hot&HitSlider != 0
}

func (hot HitObjectType) IsHitNewCombo() bool {
	return hot&HitNewCombo != 0
}

func (hot HitObjectType) IsHitNormalNewCombo() bool {
	return hot&HitNormalNewCombo != 0
}

func (hot HitObjectType) IsHitSliderNewCombo() bool {
	return hot&HitSliderNewCombo != 0
}

func (hot HitObjectType) IsHitSpinner() bool {
	return hot&HitSpinner != 0
}

func (hot HitObjectType) IsHitColourHax() bool {
	return hot&HitColourHax != 0
}

func (hot HitObjectType) IsHitHold() bool {
	return hot&HitHold != 0
}

func (hot HitObjectType) IsHitManiaLong() bool {
	return hot&HitManiaLong != 0
}

// CurveType Curve Type
type CurveType rune

const (
	PerfectCurve CurveType = 'P'
	BezierCurve  CurveType = 'B'
	LinearCurve  CurveType = 'L'
	CatmullCurve CurveType = 'C'
)

type EventType string

const (
	BackgroundEventType  EventType = "0"
	VideoEventType       EventType = "1"
	VideoStringEventType EventType = "Video"
	BreakEventType                 = "2"
	// todo make more types for events
)
