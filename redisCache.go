package ctx_cache

import (
	"context"
	"encoding/json"
	redis "github.com/Seann-Moser/ociredis"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"time"
)

var _ Cache = &RedisCache{}

type RedisCache struct {
	cacher          *redis.Client
	defaultDuration time.Duration
}

func RedisFlags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(prefix+"redis", pflag.ExitOnError)
	fs.String(prefix+"redis-addr", "", "")
	fs.String(prefix+"redis-pass", "", "")

	fs.Duration(prefix+"redis-cleanup-duration", 1*time.Minute, "")

	return fs
}
func NewRedisCacheFromFlags(ctx context.Context, prefix string) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     viper.GetString(prefix + "redis-addr"),
		Password: viper.GetString(prefix + "redis-pass"),
		Context:  ctx,
	})

	return NewRedisCache(rdb, viper.GetDuration(prefix+"redis-cleanup-duration"))
}

func NewRedisCache(cacher *redis.Client, defaultDuration time.Duration) *RedisCache {
	return &RedisCache{
		cacher:          cacher,
		defaultDuration: defaultDuration,
	}
}
func (c *RedisCache) Close() {
	_ = c.cacher.Close()
}

func (c *RedisCache) SetCache(ctx context.Context, key string, item interface{}) error {
	if c == nil {
		return ErrCacheMiss
	}
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	localClient := c.cacher.WithContext(ctx)
	stats := localClient.Set(key, data, c.defaultDuration)
	return stats.Err()
}

func (c *RedisCache) GetCache(ctx context.Context, key string) ([]byte, error) {
	if c == nil {
		return nil, ErrCacheMiss
	}
	localClient := c.cacher.WithContext(ctx)

	return localClient.Get(key).Bytes()
}

func (c *RedisCache) Ping(ctx context.Context) error {
	localClient := c.cacher.WithContext(ctx)
	return localClient.Ping().Err()
}
