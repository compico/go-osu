package filehelper

import "github.com/compico/osutools/osu"

type Osu struct {
	DataBase  *osu.OsuDB
	Skins     OsuSkins
	SongsHash map[int32]map[int32]int
	GamePath  string
	SongsPath string
	*SongsDirectory
	SkinsPath string
}

type SongsDirectory struct {
	BeatmapsDirectories []Directory
}

type Directory struct {
	SongName    string
	ArtistName  string
	CreatorName string
	Directory   []osu.Beatmap
}

type OsuSkins struct {
	skin []OsuSkin
}
type OsuSkin struct {
	path string
}
