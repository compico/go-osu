package skills

import (
	"encoding/json"
	"os"
	"slices"
	"sort"
	"testing"
)

// referenceBeatmapIDs returns the IDs of the beatmaps skillsReference refers
// to, sorted and without duplicates.
func referenceBeatmapIDs() []int {
	var ids []int
	for _, ref := range skillsReference {
		if !slices.Contains(ids, ref.BeatmapID) {
			ids = append(ids, ref.BeatmapID)
		}
	}
	sort.Ints(ids)
	return ids
}

// reference_maps.json tells internal/tools/fetchrefs which maps to download.
// If it fell out of step with skillsReference, the reference test would either
// miss maps or download ones nothing uses.
func TestReferenceManifestMatchesReferences(t *testing.T) {
	data, err := os.ReadFile("reference_maps.json")
	if err != nil {
		t.Fatal(err)
	}

	var manifest []int
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("reference_maps.json: %v", err)
	}
	sort.Ints(manifest)

	want := referenceBeatmapIDs()
	for _, id := range want {
		if !slices.Contains(manifest, id) {
			t.Errorf("beatmap %d has a reference in skilll_refs_test.go but is not in reference_maps.json", id)
		}
	}
	for _, id := range manifest {
		if !slices.Contains(want, id) {
			t.Errorf("beatmap %d is in reference_maps.json but has no reference in skilll_refs_test.go", id)
		}
	}
	for i := 1; i < len(manifest); i++ {
		if manifest[i] == manifest[i-1] {
			t.Errorf("beatmap %d is listed twice in reference_maps.json", manifest[i])
		}
	}
}
