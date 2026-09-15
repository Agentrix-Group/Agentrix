package tracer

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const (
	RequestIdKey     contextKey = "requestId"
	ParticipantIdKey contextKey = "participantId"
)

var (
	logger *zap.SugaredLogger
	once   sync.Once
)

func Errorf(ctx context.Context, format string, args ...interface{}) {
	WithContext(ctx).Errorf(format, args...)
}

func Infof(ctx context.Context, format string, args ...interface{}) {
	WithContext(ctx).Infof(format, args...)
}

func Debugf(ctx context.Context, format string, args ...interface{}) {
	WithContext(ctx).Debugf(format, args...)
}

func Warnf(ctx context.Context, format string, args ...interface{}) {
	WithContext(ctx).Warnf(format, args...)
}

func Fatalf(ctx context.Context, format string, args ...interface{}) {
	WithContext(ctx).Fatalf(format, args...)
}

func Panicf(ctx context.Context, format string, args ...interface{}) {
	WithContext(ctx).Panicf(format, args...)
}

func DPanicf(ctx context.Context, format string, args ...interface{}) {
	WithContext(ctx).DPanicf(format, args...)
}

func Error(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Error(args...)
}

func Info(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Info(args...)
}

func Debug(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Debug(args...)
}

func Warn(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Warn(args...)
}

func Fatal(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Fatal(args...)
}

func Panic(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Panic(args...)
}

func DPanic(ctx context.Context, args ...interface{}) {
	WithContext(ctx).DPanic(args...)
}

func WithContext(ctx context.Context) *zap.SugaredLogger {
	once.Do(func() {
		logger = initLogger()
	})
	l := logger
	if requestId, ok := ctx.Value(RequestIdKey).(string); ok && requestId != "" {
		if os.Getenv("MODE") != "gcp" && os.Getenv("MODE") != "railway" && len(requestId) > 8 {
			l = l.With("req", requestId[:8])
		} else {
			l = l.With("requestId", requestId)
		}
	}
	if participantId, ok := ctx.Value(ParticipantIdKey).(string); ok && participantId != "" {
		l = l.With("part", participantId)
	}
	return l
}

func initLogger() *zap.SugaredLogger {
	mode := os.Getenv("MODE")
	logLevelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))

	level := zapcore.InfoLevel
	switch logLevelStr {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "info":
		level = zapcore.InfoLevel
	default:
		level = zapcore.InfoLevel
	}

	var config zap.Config
	if mode == "gcp" || mode == "railway" {
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(level)
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	} else {
		config = zap.Config{
			Level:       zap.NewAtomicLevelAt(level),
			Development: true,
			Encoding:    "console",
			EncoderConfig: zapcore.EncoderConfig{
				MessageKey:       "message",
				LevelKey:         "level",
				TimeKey:          "timestamp",
				CallerKey:        "caller",
				StacktraceKey:    "stack",
				LineEnding:       "\n",
				EncodeLevel:      zapcore.CapitalColorLevelEncoder,
				EncodeTime:       zapcore.TimeEncoderOfLayout("15:04:05"),
				EncodeDuration:   zapcore.StringDurationEncoder,
				EncodeCaller:     zapcore.ShortCallerEncoder,
				ConsoleSeparator: " ",
			},
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
	}

	zapLogger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return zapLogger.Sugar()
}
