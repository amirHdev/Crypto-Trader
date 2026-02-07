package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	KuCoin struct {
		Key        string `mapstructure:"KUCOIN_API_KEY"`
		Secret     string `mapstructure:"KUCOIN_API_SECRET"`
		Passphrase string `mapstructure:"KUCOIN_API_PASSPHRASE"`
	}
	Telegram struct {
		BotToken string   `mapstructure:"TELEGRAM_BOT_TOKEN"`
		ChatIDs  []string `mapstructure:"TELEGRAM_CHAT_IDS"`
	}
	DataDir string `mapstructure:"DATA_DIR"`
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("No .env file found, relying on environment variables only: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Defaults
	if cfg.DataDir == "" {
		cfg.DataDir = "./data"
	}

	return &cfg, nil
}
