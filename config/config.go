package config

import (
	"strings"

	"github.com/spf13/viper"
)

var AppConfig Config

type Config struct {
	RedisURL      string
	RedisUser     string
	RedisPassword string
	RedisDatabase int
	AppPort       string
}

func LoadConfig() error {
	viper.SetConfigName("env")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	AppConfig = Config{
		RedisURL:      viper.GetString("redis.host"),
		RedisUser:     viper.GetString("redis.user"),
		RedisPassword: viper.GetString("redis.pass"),
		RedisDatabase: viper.GetInt("redis.db"),
		AppPort:       viper.GetString("app.port"),
	}

	return nil
}
