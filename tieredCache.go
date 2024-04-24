package ctx_cache

import (
	"context"
	"strconv"
	"time"

	"go.uber.org/multierr"
)

var _ Cache = &TieredCache{}

type TieredCache struct {
	cachePool []Cache
	getter    GetCache
}

func (t *TieredCache) GetParentCaches() map[string]Cache {
	data := map[string]Cache{}
	if len(t.cachePool) <= 1 {
		return data
	}
	for i, cache := range t.cachePool {
		data[strconv.FormatInt(int64(i), 10)] = cache
	}
	return data
}

func NewTieredCache(setter GetCache, cacheList ...Cache) Cache {
	return &TieredCache{
		cachePool: cacheList,
		getter:    setter,
	}
}

func (t *TieredCache) SetCacheWithExpiration(ctx context.Context, cacheTimeout time.Duration, group, key string, item interface{}) error {
	var err error
	for _, c := range t.cachePool {
		err = multierr.Combine(err, c.SetCacheWithExpiration(ctx, cacheTimeout, group, key, item))
	}
	return err
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

func (t *TieredCache) SetCache(ctx context.Context, group, key string, item interface{}) error {
	var err error
	for _, c := range t.cachePool {
		err = multierr.Combine(err, c.SetCache(ctx, group, key, item))
	}
	return err
}

func (t *TieredCache) GetCache(ctx context.Context, group, key string) ([]byte, error) {
	var missedCacheList []Cache
	var v []byte
	var err error
	defer func() {
		for _, c := range missedCacheList {
			_ = c.SetCache(ctx, group, key, v)
		}
	}()
	for _, c := range t.cachePool {
		v, err = c.GetCache(ctx, group, key)
		if err != nil || v == nil {
			missedCacheList = append(missedCacheList, c)
			continue
		}
		return v, nil
	}
	if t.getter == nil {
		return nil, ErrCacheMiss
	}
	v, err = t.getter.GetCache(ctx, group, key)
	if err != nil {
		missedCacheList = []Cache{}
		return nil, err
	}
	return v, nil
}
