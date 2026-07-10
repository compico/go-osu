package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
)

// sqliteMaxVariableNumber matches SQLITE_MAX_VARIABLE_NUMBER as configured
// in the driver build this project uses.
const sqliteMaxVariableNumber = 32766

// targetBatchSize is the row count we aim for per statement — covers ~99%
// of real upsert calls in this codebase for narrow tables. Wide tables get
// automatically capped lower by batchSizeFor so we never exceed
// sqliteMaxVariableNumber regardless of column count.
const targetBatchSize = 1000

func batchSizeFor(numCols int) int {
	if numCols <= 0 {
		return targetBatchSize
	}
	m := sqliteMaxVariableNumber / numCols
	if m < 1 {
		m = 1
	}
	if m > targetBatchSize {
		return targetBatchSize
	}
	return m
}

type tableMeta struct {
	columns    []string
	fieldIndex []int // struct field index that produced columns[i]
}

var metaCache sync.Map // reflect.Type -> tableMeta

func getTableMeta(t reflect.Type) tableMeta {
	if cached, ok := metaCache.Load(t); ok {
		return cached.(tableMeta)
	}

	meta := tableMeta{}
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		meta.columns = append(meta.columns, strings.Split(tag, ",")[0])
		meta.fieldIndex = append(meta.fieldIndex, i)
	}

	metaCache.Store(t, meta)
	return meta
}

func (m tableMeta) rowArgs(v reflect.Value) []any {
	args := make([]any, len(m.fieldIndex))
	for i, fi := range m.fieldIndex {
		args[i] = v.Field(fi).Interface()
	}
	return args
}

func (m tableMeta) placeholder() string {
	return "(" + strings.TrimSuffix(strings.Repeat("?,", len(m.columns)), ",") + ")"
}

func (m tableMeta) updateClause(conflictCols []string) string {
	skip := make(map[string]bool, len(conflictCols))
	for _, c := range conflictCols {
		skip[c] = true
	}
	sets := make([]string, 0, len(m.columns))
	for _, c := range m.columns {
		if !skip[c] {
			sets = append(sets, c+" = excluded."+c)
		}
	}
	return strings.Join(sets, ", ")
}

// UpsertBatch writes rows (must be a slice of a struct with `db` tags, e.g.
// []model.Beatmap) into table in chunks sized to stay under
// sqliteMaxVariableNumber, all inside one transaction. conflictCols are the
// PK/unique columns used for ON CONFLICT ... DO UPDATE.
//
//	err := db.UpsertBatch(ctx, "beatmaps", []string{"beatmap_id"}, beatmaps)
func (db *DB) UpsertBatch(ctx context.Context, table string, conflictCols []string, rows any) error {
	v := reflect.ValueOf(rows)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("UpsertBatch(%s): rows must be a slice, got %s", table, v.Kind())
	}
	n := v.Len()
	if n == 0 {
		return nil
	}

	meta := getTableMeta(v.Type().Elem())
	size := batchSizeFor(len(meta.columns))

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for %s: %w", table, err)
	}
	defer func(tx *sqlx.Tx) {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			db.Logger.ErrorContext(ctx, err.Error())
		}
	}(tx)

	for start := 0; start < n; start += size {
		end := start + size
		if end > n {
			end = n
		}
		if err := db.upsertChunk(ctx, tx, table, meta, conflictCols, v.Slice(start, end)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s batch: %w", table, err)
	}
	return nil
}

func (db *DB) upsertChunk(ctx context.Context, tx *sqlx.Tx, table string, meta tableMeta, conflictCols []string, chunk reflect.Value) error {
	n := chunk.Len()
	placeholders := make([]string, n)
	args := make([]any, 0, n*len(meta.columns))

	for i := 0; i < n; i++ {
		placeholders[i] = meta.placeholder()
		args = append(args, meta.rowArgs(chunk.Index(i))...)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (%s) VALUES %s
		ON CONFLICT(%s) DO UPDATE SET %s
	`,
		table,
		strings.Join(meta.columns, ", "),
		strings.Join(placeholders, ","),
		strings.Join(conflictCols, ", "),
		meta.updateClause(conflictCols),
	)

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert %s chunk (%d rows): %w", table, n, err)
	}
	return nil
}

// PluckHashes returns column[keyCol] -> column[hashCol] for every row of
// table. Used for cheap change-detection passes (e.g. beatmap_id -> md5_hash)
// without pulling full rows into memory.
func (db *DB) PluckHashes(ctx context.Context, table, keyCol, hashCol string) (map[int32]string, error) {
	rows, err := db.QueryxContext(ctx, fmt.Sprintf(`SELECT %s, %s FROM %s`, keyCol, hashCol, table))
	if err != nil {
		return nil, fmt.Errorf("pluck hashes from %s: %w", table, err)
	}
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			db.Logger.ErrorContext(ctx, err.Error())
		}
	}(rows)

	out := make(map[int32]string)
	for rows.Next() {
		var id int32
		var hash string
		if err := rows.Scan(&id, &hash); err != nil {
			return nil, fmt.Errorf("scan hash row from %s: %w", table, err)
		}
		out[id] = hash
	}
	return out, rows.Err()
}

var _ = sql.ErrNoRows // re-exported implicitly via sqlx errors, kept for readability at call sites
