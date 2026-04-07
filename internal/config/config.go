package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServerHost      string  `yaml:"server_host" env:"SERVER_HOST" env-default:"localhost"`
	ServerPort      string  `yaml:"server_port" env:"SERVER_PORT" env-default:"8080"`
	ClientPort      string  `yaml:"client_port" env:"CLIENT_PORT" env-default:"5173"`
	DBPath          string  `yaml:"db_path" env:"DB_PATH" env-required:"true"`
	TelegramToken   string  `yaml:"telegram_token" env:"TELEGRAM_TOKEN" env-required:"true"`
	MaxAge          int     `yaml:"max_age" env:"MAX_AGE" env-default:"300"`
	AdminIDs        []int64 `yaml:"admin_ids" env:"ADMIN_IDS"`
	JWTSecret       string  `yaml:"jwt_secret" env:"JWT_SECRET" env-required:"true"`
	DeepSeekKey     string  `yaml:"deepseek_key" env:"DEEPSEEK_KEY" env-required:"true"`
	GoogleCredsPath string  `yaml:"google_creds_path" env:"GOOGLE_CREDS_PATH" env-default:"credentials.json"`
}

func Load(configPath string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		fmt.Printf("could not read yaml config file %s: %w\n", configPath, err)
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("could not read environment variables: %w", err)
		}
		return &cfg, nil
	}

	return &cfg, nil
}
