package osuapi

import (
	"path/filepath"
	"strconv"

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
	difficultyID := ctx.Params("difficulty_id")
	diffID, err := strconv.Atoi(difficultyID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).SendString("Invalid difficulty_id")
	}

	filePath, err := h.osuService.GetTrackPathByDifficultyId(int32(diffID))
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
