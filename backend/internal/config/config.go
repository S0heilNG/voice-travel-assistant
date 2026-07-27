package config

import "os"

type Config struct {
	Port string
	// AnalyticsDBPath is the SQLite file for interaction logging. It must point
	// at the deploy's shared/ directory (outside any release), or logging data
	// is discarded on the next deploy. Empty disables logging entirely.
	AnalyticsDBPath string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{
		Port:            port,
		AnalyticsDBPath: os.Getenv("VTA_ANALYTICS_DB"),
	}
}
