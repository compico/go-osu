package filehelper

import (
	"errors"
	"fmt"
)

var ErrFolderNotExist = errors.New("folder not exist")

// Returns all possible information about skins
// TODO:
//		*Get images path
//		*Parse ini file and get metadata
//		*Get sounds path

func (osufolder *Osu) GetSkins() error {
	if osufolder.SkinsPath == "" {
		return fmt.Errorf("%w: %s", ErrFolderNotExist, osufolder.SkinsPath)
	}
	oskins := newOsuSkins()
	dirs, err := lsdir(osufolder.SkinsPath)
	if err != nil {
		return fmt.Errorf("cannot read directory contents: %w", err)
	}
	for i := 0; i < len(dirs); i++ {
		oskins.skin = append(oskins.skin, OsuSkin{path: dirs[i]})
	}
	return nil
}

func newOsuSkins() *OsuSkins {
	oskins := new(OsuSkins)
	return oskins
}
