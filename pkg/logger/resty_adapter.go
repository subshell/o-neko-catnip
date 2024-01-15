package logger

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"log/slog"
)

type RestyLoggerAdapter struct {
	logger *slog.Logger
}

func RestyAdapter(logger *slog.Logger) resty.Logger {
	return &RestyLoggerAdapter{
		logger: logger,
	}
}

func (a *RestyLoggerAdapter) Errorf(format string, v ...interface{}) {
	a.logger.Error(fmt.Sprintf(format, v...))
}

func (a *RestyLoggerAdapter) Warnf(format string, v ...interface{}) {
	a.logger.Warn(fmt.Sprintf(format, v...))
}

func (a *RestyLoggerAdapter) Debugf(format string, v ...interface{}) {
	a.logger.Debug(fmt.Sprintf(format, v...))
}
