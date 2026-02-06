package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Token    string
	Username string
	Org      string

}

func Load() (*Config, error) {
	_ = godotenv.Load()

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is required - set it in .env or as an environment variable")
	}

	username := os.Getenv("GITHUB_USERNAME")
	if username == "" {
		return nil, fmt.Errorf("GITHUB_USERNAME is required - set it in .env or as an environment variable")
	}

	org := os.Getenv("GITHUB_ORG")
	if org == "" {
		return nil, fmt.Errorf("GITHUB_ORG is required - set it in .env or as an environment variable")
	}

	return &Config{
		Token:    token,
		Username: username,
		Org:      org,
	}, nil
}
