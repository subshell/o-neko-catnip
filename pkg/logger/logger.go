package logger

import (
	"log/slog"
	"o-neko-catnip/pkg/config"
	"os"
	"sync"
)

var rootLogger *slog.Logger
var initOnce = sync.OnceFunc(setupRootLogger)

const LevelTrace = slog.Level(-8)

func setupRootLogger() {
	if rootLogger != nil {
		return
	}

	mode := config.Configuration().ONeko.Mode

	var (
		handler slog.Handler
	)

	if mode == config.DEVELOPMENT {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     parseLogLevel(config.Configuration().ONeko.Logging.Level),
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     parseLogLevel(config.Configuration().ONeko.Logging.Level),
		})
	}

	rootLogger = slog.New(handler)
}

func parseLogLevel(level config.LogLevel) slog.Level {
	var result slog.Level

	switch level {
	case "trace":
		result = LevelTrace
	case "debug":
		result = slog.LevelDebug
	case "info":
		result = slog.LevelInfo
	case "warn":
		result = slog.LevelWarn
	case "error":
		result = slog.LevelError
	default:
		result = slog.LevelDebug
	}

	return result
}

func New(name string) *slog.Logger {
	initOnce()
	return rootLogger.With(slog.String("logger", name))
}
