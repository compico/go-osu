package skills

import (
	"io/fs"
	"path/filepath"
)

func getOsuFiles() ([]string, error) {
	var files []string

	return files, filepath.WalkDir("./test_data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && filepath.Ext(path) == ".osu" {
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			files = append(files, abs)
		}

		return nil
	})
}
