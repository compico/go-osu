// Package http provides the Fiber-based HTTP server.
package http

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/compico/go-osu/internal/config"
	"github.com/compico/go-osu/internal/http/handler/osuapi"
	"github.com/compico/go-osu/internal/http/view"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// Developer-only constants. End users configure only the port via config.yml.
const (
	bindAddr        = "127.0.0.1"
	idleTimeout     = 60 * time.Second
	shutdownTimeout = 5 * time.Second
)

// Server wraps a fiber.App with lifecycle helpers.
type Server struct {
	app       *fiber.App
	addr      string
	cfg       *config.Config
	logger    *slog.Logger
	apiRouter fiber.Router
}

// New creates and configures the Fiber server.
func New(cfg *config.Config, logger *slog.Logger) *Server {
	app := fiber.New(fiber.Config{
		IdleTimeout: idleTimeout,
		//CBOREncoder: cbor.Marshal,
		//CBORDecoder: cbor.Unmarshal,
	})

	s := &Server{
		app:    app,
		addr:   fmt.Sprintf("%s:%d", bindAddr, cfg.HTTP.Port),
		cfg:    cfg,
		logger: logger,
	}

	return s
}

// Start begins listening. Blocks until the server is shut down.
func (s *Server) Start() error {
	s.logger.Info("http server listening", "addr", s.addr)
	return s.app.Listen(s.addr)
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	s.logger.Info("http server shutting down")
	return s.app.ShutdownWithTimeout(shutdownTimeout)
}

// App returns the underlying fiber.App for registering additional routes.
func (s *Server) App() *fiber.App { return s.app }

func (s *Server) APIRouter() fiber.Router { return s.apiRouter }

// appConfigResponse is the payload returned by GET /api/app/config.
// The frontend uses this to connect to the correct Centrifuge port and
// to know which environment it's running in.
type appConfigResponse struct {
	Env          string `json:"env"`
	RealtimePort int    `json:"realtime_port"`
}

func (s *Server) RegisterRoutes(
	v *view.View,
	osuSongsHandler *osuapi.OsuSongsHandler,
	osuSongHandler *osuapi.OsuSongHandler,
	osuTrackStreamHandler *osuapi.OsuTrackStreamHandler,
	osuGetBackgroundHandler *osuapi.OsuGetBackgroundHandler,
) {
	s.apiRouter = s.app.Group("/api")

	s.apiRouter.Get("/app/config", func(c fiber.Ctx) error {
		return c.JSON(appConfigResponse{
			Env:          s.cfg.App.Env,
			RealtimePort: s.cfg.Realtime.Port,
		})
	})

	s.apiRouter.Get("/osu/songs", compress.New(compress.Config{Level: compress.LevelBestSpeed}), osuSongsHandler.Handle)
	s.apiRouter.Get("/osu/songs/:difficulty_id<int>", osuSongHandler.Handle)
	s.apiRouter.Get("/osu/songs/:difficulty_id<int>/track", osuTrackStreamHandler.Handle)

	s.apiRouter.Get("/osu/bg/:beatmap_id<int>.jpg", osuGetBackgroundHandler.Handle)

	if !v.IsDev() {
		s.app.Use("/assets", static.New(frontendAssetsDir()))
	}

	s.app.Get("/*", v.IndexHandler)
}

// frontendAssetsDir returns the path to the built Vite assets.
// Defined here so it stays in sync with the view package constant.
func frontendAssetsDir() string { return "./frontend/dist/assets" }
