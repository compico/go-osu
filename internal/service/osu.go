package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
	"github.com/compico/go-osu/pkg/osu/database"
	"golang.org/x/sys/windows/registry"
)

type Osu struct {
	gamePath               string
	songsPath              string
	skinsPath              string
	backgroundPath         string
	dataBase               *osu.Database
	skins                  []OsuSkins
	songs                  []Directory
	songsJson              []byte
	BeatmapDifficultyIndex map[int32]map[int32]int
	BeatmapByDifficultyId  map[int32]int32
	Logger                 *slog.Logger
}

func NewOsuService(gamePath string, logger *slog.Logger) (*Osu, error) {
	srv := &Osu{
		Logger:   logger,
		gamePath: gamePath,
	}

	err := srv.initGamePath()
	if err != nil {
		return nil, err
	}

	return srv, nil
}

type OsuSkins struct {
	path string
}

func (o *Osu) initGamePath() error {
	if o.gamePath != "" {
		return nil
	}

	if path, err := getPathFromRegistry(); err != nil {
		return err
	} else {
		o.gamePath = path
	}

	if o.gamePath == "" {
		return errors.New("game path is empty")
	}

	o.initSongsPath()
	o.initSkinsPath()
	o.initBackgroundPath()

	return nil
}

func getPathFromRegistry() (string, error) {
	k, err := registry.OpenKey(registry.CLASSES_ROOT, `osustable.Uri.osu\DefaultIcon`, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("cannot open registry key: %w", err)
	}
	defer func(k registry.Key) {
		err := k.Close()
		if err != nil {
			log.Println("Error closing registry key:", err)
		}
	}(k)

	value, _, err := k.GetStringValue("")
	if err != nil {
		return "", fmt.Errorf("cannot read registry key value: %w", err)
	}

	splitValue := strings.Split(value, ",")
	if len(splitValue) != 2 {
		return "", fmt.Errorf("cannot read registry key value: %w", err)
	}

	first := splitValue[0]
	first = first[1:]
	path := filepath.Dir(first)

	return path, nil
}

func (o *Osu) initSongsPath() {
	o.songsPath = filepath.Join(o.gamePath, "Songs")
}

func (o *Osu) initSkinsPath() {
	o.skinsPath = filepath.Join(o.gamePath, "Skins")
}

func (o *Osu) initBackgroundPath() {
	o.backgroundPath = filepath.Join(o.gamePath, "Data", "bt")
}

func (o *Osu) ReadOsuDBFile() error {
	o.dataBase = &osu.Database{}

	err := database.Unmarshal(o.gamePath+"/osu!.db", o.dataBase)
	if err != nil {
		return fmt.Errorf("cannot decode osu database file: %w", err)
	}

	return o.hashing()
}

func (o *Osu) hashing() error {
	o.BeatmapDifficultyIndex = make(map[int32]map[int32]int)
	o.BeatmapByDifficultyId = make(map[int32]int32)
	for i := 0; i < len(o.dataBase.Beatmaps); i++ {
		if len(o.BeatmapDifficultyIndex[o.dataBase.Beatmaps[i].BeatmapSetID]) == 0 {
			o.BeatmapDifficultyIndex[o.dataBase.Beatmaps[i].BeatmapSetID] = make(map[int32]int)
		}
		o.BeatmapDifficultyIndex[o.dataBase.Beatmaps[i].BeatmapSetID][o.dataBase.Beatmaps[i].BeatmapID] = i
		o.BeatmapByDifficultyId[o.dataBase.Beatmaps[i].BeatmapID] = o.dataBase.Beatmaps[i].BeatmapSetID
	}
	o.directorySorting()

	var err error

	o.songsJson, err = json.Marshal(o.songs)

	return err
}

type Directory struct {
	ID         int    `json:"id"`
	BeatmapID  int32  `json:"beatmap_id"`
	SongName   string `json:"song_name"`
	ArtistName string `json:"artist_name"`
	Beatmaps   []osu.DatabaseBeatmap
}

func (o *Osu) directorySorting() {
	o.songs = make([]Directory, 0)

	for _, dirs := range o.BeatmapDifficultyIndex {
		if len(dirs) < 1 {
			continue
		}

		directory := Directory{
			Beatmaps: make([]osu.DatabaseBeatmap, 0),
		}

		firstElement := true

		for _, i := range dirs {
			if firstElement {
				firstElement = false
				directory.ID = i
				directory.BeatmapID = o.dataBase.Beatmaps[i].BeatmapSetID
				directory.SongName = o.dataBase.Beatmaps[i].SongTitle
				directory.ArtistName = o.dataBase.Beatmaps[i].ArtistName
			}

			directory.Beatmaps = append(directory.Beatmaps, o.dataBase.Beatmaps[i])
		}

		o.songs = append(o.songs, directory)
	}
}

func (o *Osu) GetSongs() []Directory {
	return o.songs
}

func (o *Osu) GetSongsJson() []byte {
	return o.songsJson
}

func (o *Osu) GetBeatmapByDifficultyId(difficultyId int32) (*osu.Beatmap, error) {
	if index, err := o.getIndexByDifficultyId(difficultyId); err != nil {
		bmFromDb := o.dataBase.Beatmaps[index]

		path := o.GetOsuFilePath(bmFromDb)

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		bm := &osu.Beatmap{}
		return bm, beatmap.Unmarshal(data, bm)
	}

	return nil, fmt.Errorf("cannot find beatmap by difficultyId: %v", difficultyId)
}

func (o *Osu) GetTrackPathByDifficultyId(difficultyId int32) (string, error) {
	if index, err := o.getIndexByDifficultyId(difficultyId); err == nil {
		bmFromDb := o.dataBase.Beatmaps[index]

		path := o.GetOsuFilePath(bmFromDb)

		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}

		bm := &osu.Beatmap{}
		if err := beatmap.Unmarshal(data, bm); err != nil {
			return "", err
		}

		return filepath.Join(o.songsPath, bmFromDb.FolderName, bm.AudioFilename), nil
	} else {
		return "", err
	}
}

func (o *Osu) GetOsuFilePath(bm osu.DatabaseBeatmap) string {
	return filepath.Join(o.songsPath, bm.FolderName, bm.NameOfTheOsuFile)
}

func (o *Osu) GetBackgroundFilePath(beatmapId int32) string {
	return filepath.Join(o.backgroundPath, fmt.Sprintf("%dl.jpg", beatmapId))
}

func (o *Osu) GetBackgroundFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	return data, nil
}

func (o *Osu) getIndexByDifficultyId(difficultyId int32) (int, error) {
	if beatmapId, ok := o.BeatmapByDifficultyId[difficultyId]; ok {
		if diffIndex, ok := o.BeatmapDifficultyIndex[beatmapId][difficultyId]; ok {
			return diffIndex, nil
		}
	}

	return -1, fmt.Errorf("cannot find beatmap by difficultyId: %v", difficultyId)
}

func (o *Osu) ParseBeatmapFile(bm osu.DatabaseBeatmap) (*osu.Beatmap, error) {
	path := o.GetOsuFilePath(bm)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read .osu file %q: %w", path, err)
	}

	full := &osu.Beatmap{}
	if err := beatmap.Unmarshal(data, full); err != nil {
		return nil, fmt.Errorf("parse .osu file %q: %w", path, err)
	}

	return full, nil
}

func (o *Osu) GetBeatmaps() []osu.DatabaseBeatmap {
	if o.dataBase == nil || o.dataBase.Beatmaps == nil {
		return nil
	}

	return o.dataBase.Beatmaps
}

func (o *Osu) GetUserID(username string) (int, error) {
	resp, err := http.Get("https://osu.ppy.sh/users/" + username)
	if err != nil {
		return 0, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			o.Logger.Warn("Failed to close response body")
		}
	}(resp.Body)

	url := resp.Request.URL.String()

	var id []byte

	// Идем справа налево до '/'
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			break
		}
		id = append(id, url[i])
	}

	// Разворачиваем слайс
	for l, r := 0, len(id)-1; l < r; l, r = l+1, r-1 {
		id[l], id[r] = id[r], id[l]
	}

	// Проверяем, что получили только цифры
	if len(id) == 0 {
		return 0, errors.New("user not found")
	}

	userID, err := strconv.Atoi(string(id))
	if err != nil {
		return 0, errors.New("user not found")
	}

	return userID, nil
}
