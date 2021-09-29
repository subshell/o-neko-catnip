package logger

import (
	"fmt"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/config"

	"go.uber.org/zap"
)

var rootLogger *zap.SugaredLogger

func setupRootLogger() {
	if rootLogger != nil {
		return
	}

	mode := config.Configuration().ONeko.Mode
	logLevelString := config.Configuration().ONeko.Logging.Level

	var (
		logger *zap.Logger
		err    error
	)

	if mode == config.DEVELOPMENT {
		logger, err = createLogger(zap.NewDevelopmentConfig(), string(logLevelString))
	} else if mode == config.PRODUCTION {
		logger, err = createLogger(zap.NewProductionConfig(), string(logLevelString))
	} else {
		logger = zap.NewNop()
	}

	if err != nil {
		panic(fmt.Sprintf("failed to setup logging: %v", err))
	}

	rootLogger = logger.Sugar()
}

func createLogger(config zap.Config, logLevelString string) (*zap.Logger, error) {
	if logLevelString != "" {
		levelRef := zap.NewAtomicLevel().Level()
		logLevel := &levelRef
		logLevel.UnmarshalText([]byte(logLevelString))
		config.Level = zap.NewAtomicLevelAt(*logLevel)
	}
	return config.Build()
}

func New(name string) *zap.SugaredLogger {
	return Root().Named(name)
}

func Root() *zap.SugaredLogger {
	setupRootLogger()
	return rootLogger
}
