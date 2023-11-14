package giu

import "github.com/redis/go-redis/v9"

type RedisParams = redis.UniversalOptions

func NewRedis(options *redis.UniversalOptions) redis.UniversalClient {
	return redis.NewUniversalClient(options)
}

var _defaultRedisOptions = redis.UniversalOptions{
	Addrs: []string{"localhost:6379"},
}

func NewStandaloneRedis(addrs string) redis.UniversalClient {
	return NewRedis(&redis.UniversalOptions{
		Addrs: []string{addrs},
	})
}

func NewClusterRedis(addrs []string) redis.UniversalClient {
	return NewRedis(&redis.UniversalOptions{
		Addrs: addrs,
	})
}

func NewFailOverRedisClient(addrs []string, masterName string) redis.UniversalClient {
	return NewRedis(&redis.UniversalOptions{
		Addrs:      addrs,
		MasterName: masterName,
	})
}

func DefaultRedis() redis.UniversalClient {
	return NewRedis(&_defaultRedisOptions)
}
