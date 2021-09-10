package logger

import (
	"fmt"
	"go.uber.org/zap"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/config"
)

var rootLogger *zap.SugaredLogger

func init() {
	mode := config.Configuration.ONeko.Mode
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
	return rootLogger.Named(name)
}

func Root() *zap.SugaredLogger {
	return rootLogger
}
