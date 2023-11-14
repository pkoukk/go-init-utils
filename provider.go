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

func newGiuProviderFromParams[T any, U any](newFunc func(U) T, params map[string]U) *GiuProvider[T] {
	itemMap := make(map[string]T)
	for k, v := range params {
		itemMap[k] = newFunc(v)
	}
	return NewGiuProvider(itemMap)
}

func newGiuProviderFromParamsError[T any, U any](newFunc func(U) (T, error), params map[string]U) (*GiuProvider[T], error) {
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

func newGiuProviderFromConfig[T any, U any](config *viper.Viper, configKey string, newFunc func(U) T) (*GiuProvider[T], error) {
	var params map[string]U
	if err := config.UnmarshalKey(configKey, &params); err != nil {
		return nil, err
	}
	return newGiuProviderFromParams[T, U](newFunc, params), nil
}

func newGiuProviderFromConfigError[T any, U any](config *viper.Viper, configKey string, newFunc func(U) (T, error)) (*GiuProvider[T], error) {
	var params map[string]U
	if err := config.UnmarshalKey(configKey, &params); err != nil {
		return nil, err
	}
	return newGiuProviderFromParamsError[T, U](newFunc, params)
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

func NewGormProvider(connections ...map[string]*gorm.DB) GormProvider {
	return newGormProvider(connections...)
}

func newGormProvider(connections ...map[string]*gorm.DB) *gormProvider {
	p := &gormProvider{
		GiuProvider: NewGiuProvider[*gorm.DB](connections...),
	}
	return p
}

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

func NewZapProvider(loggers ...map[string]*zap.Logger) ZapProvider {
	return &zapProvider{
		GiuProvider: NewGiuProvider[*zap.Logger](loggers...),
	}
}

func NewZapProviderFromParams(params map[string]*LoggerParams) ZapProvider {
	return &zapProvider{
		GiuProvider: newGiuProviderFromParams[*zap.Logger, *LoggerParams](NewZapLogger, params),
	}
}

func NewZapProviderFromConfig(config *viper.Viper) (ZapProvider, error) {
	giu, err := newGiuProviderFromConfig[*zap.Logger, *LoggerParams](config, "logger", NewZapLogger)
	if err != nil {
		return nil, err
	}
	return &zapProvider{
		GiuProvider: giu,
	}, nil
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

func NewRedisProvider(clients ...map[string]redis.UniversalClient) Provider[redis.UniversalClient] {
	return &redisProvider{
		GiuProvider: NewGiuProvider[redis.UniversalClient](clients...),
	}
}

func NewRedisProviderFromParams(params map[string]*RedisParams) Provider[redis.UniversalClient] {
	return &redisProvider{
		GiuProvider: newGiuProviderFromParams[redis.UniversalClient, *RedisParams](NewRedis, params),
	}
}

func NewRedisProviderFromConfig(config *viper.Viper) (Provider[redis.UniversalClient], error) {
	giu, err := newGiuProviderFromConfig[redis.UniversalClient, *RedisParams](config, "redis", NewRedis)
	if err != nil {
		return nil, err
	}
	return &redisProvider{
		GiuProvider: giu,
	}, nil
}
