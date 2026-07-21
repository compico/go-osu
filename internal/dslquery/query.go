package dslquery

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/mods"
)

type Operator string

const (
	OpLT  Operator = "<"
	OpLTE Operator = "<="
	OpGT  Operator = ">"
	OpGTE Operator = ">="
	OpEQ  Operator = "="
	OpNEQ Operator = "!="
)

// textOperators are the only operators meaningful for FieldText columns.
// "=" and "!=" are treated as "contains" / "does not contain" (LIKE-based)
// rather than exact string equality — artist=Camellia should match
// "Camellia feat. Nanahira", not require an exact full-string match.
var textOperators = map[Operator]bool{OpEQ: true, OpNEQ: true}

type Condition struct {
	Field    string // canonical lowercase field name as typed, e.g. "star"
	Def      FieldDef
	Operator Operator

	NumValue float64 // valid when Def.Kind is FieldBeatmap or FieldSkill
	StrValue string  // valid when Def.Kind is FieldText
}

type Query struct {
	Conditions []Condition
	FreeText   []string

	Mods    osu.Mod
	HasMods bool
}

// exprRe matches "<field><op><value>" tokens with no internal whitespace,
// e.g. "stars<10", "mode=HRHD", "artist=Camellia". Two-character operators
// are listed before their one-character prefixes in the alternation so
// "<=input" matches "<=" rather than "<" + a mis-parsed "=input" value.
var exprRe = regexp.MustCompile(`^([a-zA-Z_]+)(<=|>=|!=|=|<|>)(.+)$`)

// Parse tokenizes raw on whitespace (collapsing any run of spaces/tabs)
// and classifies each token as either a "field<op>value" expression or a
// bare free-text term. Unknown field names fall back to free text for the
// whole token, mirroring how osu!'s own search box silently ignores
// unrecognized filter syntax rather than erroring the whole query.
func Parse(raw string) (*Query, error) {
	q := &Query{}

	for _, tok := range strings.Fields(raw) {
		m := exprRe.FindStringSubmatch(tok)
		if m == nil {
			q.FreeText = append(q.FreeText, tok)
			continue
		}

		fieldRaw, opRaw, valRaw := strings.ToLower(m[1]), Operator(m[2]), m[3]
		def, ok := fieldRegistry[fieldRaw]
		if !ok {
			q.FreeText = append(q.FreeText, tok)
			continue
		}

		switch def.Kind {
		case FieldMode:
			mods, err := mods.Parse(valRaw)
			if err != nil {
				return nil, fmt.Errorf("invalid value for %q: %w", fieldRaw, err)
			}
			q.Mods = mods
			q.HasMods = true

		case FieldText:
			if !textOperators[opRaw] {
				return nil, fmt.Errorf("field %q only supports = or !=, got %q", fieldRaw, opRaw)
			}
			q.Conditions = append(q.Conditions, Condition{
				Field:    fieldRaw,
				Def:      def,
				Operator: opRaw,
				StrValue: valRaw,
			})

		default: // FieldBeatmap, FieldSkill
			val, err := strconv.ParseFloat(valRaw, 64)
			if err != nil {
				return nil, fmt.Errorf("field %q expects a number, got %q", fieldRaw, valRaw)
			}
			q.Conditions = append(q.Conditions, Condition{
				Field:    fieldRaw,
				Def:      def,
				Operator: opRaw,
				NumValue: val,
			})
		}
	}

	return q, nil
}

// NeedsSkillJoin reports whether compiling this query requires joining
// skill_cache — either because a skill field is filtered on, or because
// mode=/mods= was given (needed to pick which mod combination's row to
// join against, even if no skill field itself is filtered).
func (q *Query) NeedsSkillJoin() bool {
	if q.HasMods {
		return true
	}
	for _, c := range q.Conditions {
		if c.Def.Kind == FieldSkill {
			return true
		}
	}
	return false
}
