package config

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

type Config struct{}

func Load() *Config {
	_, thisFile, _, _ := runtime.Caller(0)
	monorepoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	// Keep .env unquoted: godotenv strips quotes but `docker --env-file` does not.
	if err := godotenv.Load(filepath.Join(monorepoRoot, ".env")); err != nil {
		log.Println("No .env file found; using host environment")
	}
	return &Config{}
}
