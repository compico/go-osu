package database

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"unsafe"

	"github.com/compico/go-osu/pkg/osu"
)

type decoder struct {
	pos  int
	data []byte
}

func newDecoder(data []byte) *decoder {
	return &decoder{
		data: data,
	}
}

func Unmarshal(filepath string, database *osu.Database) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	d := newDecoder(data)

	database.Version = d.decodeInt()
	database.FolderCount = d.decodeInt()
	database.AccountUnlocked = d.decodeBoolean()
	database.DateUnlocked = d.decodeDouble()
	database.PlayerName = d.decodeString()
	database.NumberOfBeatmaps = d.decodeInt()
	database.Beatmaps = make([]osu.DatabaseBeatmap, database.NumberOfBeatmaps)

	for i := range database.Beatmaps {
		bm := &database.Beatmaps[i]

		bm.ArtistName = d.decodeString()
		bm.ArtistNameUni = d.decodeString()
		bm.SongTitle = d.decodeString()
		bm.SongTitleUni = d.decodeString()
		bm.CreatorName = d.decodeString()
		bm.Difficulty = d.decodeString()
		bm.AudioFileName = d.decodeString()
		bm.MD5Hash = d.decodeString()
		bm.NameOfTheOsuFile = d.decodeString()
		bm.RankedStatus = d.decodeByte()
		bm.NumberOfHitcircles = d.decodeShort()
		bm.NumberOfSliders = d.decodeShort()
		bm.NumberOfSpinners = d.decodeShort()
		bm.LastModification = d.decodeLong()
		bm.ApproachRate = d.decodeSingle()
		bm.CircleSize = d.decodeSingle()
		bm.HPDrain = d.decodeSingle()
		bm.OverallDifficulty = d.decodeSingle()
		bm.SliderVelocity = d.decodeDouble()
		bm.OsuModeStars = d.decodePairsIntFloat()
		bm.TaikoModeStars = d.decodePairsIntFloat()
		bm.CTBModeStars = d.decodePairsIntFloat()
		bm.ManiaModeStars = d.decodePairsIntFloat()
		bm.DrainTime = d.decodeInt()
		bm.TotalTime = d.decodeInt()
		bm.PreviewAudioTime = d.decodeInt()
		bm.TimingPoints = d.decodeTimingPoints()
		bm.BeatmapID = d.decodeInt()
		bm.BeatmapSetID = d.decodeInt()
		bm.ThreadID = d.decodeInt()
		bm.GradeAchievedOsu = d.decodeByte()
		bm.GradeAchievedTaiko = d.decodeByte()
		bm.GradeAchievedCTB = d.decodeByte()
		bm.GradeAchievedMania = d.decodeByte()
		bm.LocalOffset = d.decodeShort()
		bm.StackLeniency = d.decodeSingle()
		bm.Mode = d.decodeByte()
		bm.SongSource = d.decodeString()
		bm.SongTags = d.decodeString()
		bm.OnlineOffset = d.decodeShort()
		bm.TitleFont = d.decodeString()
		bm.Unplayed = d.decodeBoolean()
		bm.LastPlay = d.decodeLong()
		bm.IsOsz2 = d.decodeBoolean()
		bm.FolderName = d.decodeString()
		bm.LastCheckedOsuRepo = d.decodeLong()
		bm.IgnoreSound = d.decodeBoolean()
		bm.IgnoreSkin = d.decodeBoolean()
		bm.DisableStoryboard = d.decodeBoolean()
		bm.DisableVideo = d.decodeBoolean()
		bm.VisualOverride = d.decodeBoolean()
		bm.LastModificationTime = d.decodeInt()
		bm.ManiaScrollSpeed = d.decodeByte()
	}

	database.Permissions = d.decodeInt()

	return nil
}

func (d *decoder) decodeByte() byte {
	x := d.data[d.pos]
	d.pos++
	return x
}

func (d *decoder) decodeBoolean() bool {
	return d.decodeByte() == 1
}

func (d *decoder) decodeShort() int16 {
	x := int16(binary.LittleEndian.Uint16(d.data[d.pos:]))
	d.pos += 2
	return x
}

func (d *decoder) decodeInt() int32 {
	x := int32(binary.LittleEndian.Uint32(d.data[d.pos:]))
	d.pos += 4
	return x
}

func (d *decoder) decodeLong() int64 {
	x := int64(binary.LittleEndian.Uint64(d.data[d.pos:]))
	d.pos += 8
	return x
}

func (d *decoder) decodeSingle() float32 {
	x := math.Float32frombits(binary.LittleEndian.Uint32(d.data[d.pos:]))
	d.pos += 4
	return x
}

func (d *decoder) decodeDouble() float64 {
	x := math.Float64frombits(binary.LittleEndian.Uint64(d.data[d.pos:]))
	d.pos += 8
	return x
}

func (d *decoder) decodeULEB128() uint64 {
	var result uint64
	var shift uint

	for {
		b := d.decodeByte()
		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}

	return result
}

func (d *decoder) decodeString() string {
	if d.decodeByte() != 0x0b {
		return ""
	}
	size := int(d.decodeULEB128())
	s := unsafe.String(&d.data[d.pos], size)
	d.pos += size
	return s
}

func (d *decoder) decodeTimingPoints() []osu.DatabaseTimingPoint {
	n := int(d.decodeInt())
	tp := make([]osu.DatabaseTimingPoint, n)

	for i := range tp {
		tp[i] = d.decodeTimingPoint()
	}

	return tp
}

func (d *decoder) decodeTimingPoint() osu.DatabaseTimingPoint {
	return osu.DatabaseTimingPoint{
		BeatLength: d.decodeDouble(),
		TimeOffset: d.decodeDouble(),
		Inherited:  d.decodeBoolean() == true,
	}
}

func (d *decoder) decodePairIntFloat() osu.PairIntFloat {
	if d.decodeByte() != 0x08 {
		return osu.PairIntFloat{}
	}

	i := d.decodeInt()

	if d.decodeByte() != 0x0C {
		return osu.PairIntFloat{}
	}

	f := math.Float32frombits(binary.LittleEndian.Uint32(d.data[d.pos:]))
	d.pos += 4

	return osu.PairIntFloat{
		Int:   i,
		Float: f,
	}
}

func (d *decoder) decodePairsIntFloat() []osu.PairIntFloat {
	n := int(d.decodeInt())
	pairs := make([]osu.PairIntFloat, n)

	for i := range pairs {
		pairs[i] = d.decodePairIntFloat()
	}

	return pairs
}
