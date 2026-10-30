// Package skills calculates osu! beatmap skill metrics from parsed hit
// objects, timing points, and difficulty settings.
//
// The current implementation follows the calculation structure of the
// original osuSkills port. Keep its numerical behavior covered by the
// reference values in skilll_refs_test.go when reorganizing the code.
//
// # Calculation flow
//
// ProcessBeatmap applies the requested mods to a copy of the source map and
// creates a MapData value to hold state for this one calculation. It then
// prepares derived data, calculates aim and tap strains, and aggregates those
// and other features into the final Skills result.
//
// Preparation currently builds timing-point data, slider curves and tick
// times, movement distances and velocities, target points, aim and reading
// points, angles, press intervals, and tap patterns. Later calculations reuse
// those derived slices. A MapData must not be shared between concurrent
// calculations that mutate it; the parsed source beatmap is treated as
// read-only.
//
// # Metrics
//
// Stamina aggregates the strongest tap strain. Tenacity scans stream patterns
// by interval and scores the first interval bucket containing a multi-note
// stream, using its interval and length. Agility aggregates peaks in aim
// strain. Precision scales agility using circle size. Accuracy combines
// stamina with OD and the number of circles through an SS-unranked-score
// estimate. Reaction evaluates sequences of target points for their required
// response time. Reading estimates difficulty from visible-object density,
// repeated angles, movement, and (under Hidden) visibility. Memory scores
// objects that cannot be inferred from nearby visible objects; ProcessBeatmap
// currently calculates it for Flashlight maps.
//
// These descriptions explain the current code path; the reference tests, not
// this summary, define the numerical compatibility contract.
package skills
