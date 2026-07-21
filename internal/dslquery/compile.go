package dslquery

import (
	"fmt"
	"strings"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}

// Compile turns the parsed Query into a parameterized SQL WHERE clause
// (no leading "WHERE") plus its positional args. Column names come only
// from fieldRegistry (never raw user input), and every value is bound as
// a placeholder — injection-safe regardless of what the user typed.
func (q *Query) Compile() (where string, args []any) {
	var clauses []string

	for _, c := range q.Conditions {
		switch c.Def.Kind {
		case FieldBeatmap:
			clauses = append(clauses, fmt.Sprintf("b.%s %s ?", c.Def.Column, c.Operator))
			args = append(args, c.NumValue)

		case FieldSkill:
			clauses = append(clauses, fmt.Sprintf("sc.%s %s ?", c.Def.Column, c.Operator))
			args = append(args, c.NumValue)

		case FieldText:
			like := "%" + escapeLike(c.StrValue) + "%"
			if c.Operator == OpNEQ {
				clauses = append(clauses, fmt.Sprintf(`%s NOT LIKE ? ESCAPE '\'`, c.Def.Column))
			} else {
				clauses = append(clauses, fmt.Sprintf(`%s LIKE ? ESCAPE '\'`, c.Def.Column))
			}
			args = append(args, like)
		}
	}

	for _, term := range q.FreeText {
		like := "%" + escapeLike(term) + "%"

		parts := make([]string, len(freeTextColumns))
		for i, col := range freeTextColumns {
			parts[i] = fmt.Sprintf(`%s LIKE ? ESCAPE '\'`, col)
			args = append(args, like)
		}
		clauses = append(clauses, "("+strings.Join(parts, " OR ")+")")
	}

	if len(clauses) == 0 {
		return "1 = 1", args
	}

	return strings.Join(clauses, " AND "), args
}
