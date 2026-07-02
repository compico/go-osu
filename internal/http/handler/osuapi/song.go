package osuapi

import (
	"strconv"

	"github.com/compico/go-osu/internal/service"
	"github.com/gofiber/fiber/v3"
)

type OsuSongHandler struct {
	osuService *service.Osu
}

func NewOsuSongHandler(osuService *service.Osu) *OsuSongHandler {
	return &OsuSongHandler{osuService: osuService}
}

func (h *OsuSongHandler) Handle(ctx fiber.Ctx) error {
	difficultyID := ctx.Params("difficulty_id", "")
	if difficultyID == "" {
		return ctx.Status(fiber.StatusBadRequest).SendString("difficulty_id is required")
	}

	diffID, err := strconv.Atoi(difficultyID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).SendString("Invalid difficulty_id")
	}

	song, err := h.osuService.GetBeatmapByDifficultyId(int32(diffID))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Song Not Found",
		})
	}

	return ctx.JSONP(song)
}
