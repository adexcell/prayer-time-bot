package config

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	TelegramToken string `envconfig:"TELEGRAM_BOT_TOKEN" required:"true"`
	City          string `envconfig:"CITY" required:"true"`
	Country       string `envconfig:"COUNTRY" required:"true"`
	Method        string `envconfig:"METHOD" required:"true"`
}

func Load() (Config, error) {
	var config Config
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: .env not loaded: %v", err)
	}

	if err := envconfig.Process("", &config); err != nil {
		return config, fmt.Errorf("envconfig.Process: %w", err)
	}
	return config, nil
}
