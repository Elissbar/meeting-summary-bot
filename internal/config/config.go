package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	BotToken          string `env:"BOT_TOKEN"`
	AuthURL           string `env:"AUTH_URL"`
	SaluteAuthToken   string `env:"SALUTE_AUTH_TOKEN"`
	SaluteScope       string `env:"SALUTE_SCOPE"`
	GigaChatAuthToken string `env:"GIGA_AUTH_TOKEN"`
	GigaChatScope     string `env:"GIGA_SCOPE"`
	DBConnectionURI   string `env:"DB_URI"`
	NumWorkers        int    `env:"NUM_WORKERS" envDefault:"5"`

	// Timeouts
	WaitPlaceInChan time.Duration `env:"WAIT_PLACE_IN_TASK_CHAN" envDefault:"500ms"`
	StopProcess     time.Duration `env:"WAIT_STOP_PROCESS_TASKS" envDefault:"2s"`
}

func NewConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error load .env file")
	}

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("error parse env variables")
	}
	return &cfg, nil
}
