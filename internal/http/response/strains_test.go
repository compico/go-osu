package response

import (
	"encoding/json"
	"reflect"
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

	// Windows of 400 ms of the beatmap's time (speed 1.5): the points at 0 and
	// 100 are in the first, 700 (1050 ms of the beatmap) in the third, the second
	// is empty (0).
	stamina, _ := got["stamina"].([]any)
	if len(stamina) != 3 {
		t.Fatalf("stamina = %v, want a point per window", got["stamina"])
	}
	for i, want := range []float64{3, 0, 2} {
		if p, _ := stamina[i].(map[string]any); p["value"] != want {
			t.Errorf("stamina[%d] = %v, want value %v", i, stamina[i], want)
		}
	}
	if p, _ := stamina[1].(map[string]any); p["time"] != 400.0 { // the middle of [400, 800) is 600 / 1.5
		t.Errorf("stamina[1] = %v, want time 400", stamina[1])
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

func TestNewOsuStrainsResponseMemoryKeepsItsValue(t *testing.T) {
	curves := &skills.Curves{Speed: 1, Memory: []skills.StrainPoint{{Time: 100, Value: 5}, {Time: 2500, Value: 9}}}
	got := NewOsuStrainsResponse(1, curves, 1000).Memory
	var values []float64
	for _, p := range got {
		values = append(values, p.Value)
	}
	if !reflect.DeepEqual(values, []float64{5, 5, 9}) {
		t.Errorf("memory = %v, want it to carry its value over the empty window", values)
	}
}

func TestNewOsuStrainsResponseWithoutWindowKeepsEverything(t *testing.T) {
	curves := &skills.Curves{Speed: 1, Agility: []skills.StrainPoint{{Time: 1, Value: 1}, {Time: 2, Value: 2}}}
	if got := NewOsuStrainsResponse(1, curves, 0); len(got.Agility) != 2 {
		t.Errorf("agility = %v", got.Agility)
	}
}
