// package config

// import (
// 	"fmt"
// 	"os"
// 	"time"

// 	sharedconfig "github.com/moneymate-2026/moneymate-backend/shared/config"
// 	"github.com/spf13/viper"
// )

// type ServerConfig struct {
// 	HTTPAddr     string        `mapstructure:"http_addr"`
// 	GRPCAddr     string        `mapstructure:"grpc_addr"`
// 	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
// 	WriteTimeout time.Duration `mapstructure:"write_timeout"`
// }

// type DatabaseConfig struct {
// 	User     string
// 	Password string
// 	Host     string
// 	Port     string
// 	Name     string

// 	MaxOpenConns    int           `mapstructure:"max_open_conns"`
// 	MinOpenConns    int           `mapstructure:"min_open_conns"`
// 	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
// 	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
// 	MaxIdleTime     time.Duration `mapstructure:"max_idle_time"`

// 	DSN string
// }
// type OTPConfig struct {
// 	TTL               time.Duration `mapstructure:"ttl"`
// 	Length            int           `mapstructure:"length"`
// 	ResendCooldown    time.Duration `mapstructure:"resend_cooldown"`
// 	MaxVerifyAttempts int           `mapstructure:"max_verify_attempts"`
// 	EmailVerifiedTTL  time.Duration `mapstructure:"email_verified_ttl"`
// }

// type SMTPConfig struct {
// 	Host        string `mapstructure:"host"`
// 	Port        int    `mapstructure:"port"`
// 	FromAddress string `mapstructure:"from_address"`
// 	FromName    string `mapstructure:"from_name"`
// 	Username    string // from env
// 	Password    string // from env
// }

// type RedisConfig struct {
// 	Addr          string `mapstructure:"addr"`
// 	Password      string
// 	TokenTTL      time.Duration `mapstructure:"token_ttl"`
// 	OTPTTL        time.Duration `mapstructure:"otp_ttl"`
// 	PINLockoutTTL time.Duration `mapstructure:"pin_lockout_ttl"`
// }

// type JWTConfig struct {
// 	AccessExpiryMinutes int    `mapstructure:"access_expiry_minutes"`
// 	RefreshExpiryHours  int    `mapstructure:"refresh_expiry_hours"`
// 	AccessSecret        string // from env
// 	RefreshSecret       string // from env
// }

// type Argon2Config struct {
// 	Memory     uint32 `mapstructure:"memory"`
// 	Iterations uint32 `mapstructure:"iterations"`
// 	Threads    uint8  `mapstructure:"threads"`
// 	KeyLength  uint32 `mapstructure:"key_length"`
// }

// type LogConfig struct {
// 	Level string `mapstructure:"level"`
// }

// type Config struct {
// 	Env      string
// 	Server   ServerConfig   `mapstructure:"server"`
// 	Database DatabaseConfig `mapstructure:"database"`
// 	Redis    RedisConfig    `mapstructure:"redis"`
// 	OTP      OTPConfig      `mapstructure:"otp"`
// 	JWT      JWTConfig      `mapstructure:"jwt"`
// 	Argon2   Argon2Config   `mapstructure:"argon2"`
// 	Log      LogConfig      `mapstructure:"log"`
// 	SMTP     SMTPConfig     `mapstructure:"smtp"`
// }

// func LoadConfig() (*Config, error) {
// 	yamlPath := os.Getenv("CONFIG_PATH")
// 	if yamlPath == "" {
// 		yamlPath = "./config/config.yaml" // sensible default for local dev
// 	}

// 	v := viper.New()
// 	v.SetConfigFile(yamlPath)
// 	v.AutomaticEnv()

// 	// bind secrets from env vars
// 	// these override anything in the yaml file
// 	v.BindEnv("database.user", "POSTGRES_USER")
// 	v.BindEnv("database.password", "POSTGRES_PASSWORD")
// 	v.BindEnv("database.host", "POSTGRES_HOST")
// 	v.BindEnv("database.port", "POSTGRES_PORT")
// 	v.BindEnv("database.name", "POSTGRES_DB")
// 	v.BindEnv("redis.addr", "REDIS_ADDR")
// 	v.BindEnv("redis.password", "REDIS_PASSWORD")

// 	if err := v.ReadInConfig(); err != nil {
// 		return nil, fmt.Errorf("read config: %w", err)
// 	}

// 	var cfg Config
// 	if err := v.Unmarshal(&cfg); err != nil {
// 		return nil, fmt.Errorf("unmarshal config: %w", err)
// 	}

// 	// assemble DSN from individual parts
// 	// search_path=auth scopes all queries to auth schema
// 	cfg.Database.DSN = fmt.Sprintf(
// 		"postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=auth",
// 		cfg.Database.User,
// 		cfg.Database.Password,
// 		cfg.Database.Host,
// 		cfg.Database.Port,
// 		cfg.Database.Name,
// 	)

// 	// pull secrets directly — never in yaml
// 	cfg.JWT.AccessSecret = sharedconfig.MustGet("JWT_ACCESS_SECRET")
// 	cfg.JWT.RefreshSecret = sharedconfig.MustGet("JWT_REFRESH_SECRET")
// 	cfg.SMTP.Username = sharedconfig.MustGet("SMTP_USERNAME")
// 	cfg.SMTP.Password = sharedconfig.MustGet("SMTP_PASSWORD")
// 	cfg.Env = sharedconfig.Get("ENVIRONMENT", "dev")

// 	if err := validate(&cfg); err != nil {
// 		return nil, err
// 	}

// 	return &cfg, nil
// }

// func validate(cfg *Config) error {
// 	required := []struct {
// 		value string
// 		name  string
// 	}{
// 		{cfg.Database.User, "POSTGRES_USER"},
// 		{cfg.Database.Password, "POSTGRES_PASSWORD"},
// 		{cfg.Database.Host, "POSTGRES_HOST"},
// 		{cfg.Database.Name, "POSTGRES_DB"},
// 		{cfg.Redis.Addr, "REDIS_ADDR"},
// 		{cfg.JWT.AccessSecret, "JWT_ACCESS_SECRET"},
// 		{cfg.JWT.RefreshSecret, "JWT_REFRESH_SECRET"},
// 		{cfg.SMTP.Username, "SMTP_USERNAME"},
// 		{cfg.SMTP.Password, "SMTP_PASSWORD"},
// 	}

// 	for _, r := range required {
// 		if r.value == "" {
// 			return fmt.Errorf("required env var not set: %s", r.name)
// 		}
// 	}

// 	if cfg.OTP.Length < 4 || cfg.OTP.Length > 10 {
// 		return fmt.Errorf("otp.length must be between 4 and 10, got %d", cfg.OTP.Length)
// 	}
// 	if cfg.OTP.MaxVerifyAttempts < 1 {
// 		return fmt.Errorf("otp.max_verify_attempts must be at least 1, got %d", cfg.OTP.MaxVerifyAttempts)
// 	}
// 	if cfg.OTP.TTL <= 0 {
// 		return fmt.Errorf("otp.ttl must be positive")
// 	}

// 	return nil
// }

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
	GRPCAddr     string        `mapstructure:"grpc_addr"`
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

	// All v.BindEnv calls removed — Database/SMTP/Redis/JWT are now
	// populated explicitly below via the shared Load* functions, which
	// call MustGet/Get directly instead of relying on viper's env binding.

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	// Server, OTP, Argon2, Log still populate normally via mapstructure tags.
	// Database, Redis, JWT, SMTP are zero-valued here — populated next.

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
	// All required-field checks for Database/Redis/JWT/SMTP removed —
	// MustGet inside each shared Load* function already fails loudly
	// if any of those env vars are missing. Only OTP-specific business
	// rules remain here, since those aren't generic config concerns.

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