package observability

import (
	"fmt"
	"strings"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	base *zap.SugaredLogger
}

func NewLogger(appEnv, logLevel string) (*Logger, error) {
	level, err := zapcore.ParseLevel(strings.ToLower(strings.TrimSpace(logLevel)))
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", logLevel, err)
	}

	loggerCfg := loggerConfig(appEnv)
	loggerCfg.Level = zap.NewAtomicLevelAt(level)

	base, err := loggerCfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}
	return &Logger{base: base.Sugar()}, nil
}

func loggerConfig(appEnv string) zap.Config {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "development":
		return zap.NewDevelopmentConfig()
	default:
		cfg := zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "ts"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		return cfg
	}
}

func (l *Logger) Info(msg string, keysAndValues ...any) {
	l.base.Infow(msg, keysAndValues...)
}

func (l *Logger) Warn(msg string, keysAndValues ...any) {
	l.base.Warnw(msg, keysAndValues...)
}

func (l *Logger) Error(msg string, keysAndValues ...any) {
	l.base.Errorw(msg, keysAndValues...)
}

func (l *Logger) Debug(msg string, keysAndValues ...any) {
	l.base.Debugw(msg, keysAndValues...)
}

func (l *Logger) Sync() error {
	return l.base.Sync()
}

// defaultLogger 用 atomic.Pointer 而非裸全局变量：L() 在各请求 goroutine 并发
// 读、SetDefault/懒初始化写入，裸变量是数据竞争（同仓库其它单例同样处理）。
var defaultLogger atomic.Pointer[Logger]

func SetDefault(l *Logger) {
	if l != nil {
		defaultLogger.Store(l)
	}
}

func L() *Logger {
	if l := defaultLogger.Load(); l != nil {
		return l
	}
	l, err := NewLogger("production", "info")
	if err != nil {
		panic("observability: cannot create default logger: " + err.Error())
	}
	defaultLogger.CompareAndSwap(nil, l)
	return defaultLogger.Load()
}
