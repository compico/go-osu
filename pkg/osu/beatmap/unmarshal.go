package beatmap

import (
	"github.com/compico/go-osu/pkg/osu"
)

func Unmarshal(data []byte, bm *osu.Beatmap) error {
	decoder := NewDecoder(data)

	return decoder.Decode(bm)
}
