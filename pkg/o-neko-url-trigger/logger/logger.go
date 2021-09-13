package logger

import (
	"fmt"
	"go.uber.org/zap"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/config"
)

var rootLogger *zap.SugaredLogger

func setupRootLogger() {
	if rootLogger != nil {
		return
	}

	mode := config.Configuration().ONeko.Mode
	var (
		logger *zap.Logger
		err    error
	)
	if mode == config.DEVELOPMENT {
		logger, err = zap.NewDevelopment()
	} else if mode == config.PRODUCTION {
		logger, err = zap.NewProduction()
	} else {
		logger = zap.NewNop()
	}

	if err != nil {
		panic(fmt.Sprintf("failed to setup logging: %v", err))
	}

	rootLogger = logger.Sugar()
}

func New(name string) *zap.SugaredLogger {
	return Root().Named(name)
}

func Root() *zap.SugaredLogger {
	setupRootLogger()
	return rootLogger
}
