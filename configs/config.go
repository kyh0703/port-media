package configs

import (
	"fmt"
	"os"
	"strconv"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

func NewConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("parse env config: %w", err)
	}
	if err := applyPort(&cfg); err != nil {
		return nil, err
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyPort(cfg *Config) error {
	port, ok := os.LookupEnv("PORT")
	if !ok || port == "" {
		cfg.Server.Port = 8080
		return nil
	}
	parsedPort, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("parse PORT config: %w", err)
	}
	cfg.Server.Port = parsedPort
	return nil
}

func validateConfig(cfg Config) error {
	if err := validator.New().Struct(cfg); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
}
