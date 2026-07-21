package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/compico/go-osu/internal/database"
	"github.com/compico/go-osu/internal/dslquery"
	"github.com/compico/go-osu/internal/model"
	"github.com/jmoiron/sqlx"
)

type BeatmapRepo struct {
	db *database.DB
}

func NewBeatmapRepo(db *database.DB) *BeatmapRepo {
	return &BeatmapRepo{db: db}
}

func (r *BeatmapRepo) UpsertBatch(ctx context.Context, beatmaps []model.Beatmap) error {
	return r.db.UpsertBatch(ctx, "beatmaps", []string{"beatmap_id"}, beatmaps)
}

func (r *BeatmapRepo) List(ctx context.Context) ([]model.Beatmap, error) {
	var beatmaps []model.Beatmap
	if err := r.db.SelectContext(ctx, &beatmaps, `SELECT * FROM beatmaps`); err != nil {
		return nil, fmt.Errorf("list beatmaps: %w", err)
	}
	return beatmaps, nil
}

// Hashes returns beatmap_id -> md5_hash for every beatmap currently stored.
// This is the "pluck" you're after: call it before writing a fresh parse of
// osu!.db to know what was there previously, then diff against the newly
// parsed hashes to find changed/new difficulties.
func (r *BeatmapRepo) Hashes(ctx context.Context) (map[int32]model.MD5Hash, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT beatmap_id, md5_hash FROM beatmaps`)
	if err != nil {
		return nil, fmt.Errorf("get beatmap hashes: %w", err)
	}
	defer rows.Close()

	out := make(map[int32]model.MD5Hash)
	for rows.Next() {
		var id int32
		var hash model.MD5Hash
		if err := rows.Scan(&id, &hash); err != nil {
			return nil, fmt.Errorf("scan beatmap hash row: %w", err)
		}
		out[id] = hash
	}

	return out, rows.Err()
}

// DeleteMissing removes beatmaps whose id is not in keepIDs — used after a
// sync pass to drop rows for maps the player deleted from the game since
// the last run. ON DELETE CASCADE on skill_cache handles cleanup there.
func (r *BeatmapRepo) DeleteMissing(ctx context.Context, keepIDs []int32) error {
	if len(keepIDs) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`DELETE FROM beatmaps WHERE beatmap_id NOT IN (?)`, keepIDs)
	if err != nil {
		return fmt.Errorf("build delete-missing query: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, r.db.Rebind(query), args...); err != nil {
		return fmt.Errorf("delete missing beatmaps: %w", err)
	}
	return nil
}

func (r *BeatmapRepo) Search(ctx context.Context, q *dslquery.Query, limit, offset int) ([]model.Beatmap, int, error) {
	where, whereArgs := q.Compile()

	mods := int32(0)
	if q.HasMods {
		mods = int32(q.Mods)
	}

	var sb strings.Builder
	args := make([]any, 0, len(whereArgs)+3)

	sb.WriteString(`SELECT b.* FROM beatmaps b JOIN beatmapsets bs ON bs.beatmap_set_id = b.beatmap_set_id `)

	sb.WriteString(`JOIN skill_cache sc ON sc.beatmap_id = b.beatmap_id AND sc.mods = ? `)
	args = append(args, mods)

	sb.WriteString(`WHERE `)
	sb.WriteString(where)
	args = append(args, whereArgs...)

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM (%s)`, sb.String())
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count search results: %w", err)
	}

	sb.WriteString(` LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)

	var rows []model.Beatmap
	if err := r.db.SelectContext(ctx, &rows, sb.String(), args...); err != nil {
		return nil, 0, fmt.Errorf("search beatmaps: %w", err)
	}

	return rows, total, nil
}

var ErrNoAdjacent = errors.New("no adjacent beatmap set")

type AdjacentDirection string

const (
	AdjacentNext AdjacentDirection = "next"
	AdjacentPrev AdjacentDirection = "prev"
)

// AdjacentGroup finds the next/previous beatmapset matching q's predicate,
// ordered by sortName (falls back to the default sort if unrecognized).
// GROUP BY bs.beatmap_set_id collapses the per-diff JOIN rows to one row
// per group regardless of whether sortKey's expressions are aggregated
// (e.g. "stars") or plain group columns (e.g. "artist_name") — both cases
// go through the exact same query shape, so adding a new sort key never
// requires touching this method.
func (r *BeatmapRepo) AdjacentGroup(ctx context.Context, q *dslquery.Query, currentSetID int32, dir AdjacentDirection, sortName string) (int32, error) {
	sortKey := resolveSort(sortName)

	currentValues, err := r.sortTupleFor(ctx, sortKey, currentSetID)
	if err != nil {
		return 0, fmt.Errorf("load current sort tuple: %w", err)
	}

	where, whereArgs := q.Compile()
	mods := int32(0)
	if q.HasMods {
		mods = int32(q.Mods)
	}

	cmp, order := ">", "ASC"
	if dir == AdjacentPrev {
		cmp, order = "<", "DESC"
	}

	tuple := "(" + strings.Join(sortKey.OrderExprs, ", ") + ")"
	placeholders := "(" + strings.TrimSuffix(strings.Repeat("?,", len(sortKey.OrderExprs)), ",") + ")"
	orderBy := make([]string, len(sortKey.OrderExprs))
	for i, expr := range sortKey.OrderExprs {
		orderBy[i] = expr + " " + order
	}

	var sb strings.Builder
	sb.WriteString(`SELECT bs.beatmap_set_id
		FROM beatmapsets bs
		JOIN beatmaps b ON b.beatmap_set_id = bs.beatmap_set_id `)

	args := make([]any, 0, len(whereArgs)+len(currentValues)+1)

	sb.WriteString(`JOIN skill_cache sc ON sc.beatmap_id = b.beatmap_id AND sc.mods = ? `)
	args = append(args, mods)

	fmt.Fprintf(&sb, `WHERE %s
		GROUP BY bs.beatmap_set_id
		HAVING %s %s %s
		ORDER BY %s
		LIMIT 1`, where, tuple, cmp, placeholders, strings.Join(orderBy, ", "))

	args = append(args, whereArgs...)
	args = append(args, currentValues...)

	var next struct {
		BeatmapSetID int32 `db:"beatmap_set_id"`
	}
	err = r.db.GetContext(ctx, &next, sb.String(), args...)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNoAdjacent
	}
	if err != nil {
		return 0, fmt.Errorf("find adjacent group: %w", err)
	}

	return next.BeatmapSetID, nil
}

// sortTupleFor evaluates sortKey.OrderExprs for a single known
// beatmap_set_id, returning the values in the same order/shape needed for
// the row-value comparison in AdjacentGroup. Always joins beatmaps and
// GROUP BYs, same as the main query, so aggregate expressions like
// MIN(b.stars_nomod) resolve identically in both places.
func (r *BeatmapRepo) sortTupleFor(ctx context.Context, sortKey SortKey, beatmapSetID int32) ([]any, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM beatmapsets bs
		JOIN beatmaps b ON b.beatmap_set_id = bs.beatmap_set_id
		WHERE bs.beatmap_set_id = ?
		GROUP BY bs.beatmap_set_id`,
		strings.Join(sortKey.OrderExprs, ", "),
	)

	rows, err := r.db.QueryxContext(ctx, query, beatmapSetID)
	if err != nil {
		return nil, fmt.Errorf("query sort tuple for set %d: %w", beatmapSetID, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("beatmap set %d not found or has no beatmaps", beatmapSetID)
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("read columns: %w", err)
	}

	values := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, fmt.Errorf("scan sort tuple: %w", err)
	}

	return values, rows.Err()
}

// FirstDiff picks the diff shown/played first for a group — lowest star
// rating, mirroring the current frontend's enrich() sort.
func (r *BeatmapRepo) FirstDiff(ctx context.Context, beatmapSetID int32) (*model.Beatmap, error) {
	var bm model.Beatmap
	err := r.db.GetContext(ctx, &bm, `
		SELECT * FROM beatmaps
		WHERE beatmap_set_id = ?
		ORDER BY stars_nomod ASC, beatmap_id ASC
		LIMIT 1`, beatmapSetID)
	if err != nil {
		return nil, fmt.Errorf("first diff of set %d: %w", beatmapSetID, err)
	}
	return &bm, nil
}

func (r *BeatmapRepo) Get(ctx context.Context, beatmapID int32) (*model.Beatmap, error) {
	var bm model.Beatmap
	err := r.db.GetContext(ctx, &bm, `SELECT * FROM beatmaps WHERE beatmap_id = ?`, beatmapID)
	if err != nil {
		return nil, fmt.Errorf("get beatmap %d: %w", beatmapID, err)
	}
	return &bm, nil
}

type GroupResult struct {
	Set   model.BeatmapSet
	Diffs []model.Beatmap
}

// SearchGroups is the keyset-paginated equivalent of Laravel's chunkById,
// generalized to any sort key: instead of "WHERE id > ?" it uses SQLite's
// row-value comparison "WHERE (col1, col2, ...) > (?, ?, ...)" against the
// sort tuple of the last item on the previous page. No OFFSET anywhere —
// cost stays flat regardless of how deep the user has scrolled.
func (r *BeatmapRepo) SearchGroups(ctx context.Context, q *dslquery.Query, sortName, cursor string, limit int) ([]GroupResult, string, error) {
	sortKey := resolveSort(sortName)

	cursorValues, err := DecodeCursor(cursor, sortKey)
	if err != nil {
		return nil, "", fmt.Errorf("decode cursor: %w", err)
	}

	where, whereArgs := q.Compile()
	mods := int32(0)
	if q.HasMods {
		mods = int32(q.Mods)
	}

	tuple := "(" + strings.Join(sortKey.OrderExprs, ", ") + ")"
	orderBy := make([]string, len(sortKey.OrderExprs))
	for i, expr := range sortKey.OrderExprs {
		orderBy[i] = expr + " ASC"
	}

	var sb strings.Builder
	sb.WriteString(`SELECT bs.beatmap_set_id
		FROM beatmapsets bs
		JOIN beatmaps b ON b.beatmap_set_id = bs.beatmap_set_id `)

	args := make([]any, 0, len(whereArgs)+len(cursorValues)+1)

	sb.WriteString(`JOIN skill_cache sc ON sc.beatmap_id = b.beatmap_id AND sc.mods = ? `)
	args = append(args, mods)

	sb.WriteString(`WHERE `)
	sb.WriteString(where)
	args = append(args, whereArgs...)

	sb.WriteString(` GROUP BY bs.beatmap_set_id`)
	if len(cursorValues) > 0 {
		placeholders := "(" + strings.TrimSuffix(strings.Repeat("?,", len(sortKey.OrderExprs)), ",") + ")"
		fmt.Fprintf(&sb, ` HAVING %s > %s`, tuple, placeholders)
		args = append(args, cursorValues...)
	}

	fmt.Fprintf(&sb, ` ORDER BY %s LIMIT ?`, strings.Join(orderBy, ", "))
	args = append(args, limit)

	var setIDs []int32
	if err := r.db.SelectContext(ctx, &setIDs, sb.String(), args...); err != nil {
		return nil, "", fmt.Errorf("search groups: %w", err)
	}
	if len(setIDs) == 0 {
		return nil, "", nil
	}

	sets, err := r.beatmapSetsByIDsOrdered(ctx, setIDs)
	if err != nil {
		return nil, "", err
	}
	diffsBySet, err := r.diffsGroupedBySetID(ctx, q, setIDs) // теперь передаём q
	if err != nil {
		return nil, "", err
	}

	results := make([]GroupResult, 0, len(setIDs))
	for _, id := range setIDs {
		results = append(results, GroupResult{Set: sets[id], Diffs: diffsBySet[id]})
	}

	nextTuple, err := r.sortTupleFor(ctx, sortKey, setIDs[len(setIDs)-1])
	if err != nil {
		return nil, "", err
	}
	nextCursor, err := EncodeCursor(nextTuple)
	if err != nil {
		return nil, "", err
	}

	return results, nextCursor, nil
}

func (r *BeatmapRepo) beatmapSetsByIDsOrdered(ctx context.Context, ids []int32) (map[int32]model.BeatmapSet, error) {
	query, args, err := sqlx.In(`SELECT * FROM beatmapsets WHERE beatmap_set_id IN (?)`, ids)
	if err != nil {
		return nil, fmt.Errorf("build beatmapsets query: %w", err)
	}
	var rows []model.BeatmapSet
	if err := r.db.SelectContext(ctx, &rows, r.db.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("load beatmapsets: %w", err)
	}
	out := make(map[int32]model.BeatmapSet, len(rows))
	for _, row := range rows {
		out[row.BeatmapSetID] = row
	}
	return out, nil
}

func (r *BeatmapRepo) diffsGroupedBySetID(ctx context.Context, q *dslquery.Query, setIDs []int32) (map[int32][]model.Beatmap, error) {
	where, whereArgs := q.Compile()
	mods := int32(0)
	if q.HasMods {
		mods = int32(q.Mods)
	}

	inPlaceholders := "(" + strings.TrimSuffix(strings.Repeat("?,", len(setIDs)), ",") + ")"

	var sb strings.Builder
	sb.WriteString(`SELECT b.*
		FROM beatmaps b
		JOIN beatmapsets bs ON bs.beatmap_set_id = b.beatmap_set_id
		JOIN skill_cache sc ON sc.beatmap_id = b.beatmap_id AND sc.mods = ? `)

	args := make([]any, 0, 1+len(whereArgs)+len(setIDs))
	args = append(args, mods)

	fmt.Fprintf(&sb, `WHERE b.beatmap_set_id IN %s AND %s
		ORDER BY b.stars_nomod ASC`, inPlaceholders, where)

	for _, id := range setIDs {
		args = append(args, id)
	}
	args = append(args, whereArgs...)

	var rows []model.Beatmap
	if err := r.db.SelectContext(ctx, &rows, sb.String(), args...); err != nil {
		return nil, fmt.Errorf("load filtered diffs: %w", err)
	}

	out := make(map[int32][]model.Beatmap)
	for _, row := range rows {
		out[row.BeatmapSetID] = append(out[row.BeatmapSetID], row)
	}
	return out, nil
}
