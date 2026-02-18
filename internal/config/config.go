package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	APIKey string
	DBPath string
}

func Load() (*Config, error) {
	godotenv.Load()

	viper.SetDefault("database_path", "usage.db")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.BindEnv("api_key", "SYNTHETIC_API_KEY")
	viper.BindEnv("database_path", "DATABASE_PATH")

	cfg := &Config{
		APIKey: viper.GetString("api_key"),
		DBPath: viper.GetString("database_path"),
	}

	return cfg, nil
}
