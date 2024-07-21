package config

import (
	"strings"

	"github.com/spf13/viper"
)

var Env Config

type Config struct {
	AppPort       string
	RedisURL      string
	RedisUser     string
	RedisPassword string
	RedisDatabase int
	KafkaUrl      string
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

	Env = Config{
		AppPort:       viper.GetString("app.port"),
		RedisURL:      viper.GetString("redis.host"),
		RedisUser:     viper.GetString("redis.user"),
		RedisPassword: viper.GetString("redis.pass"),
		RedisDatabase: viper.GetInt("redis.db"),
		KafkaUrl:      viper.GetString("kafka.url"),
	}

	return nil
}
