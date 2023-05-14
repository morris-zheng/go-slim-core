package logger

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Fatal(ctx context.Context, message string, params ...interface{})
	Error(ctx context.Context, message string, params ...interface{})
	Warn(ctx context.Context, message string, params ...interface{})
	Debug(ctx context.Context, message string, params ...interface{})
	Info(ctx context.Context, message string, params ...interface{})
}

type logger struct {
	zapLogger *zap.SugaredLogger
}

type Level string

const (
	FATAL Level = `FATAL`
	ERROR Level = `ERROR`
	WARN  Level = `WARN`
	INFO  Level = `INFO`
	DEBUG Level = `DEBUG`
)

var logTypes = map[Level]int{
	DEBUG: -1,
	INFO:  0,
	WARN:  1,
	ERROR: 2,
	FATAL: 5,
}

var errLevelNotDefined = errors.New("log level not defined")

func NewLogger(level Level) (Logger, error) {
	logLevel, ok := logTypes[level]
	if !ok {
		return nil, errLevelNotDefined
	}

	lev := zapcore.Level(logLevel)
	customTimeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format(time.RFC3339Nano))
	}

	config := zap.NewProductionConfig()
	config.DisableStacktrace = true
	config.Encoding = "json"
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stdout"}
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = customTimeEncoder
	config.EncoderConfig.CallerKey = "caller"
	config.Level.SetLevel(lev)

	l, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	// flushes buffer
	defer l.Sync()

	sugar := l.Sugar()

	return &logger{
		zapLogger: sugar,
	}, nil
}

func (l *logger) Fatal(ctx context.Context, message string, params ...interface{}) {
	l.zapLogger.Fatalw(message, params...)
}

func (l *logger) Error(ctx context.Context, message string, params ...interface{}) {
	l.zapLogger.Errorw(message, params...)
}

func (l *logger) Warn(ctx context.Context, message string, params ...interface{}) {
	l.zapLogger.Warnw(message, params...)
}

func (l *logger) Debug(ctx context.Context, message string, params ...interface{}) {
	l.zapLogger.Debugw(message, params...)
}

func (l *logger) Info(ctx context.Context, message string, params ...interface{}) {
	l.zapLogger.Infow(message, params...)
}
