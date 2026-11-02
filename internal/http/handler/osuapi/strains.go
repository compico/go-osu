package osuapi

import (
	"database/sql"
	"errors"
	"io/fs"

	"github.com/compico/go-osu/internal/http/response"
	"github.com/compico/go-osu/internal/service"
	"github.com/compico/go-osu/pkg/osu/mods"
	"github.com/gofiber/fiber/v3"
)

const (
	// defaultStrainsWindow is the default resampling of the curves, in ms of the
	// beatmap's timeline. Per-note values are far too noisy to read as a line;
	// the strongest value in 5 s follows how the difficulty develops.
	defaultStrainsWindow = 5000
	maxStrainsWindow     = 60000
)

type OsuStrainsHandler struct {
	osuService *service.Osu
}

func NewOsuStrainsHandler(osuService *service.Osu) *OsuStrainsHandler {
	return &OsuStrainsHandler{osuService: osuService}
}

// Handle serves GET /api/osu/songs/:difficulty_id/strains?mods=HDDT&window=5000.
//
// mods is a concatenated mod string like the search DSL's mods= (default: no
// mods). window is the width of the resampling windows in ms of the beatmap's
// timeline (default 5000; 0 returns every point of every curve, per note).
func (h *OsuStrainsHandler) Handle(ctx fiber.Ctx) error {
	diffID := fiber.Params[int32](ctx, "difficulty_id", -100)
	if diffID == -100 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid difficulty_id"})
	}

	modsValue, err := mods.Parse(ctx.Query("mods", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	window := fiber.Query[int](ctx, "window", defaultStrainsWindow)
	if window < 0 || window > maxStrainsWindow {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "window must be between 0 and 60000 ms"})
	}

	curves, err := h.osuService.StrainCurves(ctx, diffID, modsValue)
	switch {
	case err == nil:
		return ctx.JSON(response.NewOsuStrainsResponse(diffID, curves, window))
	case errors.Is(err, sql.ErrNoRows), errors.Is(err, fs.ErrNotExist):
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Beatmap not found"})
	case errors.Is(err, service.ErrStrainCurvesTimeout):
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	default:
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
}
