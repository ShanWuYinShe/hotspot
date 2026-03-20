package observability

import (
	"go.uber.org/zap"
)

type Logger struct {
	base *zap.Logger
}

func NewLogger() (*Logger, error) {
	base, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &Logger{base: base}, nil
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
