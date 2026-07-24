package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	sharedconfig "github.com/moneymate-2026/moneymate-backend/shared/config"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	HTTPAddr string `mapstructure:"http_addr"`
}

type Config struct {
	Env      string
	Server   ServerConfig `mapstructure:"server"`
	Database sharedconfig.DatabaseConfig
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()
	yamlPath := os.Getenv("CONFIG_PATH")
	if yamlPath == "" {
		yamlPath = "./config/config.yaml"
	}

	v := viper.New()
	v.SetConfigFile(yamlPath)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		// Ignore if file doesn't exist, rely on env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("Warning: failed to read config file: %v\n", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Database = sharedconfig.LoadDatabaseConfig(v, "merchant")
	cfg.Env = sharedconfig.Get("ENVIRONMENT", "dev")

	return &cfg, nil
}
