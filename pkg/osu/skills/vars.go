package skills

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Vars contains the formula parameters used by skill calculations.
//
// Create values with NewVars or LoadVars and treat them as immutable while
// sharing them between concurrent calculations.
type Vars struct {
	Stamina   StaminaVars   `json:"stamina"`
	Tenacity  TenacityVars  `json:"tenacity"`
	Agility   AgilityVars   `json:"agility"`
	Accuracy  AccuracyVars  `json:"accuracy"`
	Precision PrecisionVars `json:"precision"`
	Memory    MemoryVars    `json:"memory"`
	Reaction  ReactionVars  `json:"reaction"`
	Reading   ReadingVars   `json:"reading"`
}

// StaminaVars contains the parameters for Stamina calculations.
type StaminaVars struct {
	Decay           float64 `json:"decay"`
	Mult            float64 `json:"mult"`
	Pow             float64 `json:"pow"`
	DecayMax        float64 `json:"decayMax"`
	Scale           float64 `json:"scale"`
	LargestInterval float64 `json:"largestInterval"`
	TotalMult       float64 `json:"totalMult"`
	TotalPow        float64 `json:"totalPow"`
}

// TenacityVars contains the parameters for Tenacity calculations.
type TenacityVars struct {
	IntervalMult  float64 `json:"intervalMult"`
	IntervalMult2 float64 `json:"intervalMult2"`
	IntervalPow   float64 `json:"intervalPow"`
	LengthDivisor float64 `json:"lengthDivisor"`
	LengthMult    float64 `json:"lengthMult"`
	TotalMult     float64 `json:"totalMult"`
	TotalPow      float64 `json:"totalPow"`
}

// AgilityVars contains the parameters for Agility calculations.
type AgilityVars struct {
	DistMult          float64 `json:"distMult"`
	DistPow           float64 `json:"distPow"`
	DistDivisor       float64 `json:"distDivisor"`
	TimeMult          float64 `json:"timeMult"`
	TimePow           float64 `json:"timePow"`
	StrainDecay       float64 `json:"strainDecay"`
	AngleMult         float64 `json:"angleMult"`
	SliderStrainDecay float64 `json:"sliderStrainDecay"`
	Weighting         float64 `json:"weighting"`
	TotalMult         float64 `json:"totalMult"`
	TotalPow          float64 `json:"totalPow"`
}

// AccuracyVars contains the parameters for Accuracy calculations.
type AccuracyVars struct {
	AccScale  float64 `json:"accScale"`
	VerScale  float64 `json:"verScale"`
	TotalMult float64 `json:"totalMult"`
	TotalPow  float64 `json:"totalPow"`
}

// PrecisionVars contains the parameters for Precision calculations.
type PrecisionVars struct {
	AgilityLimit    float64 `json:"agilityLimit"`
	AgilityPow      float64 `json:"agilityPow"`
	AgilitySubtract float64 `json:"agilitySubtract"`
	TotalMult       float64 `json:"totalMult"`
	TotalPow        float64 `json:"totalPow"`
}

// MemoryVars contains the parameters for Memory calculations.
type MemoryVars struct {
	FollowpointsNerf float64 `json:"followpointsNerf"`
	SliderBuff       float64 `json:"sliderBuff"`
	TotalMult        float64 `json:"totalMult"`
	TotalPow         float64 `json:"totalPow"`
}

// ReactionVars contains the parameters for Reaction calculations.
type ReactionVars struct {
	AvgWeighting   float64 `json:"avgWeighting"`
	PatternDamping float64 `json:"patternDamping"`
	FadeinPercent  float64 `json:"fadeinPercent"`
	VerScale       float64 `json:"verScale"`
	CurveExp       float64 `json:"curveExp"`
}

// ReadingVars contains the parameters for Reading calculations.
type ReadingVars struct {
	StrainDecay float64 `json:"strainDecay"`
	Weighting   float64 `json:"weighting"`
	TotalMult   float64 `json:"totalMult"`
	TotalPow    float64 `json:"totalPow"`
}

// NewVars returns the built-in formula parameters.
func NewVars() *Vars {
	return &Vars{
		Stamina: StaminaVars{
			Decay:           0.94,
			Mult:            0.8,
			Pow:             0.1,
			DecayMax:        0,
			Scale:           7000,
			LargestInterval: 50550,
			TotalMult:       4.6,
			TotalPow:        0.75,
		},
		Tenacity: TenacityVars{
			IntervalMult:  0.37,
			IntervalMult2: 13000,
			IntervalPow:   0.143,
			LengthDivisor: 0.08,
			LengthMult:    15,
			TotalMult:     5,
			TotalPow:      0.75,
		},
		Agility: AgilityVars{
			DistMult:          1,
			DistPow:           1,
			DistDivisor:       2,
			TimeMult:          0.001,
			TimePow:           1.04,
			StrainDecay:       16.9201,
			AngleMult:         4,
			SliderStrainDecay: 2,
			Weighting:         0.78,
			TotalMult:         30,
			TotalPow:          0.28,
		},
		Accuracy: AccuracyVars{
			AccScale:  0.01,
			VerScale:  0.30,
			TotalMult: 23100,
			TotalPow:  1.3,
		},
		Precision: PrecisionVars{
			AgilityLimit:    700,
			AgilityPow:      0.1,
			AgilitySubtract: 0.995462,
			TotalMult:       20,
			TotalPow:        2,
		},
		Memory: MemoryVars{
			FollowpointsNerf: 0.8,
			SliderBuff:       1.1,
			TotalMult:        205,
			TotalPow:         0.3,
		},
		Reaction: ReactionVars{
			AvgWeighting:   0.7,
			PatternDamping: 0.15,
			FadeinPercent:  0.1,
			VerScale:       12.2,
			CurveExp:       0.64,
		},
		Reading: ReadingVars{
			StrainDecay: 0.01,
			Weighting:   0.78,
			TotalMult:   1,
			TotalPow:    1,
		},
	}
}

// LoadVars creates defaults and overlays the JSON values from path.
// Omitted fields keep their defaults. Unknown fields are rejected to catch
// misspelled parameter names before calculations begin. The JSON uses skill
// names and camelCase parameter names, for example: {"agility":{"timePow":1.1}}.
func LoadVars(path string) (*Vars, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skills vars %q: %w", path, err)
	}
	if err := validateVarsJSON(data); err != nil {
		return nil, fmt.Errorf("validate skills vars %q: %w", path, err)
	}

	vars := NewVars()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(vars); err != nil {
		return nil, fmt.Errorf("decode skills vars %q: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode skills vars %q: multiple JSON values", path)
		}
		return nil, fmt.Errorf("decode skills vars %q: %w", path, err)
	}

	return vars, nil
}

func validateVarsJSON(data []byte) error {
	var sections map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &sections); err != nil {
		return err
	}
	if sections == nil {
		return fmt.Errorf("expected a JSON object")
	}

	for section, fields := range sections {
		if fields == nil {
			return fmt.Errorf("section %q must be an object", section)
		}
		for name, value := range fields {
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return fmt.Errorf("field %q.%q cannot be null", section, name)
			}
		}
	}
	return nil
}

// DefaultVars is kept for source compatibility. New code should use NewVars.
//
// Deprecated: use NewVars.
func DefaultVars() *Vars {
	return NewVars()
}
