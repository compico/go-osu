package beatmap

import "strings"

type section string

const (
	formatVersionSection section = ""
	generalSection       section = "[General]"
	editorSection        section = "[Editor]"
	metadataSection      section = "[Metadata]"
	difficultySection    section = "[Difficulty]"
	eventsSection        section = "[Events]"
	timingPointsSection  section = "[TimingPoints]"
	coloursSection       section = "[Colours]"
	hitObjectsSection    section = "[HitObjects]"
)

var allSections = []section{
	generalSection,
	editorSection,
	metadataSection,
	difficultySection,
	eventsSection,
	timingPointsSection,
	coloursSection,
	hitObjectsSection,
}

type sectionMask uint16

const (
	formatFlag sectionMask = 1 << iota
	generalFlag
	editorFlag
	metadataFlag
	difficultyFlag
	eventsFlag
	timingPointsFlag
	coloursFlag
	hitObjectsFlag
)

var sectionToFlag = map[section]sectionMask{
	formatVersionSection: formatFlag,
	generalSection:       generalFlag,
	editorSection:        editorFlag,
	metadataSection:      metadataFlag,
	difficultySection:    difficultyFlag,
	eventsSection:        eventsFlag,
	timingPointsSection:  timingPointsFlag,
	coloursSection:       coloursFlag,
	hitObjectsSection:    hitObjectsFlag,
}

const allSectionFlags sectionMask = formatFlag |
	generalFlag |
	editorFlag |
	metadataFlag |
	difficultyFlag |
	eventsFlag |
	timingPointsFlag |
	coloursFlag |
	hitObjectsFlag

func (m *sectionMask) Set(flag sectionMask) {
	*m |= flag
}

func (m *sectionMask) Has(flag sectionMask) bool {
	return *m&flag != 0
}

func (m *sectionMask) MissingString() string {
	missing := allSectionFlags &^ *m
	if missing == 0 {
		return ""
	}

	var parts []string

	if missing.Has(generalFlag) {
		parts = append(parts, string(generalSection))
	}
	if missing.Has(editorFlag) {
		parts = append(parts, string(editorSection))
	}
	if missing.Has(metadataFlag) {
		parts = append(parts, string(metadataSection))
	}
	if missing.Has(difficultyFlag) {
		parts = append(parts, string(difficultySection))
	}
	if missing.Has(eventsFlag) {
		parts = append(parts, string(eventsSection))
	}
	if missing.Has(timingPointsFlag) {
		parts = append(parts, string(timingPointsSection))
	}
	if missing.Has(coloursFlag) {
		parts = append(parts, string(coloursSection))
	}
	if missing.Has(hitObjectsFlag) {
		parts = append(parts, string(hitObjectsSection))
	}

	return strings.Join(parts, ", ")
}
