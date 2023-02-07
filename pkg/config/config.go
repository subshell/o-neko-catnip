package config

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var configuration *Config

func Configuration() *Config {
	if configuration == nil {
		configuration = readInConfig()
	}
	return configuration
}

// Use this to set a configuration in tests
func OverrideConfiguration(c *Config) {
	configuration = c
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
		if _, isConfigFileNotFoundError := err.(viper.ConfigFileNotFoundError); !isConfigFileNotFoundError {
			panic(fmt.Errorf("failed to read in config file: %w \n", err))
		}
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	readConfig := Config{}
	if err := viper.Unmarshal(&readConfig); err != nil {
		panic(fmt.Errorf("failed to parse config: %w \n", err))
	}

	if err := validateConfig(readConfig); err != nil {
		panic(fmt.Errorf("config is invalid: %w \n", err))
	}

	return &readConfig
}

func validateConfig(c Config) error {
	validate := validator.New()

	if err := validate.Struct(c); err != nil {
		return err
	}

	return nil
}

type Config struct {
	ONeko ONekoConfig `yaml:"oneko"`
}

type ONekoConfig struct {
	Api       ApiConfig     `yaml:"api" validate:"required,dive"`
	Mode      Mode          `yaml:"mode" validate:"required,oneof='development' 'production'"`
	Server    ServerConfig  `yaml:"server" validate:"required,dive"`
	CatnipUrl string        `yaml:"catnipUrl" validate:"required,uri"`
	Logging   LoggingConfig `yaml:"logging" validate:"required,dive"`
}

type LoggingConfig struct {
	Level LogLevel `yaml:"level" validate:"oneof='' 'debug' 'info' 'warn' 'error'"`
}

type ApiConfig struct {
	BaseUrl              string        `yaml:"baseUrl" validate:"required,uri"`
	Auth                 AuthConfig    `yaml:"auth" validate:"required,dive"`
	ApiCallCacheDuration time.Duration `yaml:"apiCallCacheDuration" validate:"required,max=5m"`
}

type AuthConfig struct {
	Username string `yaml:"username" validate:"required"`
	Password string `yaml:"password" validate:"required"`
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
	Port int `yaml:"port" validate:"required,number"`
}
