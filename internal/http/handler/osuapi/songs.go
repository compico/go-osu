package osuapi

import (
	"github.com/compico/go-osu/internal/dslquery"
	"github.com/compico/go-osu/internal/repository"
	"github.com/gofiber/fiber/v3"
)

type OsuSongsHandler struct {
	repos *repository.Repos
}

func NewOsuSongsHandler(repos *repository.Repos) *OsuSongsHandler {
	return &OsuSongsHandler{repos: repos}
}

func (h *OsuSongsHandler) Handle(ctx fiber.Ctx) error {
	q, err := dslquery.Parse(ctx.Query("q"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	limit := fiber.Query[int](ctx, "limit", 50)
	offset := fiber.Query[int](ctx, "offset", 0)

	beatmaps, total, err := h.repos.Beatmaps.Search(ctx.Context(), q, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"total": total, "items": beatmaps})
}
