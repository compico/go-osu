package osuapi

import (
	"github.com/compico/go-osu/internal/dslquery"
	"github.com/compico/go-osu/internal/repository"
	"github.com/gofiber/fiber/v3"
)

type OsuSongsSearchHandler struct {
	repos *repository.Repos
}

func NewOsuSongsSearchHandler(repos *repository.Repos) *OsuSongsSearchHandler {
	return &OsuSongsSearchHandler{repos: repos}
}

type searchPage struct {
	Items      []GroupDTO `json:"items"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

func (h *OsuSongsSearchHandler) Handle(ctx fiber.Ctx) error {
	q, err := dslquery.Parse(ctx.Query("q"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	sortName := ctx.Query("sort", "artist_name")
	cursor := ctx.Query("cursor", "")
	limit := fiber.Query[int](ctx, "limit", 50)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	groups, nextCursor, err := h.repos.Beatmaps.SearchGroups(ctx.Context(), q, sortName, cursor, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Skills are always fetched now — NoMod (mods=0) is a fully computed
	// combination just like HDDT etc., so there's no reason to hide the
	// skills block just because the user hasn't typed mode= yet.
	mods := int32(0)
	if q.HasMods {
		mods = int32(q.Mods)
	}

	beatmapIDs := make([]int32, 0)
	for _, g := range groups {
		for _, d := range g.Diffs {
			beatmapIDs = append(beatmapIDs, d.BeatmapID)
		}
	}
	skillsByID, err := h.repos.Skills.ForModsAndBeatmapIDs(ctx.Context(), mods, beatmapIDs)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	items := make([]GroupDTO, 0, len(groups))
	for _, g := range groups {
		items = append(items, toGroupDTO(g, skillsByID))
	}

	return ctx.JSON(searchPage{Items: items, NextCursor: nextCursor})
}
