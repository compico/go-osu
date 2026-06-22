//go:build windows
// +build windows

package filehelper

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func (osufolder *Osu) InitGamePathByReg() error {
	k, err := registry.OpenKey(registry.CLASSES_ROOT, `osustable.Uri.osu\DefaultIcon`, registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("cannot open registry key: %w", err)
	}
	defer k.Close()

	path, _, err := k.GetStringValue("")
	if err != nil {
		return fmt.Errorf("cannot read registry key value: %w", err)
	}
	path = path[1:]
	path = filepath.Dir(path)

	osufolder.GamePath = path

	osufolder.initSongsPath()
	osufolder.initSkinsPath()

	return nil
}
