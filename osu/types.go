package osu

type (
	PairIntFloat struct {
		Int   int32
		Float float32
	}
	TimingPoint struct {
		Todo []byte
	}
	DateTime float64
)

type OsuDB struct {
	Beatmaps         []Beatmap
	PlayerName       string
	DateUnlocked     float64
	Version          int32
	FolderCount      int32
	NumberOfBeatmaps int32
	Permissions      int32
	AccountUnlocked  bool
}

type Beatmap struct {
	TimingPoints         []TimingPoint
	OsuModeStars         []PairIntFloat
	TaikoModeStars       []PairIntFloat
	CTBModeStars         []PairIntFloat
	ManiaModeStars       []PairIntFloat
	ArtistName           string
	ArtistNameUni        string
	SongTitle            string
	SongTitleUni         string
	CreatorName          string
	Difficulty           string
	AudioFileName        string
	MD5Hash              string
	NameOfTheOsuFile     string
	SongSource           string
	SongTags             string
	TitleFont            string
	FolderName           string
	SliderVelocity       float64
	LastCheckedOsuRepo   int64
	LastModification     int64
	LastPlay             int64
	ApproachRate         float32
	CircleSize           float32
	HPDrain              float32
	OverallDifficulty    float32
	StackLeniency        float32
	DrainTime            int32
	TotalTime            int32
	PreviewAudioTime     int32
	DifficultyID         int32
	BeatmapID            int32
	ThreadID             int32
	LastModificationTime int32
	NumberOfHitcircles   int16
	NumberOfSliders      int16
	NumberOfSpinners     int16
	LocalOffset          int16
	OnlineOffset         int16
	Unknown              int16
	RankedStatus         byte
	GradeAchievedOsu     byte
	GradeAchievedTaiko   byte
	GradeAchievedCTB     byte
	GradeAchievedMania   byte
	Mode                 byte //0x00 = osu, 0x01 = taiko, 0x02 ctb, 0x03 = mania
	ManiaScrollSpeed     byte
	Unplayed             bool
	IsOsz2               bool
	IgnoreSound          bool
	IgnoreSkin           bool
	DisableStoryboard    bool
	DisableVideo         bool
	VisualOverride       bool
}
