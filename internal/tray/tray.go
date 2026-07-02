// Package tray provides the system-tray icon and menu.
//
// fyne.io/systray requires Run to be called from the OS main thread.
// The typical usage pattern inside a cli.Command Action:
//
//	go server.Start()
//	tray.Run(onQuit)  // blocks until user clicks Quit
package tray

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"

	"fyne.io/systray"
	"github.com/compico/go-osu/assets"
)

// Developer-only constants.
const (
	appName  = "Go-osu!"
	tooltip  = "Go-osu is running"
	bindAddr = "127.0.0.1"
)

// Tray manages the system-tray icon and reacts to menu events.
type Tray struct {
	httpPort int
	logger   *slog.Logger
	onQuit   func()
}

// New creates a Tray.
// httpPort is used to build the "Open in browser" URL.
func New(httpPort int, logger *slog.Logger) *Tray {
	return &Tray{
		httpPort: httpPort,
		logger:   logger,
	}
}

// Run blocks the calling goroutine (must be the OS main thread) until the
// user chooses Quit from the tray menu. onQuit is invoked before returning.
func (t *Tray) Run(onQuit func()) {
	t.onQuit = onQuit
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetIcon(assets.IconIco)
	systray.SetTitle(appName)
	systray.SetTooltip(tooltip)

	mOpen := systray.AddMenuItem("Open", "Open in browser")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", fmt.Sprintf("Quit %s", appName))

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				t.openBrowser()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (t *Tray) onExit() {
	if t.onQuit != nil {
		t.onQuit()
	}
}

// openBrowser launches the default system browser pointing at the app URL.
func (t *Tray) openBrowser() {
	url := fmt.Sprintf("http://%s:%d", bindAddr, t.httpPort)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // linux
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		t.logger.Error("tray: open browser", "err", err)
	}
}
