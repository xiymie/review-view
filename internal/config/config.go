package config

import "os"

type Config struct {
	Addr        string
	DatabaseDSN string
}

func Load() Config {
	cfg := Config{
		Addr:        ":18083",
		DatabaseDSN: "file:review-view.db?_foreign_keys=on",
	}
	if v := os.Getenv("APP_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		cfg.DatabaseDSN = v
	}
	return cfg
}
