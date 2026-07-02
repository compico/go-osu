package config

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"go.yaml.in/yaml/v2"
)

// Config holds only the settings that are meaningful for end users.
// Developer/infrastructure defaults (bind addresses, timeouts, Vite paths, tray
// labels, etc.) are hardcoded as constants inside their respective packages.
type Config struct {
	App      AppConfig      `yaml:"app"`
	HTTP     HTTPConfig     `yaml:"http"`
	Database DatabaseConfig `yaml:"database"`
	Log      LogConfig      `yaml:"log"`
	Realtime RealtimeConfig `yaml:"realtime"`
	Osu      Osu            `yaml:"osu"`
}

// Flag is the CLI flag name for the config file path.
const Flag = "config"

// Flags returns CLI flags to add to a cli.Command.
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    Flag,
			Aliases: []string{"c"},
			Usage:   "path to YAML config file",
			Value:   "config.yml",
			Sources: cli.EnvVars("APP_CONFIG"),
		},
	}
}

func defaultConfig() Config {
	return Config{
		App: AppConfig{
			Env: "production",
		},
		HTTP: HTTPConfig{
			Port: 3000,
		},
		Database: DatabaseConfig{
			Path: "goosu.db",
		},
		Log: LogConfig{
			Level: "info",
		},
		Realtime: RealtimeConfig{
			Port: 3001,
		},
		Osu: Osu{
			Path: "",
		},
	}
}

// Load reads the YAML config file specified via --config flag.
// If the file does not exist, defaults are used silently.
func Load(c *cli.Command) (*Config, error) {
	cfg := defaultConfig()

	path := c.String(Flag)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("config: read %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}

	return &cfg, nil
}
