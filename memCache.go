package ctx_cache

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/orijtech/gomemcache/memcache"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"time"
)

var _ Cache = &MemCache{}

type MemCache struct {
	memcacheClient  *memcache.Client
	defaultDuration time.Duration
}

func MemcacheFlags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(prefix+"memcache", pflag.ExitOnError)
	fs.StringSlice(prefix+"memcache-addrs", []string{}, "")
	fs.Duration(prefix+"memcache-default-duration", 1*time.Minute, "")
	return fs
}
func NewMemcacheFromFlags(prefix string) *MemCache {
	return NewMemcache(memcache.New(viper.GetStringSlice(prefix+"memcache-addrs")...), viper.GetDuration(prefix+"memcache-default-duration"))
}

func NewMemcache(cacher *memcache.Client, defaultDuration time.Duration) *MemCache {
	return &MemCache{
		memcacheClient:  cacher,
		defaultDuration: defaultDuration,
	}
}

func (c *MemCache) Close() {

}

func (c *MemCache) Ping(ctx context.Context) error {
	return nil
}

func (c *MemCache) SetCache(ctx context.Context, key string, item interface{}) error {
	if c == nil {
		return ErrCacheMiss
	}
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	return c.memcacheClient.Set(ctx, &memcache.Item{
		Key:        key,
		Value:      data,
		Expiration: int32(c.defaultDuration.Seconds()),
	})
}

func (c *MemCache) GetCache(ctx context.Context, key string) ([]byte, error) {
	if c == nil {
		return nil, ErrCacheMiss
	}
	it, err := c.memcacheClient.Get(ctx, key)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	return it.Value, nil
}
