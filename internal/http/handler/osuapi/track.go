package osuapi

import (
	"path/filepath"

	"github.com/compico/go-osu/internal/service"
	"github.com/gofiber/fiber/v3"
)

type OsuTrackStreamHandler struct {
	osuService *service.Osu
}

func NewOsuTrackStreamHandler(service *service.Osu) *OsuTrackStreamHandler {
	return &OsuTrackStreamHandler{osuService: service}
}

func (h *OsuTrackStreamHandler) Handle(ctx fiber.Ctx) error {
	diffID := fiber.Params[int32](ctx, "difficulty_id", -100)
	if diffID == -100 {
		return fiber.NewError(400, "Invalid difficulty_id")
	}

	filePath, err := h.osuService.GetTrackPathByDifficultyId(ctx, diffID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).SendString("Track not found")
	}

	return ctx.Type(h.getContentType(filePath)).SendFile(filePath)
}

func (h *OsuTrackStreamHandler) getContentType(file string) string {
	ext := filepath.Ext(file)
	if ext == "" {
		return "mp3"
	}

	return ext
}
