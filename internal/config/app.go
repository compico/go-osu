package config

// AppConfig holds application-level settings exposed to the end user.
type AppConfig struct {
	// Env controls which mode the app runs in.
	// "development" enables the Vite dev server integration and verbose output.
	// "production" (default) serves built assets from frontend/dist.
	Env string `yaml:"env"`
}

// IsDev reports whether the application is running in development mode.
func (a AppConfig) IsDev() bool { return a.Env == "development" }
