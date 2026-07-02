package config

// HTTPConfig holds user-facing HTTP server settings.
// Low-level internals (bind address, timeouts) are hardcoded in the server package.
type HTTPConfig struct {
	Port int `yaml:"port"`
}
