package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// EncodeCursor serializes a sort tuple (as returned by sortTupleFor) into
// an opaque string safe to hand to the frontend and receive back verbatim.
func EncodeCursor(values []any) (string, error) {
	raw, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeCursor reverses EncodeCursor and coerces each element to the Go
// type sortKey.Types expects, since JSON round-trips numbers as float64
// and SQLite driver binding cares about matching Go types to column
// affinity for correct row-value comparisons.
func DecodeCursor(cursor string, sortKey SortKey) ([]any, error) {
	if cursor == "" {
		return nil, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var vals []any
	if err := json.Unmarshal(raw, &vals); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}
	if len(vals) != len(sortKey.Types) {
		return nil, fmt.Errorf("cursor has %d values, sort key %q expects %d", len(vals), sortKey.Name, len(sortKey.Types))
	}

	for i, t := range sortKey.Types {
		switch t {
		case SortText:
			s, ok := vals[i].(string)
			if !ok {
				return nil, fmt.Errorf("cursor value %d: expected string for sort %q", i, sortKey.Name)
			}
			vals[i] = s
		case SortNum:
			f, ok := vals[i].(float64) // JSON numbers decode as float64
			if !ok {
				return nil, fmt.Errorf("cursor value %d: expected number for sort %q", i, sortKey.Name)
			}
			vals[i] = f
		}
	}

	return vals, nil
}
