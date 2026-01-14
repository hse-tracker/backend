package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServerHost          string `yaml:"server_host" env:"SERVER_HOST" env-default:"localhost"`
	ServerPort          string `yaml:"server_port" env:"SERVER_PORT" env-default:"8080"`
	DBPath              string `yaml:"db_path" env:"DB_PATH" env-required:"true"`
	TelegramToken       string `yaml:"telegram_token" env:"TELEGRAM_TOKEN" env-required:"true"`
	ParseInterval       string `yaml:"parse_interval" env:"PARSE_INTERVAL" env-default:"10m"`
	ParseIntervalParsed time.Duration
}

func Load(configPath string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("could not read config file %s: %w", configPath, err)
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("could not read environment variables: %w", err)
	}

	interval, err := time.ParseDuration(cfg.ParseInterval)
	if err != nil {
		return nil, fmt.Errorf("wrong 'check_interval' format: %w", err)
	}
	cfg.ParseIntervalParsed = interval

	return &cfg, nil
}
