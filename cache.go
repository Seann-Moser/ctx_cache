package ctx_cache

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/trace"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	CTX_CACHE = "cache_ctx"
)

var (
	ErrCacheMiss    = errors.New("cache missed")
	ErrCacheUpdated = errors.New("cache updated")
	ErrCacheGet     = errors.New("cache get")
	DefaultCache    Cache
	UseHash         bool = false
)

type CacheObject interface {
	GetName() string
}

type Cache interface {
	SetCache
	GetCache
	DeleteKey(ctx context.Context, key string) error
	Ping(ctx context.Context) error
	Close()
	GetName() string
	GetParentCaches() map[string]Cache
}

type SetCache interface {
	SetCache(ctx context.Context, group, key string, item interface{}) error
	SetCacheWithExpiration(ctx context.Context, cacheTimeout time.Duration, group, key string, item interface{}) error
}

type GetCache interface {
	GetCache(ctx context.Context, group, key string) ([]byte, error)
}

func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func GetKey[T any](key1, key2 string) string {
	return GetTypeReflect[T]() + key1 + key2

}

func Set[T any](ctx context.Context, group, key string, data T) error {
	k := GetKey[T](group, key)
	err := GetCacheFromContext(ctx).SetCache(ctx, group, k, Wrapper[T]{Data: data}.Get())
	if err != nil {
		return err
	}
	if strings.EqualFold(group, GroupPrefix) || group == "" || group == key {
		return nil
	}
	return GlobalCacheMonitor.UpdateCache(ctx, group, k)
}

func Delete[T any](ctx context.Context, group, key string) error {
	return GetCacheFromContext(ctx).DeleteKey(ctx, GetKey[T](group, key))
}

func DeleteKey(ctx context.Context, key string) error {
	return GetCacheFromContext(ctx).DeleteKey(ctx, key)
}

func SetWithExpiration[T any](ctx context.Context, cacheTimeout time.Duration, group, key string, data T) error {
	getSetKey := trace.StartRegion(ctx, "get_set_key")
	c := GetCacheFromContext(ctx)
	k := GetKey[T](group, key)
	getSetKey.End()
	encodedData := trace.StartRegion(ctx, "encodeData")

	w := Wrapper[T]{Data: data}.Get()
	encodedData.End()
	setCache := trace.StartRegion(ctx, "setCache")
	err := c.SetCacheWithExpiration(ctx, cacheTimeout, group, k, w)
	setCache.End()
	if err != nil {
		return err
	}
	if strings.EqualFold(group, GroupPrefix) || group == "" || group == key {
		return nil
	}
	updateGlobal := trace.StartRegion(ctx, "update_global")
	defer updateGlobal.End()
	return GlobalCacheMonitor.UpdateCache(ctx, group, k)
}

func SetFromCache[T any](ctx context.Context, cache Cache, group, key string, data T) error {
	return cache.SetCache(ctx, group, GetKey[T](group, key), Wrapper[T]{Data: data}.Get())
}
func SetFromCacheWithExpiration[T any](ctx context.Context, cache Cache, cacheTimeout time.Duration, group, key string, data T) error {
	return cache.SetCacheWithExpiration(ctx, cacheTimeout, group, GetKey[T](group, key), Wrapper[T]{Data: data}.Get())
}

type Wrapper[T any] struct {
	Data T `json:"data"`
}

func (w Wrapper[T]) Get() interface{} {
	if CheckPrimaryType[T](w.Data) {
		return w.Data
	}
	return w
}

func UnmarshalWrappert[T any](data []byte) (*T, error) {
	var output Wrapper[T]
	err := json.Unmarshal(data, &output)
	if err != nil {
		err = json.Unmarshal(data, &output)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal data in UnmarshalWrappert: %w", err)
		}
	}
	return &output.Data, nil
}

func Get[T any](ctx context.Context, group, key string) (*T, error) {
	//if group != GroupPrefix && group != "" {
	//
	//	if GlobalCacheMonitor.HasGroupKeyBeenUpdated(ctx, group) {
	//		return nil, ErrCacheUpdated
	//	}
	//}
	getKey := trace.StartRegion(ctx, "get_key")
	key = GetKey[T](group, key)
	c := GetCacheFromContext(ctx)
	getKey.End()
	getCache := trace.StartRegion(ctx, "get_cache")
	data, err := c.GetCache(ctx, group, key)
	getCache.End()
	if err != nil {
		return nil, err
	}

	convert := trace.StartRegion(ctx, "convert")
	defer convert.End()
	if CheckPrimaryType[T](*new(T)) {
		t, err := ConvertBytesToType[T](data)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}

	return UnmarshalWrappert[T](data)
	//var output Wrapper[T]
	//err = json.Unmarshal(data, &output)
	//if err != nil {
	//	return nil, err
	//}
	//return &output.Data, nil
}

func GetSet[T any](ctx context.Context, cacheTimeout time.Duration, group, key string, refresh bool, gtr func(ctx context.Context) (T, error)) (T, error) {
	if refresh {
		nv, err := gtr(ctx)
		if err != nil {
			var tmp T
			return tmp, err
		}
		return nv, SetWithExpiration[T](ctx, cacheTimeout, group, key, nv)
	}
	if v, err := Get[T](ctx, group, key); errors.Is(err, ErrCacheMiss) || errors.Is(err, ErrCacheUpdated) || v == nil {
		nv, err := gtr(ctx)
		if err != nil {
			var tmp T
			return tmp, err
		}
		return nv, SetWithExpiration[T](ctx, cacheTimeout, group, key, nv)
	} else {
		return *v, nil
	}
}

func GetSetP[T any](ctx context.Context, cacheTimeout time.Duration, group, key string, refresh bool, gtr func(ctx context.Context) (*T, error)) (*T, error) {
	if refresh {
		nv, err := gtr(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed getting cache value(group:%s, key:%s): %w", group, key, err)
		}
		if nv == nil {
			return nil, ErrCacheGet
		}
		return nv, SetWithExpiration[T](ctx, cacheTimeout, group, key, *nv)
	}
	if v, err := Get[T](ctx, group, key); errors.Is(err, ErrCacheMiss) || errors.Is(err, ErrCacheUpdated) || v == nil {
		nv, err := gtr(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed getting cache value(group:%s, key:%s): %w", group, key, err)
		}
		if nv == nil {
			return nil, ErrCacheGet
		}
		return nv, SetWithExpiration[T](ctx, cacheTimeout, group, key, *nv)
	} else {
		return v, nil
	}
}

func GetSetCheck[T any](ctx context.Context, cacheTimeout time.Duration, group, key string, refresh bool, isValid func(ctx context.Context, data *T) bool, gtr func(ctx context.Context) (T, error)) (T, error) {
	if refresh {
		nv, err := gtr(ctx)
		if err != nil {
			var tmp T
			return tmp, err
		}
		return nv, SetWithExpiration[T](ctx, cacheTimeout, group, key, nv)
	}
	if v, err := Get[T](ctx, group, key); errors.Is(err, ErrCacheMiss) || errors.Is(err, ErrCacheUpdated) || v == nil || !isValid(ctx, v) {
		nv, err := gtr(ctx)
		if err != nil {
			var tmp T
			return tmp, err
		}
		return nv, SetWithExpiration[T](ctx, cacheTimeout, group, key, nv)
	} else {
		return *v, nil
	}
}

func GetSetCheckP[T any](ctx context.Context, cacheTimeout time.Duration, group, key string, refresh bool, isValid func(ctx context.Context, data *T) bool, gtr func(ctx context.Context) (*T, error)) (*T, error) {
	if refresh {
		refresh := trace.StartRegion(ctx, "refresh_function")
		nv, err := gtr(ctx)
		refresh.End()
		if err != nil {
			return nil, fmt.Errorf("failed getting cache value(group:%s, key:%s): %w", group, key, err)
		}
		if nv == nil {
			return nil, ErrCacheGet
		}
		setWithExpiration := trace.StartRegion(ctx, "set_with_expiration")
		defer setWithExpiration.End()
		return nv, SetWithExpiration[T](ctx, cacheTimeout, group, key, *nv)
	}
	get := trace.StartRegion(ctx, "get")
	v, err := Get[T](ctx, group, key)
	get.End()
	if errors.Is(err, ErrCacheMiss) || errors.Is(err, ErrCacheUpdated) || v == nil || !isValid(ctx, v) {
		setFunctionCall := trace.StartRegion(ctx, "set_function_call")
		nv, err := gtr(ctx)
		setFunctionCall.End()
		if err != nil {
			return nil, fmt.Errorf("failed getting cache value(group:%s, key:%s): %w", group, key, err)
		}
		if nv == nil {
			return nil, ErrCacheGet
		}
		setWithExpiration := trace.StartRegion(ctx, "set_with_expiration")
		defer setWithExpiration.End()
		return nv, SetWithExpiration[T](ctx, cacheTimeout, group, key, *nv)
	} else {
		return v, nil
	}
}

func GetFromCache[T any](ctx context.Context, cache Cache, group, key string) (*T, error) {
	if GlobalCacheMonitor.HasGroupKeyBeenUpdated(ctx, group) {
		return nil, ErrCacheUpdated
	}
	data, err := cache.GetCache(ctx, group, GetKey[T](group, key))
	if err != nil {
		return nil, err
	}
	var output Wrapper[T]
	err = json.Unmarshal(data, &output)
	if err != nil {
		return nil, err
	}
	return &output.Data, nil
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
