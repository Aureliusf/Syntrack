package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	APIKey     string
	DBPath     string
	AuthTokens []string
}

func Load() (*Config, error) {
	godotenv.Load()

	viper.SetDefault("database_path", "usage.db")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.BindEnv("api_key", "SYNTHETIC_API_KEY")
	viper.BindEnv("database_path", "DATABASE_PATH")
	viper.BindEnv("auth_tokens", "SYNTRACK_AUTH_TOKENS")

	cfg := &Config{
		APIKey:     viper.GetString("api_key"),
		DBPath:     viper.GetString("database_path"),
		AuthTokens: loadAuthTokens(),
	}

	return cfg, nil
}

func loadAuthTokens() []string {
	var tokens []string

	// Load from environment variable (comma-separated)
	if envTokens := viper.GetString("auth_tokens"); envTokens != "" {
		for _, token := range strings.Split(envTokens, ",") {
			token = strings.TrimSpace(token)
			if token != "" {
				tokens = append(tokens, token)
			}
		}
	}

	// Load from file (~/.syntrack/tokens)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		tokenFile := filepath.Join(homeDir, ".syntrack", "tokens")
		if fileTokens, err := loadTokensFromFile(tokenFile); err == nil {
			tokens = append(tokens, fileTokens...)
		}
	}

	return tokens
}

func loadTokensFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tokens []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		token := strings.TrimSpace(scanner.Text())
		if token != "" && !strings.HasPrefix(token, "#") {
			tokens = append(tokens, token)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}
