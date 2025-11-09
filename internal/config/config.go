package config

import (
	"errors"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AsanaPAT   string
	HTTPTimeout time.Duration
	OutDir      string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	pat := os.Getenv("ASANA_PAT")
	if pat == "" {
		return nil, errors.New("ASANA_PAT is not set (.env or env var)")
	}

	out := os.Getenv("OUT_DIR")
	if out == "" {
		out = "out"
	}

	return &Config{
		AsanaPAT:    pat,
		HTTPTimeout: 30 * time.Second,
		OutDir:      out,
	}, nil
}
