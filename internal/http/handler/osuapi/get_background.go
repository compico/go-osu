package osuapi

import (
	"strconv"

	"github.com/compico/go-osu/internal/service"
	"github.com/gofiber/fiber/v3"
)

type OsuGetBackgroundHandler struct {
	osuService *service.Osu
}

func NewOsuGetBackgroundHandler(osuService *service.Osu) *OsuGetBackgroundHandler {
	return &OsuGetBackgroundHandler{
		osuService: osuService,
	}
}

func (o *OsuGetBackgroundHandler) Handle(ctx fiber.Ctx) error {
	bmParam := ctx.Params("beatmap_id", "")
	bmId, err := strconv.Atoi(bmParam)
	if err != nil {
		return err
	}

	bgPath := o.osuService.GetBackgroundFilePath(int32(bmId))
	image, err := o.osuService.GetBackgroundFile(bgPath)
	if err != nil {
		return err
	}

	if image == nil {
		return fiber.NewError(fiber.StatusNotFound, "background file not found")
	}

	return ctx.Type("jpeg").Send(image)
}
