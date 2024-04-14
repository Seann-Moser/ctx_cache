package ctx_cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/patrickmn/go-cache"

	"go.uber.org/multierr"
)

var (
	ErrCacheMiss = errors.New("cache missed")
)

type Cache interface {
	SetCache
	GetCache
	DeleteKey(ctx context.Context, key string) error
	Ping(ctx context.Context) error
	Close()
}

type SetCache interface {
	SetCache(ctx context.Context, key string, item interface{}) error
	SetCacheWithExpiration(ctx context.Context, cacheTimeout time.Duration, key string, item interface{}) error
}

type GetCache interface {
	GetCache(ctx context.Context, key string) ([]byte, error)
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
		if t.Elem().Kind() == reflect.Struct {
			return t.Elem().Name()
		}
		return t.String()
	}
}
func GetKey[T any](key string) string {
	var d T
	return fmt.Sprintf("%s_%s", getType(d), key)
}

func Set[T any](ctx context.Context, key string, data T) error {
	return GetCacheFromContext(ctx).SetCache(ctx, GetKey[T](key), data)
}

func SetWithExpiration[T any](ctx context.Context, cacheTimeout time.Duration, key string, data T) error {
	return GetCacheFromContext(ctx).SetCacheWithExpiration(ctx, cacheTimeout, GetKey[T](key), data)
}

func SetFromCache[T any](ctx context.Context, cache Cache, key string, data T) error {
	return cache.SetCache(ctx, GetKey[T](key), data)
}
func SetFromCacheWithExpiration[T any](ctx context.Context, cache Cache, cacheTimeout time.Duration, key string, data T) error {
	return cache.SetCacheWithExpiration(ctx, cacheTimeout, GetKey[T](key), data)
}

func Get[T any](ctx context.Context, key string) (*T, error) {
	data, err := GetCacheFromContext(ctx).GetCache(ctx, GetKey[T](key))
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

func GetFromCache[T any](ctx context.Context, cache Cache, key string) (*T, error) {
	data, err := cache.GetCache(ctx, GetKey[T](key))
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

const (
	CTX_CACHE = "cache_ctx"
)

func ContextWithCache(ctx context.Context, cache Cache) context.Context {
	return context.WithValue(ctx, CTX_CACHE, cache) //nolint:staticcheck
}

func GetCacheFromContext(ctx context.Context) Cache {
	if ctx == nil {
		return &GoCache{
			defaultDuration: 0,
			cacher:          cache.New(time.Minute, time.Minute),
			cacheTags:       NewCacheTags("go-cache", "backup"),
		}
	}
	gCache := ctx.Value(CTX_CACHE)
	if gCache == nil {
		return &GoCache{
			defaultDuration: time.Minute,
			cacher:          cache.New(time.Minute, time.Minute),
			cacheTags:       NewCacheTags("go-cache", "backup"),
		}
	}
	return gCache.(Cache)
}

var _ Cache = &TieredCache{}

type TieredCache struct {
	cachePool []Cache
	getter    GetCache
}

func (t *TieredCache) SetCacheWithExpiration(ctx context.Context, cacheTimeout time.Duration, key string, item interface{}) error {
	var err error
	for _, c := range t.cachePool {
		err = multierr.Combine(err, c.SetCacheWithExpiration(ctx, cacheTimeout, key, item))
	}
	return err
}

func NewTieredCache(setter GetCache, cacheList ...Cache) Cache {
	return &TieredCache{
		cachePool: cacheList,
		getter:    setter,
	}
}

func (t *TieredCache) DeleteKey(ctx context.Context, key string) error {
	var err error
	for _, c := range t.cachePool {
		err = multierr.Combine(err, c.DeleteKey(ctx, key))
	}
	return err
}

func (t *TieredCache) Ping(ctx context.Context) error {
	var err error
	for _, c := range t.cachePool {
		err = multierr.Combine(err, c.Ping(ctx))
	}
	return err
}

func (t *TieredCache) Close() {
	for _, c := range t.cachePool {
		c.Close()
	}
}

func (t *TieredCache) SetCache(ctx context.Context, key string, item interface{}) error {
	var err error
	for _, c := range t.cachePool {
		err = multierr.Combine(err, c.SetCache(ctx, key, item))
	}
	return err
}

func (t *TieredCache) GetCache(ctx context.Context, key string) ([]byte, error) {
	var missedCacheList []Cache
	var v []byte
	var err error
	defer func() {
		for _, c := range missedCacheList {
			_ = c.SetCache(ctx, key, v)
		}
	}()
	for _, c := range t.cachePool {
		v, err = c.GetCache(ctx, key)
		if err != nil || v == nil {
			missedCacheList = append(missedCacheList, c)
			continue
		}
		return v, nil
	}
	if t.getter == nil {
		return nil, ErrCacheMiss
	}
	v, err = t.getter.GetCache(ctx, key)
	if err != nil {
		missedCacheList = []Cache{}
		return nil, err
	}
	return v, nil
}
