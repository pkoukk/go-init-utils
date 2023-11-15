package giu

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggerParams struct {
	LogName   string // log file name with path
	LogLevel  string // log level: info, debug, warn, error, dpanic, panic, fatal
	MaxSize   int    // size in megabytes
	MaxBackup int    // max backup files
	MaxAge    int    // max age in days
	Compress  bool   // compress
	Tag       string // log tag
}

var (
	ERR_LOGGER_NOT_INIT = errors.New("logger is nil, please init logger first")
)

const (
	LOG_LEVEL_DEBUG  = "debug"
	LOG_LEVEL_INFO   = "info"
	LOG_LEVEL_WARN   = "warn"
	LOG_LEVEL_ERROR  = "error"
	LOG_LEVEL_PANIC  = "panic"
	LOG_LEVEL_DPANIC = "dpanic"
	LOG_LEVEL_FATAL  = "fatal"
)

var _defaultLoggerParams = LoggerParams{
	LogName:   "logs/log.log",
	LogLevel:  "debug",
	MaxSize:   10,
	MaxBackup: 10,
	MaxAge:    7,
	Compress:  true,
	Tag:       "default",
}

func NewZapLogger(params *LoggerParams) *zap.Logger {
	core := newZapCore(params.LogName, params.LogLevel, params.MaxSize, params.MaxBackup, params.MaxAge, params.Compress)
	return zap.New(core, zap.AddCaller(), zap.Development(), zap.Fields(zap.String("tag", params.Tag)))
}

func DefaultZapLogger() *zap.Logger {
	return NewZapLogger(&_defaultLoggerParams)
}

func newZapCore(fileName string, level string, maxSize int, maxBackups int, maxAge int, compress bool) zapcore.Core {
	hook := lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   compress,
	}
	atomicLevel := zap.NewAtomicLevel()
	logLevel := convertZapLevel(level)
	atomicLevel.SetLevel(logLevel)
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		// EncodeCaller:   zapcore.FullCallerEncoder,
		EncodeName: zapcore.FullNameEncoder,
	}

	syncer := zapcore.AddSync(&hook)
	if logLevel <= zapcore.InfoLevel {
		// log to stdout when log level is info or lower
		syncer = zapcore.NewMultiWriteSyncer(syncer, zapcore.AddSync(os.Stdout))
	}

	return zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		syncer,
		atomicLevel,
	)
}

type ZapLogger struct {
	*zap.Logger
}

func (zl *ZapLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	if zl.Logger != nil {
		zl.Logger.Sugar().Infof(format, v...)
	}
}

func convertZapLevel(logLevel string) zapcore.Level {
	var level zapcore.Level
	switch logLevel {
	case LOG_LEVEL_INFO:
		level = zapcore.InfoLevel
	case LOG_LEVEL_DEBUG:
		level = zapcore.DebugLevel
	case LOG_LEVEL_WARN:
		level = zapcore.WarnLevel
	case LOG_LEVEL_ERROR:
		level = zapcore.ErrorLevel
	case LOG_LEVEL_DPANIC:
		level = zapcore.DPanicLevel
	case LOG_LEVEL_PANIC:
		level = zapcore.PanicLevel
	case LOG_LEVEL_FATAL:
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}
	return level
}

func convertSLogLevel(logLevel string) slog.Level {
	var level slog.Level
	switch logLevel {
	case LOG_LEVEL_INFO:
		level = slog.LevelInfo
	case LOG_LEVEL_DEBUG:
		level = slog.LevelDebug
	case LOG_LEVEL_WARN:
		level = slog.LevelWarn
	case LOG_LEVEL_ERROR:
		level = slog.LevelError
	case LOG_LEVEL_DPANIC:
		level = slog.LevelError
	case LOG_LEVEL_PANIC:
		level = slog.LevelError
	case LOG_LEVEL_FATAL:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	return level
}

func NewSLogger(params LoggerParams) *slog.Logger {
	var writer io.Writer
	hook := lumberjack.Logger{
		Filename:   params.LogName,
		MaxSize:    params.MaxSize,
		MaxBackups: params.MaxBackup,
		MaxAge:     params.MaxAge,
		Compress:   params.Compress,
	}
	logLevel := convertSLogLevel(params.LogLevel)
	if logLevel < slog.LevelInfo {
		writer = io.MultiWriter(&hook, os.Stdout)
	} else {
		writer = &hook
	}
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: logLevel})
	logger := slog.New(handler)
	if params.Tag != "" {
		logger = logger.With(slog.String("tag", params.Tag))
	}
	return logger
}

func DefaultSLogger() *slog.Logger {
	return NewSLogger(_defaultLoggerParams)
}
