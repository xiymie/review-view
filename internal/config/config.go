package config

import "os"

type Config struct {
	Addr               string
	DatabaseDSN        string
	ReviewWorkflowMode string
}

func Load() Config {
	cfg := Config{
		Addr:               ":18083",
		DatabaseDSN:        "file:review-view.db?_foreign_keys=on",
		ReviewWorkflowMode: "legacy",
	}
	if v := os.Getenv("APP_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		cfg.DatabaseDSN = v
	}
	if v := os.Getenv("REVIEW_WORKFLOW_MODE"); v != "" {
		cfg.ReviewWorkflowMode = v
	}
	return cfg
}
