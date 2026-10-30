package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadVarsMergesJSONOverridesWithDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vars.json")
	data := `{"stamina":{"decay":0.5},"agility":{"timePow":1.2}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	vars, err := LoadVars(path)
	if err != nil {
		t.Fatal(err)
	}

	if vars.Stamina.Decay != 0.5 {
		t.Errorf("Stamina.Decay = %v, want override 0.5", vars.Stamina.Decay)
	}
	if vars.Stamina.Scale != 7000 {
		t.Errorf("Stamina.Scale = %v, want default 7000", vars.Stamina.Scale)
	}
	if vars.Agility.TimePow != 1.2 {
		t.Errorf("Agility.TimePow = %v, want override 1.2", vars.Agility.TimePow)
	}
	if vars.Agility.TimeMult != 0.001 {
		t.Errorf("Agility.TimeMult = %v, want default 0.001", vars.Agility.TimeMult)
	}
}

func TestLoadVarsRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vars.json")
	data := `{"agility":{"timePwo":1.2}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadVars(path)
	if err == nil {
		t.Fatal("LoadVars accepted an unknown field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("LoadVars error = %q, want unknown-field error", err)
	}
}

func TestLoadVarsRejectsNullValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vars.json")
	data := `{"stamina":{"decay":null}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadVars(path); err == nil {
		t.Fatal("LoadVars accepted a null formula value")
	}
}
