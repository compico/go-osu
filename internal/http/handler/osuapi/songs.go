package osuapi

import (
	"github.com/compico/go-osu/internal/service"
	"github.com/gofiber/fiber/v3"
)

type OsuSongsHandler struct {
	osuService *service.Osu
}

func NewOsuSongsHandler(osuService *service.Osu) *OsuSongsHandler {
	return &OsuSongsHandler{osuService: osuService}
}

func (h *OsuSongsHandler) Handle(ctx fiber.Ctx) error {
	return ctx.Type("json").Send(h.osuService.GetSongsJson())
}
