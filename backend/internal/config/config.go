package config

import "os"

type Config struct {
	Port          string
	DBPath        string
	WidgetPath    string
	DashboardPath string
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8080"),
		DBPath:        getEnv("DB_PATH", "./data/tracker.db"),
		WidgetPath:    getEnv("WIDGET_PATH", "../widget/widget.js"),
		DashboardPath: getEnv("DASHBOARD_PATH", "../dashboard/index.html"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
