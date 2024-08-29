package ctx_cache

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
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
	err := GetCacheFromContext(ctx).SetCache(ctx, group, GetKey[T](group, key), Wrapper[T]{Data: data}.Get())
	if err != nil {
		return err
	}
	if strings.EqualFold(group, GroupPrefix) || group == "" {
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
	c := GetCacheFromContext(ctx)
	k := GetKey[T](group, key)
	w := Wrapper[T]{Data: data}.Get()

	err := c.SetCacheWithExpiration(ctx, cacheTimeout, group, k, w)
	if err != nil {
		return err
	}
	if strings.EqualFold(group, GroupPrefix) || group == "" {
		return nil
	}
	return GlobalCacheMonitor.UpdateCache(ctx, group, key)
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
	err := jsoniter.Unmarshal(data, &output)
	if err != nil {
		err = json.Unmarshal(data, &output)
		if err != nil {
			return nil, err
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
	key = GetKey[T](group, key)
	c := GetCacheFromContext(ctx)
	data, err := c.GetCache(ctx, group, key)
	if err != nil {
		return nil, err
	}

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
		nv, err := gtr(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed getting cache value(group:%s, key:%s): %w", group, key, err)
		}
		if nv == nil {
			return nil, ErrCacheGet
		}
		return nv, SetWithExpiration[T](ctx, cacheTimeout, group, key, *nv)
	}
	v, err := Get[T](ctx, group, key)
	if errors.Is(err, ErrCacheMiss) || errors.Is(err, ErrCacheUpdated) || v == nil || !isValid(ctx, v) {
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
