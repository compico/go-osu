package model

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
)

// MD5Hash stores an MD5 digest as raw bytes rather than a 32-character hex
// string — halves storage per row (16 bytes vs 32) and makes index/equality
// comparisons operate on raw bytes instead of ASCII text. Comparable with
// == since it's a fixed-size array.
type MD5Hash [16]byte

// ParseMD5Hash decodes a 32-character hex string (as found in osu!.db /
// osu.DatabaseBeatmap.MD5Hash) into an MD5Hash.
func ParseMD5Hash(hexStr string) (MD5Hash, error) {
	var h MD5Hash
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return h, fmt.Errorf("decode md5 hex %q: %w", hexStr, err)
	}
	if len(b) != len(h) {
		return h, fmt.Errorf("md5 hex %q decodes to %d bytes, want %d", hexStr, len(b), len(h))
	}
	copy(h[:], b)
	return h, nil
}

// MustParseMD5Hash is ParseMD5Hash but panics on error — use only where the
// input is already known-good (e.g. constants in tests).
func MustParseMD5Hash(hexStr string) MD5Hash {
	h, err := ParseMD5Hash(hexStr)
	if err != nil {
		panic(err)
	}
	return h
}

func (h MD5Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Value implements driver.Valuer so sqlx/database-sql writes this as a
// BLOB automatically.
func (h MD5Hash) Value() (driver.Value, error) {
	return h[:], nil
}

// Scan implements sql.Scanner so sqlx/database-sql reads a BLOB column
// directly into this type.
func (h *MD5Hash) Scan(src any) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("MD5Hash.Scan: unsupported type %T", src)
	}
	if len(b) != len(h) {
		return fmt.Errorf("MD5Hash.Scan: got %d bytes, want %d", len(b), len(h))
	}
	copy(h[:], b)
	return nil
}
