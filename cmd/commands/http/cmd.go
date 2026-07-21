package http

import (
	"context"
	"errors"
	"log"
	"net"
	"time"

	"github.com/compico/go-osu/cmd/commands"
	appcfg "github.com/compico/go-osu/internal/config"
	"github.com/compico/go-osu/internal/database"
	httpserver "github.com/compico/go-osu/internal/http"
	"github.com/compico/go-osu/internal/http/handler/osuapi"
	"github.com/compico/go-osu/internal/http/view"
	"github.com/compico/go-osu/internal/logger"
	"github.com/compico/go-osu/internal/realtime"
	"github.com/compico/go-osu/internal/repository"
	"github.com/compico/go-osu/internal/service"
	"github.com/compico/go-osu/internal/tray"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

func init() {
	commands.Append(&cli.Command{
		Name:  "http",
		Usage: "start the HTTP server and system tray",
		Flags: appcfg.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			return run(ctx, c)
		},
	})
}

func run(ctx context.Context, c *cli.Command) error {
	cfg, err := appcfg.Load(c)
	if err != nil {
		return err
	}

	slogger, browserHandler, err := logger.New(&cfg.Log)
	if err != nil {
		return err
	}

	db, err := database.New(&cfg.Database, slogger)
	if err != nil {
		return err
	}
	defer func(db *database.DB) {
		err := db.Close()
		if err != nil {
			slogger.Error(err.Error())
		}
	}(db)

	if err := db.Migrate(ctx); err != nil {
		return err
	}

	repos := repository.New(db)

	osuService, err := service.NewOsuService(cfg.Osu.Path, slogger, repos)
	if err != nil {
		return err
	}

	syncService := service.NewSyncer(osuService, repos, slogger)

	if err = syncService.Run(ctx); err != nil {
		return err
	}

	viteView, err := view.New(cfg.App.IsDev())
	if err != nil {
		return err
	}

	rt, err := realtime.New(&cfg.Realtime, slogger)
	if err != nil {
		return err
	}
	if err := rt.Start(); err != nil {
		return err
	}

	osuSongHandler := osuapi.NewOsuSongHandler(osuService)
	osuTrackStreamHandler := osuapi.NewOsuTrackStreamHandler(osuService)
	osuGetBackgroundHandler := osuapi.NewOsuGetBackgroundHandler(osuService)
	osuQueueAdjacentHandler := osuapi.NewOsuQueueAdjacentHandler(repos)
	osuSongsSearchHandler := osuapi.NewOsuSongsSearchHandler(repos)

	http := httpserver.New(cfg, slogger)
	http.RegisterRoutes(viteView, osuSongHandler, osuTrackStreamHandler, osuGetBackgroundHandler, osuQueueAdjacentHandler, osuSongsSearchHandler)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := http.Start(); err != nil && !errors.Is(err, net.ErrClosed) {
			return err
		}
		return nil
	})

	go rt.DrainLogs(gCtx, browserHandler.Chan())

	trayIcon := tray.New(cfg.HTTP.Port, slogger)

	onQuit := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		slogger.Info("shutting down...")

		if err := rt.Stop(shutdownCtx); err != nil {
			log.Printf("realtime shutdown: %v", err)
		}
		if err := http.Stop(); err != nil {
			log.Printf("http shutdown: %v", err)
		}
	}

	trayIcon.Run(onQuit)

	return g.Wait()
}
