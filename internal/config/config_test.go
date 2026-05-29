package config_test

import (
	"testing"

	"review-view/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ADDR", "")
	t.Setenv("DATABASE_DSN", "")

	cfg := config.Load()
	if cfg.Addr != ":18083" {
		t.Fatalf("expected default addr :18083, got %q", cfg.Addr)
	}
	if cfg.DatabaseDSN != "file:review-view.db?_foreign_keys=on" {
		t.Fatalf("expected default dsn, got %q", cfg.DatabaseDSN)
	}
}
