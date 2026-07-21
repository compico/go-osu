package dslquery

import "testing"

func TestParse_CollapsesExtraWhitespace(t *testing.T) {
	// The literal case from the task: multiple spaces after "stars<10".
	q, err := Parse("stars<10   bpm>200 freedom mode=HRHD stamina>30 ar=9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(q.Conditions) != 4 {
		t.Fatalf("expected 4 conditions, got %d: %+v", len(q.Conditions), q.Conditions)
	}
	if len(q.FreeText) != 1 || q.FreeText[0] != "freedom" {
		t.Fatalf("expected free text [freedom], got %v", q.FreeText)
	}
	if !q.HasMods {
		t.Fatal("expected HasMods to be true")
	}
}

func TestParse_TextFieldsAndStarAlias(t *testing.T) {
	q, err := Parse("star<6 artist=Camellia creator!=Nathan title=Bohemian difficulty=Insane")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(q.Conditions) != 5 {
		t.Fatalf("expected 5 conditions, got %d: %+v", len(q.Conditions), q.Conditions)
	}

	star := q.Conditions[0]
	if star.Def.Column != "stars_nomod" {
		t.Errorf("expected star to alias to stars_nomod column, got %q", star.Def.Column)
	}

	artist := q.Conditions[1]
	if artist.Def.Kind != FieldText || artist.StrValue != "Camellia" {
		t.Errorf("unexpected artist condition: %+v", artist)
	}

	creator := q.Conditions[2]
	if creator.Operator != OpNEQ {
		t.Errorf("expected != on creator, got %q", creator.Operator)
	}
}

func TestParse_TextFieldRejectsComparisonOperators(t *testing.T) {
	_, err := Parse("artist<Camellia")
	if err == nil {
		t.Fatal("expected error for < on a text field, got nil")
	}
}
