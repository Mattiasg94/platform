package config

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

type Config struct{}

// Load reads the monorepo-root .env into the process environment. The path is
// resolved relative to this source file (not the working directory), so it
// works whether the orchestrator is run from the repo root or from
// orchestrator/. godotenv strips surrounding quotes, so a quoted key here is
// fine — but note `docker --env-file` does not, so keep .env unquoted.
func Load() *Config {
	_, thisFile, _, _ := runtime.Caller(0)
	// this file: platform/orchestrator/internal/config/config.go
	monorepoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	if err := godotenv.Load(filepath.Join(monorepoRoot, ".env")); err != nil {
		log.Println("No .env file found; using host environment")
	}
	return &Config{}
}
