package skills

// Vars holds the tweakable formula constants used by skill calculations
// (ported from tweakvars.cpp's VARS map + GetVar()).
//
// Thread-safety note: in the original C++, VARS is a package-level mutable
// unordered_map that gets written to at startup (ResetFormulaVars /
// LoadFormulaVars) and then only read from during calculations. That's safe
// in a single-threaded app, but is a data race waiting to happen if any two
// goroutines ever called a "load" and a "calculate" concurrently.
//
// Here, *Vars is built once by DefaultVars() (or, later, LoadVars(path))
// and never mutated afterwards — every field is set exactly once during
// construction. Because of that, a single *Vars instance can be freely
// shared and read concurrently by any number of goroutines without a mutex:
// there is no writer to race with a reader.
type Vars struct {
	values map[string]map[string]float64
}

// Get returns the named formula variable for a skill. Like the original
// VARS[skill][name] lookup on a nested map, an unknown skill or variable
// name simply returns the zero value (0.0) rather than an error.
func (v *Vars) Get(skill, name string) float64 {
	return v.values[skill][name]
}

// DefaultVars returns the built-in default values (ResetFormulaVars in the
// original code). Only the "Reading" section is populated so far — the
// other skills' sections will be filled in as each of them gets ported.
func DefaultVars() *Vars {
	return &Vars{
		values: map[string]map[string]float64{
			"Reading": {
				"StrainDecay": 0.01,
				"Weighting":   0.78,
				"TotalMult":   1,
				"TotalPow":    1,
			},
			"Stamina": {
				"Weighting": 0.78,
				"TotalMult": 1,
				"TotalPow":  1,
			},
			"Agility": {
				"DistMult":          1,
				"DistPow":           1,
				"DistDivisor":       2,
				"TimeMult":          0.001,
				"TimePow":           1.04,
				"StrainDecay":       16.9201,
				"AngleMult":         4,
				"SliderStrainDecay": 2,
				"Weighting":         0.78,
				"TotalMult":         30,
				"TotalPow":          0.28,
			},
			"Tenacity": {
				"Weighting": 0.78,
				"TotalMult": 1,
				"TotalPow":  1,
			},
			"Precision": {
				"Weighting": 0.78,
				"TotalMult": 1,
				"TotalPow":  1,
			},
			"Reaction": {
				"Weighting": 0.78,
				"TotalMult": 1,
				"TotalPow":  1,
			},
			"Memory": {
				"Weighting": 0.78,
				"TotalMult": 1,
				"TotalPow":  1,
			},
			"Accuracy": {
				"Weighting": 0.78,
				"TotalMult": 1,
				"TotalPow":  1,
			},
		},
	}
}
