package osuapi

import (
	"errors"

	"github.com/compico/go-osu/internal/dslquery"
	"github.com/compico/go-osu/internal/http/response"
	"github.com/compico/go-osu/internal/repository"
	"github.com/gofiber/fiber/v3"
)

type OsuQueueAdjacentHandler struct {
	repos *repository.Repos
}

func NewOsuQueueAdjacentHandler(repos *repository.Repos) *OsuQueueAdjacentHandler {
	return &OsuQueueAdjacentHandler{repos}
}

func (h *OsuQueueAdjacentHandler) Handle(ctx fiber.Ctx) error {
	q, err := dslquery.Parse(ctx.Query("q"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	currentSetID := fiber.Query[int32](ctx, "current_set_id")
	dir := repository.AdjacentDirection(ctx.Query("dir", "next"))
	sort := ctx.Query("sort", "artist_name") // default now; frontend sort selector plugs in later without backend changes

	nextSetID, err := h.repos.Beatmaps.AdjacentGroup(ctx.Context(), q, currentSetID, dir, sort)
	if errors.Is(err, repository.ErrNoAdjacent) {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no adjacent track"})
	}
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	diff, err := h.repos.Beatmaps.FirstDiff(ctx.Context(), nextSetID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	set, err := h.repos.BeatmapSets.Get(ctx.Context(), nextSetID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(response.NewOsuQueueAdjacentResponse(set, diff))
}
