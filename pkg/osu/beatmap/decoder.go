package beatmap

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

type Decoder struct {
	data []byte
	pos  int
	line []byte

	section             section
	seen                sectionMask
	isSearchSection     bool
	previousTimingPoint *osu.TimingPoint
}

func NewDecoder(data []byte) *Decoder {
	return &Decoder{
		data:            data,
		section:         formatVersionSection,
		isSearchSection: false,
	}
}

func (d *Decoder) Decode(bm *osu.Beatmap) error {
	for d.next() {
		if len(d.line) == 0 || bytes.Equal(d.line[:1], []byte("[")) {
			d.seen.Set(sectionToFlag[d.section])
			d.isSearchSection = true
		}

		if len(d.line) >= 2 && bytes.Equal(d.line[:2], []byte("//")) {
			continue
		}

		err := d.handleSection(bm)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) next() bool {
	if d.pos >= len(d.data) {
		return false
	}

	start := d.pos

	for d.pos < len(d.data) && d.data[d.pos] != '\n' {
		d.pos++
	}

	d.line = d.data[start:d.pos]

	if n := len(d.line); n > 0 && d.line[n-1] == '\r' {
		d.line = d.line[:n-1]
	}

	if d.pos < len(d.data) {
		d.pos++
	}

	return true
}

func (d *Decoder) handleSection(bm *osu.Beatmap) error {
	if d.isSearchSection {
		d.searchSection()

		return nil
	}

	// todo make events and colours section parse
	switch d.section {
	case formatVersionSection:
		return d.parseFormatVersion(bm)
	case generalSection:
		return d.parseGeneralSection(bm)
	case editorSection:
		return d.parseEditorSection(bm)
	case metadataSection:
		return d.parseMetaSection(bm)
	case timingPointsSection:
		return d.parseTimingPointsSection(bm)
	case difficultySection:
		return d.parseDifficultySection(bm)
	case hitObjectsSection:
		return d.parseHitObjectsSection(bm)
	}

	return nil
}

func (d *Decoder) searchSection() {
	line := bytes.TrimSpace(d.line)

	for _, section := range allSections {
		if bytes.Equal(line, []byte(section)) {
			d.section = section
			d.isSearchSection = false
		}
	}
}

func (d *Decoder) parseFormatVersion(bm *osu.Beatmap) error {
	for i := len(d.line) - 1; i >= 0; i-- {
		if d.line[i] != 'v' {
			continue
		}

		version, err := strconv.Atoi(string(d.line[i+1 : len(d.line)]))
		if err != nil {
			return err
		}

		bm.Format = version
		d.isSearchSection = true

		return nil
	}

	return nil
}

func (d *Decoder) parseGeneralSection(bm *osu.Beatmap) error {
	k, v, err := d.getKeyValue()
	if err != nil {
		return err
	}

	switch k {
	case "AudioFilename":
		bm.AudioFilename = v
		break
	case "AudioLeadIn":
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		bm.AudioLeadIn = value
		break
	case "AudioHash":
		bm.AudioHash = v
		break
	case "PreviewTime":
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		bm.PreviewTime = value
		break
	case "Countdown":
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		bm.Countdown = osu.Countdown(value)
		break
	case "SampleSet":
		bm.SampleSet = v
		break
	case "StackLeniency":
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		bm.StackLeniency = value
		break
	case "Mode":
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		bm.Mode = osu.GameMode(value)
		break
	case "LetterboxInBreaks":
		bm.LetterboxInBreaks = v == "1"
		break
	case "StoryFireInFront":
		bm.StoryFireInFront = v == "1"
		break
	case "UseSkinSprites":
		bm.UseSkinSprites = v == "1"
		break
	case "AlwaysShowPlayfield":
		bm.AlwaysShowPlayfield = v == "1"
		break
	case "OverlayPosition":
		bm.OverlayPosition = v
		break
	case "SkinPreference":
		bm.SkinPreference = v == "1"
		break
	case "EpilepsyWarning":
		bm.EpilepsyWarning = v == "1"
		break
	case "CountdownOffset":
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		bm.CountdownOffset = value
		break
	case "SpecialStyle":
		bm.SpecialStyle = v == "1"
		break
	case "WidescreenStoryboard":
		bm.WidescreenStoryboard = v == "1"
		break
	case "SamplesMatchPlaybackRate":
		bm.SamplesMatchPlaybackRate = v == "1"
		break
	default:
		return fmt.Errorf("invalid general section: %s", k)
	}

	return nil
}

func (d *Decoder) parseEditorSection(bm *osu.Beatmap) error {
	k, v, err := d.getKeyValue()
	if err != nil {
		return err
	}

	switch k {
	case "Bookmarks":
		bm.Bookmarks = []int{}
		for _, bookmark := range strings.Split(v, ",") {
			bookmark = strings.TrimSpace(bookmark)
			if bookmark == "" {
				continue
			}
			bookmarkInt, err := strconv.Atoi(bookmark)
			if err != nil {
				return err
			}
			bm.Bookmarks = append(bm.Bookmarks, bookmarkInt)
		}
		break
	case "DistanceSpacing":
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		bm.DistanceSpacing = value
		break
	case "BeatDivisor":
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		bm.BeatDivisor = value
		break
	case "GridSize":
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		bm.GridSize = value
		break
	case "TimelineZoom":
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		bm.TimelineZoom = value
		break
	default:
		return fmt.Errorf("invalid editor section: %s", k)
	}

	return nil
}

func (d *Decoder) parseMetaSection(bm *osu.Beatmap) error {
	k, v, err := d.getKeyValue()
	if err != nil {
		return err
	}

	switch k {
	case "Title":
		bm.Title = v
	case "TitleUnicode":
		bm.TitleUnicode = v
	case "Artist":
		bm.Artist = v
	case "ArtistUnicode":
		bm.ArtistUnicode = v
	case "Creator":
		bm.Creator = v
	case "Version":
		bm.Version = v
	case "Source":
		bm.Source = v
	case "Tags":
		bm.Tags = strings.Split(v, ",")
	case "BeatmapID":
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		bm.BeatmapID = value
		break
	case "BeatmapSetID":
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		bm.BeatmapSetID = value
		break
	default:
		return fmt.Errorf("invalid meta section: %s", k)
	}

	return nil
}

func (d *Decoder) parseDifficultySection(bm *osu.Beatmap) error {
	k, v, err := d.getKeyValue()
	if err != nil {
		return err
	}

	switch k {
	case "HPDrainRate":
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		bm.HPDrainRate = value
		break
	case "CircleSize":
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		bm.CircleSize = value
		break
	case "OverallDifficulty":
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		bm.OverallDifficulty = value
		break
	case "ApproachRate":
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		bm.ApproachRate = value
		break
	case "SliderMultiplier":
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		bm.SliderMultiplier = value
		break
	case "SliderTickRate":
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		bm.SliderTickRate = value
		break
	}

	return nil
}

func (d *Decoder) parseTimingPointsSection(bm *osu.Beatmap) error {
	timeVal, err := d.parseFloat()
	if err != nil {
		return fmt.Errorf("timing point time: %w", err)
	}
	beatLength, err := d.parseFloat()
	if err != nil {
		return fmt.Errorf("timing point beatLength: %w", err)
	}
	meter, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("timing point meter: %w", err)
	}
	sampleSet, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("timing point sampleSet: %w", err)
	}
	sampleIndex, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("timing point sampleIndex: %w", err)
	}
	volume, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("timing point volume: %w", err)
	}
	uninheritedRaw, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("timing point uninherited: %w", err)
	}
	effects, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("timing point effects: %w", err)
	}

	bm.TimingPoints = append(bm.TimingPoints, osu.TimingPoint{
		Time:                timeVal,
		BeatLength:          beatLength,
		Meter:               meter,
		SampleSet:           osu.SampleSet(sampleSet),
		SampleIndex:         sampleIndex,
		Volume:              volume,
		Uninherited:         uninheritedRaw == 1,
		Effects:             effects,
		PreviousTimingPoint: d.previousTimingPoint,
	})

	d.previousTimingPoint = &bm.TimingPoints[len(bm.TimingPoints)-1]

	return nil
}

func (d *Decoder) parseHitObjectsSection(bm *osu.Beatmap) error {
	x, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("hit object x: %w", err)
	}
	y, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("hit object y: %w", err)
	}
	timeVal, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("hit object time: %w", err)
	}
	typeVal, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("hit object type: %w", err)
	}

	// hitSound (bit flags: normal/whistle/finish/clap), пока никуда не сохраняем
	if _, err := d.parseInt(); err != nil {
		return fmt.Errorf("hit object hitSound: %w", err)
	}

	ho := osu.HitObject{
		Pos:  vector2d.Vector2dd{X: float64(x), Y: float64(y)},
		Time: timeVal,
		Type: typeVal,
	}

	switch {
	case typeVal&int(osu.HitSlider) != 0:
		if err := d.parseSliderParams(&ho); err != nil {
			return fmt.Errorf("hit object slider params: %w", err)
		}
	case typeVal&int(osu.HitSpinner) != 0:
		endTime, err := d.parseInt()
		if err != nil {
			return fmt.Errorf("hit object spinner endTime: %w", err)
		}
		ho.EndTime = endTime
	case typeVal&int(osu.HitHold) != 0:
		endTime, err := d.parseHoldEndTime()
		if err != nil {
			return fmt.Errorf("hit object hold endTime: %w", err)
		}
		ho.EndTime = endTime
	}
	// case typeVal&int(osu.HitNormal) != 0: доп. параметров нет

	bm.HitObjects = append(bm.HitObjects, ho)

	return nil
}

func (d *Decoder) parseSliderParams(ho *osu.HitObject) error {
	curveField, ok := d.nextField()
	if !ok {
		return fmt.Errorf("unexpected end of line")
	}

	parts := bytes.Split(curveField, []byte("|"))
	if len(parts) == 0 || len(parts[0]) == 0 {
		return fmt.Errorf("invalid curve type")
	}

	ho.CurveType = rune(parts[0][0])

	for _, p := range parts[1:] {
		xy := bytes.SplitN(p, []byte(":"), 2)
		if len(xy) != 2 {
			return fmt.Errorf("invalid curve point: %s", p)
		}

		px, err := strconv.ParseFloat(string(xy[0]), 64)
		if err != nil {
			return err
		}
		py, err := strconv.ParseFloat(string(xy[1]), 64)
		if err != nil {
			return err
		}

		ho.Curves = append(ho.Curves, vector2d.Vector2dd{X: px, Y: py})
	}

	slides, err := d.parseInt()
	if err != nil {
		return fmt.Errorf("slides: %w", err)
	}
	ho.Repeat = slides

	length, err := d.parseFloat()
	if err != nil {
		return fmt.Errorf("length: %w", err)
	}
	ho.PixelLength = length

	// edgeSounds, edgeSets, hitSample опциональны — если их нет, строка уже закончилась
	if _, ok := d.nextField(); !ok {
		return nil
	}
	if _, ok := d.nextField(); !ok {
		return nil
	}
	d.nextField() // hitSample, пока игнорируем

	return nil
}

func (d *Decoder) parseHoldEndTime() (int, error) {
	f, ok := d.nextField()
	if !ok {
		return 0, fmt.Errorf("unexpected end of line")
	}

	if idx := bytes.IndexByte(f, ':'); idx != -1 {
		f = f[:idx]
	}

	return strconv.Atoi(string(f))
}

func (d *Decoder) getKeyValue() (string, string, error) {
	idx := bytes.IndexByte(d.line, ':')
	if idx == -1 {
		return "", "", fmt.Errorf("invalid general section")
	}

	key := d.line[:idx]

	valueStart := idx + 1
	if valueStart < len(d.line) && d.line[valueStart] == ' ' {
		valueStart++
	}

	value := d.line[valueStart:]

	return string(key), string(value), nil
}

func (d *Decoder) nextField() ([]byte, bool) {
	i := bytes.IndexByte(d.line, ',')
	if i < 0 {
		f := d.line
		d.line = nil
		return f, len(f) > 0
	}
	f := d.line[:i]
	d.line = d.line[i+1:]
	return f, true
}

func (d *Decoder) parseFloat() (float64, error) {
	f, ok := d.nextField()
	if !ok {
		return 0, fmt.Errorf("unexpected end of line")
	}
	return strconv.ParseFloat(string(f), 64)
}

func (d *Decoder) parseInt() (int, error) {
	f, ok := d.nextField()
	if !ok {
		return 0, fmt.Errorf("unexpected end of line")
	}
	return strconv.Atoi(string(f))
}
