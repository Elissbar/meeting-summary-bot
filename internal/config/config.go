package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	BotToken          string `env:"BOT_TOKEN"`
	SaluteAuthURL     string `env:"SALUTE_AUTH_URL"`
	SaluteAuthToken   string `env:"SALUTE_AUTH_TOKEN"`
	SaluteScope       string `env:"SALUTE_SCOPE"`
	GigaChatAuthURL   string `env:"GIGA_AUTH_URL"`
	GigaChatAuthToken string `env:"GIGA_AUTH_TOKEN"`
}

func NewConfig() (Config, error) {
	if err := godotenv.Load(); err != nil {
		return Config{}, fmt.Errorf("error load .env file")
	}

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("error parse env variables")
	}
	return cfg, nil
}
