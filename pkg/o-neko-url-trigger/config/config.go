package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
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
	Api     ApiConfig     `yaml:"api"`
	Mode    Mode          `yaml:"mode"`
	Server  ServerConfig  `yaml:"server"`
	Logging LoggingConfig `yaml:"logging"`
}

type LoggingConfig struct {
	Level LogLevel `yaml:"level"`
}

type ApiConfig struct {
	BaseUrl                string     `yaml:"baseUrl"`
	Auth                   AuthConfig `yaml:"auth"`
	CacheRequestsInMinutes int        `yaml:"cacheRequestsInMinutes"`
}

type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Mode string

const (
	DEVELOPMENT Mode = "development"
	PRODUCTION  Mode = "production"
)

type LogLevel string

const (
	DEBUG LogLevel = "debug"
	INFO  LogLevel = "info"
	WARN  LogLevel = "warn"
	ERROR LogLevel = "error"
)

type ServerConfig struct {
	Port int `yaml:"port"`
}
