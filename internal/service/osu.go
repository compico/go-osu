package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/compico/go-osu/internal/repository"
	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
	"github.com/compico/go-osu/pkg/osu/database"
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
	repos                  *repository.Repos
}

func NewOsuService(gamePath string, logger *slog.Logger, repos *repository.Repos) (*Osu, error) {
	srv := &Osu{
		Logger:   logger,
		gamePath: gamePath,
		repos:    repos,
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

	if err := database.Unmarshal(o.gamePath+"/osu!.db", o.dataBase); err != nil {
		return fmt.Errorf("cannot decode osu database file: %w", err)
	}

	return nil
}

type Directory struct {
	ID         int    `json:"id"`
	BeatmapID  int32  `json:"beatmap_id"`
	SongName   string `json:"song_name"`
	ArtistName string `json:"artist_name"`
	Beatmaps   []osu.DatabaseBeatmap
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

func (o *Osu) GetTrackPathByDifficultyId(ctx context.Context, difficultyId int32) (string, error) {
	bm, err := o.repos.Beatmaps.Get(ctx, difficultyId)
	if err != nil {
		return "", err
	}

	return filepath.Join(o.songsPath, bm.FolderName, bm.AudioFileName), nil
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
