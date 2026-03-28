package observability

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	base *zap.Logger
}

func NewLogger(appEnv, logLevel string) (*Logger, error) {
	level, err := zapcore.ParseLevel(strings.ToLower(strings.TrimSpace(logLevel)))
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", logLevel, err)
	}

	loggerCfg, err := loggerConfig(appEnv)
	if err != nil {
		return nil, err
	}
	loggerCfg.Level = zap.NewAtomicLevelAt(level)

	base, err := loggerCfg.Build()
	if err != nil {
		return nil, err
	}
	return &Logger{base: base}, nil
}

func loggerConfig(appEnv string) (zap.Config, error) {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "development":
		return zap.NewDevelopmentConfig(), nil
	default:
		return zap.NewProductionConfig(), nil
	}
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.base.Info(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.base.Error(msg, fields...)
}

func (l *Logger) Sync() error {
	return l.base.Sync()
}

func Field(key string, value any) zap.Field {
	return zap.Any(key, value)
}
