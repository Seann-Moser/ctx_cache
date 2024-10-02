package ctx_cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"go.uber.org/zap"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var _ Cache = (*RedisCache)(nil)

type RedisCache struct {
	cacher          *redis.Client
	defaultDuration time.Duration
	cacheTags       CacheTags
	enabled         bool
}

func (c *RedisCache) GetParentCaches() map[string]Cache {
	return map[string]Cache{}
}

func RedisFlags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(prefix+"redis", pflag.ExitOnError)
	fs.String(prefix+"redis-addr", "", "")
	fs.String(prefix+"redis-pass", "", "")
	fs.Bool(prefix+"redis-enabled", false, "")
	fs.String(prefix+"redis-instance", "default", "")
	fs.Duration(prefix+"redis-cleanup-duration", 1*time.Minute, "")

	return fs
}

func NewRedisCacheFromFlags(ctx context.Context, prefix string) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     viper.GetString(prefix + "redis-addr"),
		Password: viper.GetString(prefix + "redis-pass"),
	})

	return NewRedisCache(rdb, viper.GetDuration(prefix+"redis-cleanup-duration"), viper.GetString(prefix+"redis-instance"), viper.GetBool(prefix+"redis-enabled"))
}

func NewRedisCache(cacher *redis.Client, defaultDuration time.Duration, instance string, enabled bool) *RedisCache {

	return &RedisCache{
		cacher:          cacher,
		defaultDuration: defaultDuration,
		cacheTags:       NewCacheTags("redis", instance),
		enabled:         enabled,
	}
}
func (c *RedisCache) Close() {
	_ = c.cacher.Close()
}
func (c *RedisCache) GetName() string {
	return fmt.Sprintf("REDISCACHE_%s", c.cacheTags.instance)
}
func (c *RedisCache) DeleteKey(ctx context.Context, key string) error {
	stat, err := c.cacher.Del(ctx, key).Result()
	if err != nil {
		ctxLogger.Warn(ctx, "failed deleting redis cache key", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	if stat == 0 {
		ctxLogger.Info(ctx, "redis key not found for deletion", zap.String("key", key))
	} else {
		ctxLogger.Debug(ctx, "deleted redis cache key", zap.String("key", key))
	}
	return nil
}

func (c *RedisCache) SetCacheWithExpiration(ctx context.Context, cacheTimeout time.Duration, group, key string, item interface{}) error {
	if !c.enabled {
		return nil
	}
	if item == nil {
		return nil
	}
	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}
	return c.cacher.Set(ctx, key, data, cacheTimeout).Err()
}

func (c *RedisCache) SetCache(ctx context.Context, group, key string, item interface{}) error {
	return c.SetCacheWithExpiration(ctx, c.defaultDuration, group, key, item)
}

func (c *RedisCache) GetCache(ctx context.Context, group, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	data, err := c.cacher.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return data, nil
}

func (c *RedisCache) Ping(ctx context.Context) error {
	if err := c.cacher.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}
