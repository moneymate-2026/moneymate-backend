package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	sharedconfig "github.com/moneymate-2026/moneymate-backend/shared/config"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	HTTPAddr     string        `mapstructure:"http_addr"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type OTPConfig struct {
	TTL               time.Duration `mapstructure:"ttl"`
	Length            int           `mapstructure:"length"`
	ResendCooldown    time.Duration `mapstructure:"resend_cooldown"`
	MaxVerifyAttempts int           `mapstructure:"max_verify_attempts"`
	EmailVerifiedTTL  time.Duration `mapstructure:"email_verified_ttl"`
}

type Argon2Config struct {
	Memory     uint32 `mapstructure:"memory"`
	Iterations uint32 `mapstructure:"iterations"`
	Threads    uint8  `mapstructure:"threads"`
	KeyLength  uint32 `mapstructure:"key_length"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type Config struct {
	Env      string
	Server   ServerConfig             `mapstructure:"server"`
	Database sharedconfig.DatabaseConfig
	Redis    sharedconfig.RedisConfig
	OTP      OTPConfig                `mapstructure:"otp"`
	JWT      sharedconfig.JWTConfig
	Argon2   Argon2Config             `mapstructure:"argon2"`
	Log      LogConfig                `mapstructure:"log"`
	SMTP     sharedconfig.SMTPConfig
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
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Database = sharedconfig.LoadDatabaseConfig(v, "auth")
	cfg.SMTP = sharedconfig.LoadSMTPConfig(v)
	cfg.Redis = sharedconfig.LoadRedisConfig(v)
	cfg.JWT = sharedconfig.LoadJWTConfig(v)

	cfg.Env = sharedconfig.Get("ENVIRONMENT", "dev")

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {

	if cfg.OTP.Length < 4 || cfg.OTP.Length > 10 {
		return fmt.Errorf("otp.length must be between 4 and 10, got %d", cfg.OTP.Length)
	}
	if cfg.OTP.MaxVerifyAttempts < 1 {
		return fmt.Errorf("otp.max_verify_attempts must be at least 1, got %d", cfg.OTP.MaxVerifyAttempts)
	}
	if cfg.OTP.TTL <= 0 {
		return fmt.Errorf("otp.ttl must be positive")
	}

	return nil
}