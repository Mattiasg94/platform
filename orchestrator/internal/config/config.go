package config

import (
	"log"

	"github.com/joho/godotenv"
)

type Config struct{}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using host environment")
	}

	return &Config{}
}
