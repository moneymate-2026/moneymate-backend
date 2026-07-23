	package sharedconfig

	import "github.com/spf13/viper"

	type SMTPConfig struct {
		Host        string
		Port        int
		FromAddress string
		FromName    string
		Username    string
		Password    string
	}

	func LoadSMTPConfig(v *viper.Viper) SMTPConfig {
		return SMTPConfig{
			Host:        v.GetString("smtp.host"),
			Port:        v.GetInt("smtp.port"),
			FromAddress: v.GetString("smtp.from_address"),
			FromName:    v.GetString("smtp.from_name"),
			Username:    MustGet("SMTP_USERNAME"),
			Password:    MustGet("SMTP_PASSWORD"),
		}
	}