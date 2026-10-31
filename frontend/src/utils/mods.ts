// Mod handling for the search box. It mirrors the backend search DSL
// (internal/dslquery.Parse and pkg/osu/mods.Parse) so the badge in the
// details panel always names the mod combination the backend actually used
// for the current query.

/** Label for a query that selects no mods (or has no mods filter at all). */
export const NO_MOD_LABEL = 'NoMod'

// Field names the backend registers as FieldMode; "mods" is an alias of "mode".
const MOD_FIELDS = new Set(['mode', 'mods'])

// Same token shape as exprRe in internal/dslquery/query.go.
const EXPR_RE = /^([a-zA-Z_]+)(<=|>=|!=|=|<|>)(.+)$/

// Same set as modAbbreviations in pkg/osu/mods/parser.go.
const MOD_CODES = new Set(['NF', 'EZ', 'HD', 'HR', 'SD', 'DT', 'RL', 'HT', 'FL', 'AU', 'SO', 'AP'])

// Spellings the backend treats as "no mods" besides an empty value.
const NO_MOD_VALUES = new Set(['NM', 'NOMOD', 'NONE'])

/**
 * Turns the value of a mode=/mods= token (e.g. "hddt") into its label
 * ("HDDT"). Returns null when the backend would reject the value.
 */
function parseModsValue(value: string): string | null {
  const s = value.trim().toUpperCase()
  if (s === '' || NO_MOD_VALUES.has(s)) return NO_MOD_LABEL
  if (s.length % 2 !== 0) return null

  for (let i = 0; i < s.length; i += 2) {
    if (!MOD_CODES.has(s.slice(i, i + 2))) return null
  }
  return s
}

/**
 * Returns the mods label for a search query: "HDDT" for "mode=HDDT" or
 * "mods=hddt", "NoMod" when the query has no mods filter or selects none.
 *
 * Like the backend, tokens are split on whitespace, field names are
 * case-insensitive, and the last mode=/mods= token wins. A value the
 * backend rejects (it answers 400, so no results are shown) falls back to
 * "NoMod" rather than echoing the invalid text.
 */
export function modsLabelFromQuery(query: string): string {
  let label = NO_MOD_LABEL

  for (const token of query.split(/\s+/)) {
    const m = EXPR_RE.exec(token)
    if (!m || !MOD_FIELDS.has(m[1].toLowerCase())) continue
    label = parseModsValue(m[3]) ?? NO_MOD_LABEL
  }

  return label
}
