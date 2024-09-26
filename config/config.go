package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Producer struct {
		Max_Backlog     int
		Prod_Rate       int
		Flow_Size_Limit int64
	}
	Consumer struct {
		Rate_Limit      int
		Flow_Size_Limit int64
	}
	Database struct {
		Host     string
		User     string
		Password string
		Dbname   string
		Sslmode  string
	}
}

func LoadConfig() *Config {
	viper.SetConfigFile("config/config.yaml")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	return &cfg
}
