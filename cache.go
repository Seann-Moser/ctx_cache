package ctx_cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/patrickmn/go-cache"
	"reflect"
	"strings"
	"time"
)

const (
	CTX_CACHE = "cache_ctx"
)

var (
	ErrCacheMiss = errors.New("cache missed")
	DefaultCache Cache
)

type Cache interface {
	SetCache
	GetCache
	DeleteKey(ctx context.Context, key string) error
	Ping(ctx context.Context) error
	Close()
	GetParentCaches() map[string]Cache
}

type SetCache interface {
	SetCache(ctx context.Context, group, key string, item interface{}) error
	SetCacheWithExpiration(ctx context.Context, cacheTimeout time.Duration, group, key string, item interface{}) error
}

type GetCache interface {
	GetCache(ctx context.Context, group, key string) ([]byte, error)
}

func getType(myVar interface{}) string {
	if myVar == nil {
		return "nil"
	}
	t := reflect.TypeOf(myVar)
	if t.Kind() == reflect.Ptr {
		if t.Elem().Kind() == reflect.Struct {
			return t.Elem().Name()
		}
		return t.Elem().String()
	} else {
		if t.Kind() == reflect.Struct {
			return t.Name()
		}
		return t.String()
	}
}
func GetKey[T any](key ...string) string {
	var d T
	return fmt.Sprintf("%s_%s", getType(d), strings.Join(key, "_"))
}

func Set[T any](ctx context.Context, group, key string, data T) error {
	err := GetCacheFromContext(ctx).SetCache(ctx, group, GetKey[T](group, key), data)
	if err != nil {
		return err
	}
	if strings.EqualFold(group, GroupPrefix) {
		return nil
	}
	return GlobalCacheMonitor.UpdateCache(ctx, group, key)
}

func Delete[T any](ctx context.Context, group, key string) error {
	return GetCacheFromContext(ctx).DeleteKey(ctx, GetKey[T](group, key))
}

func DeleteKey(ctx context.Context, key string) error {
	return GetCacheFromContext(ctx).DeleteKey(ctx, key)
}

func SetWithExpiration[T any](ctx context.Context, cacheTimeout time.Duration, group, key string, data T) error {
	err := GetCacheFromContext(ctx).SetCacheWithExpiration(ctx, cacheTimeout, group, GetKey[T](group, key), data)
	if err != nil {
		return err
	}
	if strings.EqualFold(group, GroupPrefix) {
		return nil
	}
	return GlobalCacheMonitor.UpdateCache(ctx, group, key)
}

func SetFromCache[T any](ctx context.Context, cache Cache, group, key string, data T) error {
	return cache.SetCache(ctx, group, GetKey[T](group, key), data)
}
func SetFromCacheWithExpiration[T any](ctx context.Context, cache Cache, cacheTimeout time.Duration, group, key string, data T) error {
	return cache.SetCacheWithExpiration(ctx, cacheTimeout, group, GetKey[T](group, key), data)
}

func Get[T any](ctx context.Context, group, key string) (*T, error) {
	if GlobalCacheMonitor.HasGroupKeyBeenUpdated(ctx, group) {
		return nil, ErrCacheMiss
	}
	data, err := GetCacheFromContext(ctx).GetCache(ctx, group, GetKey[T](group, key))
	if err != nil {
		return nil, err
	}
	var output T
	err = json.Unmarshal(data, &output)
	if err != nil {
		return nil, err
	}
	return &output, nil
}

func GetFromCache[T any](ctx context.Context, cache Cache, group, key string) (*T, error) {
	if GlobalCacheMonitor.HasGroupKeyBeenUpdated(ctx, group) {
		return nil, ErrCacheMiss
	}
	data, err := cache.GetCache(ctx, group, GetKey[T](group, key))
	if err != nil {
		return nil, err
	}
	var output T
	err = json.Unmarshal(data, &output)
	if err != nil {
		return nil, err
	}
	return &output, nil
}

func ContextWithCache(ctx context.Context, cache Cache) context.Context {
	return context.WithValue(ctx, CTX_CACHE, cache) //nolint:staticcheck
}

func GetCacheFromContext(ctx context.Context) Cache {
	if ctx == nil {
		if DefaultCache == nil {
			DefaultCache = &GoCache{
				defaultDuration: cache.DefaultExpiration,
				cacher:          cache.New(5*time.Minute, time.Minute),
				cacheTags:       NewCacheTags("go-cache", "backup"),
			}
		}
		return DefaultCache
	}
	gCache := ctx.Value(CTX_CACHE)
	if gCache == nil {
		if DefaultCache == nil {
			DefaultCache = &GoCache{
				defaultDuration: cache.DefaultExpiration,
				cacher:          cache.New(5*time.Minute, time.Minute),
				cacheTags:       NewCacheTags("go-cache", "backup"),
			}
		}
		return DefaultCache
	}
	return gCache.(Cache)
}
