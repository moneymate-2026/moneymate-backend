package sharedconfig

import "github.com/spf13/viper"

type JWTConfig struct {
	AccessExpiryMinutes int
	RefreshExpiryHours  int
	AccessSecret        string
	RefreshSecret       string
}

func LoadJWTConfig(v *viper.Viper) JWTConfig {
	return JWTConfig{
		AccessExpiryMinutes: v.GetInt("jwt.access_expiry_minutes"),
		RefreshExpiryHours:  v.GetInt("jwt.refresh_expiry_hours"),
		AccessSecret:        MustGet("JWT_ACCESS_SECRET"),
		RefreshSecret:       MustGet("JWT_REFRESH_SECRET"),
	}
}


