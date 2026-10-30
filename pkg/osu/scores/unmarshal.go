package scorestype

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

func Unmarshal(filepath string, database *osu.ScoreDatabase) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	d := newDecoder(data)

	database.Version = d.decodeInt()
	numberOfBeatmaps := d.decodeInt()

	database.Beatmaps = make([]osu.BeatmapScores, 0, numberOfBeatmaps)
	for i := int32(0); i < numberOfBeatmaps; i++ {
		bm := osu.BeatmapScores{}
		bm.MD5Hash = d.decodeString()
		numberOfScores := d.decodeInt()
		bm.Scores = make([]osu.Score, 0, numberOfScores)
		for i := int32(0); i < numberOfScores; i++ {
			s := osu.Score{}

			s.Mode = d.decodeByte()
			s.Version = d.decodeInt()
			s.BeatmapMD5Hash = d.decodeString()
			s.PlayerName = d.decodeString()
			s.ScoreMD5Hash = d.decodeString()
			s.Number300 = d.decodeShort()
			s.Number100 = d.decodeShort()
			s.Number50 = d.decodeShort()
			s.NumberGeki = d.decodeShort()
			s.NumberKatu = d.decodeShort()
			s.NumberMiss = d.decodeShort()
			s.ReplayScore = d.decodeInt()
			s.MaxCombo = d.decodeShort()
			s.PerfectCombo = d.decodeBoolean()
			s.Mods = osu.Mod(d.decodeInt())
			_ = d.decodeString()
			s.Timestamp = d.decodeLong()
			_ = d.decodeInt()
			s.OnlineID = d.decodeLong()

			// ModTargetPractice
			if s.Mods&(1<<23) != 0 {
				s.AdditionalModInfo = d.decodeDouble()
			}

			bm.Scores = append(bm.Scores, s)
		}

		database.Beatmaps = append(database.Beatmaps, bm)
	}

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
		Inherited:  d.decodeBoolean() == false,
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
