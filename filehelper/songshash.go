package filehelper

import (
	"github.com/compico/osutools/osu"
)

func (folder *Osu) DirectorySorting() {
	folder.SongsDirectory = new(SongsDirectory)
	folder.SongsDirectory.BeatmapsDirectories = make([]Directory, 0)

	for _, dirs := range folder.SongsHash {
		if len(dirs) < 1 {
			continue
		}
		directory := Directory{
			Directory: make([]osu.Beatmap, 0),
		}
		fsthel := true
		for _, i := range dirs {
			if fsthel {
				fsthel = false
				directory.SongName = folder.DataBase.Beatmaps[i].SongTitle
				directory.ArtistName = folder.DataBase.Beatmaps[i].ArtistName
				directory.CreatorName = folder.DataBase.Beatmaps[i].CreatorName
			}
			directory.Directory = append(directory.Directory, folder.DataBase.Beatmaps[i])
		}
		folder.SongsDirectory.BeatmapsDirectories = append(folder.SongsDirectory.BeatmapsDirectories, directory)
	}
}
