package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ProjectRepoURL string
	ProjectRef     string
	ProjectOwner   string
	ProjectRepo    string
	GitHubToken    string
}

func Load() (*Config, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	monorepoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	// Keep .env unquoted: godotenv strips quotes but `docker --env-file` does not.
	if err := godotenv.Load(filepath.Join(monorepoRoot, ".env")); err != nil {
		log.Println("No .env file found; using host environment")
	}

	cfg := &Config{
		ProjectRepoURL: os.Getenv("PROJECT_REPO_URL"),
		ProjectRef:     envOr("PROJECT_REF", "main"),
		GitHubToken:    os.Getenv("GITHUB_TOKEN"),
	}
	if cfg.ProjectRepoURL == "" {
		return nil, fmt.Errorf("PROJECT_REPO_URL not set (checked .env and host env)")
	}
	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN not set (checked .env and host env)")
	}

	owner, repo, err := parseOwnerRepo(cfg.ProjectRepoURL)
	if err != nil {
		return nil, err
	}
	cfg.ProjectOwner = owner
	cfg.ProjectRepo = repo

	return cfg, nil
}

func parseOwnerRepo(repoURL string) (owner, repo string, err error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("parse repo url: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("repo url %q lacks an owner/name path", repoURL)
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
