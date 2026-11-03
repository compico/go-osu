// Package outliers adds the "outliers" command: it lists the beatmaps whose
// skill is far above (or below) that of beatmaps with a similar star rating.
//
// It is a way to find suspects, not a target: skills do not have to follow the
// star rating, they are meant to replace it. A listed beatmap is one to look at,
// to see whether its skill is a formula's mistake or a real difference.
package outliers

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/compico/go-osu/cmd/commands"
	appcfg "github.com/compico/go-osu/internal/config"
	"github.com/compico/go-osu/internal/database"
	"github.com/compico/go-osu/internal/outliers"
	"github.com/urfave/cli/v3"
)

// skillColumns maps a --skill value to its skill_cache column. The column is
// put into the query, so only these names are accepted.
var skillColumns = map[string]string{
	"stamina":   "stamina",
	"tenacity":  "tenacity",
	"agility":   "agility",
	"precision": "precision",
	"reading":   "reading",
	"memory":    "memory",
	"accuracy":  "accuracy",
	"reaction":  "reaction",
}

func init() {
	flags := append(appcfg.Flags(),
		&cli.StringFlag{Name: "skill", Value: "reaction", Usage: "skill to check: stamina, tenacity, agility, precision, reading, memory, accuracy, reaction"},
		&cli.IntFlag{Name: "mods", Value: 0, Usage: "mod combination as a bit mask (0 = no mods, 64 = DT, 16 = HR)"},
		&cli.IntFlag{Name: "limit", Value: 50, Usage: "how many beatmaps to list"},
		&cli.BoolFlag{Name: "lowest", Usage: "list the beatmaps furthest below the star rating instead"},
		&cli.FloatFlag{Name: "min-stars", Value: 0, Usage: "ignore beatmaps below this star rating"},
		&cli.BoolFlag{Name: "csv", Usage: "print CSV instead of a table"},
		&cli.StringFlag{Name: "db", Usage: "path to the SQLite file (default: from the config)"},
	)

	commands.Append(&cli.Command{
		Name:  "outliers",
		Usage: "list beatmaps whose skill stands out from beatmaps of a similar star rating, to look for formula mistakes",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			return run(ctx, c, os.Stdout)
		},
	})
}

// row is one beatmap with its star rating and the skill that is checked.
type row struct {
	BeatmapID  int32   `db:"beatmap_id"`
	Title      string  `db:"song_title"`
	Artist     string  `db:"artist_name"`
	Difficulty string  `db:"difficulty"`
	Stars      float64 `db:"stars"`
	Skill      float64 `db:"skill"`
}

func run(ctx context.Context, c *cli.Command, out io.Writer) error {
	column, ok := skillColumns[c.String("skill")]
	if !ok {
		return fmt.Errorf("outliers: unknown skill %q", c.String("skill"))
	}

	cfg, err := appcfg.Load(c)
	if err != nil {
		return err
	}
	if p := c.String("db"); p != "" {
		cfg.Database.Path = p
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		return err
	}
	defer db.Close()

	// Only the standard game mode: the skills are defined for it.
	query := `SELECT b.beatmap_id AS beatmap_id,
	                 s.song_title AS song_title,
	                 s.artist_name AS artist_name,
	                 b.difficulty AS difficulty,
	                 b.stars_nomod AS stars,
	                 sc.` + column + ` AS skill
	          FROM skill_cache sc
	          JOIN beatmaps b ON b.beatmap_id = sc.beatmap_id
	          JOIN beatmapsets s ON s.beatmap_set_id = b.beatmap_set_id
	          WHERE sc.mods = ? AND b.mode = 0 AND b.stars_nomod > ?`
	var rows []row
	if err := db.SelectContext(ctx, &rows, query, c.Int("mods"), c.Float("min-stars")); err != nil {
		return fmt.Errorf("outliers: read the library: %w", err)
	}

	data := make([]outliers.Row, len(rows))
	for i, r := range rows {
		data[i] = outliers.Row{BeatmapID: r.BeatmapID, Title: r.Title, Artist: r.Artist, Difficulty: r.Difficulty, Stars: r.Stars, Skill: r.Skill}
	}

	model, err := outliers.Fit(data)
	if err != nil {
		return err
	}
	ranked := outliers.Rank(data, model)
	if c.Bool("lowest") {
		for i, j := 0, len(ranked)-1; i < j; i, j = i+1, j-1 {
			ranked[i], ranked[j] = ranked[j], ranked[i]
		}
	}
	if limit := c.Int("limit"); limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}

	fmt.Fprintf(os.Stderr, "%s, mods %d: %d beatmaps, %s ~ stars^%.2f, spread (sigma of ln) %.2f\n",
		c.String("skill"), c.Int("mods"), model.N, c.String("skill"), model.B, model.Sigma)

	if c.Bool("csv") {
		return writeCSV(out, ranked)
	}
	return writeTable(out, ranked)
}

func name(r outliers.Ranked) string {
	return fmt.Sprintf("%s - %s [%s]", r.Artist, r.Title, r.Difficulty)
}

func writeTable(out io.Writer, ranked []outliers.Ranked) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tRATIO\tZ\tSTARS\tSKILL\tEXPECTED\tBEATMAP")
	for _, r := range ranked {
		fmt.Fprintf(w, "%d\tx%.2f\t%+.1f\t%.2f\t%.0f\t%.0f\t%s\n", r.BeatmapID, r.Ratio, r.Z, r.Stars, r.Skill, r.Expected, name(r))
	}
	return w.Flush()
}

func writeCSV(out io.Writer, ranked []outliers.Ranked) error {
	w := csv.NewWriter(out)
	_ = w.Write([]string{"beatmap_id", "ratio", "z", "stars", "skill", "expected", "beatmap"})
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) }
	for _, r := range ranked {
		_ = w.Write([]string{strconv.Itoa(int(r.BeatmapID)), f(r.Ratio), f(r.Z), f(r.Stars), f(r.Skill), f(r.Expected), name(r)})
	}
	w.Flush()
	return w.Error()
}
