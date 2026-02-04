package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	KuCoin struct {
		Key        string `mapstructure:"api_key"`
		Secret     string `mapstructure:"api_secret"`
		Passphrase string `mapstructure:"api_passphrase"`
	} `mapstructure:"kucoin"`

	Telegram struct {
		BotToken string   `mapstructure:"bot_token"`
		ChatIDs  []string `mapstructure:"chat_ids"`
	} `mapstructure:"telegram"`

	DataDir string `mapstructure:"data_dir"`
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
