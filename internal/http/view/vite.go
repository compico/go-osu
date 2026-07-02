// Package view provides Vite backend integration for fiber.
//
// It generates the correct <script>/<link> tags for the HTML <head> section
// without serving index.html itself — the Go server owns the HTML shell.
//
// Dev mode:  tags point to the Vite dev server (HMR + hot reload).
// Prod mode: tags use content-hashed filenames read from dist/.vite/manifest.json.
//
// Vite backend integration reference:
// https://vite.dev/guide/backend-integration.html
package view

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/olivere/vite"
)

// Developer-only constants. End users never need to change these.
const (
	devServerURL = "http://localhost:5173" // Vite dev server base URL
	frontendDir  = "./frontend"            // project-relative path to Vue sources
	entryPoint   = "src/main.ts"           // must match rollupOptions.input in vite.config.ts
)

// indexTmpl is the HTML shell rendered by the Go backend.
// index.html in the frontend project is intentionally ignored.
var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>{{.Title}}</title>
  {{.ViteTags}}
</head>
<body>
  <div id="app"></div>
</body>
</html>`))

// View wraps olivere/vite.Fragment and exposes fiber-friendly helpers.
type View struct {
	isDev    bool
	fragment *vite.Fragment
}

// New builds a View. isDev comes from AppConfig.IsDev().
//
//   - Dev:  generates @vite/client + entry script tags for the Vite dev server.
//   - Prod: reads dist/.vite/manifest.json and generates hashed asset tags.
func New(isDev bool) (*View, error) {
	cfg := vite.Config{
		IsDev:     isDev,
		ViteURL:   devServerURL,
		ViteEntry: entryPoint,
	}

	if !isDev {
		cfg.FS = os.DirFS(frontendDir + "/dist")
	}

	fragment, err := vite.HTMLFragment(cfg)
	if err != nil {
		return nil, fmt.Errorf("vite view: %w", err)
	}

	return &View{isDev: isDev, fragment: fragment}, nil
}

// IsDev reports whether the view is in development mode.
func (v *View) IsDev() bool { return v.isDev }

// Tags returns the HTML tags to inject inside <head>.
// Pre-escaped as template.HTML — safe for Go templates.
func (v *View) Tags() template.HTML { return v.fragment.Tags }

// IndexHandler serves the SPA shell page for every unmatched route.
//
//	app.Get("/*", viteView.IndexHandler)
func (v *View) IndexHandler(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)

	var buf bytes.Buffer
	if err := indexTmpl.Execute(&buf, map[string]any{
		"Title":    "Go-osu",
		"ViteTags": v.fragment.Tags,
	}); err != nil {
		return fmt.Errorf("vite view: render: %w", err)
	}

	return c.SendString(buf.String())
}
