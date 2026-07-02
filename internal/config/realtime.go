package config

// RealtimeConfig holds user-facing Centrifuge settings.
// The bind address is hardcoded to 127.0.0.1 in the realtime package.
type RealtimeConfig struct {
	Port int `yaml:"port"`
}
