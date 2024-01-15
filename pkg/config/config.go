package config

import (
	"context"
	"fmt"
	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"regexp"
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

	if err := transformConfig(&readConfig); err != nil {
		panic(fmt.Errorf("failed to transform config: %w \n", err))
	}

	if err := validateConfig(readConfig); err != nil {
		panic(fmt.Errorf("config is invalid: %w \n", err))
	}

	return &readConfig
}

func transformConfig(c *Config) error {
	tf := modifiers.New()
	tf.Register("urlWithoutProtocol", func(ctx context.Context, fl mold.FieldLevel) error {
		protocolRegex := regexp.MustCompile(`^\w+://`)
		if protocolRegex.MatchString(fl.Field().String()) {
			literalString := protocolRegex.ReplaceAllLiteralString(fl.Field().String(), "")
			fl.Field().SetString(literalString)
		}
		return nil
	})
	return tf.Struct(context.Background(), c)
}

func validateConfig(c Config) error {
	validate := validator.New()

	err := validate.RegisterValidation("urlWithOptionalPort", func(fl validator.FieldLevel) bool {
		hostnameAndOptionalPortRegex := regexp.MustCompile(`[\w.\-]+(:\d{1,5})?`)
		return hostnameAndOptionalPortRegex.MatchString(fl.Field().String())
	}, false)

	if err != nil {
		return err
	}

	if err := validate.Struct(c); err != nil {
		return err
	}

	return nil
}

type Config struct {
	ONeko ONekoConfig `yaml:"oneko"`
}

type ONekoConfig struct {
	Api       ApiConfig     `yaml:"api" validate:"required"`
	Mode      Mode          `yaml:"mode" validate:"required,oneof='development' 'production'"`
	Server    ServerConfig  `yaml:"server" validate:"required"`
	CatnipUrl string        `yaml:"catnipUrl" validate:"required,urlWithOptionalPort" mod:"trim,lcase,urlWithoutProtocol"`
	Logging   LoggingConfig `yaml:"logging" validate:"required"`
}

type LoggingConfig struct {
	Level LogLevel `yaml:"level" validate:"oneof='' 'debug' 'info' 'warn' 'error'"`
}

type ApiConfig struct {
	BaseUrl              string        `yaml:"baseUrl" validate:"required,uri"`
	Auth                 AuthConfig    `yaml:"auth" validate:"required"`
	ApiCallCacheDuration time.Duration `yaml:"apiCallCacheDuration" validate:"required,min=15s,max=10m"`
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
	TRACE LogLevel = "trace"
	DEBUG LogLevel = "debug"
	INFO  LogLevel = "info"
	WARN  LogLevel = "warn"
	ERROR LogLevel = "error"
)

type ServerConfig struct {
	Port        int `yaml:"port" validate:"required,number"`
	MetricsPort int `yaml:"metricsPort" validate:"required,number"`
}
