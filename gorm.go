package giu

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type GormConnectionParams struct {
	Driver   string
	Host     string
	Port     uint
	User     string
	Password string
	Database string
}

type GormConfigParams struct {
	*gorm.Config
	Logger   *zap.Logger
	LogLevel string
}

var _defaultGormParams = GormConnectionParams{
	Driver:   "mysql",
	Host:     "localhost",
	Port:     3306,
	User:     "root",
	Password: "root",
	Database: "test",
}

const (
	GORM_DRIVER_MYSQL      = "mysql"
	GORM_DRIVER_PG         = "postgres"
	GORM_DRIVER_PG_SHORTEN = "pg"
	GORM_DRIVER_SQLITE     = "sqlite"
	GORM_DRIVER_SQLSERVER  = "sqlserver"
)

func NewGorm(params GormConnectionParams, configParams ...*GormConfigParams) (*gorm.DB, error) {
	config := &gorm.Config{}
	if len(configParams) > 0 && configParams[0] != nil {
		configParam := configParams[0]
		if configParam.Config != nil {
			config = configParam.Config
		}
		if configParam.Logger != nil {
			config.Logger = NewZapGormLogger(configParam.Logger, configParam.LogLevel)
		}
	}

	switch params.Driver {
	case GORM_DRIVER_MYSQL:
		return gorm.Open(NewGormMysql(params))
	case GORM_DRIVER_PG, GORM_DRIVER_PG_SHORTEN:
		return gorm.Open(NewGormPostgres(params))
	case GORM_DRIVER_SQLITE:
		return gorm.Open(NewGormSQLite(params))
	case GORM_DRIVER_SQLSERVER:
		return gorm.Open(NewGormSQLServer(params))
	default:
		return nil, fmt.Errorf("unsupported gorm driver: %s", params.Driver)
	}
}

func DefaultGorm() (*gorm.DB, error) {
	return NewGorm(_defaultGormParams)
}

func NewGormMysql(params GormConnectionParams) gorm.Dialector {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", params.User, params.Password, params.Host, params.Port, params.Database)
	return mysql.Open(dsn)
}

func NewGormPostgres(params GormConnectionParams) gorm.Dialector {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable", params.Host, params.User, params.Password, params.Database, params.Port)
	return postgres.Open(dsn)
}

func NewGormSQLServer(params GormConnectionParams) gorm.Dialector {
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", params.User, params.Password, params.Host, params.Port, params.Database)
	return mysql.Open(dsn)
}

func NewGormSQLite(params GormConnectionParams) gorm.Dialector {
	dsn := fmt.Sprintf("%s.db", params.Database)
	return mysql.Open(dsn)
}

type ZapGormLogger struct {
	logger                    *zap.Logger
	logLevel                  logger.LogLevel
	SlowThreshold             time.Duration
	SkipCallerLookup          bool
	IgnoreRecordNotFoundError bool
	TraceWarnStr              string
	TraceErrStr               string
	TraceStr                  string
}

func NewZapGormLogger(zl *zap.Logger, logLevel string) *ZapGormLogger {
	gLevel := convertGormLogLevel(logLevel)
	return &ZapGormLogger{
		logger:                    zl,
		logLevel:                  gLevel,
		SlowThreshold:             200 * time.Millisecond,
		SkipCallerLookup:          true,
		IgnoreRecordNotFoundError: true,
		TraceWarnStr:              "gorm: warning",
		TraceErrStr:               "gorm: error",
		TraceStr:                  "gorm: info",
	}
}

func convertGormLogLevel(level string) logger.LogLevel {
	switch level {
	case LOG_LEVEL_DEBUG:
		return logger.Info
	case LOG_LEVEL_INFO:
		return logger.Info
	case LOG_LEVEL_WARN:
		return logger.Warn
	case LOG_LEVEL_ERROR:
		return logger.Error
	case LOG_LEVEL_DPANIC:
		return logger.Error
	case LOG_LEVEL_PANIC:
		return logger.Error
	case LOG_LEVEL_FATAL:
		return logger.Error
	default:
		return logger.Silent
	}
}

func (z *ZapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *z
	newLogger.logLevel = level
	return &newLogger
}

func (z *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if z.logLevel >= logger.Info {
		z.logger.Sugar().Infof(msg, data...)
	}
}

func (z *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if z.logLevel >= logger.Warn {
		z.logger.Sugar().Warnf(msg, data...)
	}
}

func (z *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if z.logLevel >= logger.Error {
		z.logger.Sugar().Errorf(msg, data...)
	}
}

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	switch {
	case err != nil && l.logLevel >= logger.Error && (!errors.Is(err, logger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			l.logger.Sugar().Errorf(l.TraceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.logger.Sugar().Errorf(l.TraceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.logLevel >= logger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			l.logger.Sugar().Warn(l.TraceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.logger.Sugar().Warn(l.TraceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.logLevel == logger.Info:
		sql, rows := fc()
		if rows == -1 {
			l.logger.Sugar().Infof(l.TraceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.logger.Sugar().Infof(l.TraceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}
