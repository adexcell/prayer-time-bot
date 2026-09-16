package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	TelegramToken string `envconfig:"TELEGRAM_BOT_TOKEN" required:"true"`
	City          string `envconfig:"CITY" required:"true"`
	Country       string `envconfig:"COUNTRY" required:"true"`
	Method        string `envconfig:"METHOD" required:"true"`
	AdminIDs      string `envconfig:"ADMIN_IDS" default:""`
}

func (c *Config) GetAdminIDList() []int64 {
	var ids []int64
	if c.AdminIDs == "" {
		return ids
	}
	for _, part := range strings.Split(c.AdminIDs, ",") {
		clean := strings.TrimSpace(part)
		if clean == "" {
			continue
		}
		var id int64
		if _, err := fmt.Sscanf(clean, "%d", &id); err == nil && id != 0 {
			ids = append(ids, id)
		}
	}
	return ids
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
