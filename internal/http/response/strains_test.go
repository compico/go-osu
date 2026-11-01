package response

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/skills"
)

func TestNewOsuStrainsResponseJSON(t *testing.T) {
	curves := &skills.Curves{
		Mods:       osu.DT,
		ModsString: "DT",
		Speed:      1.5,
		Stamina:    []skills.StrainPoint{{Time: 0, Value: 1}, {Time: 100, Value: 3}, {Time: 700, Value: 2}},
	}

	data, err := json.Marshal(NewOsuStrainsResponse(42, curves, 400))
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["beatmap_id"] != 42.0 || got["mods"] != "DT" || got["speed"] != 1.5 || got["window"] != 400.0 {
		t.Errorf("header = %v", got)
	}

	stamina, _ := got["stamina"].([]any)
	if len(stamina) != 2 {
		t.Fatalf("stamina = %v, want the two windows", got["stamina"])
	}
	if first, _ := stamina[0].(map[string]any); first["time"] != 100.0 || first["value"] != 3.0 {
		t.Errorf("first stamina point = %v", stamina[0])
	}

	// Curves the map does not have are [], not null.
	for _, name := range []string{"tenacity", "agility", "precision", "reading", "memory", "accuracy", "reaction"} {
		list, ok := got[name].([]any)
		if !ok || len(list) != 0 {
			t.Errorf("%s = %v, want []", name, got[name])
		}
	}
	if strings.Contains(string(data), "null") {
		t.Errorf("payload has null: %s", data)
	}
}

func TestNewOsuStrainsResponseWithoutWindowKeepsEverything(t *testing.T) {
	curves := &skills.Curves{Speed: 1, Agility: []skills.StrainPoint{{Time: 1, Value: 1}, {Time: 2, Value: 2}}}
	if got := NewOsuStrainsResponse(1, curves, 0); len(got.Agility) != 2 {
		t.Errorf("agility = %v", got.Agility)
	}
}
