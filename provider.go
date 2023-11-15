package giu

import (
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Provider[T any] interface {
	Add(name string, d T, isDefault ...bool)
	Get(name string) (T, bool)
	Default() T
	SetDefault(name string) bool
	Shutdown() error
}

type GiuProvider[T any] struct {
	lock      sync.RWMutex
	d         T
	container map[string]T
}

func MapToSet[T any](m map[string]T) []Set[T] {
	var s []Set[T]
	for k, v := range m {
		s = append(s, Set[T]{Name: k, Value: v})
	}
	return s
}

type Set[T any] struct {
	Name  string
	Value T
}

// NewProvider creates a generic provider, if items is not empty, the first item will be set as default
func NewProvider[T any](items ...map[string]T) Provider[T] {
	return NewGiuProvider[T](items...)
}

// NewGiuProvider creates a generic provider, if items is not empty, the first item will be set as default
func NewGiuProvider[T any](items ...map[string]T) *GiuProvider[T] {
	g := &GiuProvider[T]{
		lock:      sync.RWMutex{},
		container: make(map[string]T)}
	if len(items) > 0 {
		for k, v := range items[0] {
			g.Add(k, v)
		}
	}
	return g
}

// NewGiuProviderWithLogger creates a generic provider with item init function and the params used in the init function
func NewGiuProviderFromParams[T any, U any](newFunc func(U) T, params map[string]U) *GiuProvider[T] {
	itemMap := make(map[string]T)
	for k, v := range params {
		itemMap[k] = newFunc(v)
	}
	return NewGiuProvider(itemMap)
}

// NewGiuProviderWithLogger creates a generic provider with item init function and the params used in the init function.
// The item needs a zap logger to init, so the logger is also passed in.
func NewGiuProviderWithLoggerFromParams[T any, U any](newFunc func(U, *zap.Logger) T, params map[string]U, logger *zap.Logger) *GiuProvider[T] {
	itemMap := make(map[string]T)
	for k, v := range params {
		itemMap[k] = newFunc(v, logger)
	}
	return NewGiuProvider(itemMap)
}

// NewGiuProviderWithLogger creates a generic provider with item init function and the params used in the init function.
// The init function needs a logger and it may return an error.
func NewGiuProviderWithLoggerFromParamsError[T any, U any](newFunc func(U, *zap.Logger) (T, error), params map[string]U, logger *zap.Logger) (*GiuProvider[T], error) {
	itemMap := make(map[string]T)
	for k, v := range params {
		item, err := newFunc(v, logger)
		if err != nil {
			return nil, err
		}
		itemMap[k] = item
	}
	return NewGiuProvider(itemMap), nil
}

// NewGiuProviderWithLogger creates a generic provider with item init function and the params used in the init function.
// The init function may return an error.
func NewGiuProviderFromParamsError[T any, U any](newFunc func(U) (T, error), params map[string]U) (*GiuProvider[T], error) {
	itemMap := make(map[string]T)
	for k, v := range params {
		item, err := newFunc(v)
		if err != nil {
			return nil, err
		}
		itemMap[k] = item
	}
	return NewGiuProvider(itemMap), nil
}

// NewGiuProviderFromConfig creates a generic provider with item init function and read the init params from viper config.
func NewGiuProviderFromConfig[T any, U any](config *viper.Viper, configKey string, newFunc func(U) T) (*GiuProvider[T], error) {
	var params map[string]U
	if err := config.UnmarshalKey(configKey, &params); err != nil {
		return nil, err
	}
	return NewGiuProviderFromParams[T, U](newFunc, params), nil
}

// NewGiuProviderWithLoggerFromConfig creates a generic provider with item init function and read the init params from viper config.
// The item needs a zap logger to init.
func NewGiuProviderWithLoggerFromConfig[T any, U any](config *viper.Viper, configKey string, newFunc func(U, *zap.Logger) T, logger *zap.Logger) (*GiuProvider[T], error) {
	var params map[string]U
	if err := config.UnmarshalKey(configKey, &params); err != nil {
		return nil, err
	}
	return NewGiuProviderWithLoggerFromParams[T, U](newFunc, params, logger), nil
}

// NewGiuProviderWithLoggerFromConfig creates a generic provider with item init function and read the init params from viper config.
// The function may return an error.
func NewGiuProviderFromConfigError[T any, U any](config *viper.Viper, configKey string, newFunc func(U) (T, error)) (*GiuProvider[T], error) {
	var params map[string]U
	if err := config.UnmarshalKey(configKey, &params); err != nil {
		return nil, err
	}
	return NewGiuProviderFromParamsError[T, U](newFunc, params)
}

// NewGiuProviderWithLoggerFromConfig creates a generic provider with item init function and read the init params from viper config.
// The function needs a zap logger to init and may return an error.
func NewGiuProviderWithLoggerFromConfigError[T any, U any](config *viper.Viper, configKey string, newFunc func(U, *zap.Logger) (T, error), logger *zap.Logger) (*GiuProvider[T], error) {
	var params map[string]U
	if err := config.UnmarshalKey(configKey, &params); err != nil {
		return nil, err
	}
	return NewGiuProviderWithLoggerFromParamsError[T, U](newFunc, params, logger)
}

// Add adds a value to the generic provider
func (p *GiuProvider[T]) Add(name string, d T, isDefault ...bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(isDefault) > 0 && isDefault[0] {
		p.d = d
	}
	if len(p.container) == 0 {
		p.d = d
	}
	p.container[name] = d
}

// Get returns the value of the generic provider, if the name is not found, it returns false
func (p *GiuProvider[T]) Get(name string) (T, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	v, ok := p.container[name]
	return v, ok
}

// Default returns the default value of the generic provider, if no default value is set, it returns the first value
func (p *GiuProvider[T]) Default() T {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.d
}

// SetDefault sets the default value of the generic provider, if the name is not found, it returns false
func (p *GiuProvider[T]) SetDefault(name string) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.container[name]; ok {
		p.d = p.container[name]
		return true
	}
	return false

}

// Shutdown is a placeholder for the generic provider, it should be implemented by the specific provider
func (p *GiuProvider[T]) Shutdown() error {
	return nil
}

type GormProvider interface {
	Provider[*gorm.DB]
}

type gormProvider struct {
	*GiuProvider[*gorm.DB]
}

func (gp *gormProvider) Shutdown() error {
	for _, v := range gp.container {
		if db, err := v.DB(); err == nil {
			if err := db.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

// NewGormProvider creates a gorm provider from existing connection, if items is not empty, the first item will be set as default
func NewGormProvider(connections ...map[string]*gorm.DB) GormProvider {
	return newGormProvider(connections...)
}

func newGormProvider(connections ...map[string]*gorm.DB) *gormProvider {
	p := &gormProvider{
		GiuProvider: NewGiuProvider[*gorm.DB](connections...),
	}
	return p
}

// NewGormProviderFromParams creates a gorm provider from params, if items is not empty, the first item will be set as default
func NewGormProviderFromParams(configParams *GormConfigParams, connectionParams map[string]*GormConnectionParams) (GormProvider, error) {
	connections := make(map[string]*gorm.DB)
	for k, v := range connectionParams {
		conn, err := NewGorm(*v, configParams)
		if err != nil {
			return nil, err
		}
		connections[k] = conn
	}
	return NewGormProvider(connections), nil
}

// NewGormProviderFromConfig creates a gorm provider from viper config and GiuConfig struct, if items is not empty, the first item will be set as default
func NewGormProviderFromConfig(config *viper.Viper) (GormProvider, error) {
	var c GormConfigParams
	var connections map[string]*GormConnectionParams
	if err := config.UnmarshalKey("gorm_config", &c); err != nil {
		return nil, err
	}
	if err := config.UnmarshalKey("gorm_connection", &connections); err != nil {
		return nil, err
	}
	return NewGormProviderFromParams(&c, connections)
}

// NewGormProviderWithLoggerFromConfig creates a gorm provider from viper config and GiuConfig struct and replace default logger with zap logger, if items is not empty, the first item will be set as default
func NewGormProviderWithLoggerFromConfig(config *viper.Viper, logger *zap.Logger) (GormProvider, error) {
	var c GormConfigParams
	var connectionParams map[string]*GormConnectionParams
	if err := config.UnmarshalKey("gorm_config", &c); err != nil {
		return nil, err
	}
	if err := config.UnmarshalKey("gorm_connection", &connectionParams); err != nil {
		return nil, err
	}
	connections := make(map[string]*gorm.DB)
	for k, v := range connectionParams {
		conn, err := NewGormWithLogger(*v, logger, &c)
		if err != nil {
			return nil, err
		}
		connections[k] = conn
	}
	return NewGormProvider(connections), nil
}

type ZapProvider interface {
	Provider[*zap.Logger]
}

type zapProvider struct {
	*GiuProvider[*zap.Logger]
}

func (zp *zapProvider) Shutdown() error {
	for _, v := range zp.container {
		if err := v.Sync(); err != nil {
			return err
		}
	}
	return nil
}

// NewZapProvider creates a zap provider from existing logger, if items is not empty, the first item will be set as default
func NewZapProvider(loggers ...map[string]*zap.Logger) ZapProvider {
	return &zapProvider{
		GiuProvider: NewGiuProvider[*zap.Logger](loggers...),
	}
}

// NewZapProviderFromParams creates a zap provider from params, if items is not empty, the first item will be set as default
func NewZapProviderFromParams(params map[string]*LoggerParams) ZapProvider {
	return &zapProvider{
		GiuProvider: NewGiuProviderFromParams[*zap.Logger, *LoggerParams](NewZapLogger, params),
	}
}

// NewZapProviderFromConfig creates a zap provider from viper config and GiuConfig struct, if items is not empty, the first item will be set as default
func NewZapProviderFromConfig(config *viper.Viper) (ZapProvider, error) {
	giu, err := NewGiuProviderFromConfig[*zap.Logger, *LoggerParams](config, "logger", NewZapLogger)
	if err != nil {
		return nil, err
	}
	return &zapProvider{
		GiuProvider: giu,
	}, nil
}

type RedisProvider interface {
	Provider[redis.UniversalClient]
}

type redisProvider struct {
	*GiuProvider[redis.UniversalClient]
}

func (rp *redisProvider) Shutdown() error {
	for _, v := range rp.container {
		if err := v.Close(); err != nil {
			return err
		}
	}
	return nil
}

// NewRedisProvider creates a redis provider from existing connection, if items is not empty, the first item will be set as default
func NewRedisProvider(clients ...map[string]redis.UniversalClient) Provider[redis.UniversalClient] {
	return &redisProvider{
		GiuProvider: NewGiuProvider[redis.UniversalClient](clients...),
	}
}

// NewRedisProviderFromParams creates a redis provider from params, if items is not empty, the first item will be set as default
func NewRedisProviderFromParams(params map[string]*RedisParams) Provider[redis.UniversalClient] {
	return &redisProvider{
		GiuProvider: NewGiuProviderFromParams[redis.UniversalClient, *RedisParams](NewRedis, params),
	}
}

// NewRedisProviderFromConfig creates a redis provider from viper config and GiuConfig struct, if items is not empty, the first item will be set as default.
// NOTE: it's not a good idea to log redis cmd, so we don't use zap logger here.
func NewRedisProviderFromConfig(config *viper.Viper) (Provider[redis.UniversalClient], error) {
	giu, err := NewGiuProviderFromConfig[redis.UniversalClient, *RedisParams](config, "redis", NewRedis)
	if err != nil {
		return nil, err
	}
	return &redisProvider{
		GiuProvider: giu,
	}, nil
}
