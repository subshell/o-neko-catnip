package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

var configuration *Config

func Configuration() *Config {
	if configuration == nil {
		configuration = readInConfig()
	}
	return configuration
}

func readInConfig() *Config {
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config/")
	viper.AddConfigPath(".")
	viper.SetConfigName("application-default")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("failed to read in default config: %w \n", err))
	}

	viper.SetConfigName("application.yaml")
	if err := viper.MergeInConfig(); err != nil {
		// maybe we don't need this
		// fmt.Println()fmt.Errorf("failed to read in config: %w \n", err)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	readConfig := Config{}
	if err := viper.Unmarshal(&readConfig); err != nil {
		panic(fmt.Errorf("failed to parse config: %w \n", err))
	}
	return &readConfig
}

type Config struct {
	ONeko ONekoConfig `yaml:"oneko"`
}

type ONekoConfig struct {
	Mode   Mode         `yaml:"mode"`
	Server ServerConfig `yaml:"server"`
}

type Mode string
const (
	DEVELOPMENT Mode = "development"
	PRODUCTION  Mode = "production"
)

type ServerConfig struct {
	Port int `yaml:"port"`
}
