package ctx_cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var GlobalCacheMonitor CacheMonitor = NewMonitor()

const GroupPrefix = "[CTX_CACHE_GROUP]"

type CacheMonitor interface {
	AddGroupKeys(ctx context.Context, group string, newKeys ...string) error
	HasGroupKeyBeenUpdated(ctx context.Context, group string) bool
	GetGroupKeys(ctx context.Context, group string) (map[string]struct{}, error)
	DeleteCache(ctx context.Context, group string) error
	UpdateCache(ctx context.Context, group string, key string) error
}

type CacheMonitorImpl struct {
	cleanDuration time.Duration
	Mutex         *sync.RWMutex
	groupKeys     map[string]int64
}

func NewMonitor() CacheMonitor {
	return &CacheMonitorImpl{
		cleanDuration: time.Minute,
		groupKeys:     make(map[string]int64),
		Mutex:         &sync.RWMutex{},
	}
}

func (c *CacheMonitorImpl) UpdateCache(ctx context.Context, group string, key string) error {
	err := c.AddGroupKeys(ctx, group, key)
	if err != nil {
		return err
	}
	now := time.Now()
	groupKey := fmt.Sprintf("%s_%s_updated", GroupPrefix, group)
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.groupKeys[groupKey] = now.Unix()
	err = SetWithExpiration[int64](ctx, 60*time.Minute, GroupPrefix, groupKey, now.Unix())
	if err != nil {
		return err
	}
	ctxLogger.Debug(ctx, "setting cache", zap.String("group", group), zap.String("key", key))
	return nil
}

func (c *CacheMonitorImpl) DeleteCache(ctx context.Context, group string) error {
	keys, err := c.GetGroupKeys(ctx, group)
	if err != nil {
		return err
	}
	for k := range keys {
		err = multierr.Combine(err, DeleteKey(ctx, k))
	}
	return err
}

func (c *CacheMonitorImpl) GetGroupKeys(ctx context.Context, group string) (map[string]struct{}, error) {
	key := fmt.Sprintf("%s_%s_keys", GroupPrefix, group)
	keys, err := Get[map[string]struct{}](ctx, GroupPrefix, key)
	var foundKeys map[string]struct{}
	if err != nil {
		return nil, err
	} else {
		foundKeys = *keys
	}
	return foundKeys, nil
}

func (c *CacheMonitorImpl) AddGroupKeys(ctx context.Context, group string, newKeys ...string) error {
	if len(newKeys) == 0 {
		return nil
	}
	key := fmt.Sprintf("%s_%s_keys", GroupPrefix, group)
	keys, err := Get[map[string]struct{}](ctx, GroupPrefix, key)
	var foundKeys map[string]struct{}
	if err != nil || keys == nil {
		foundKeys = map[string]struct{}{}
	} else {
		foundKeys = *keys
	}
	for _, k := range newKeys {
		foundKeys[k] = struct{}{}
	}
	return SetWithExpiration[map[string]struct{}](ctx, 60*time.Minute, GroupPrefix, key, foundKeys)
}

// HasGroupKeyBeenUpdated is the time does not match then the key value has been updated, if it has been updated invalidate all cache
func (c *CacheMonitorImpl) HasGroupKeyBeenUpdated(ctx context.Context, group string) bool {
	if group == GroupPrefix {
		return false
	}
	key := fmt.Sprintf("%s_%s_updated", GroupPrefix, group)
	lastUpdated, err := Get[int64](ctx, GroupPrefix, key)
	if err != nil {
		ctxLogger.Debug(ctx, "failed getting last updated group", zap.Error(err))
		err = SetWithExpiration[int64](ctx, 60*time.Minute, GroupPrefix, key, time.Now().Unix())
		if err != nil {
			return true
		}
		return true
	}
	for _, c := range GetCacheFromContext(ctx).GetParentCaches() {
		i, err := GetFromCache[int64](ctx, c, GroupPrefix, key)
		if err != nil || *i != *lastUpdated {
			if i != nil {
				ctxLogger.Debug(ctx, "last updated does not match", zap.Int64("lastUpdated", *lastUpdated), zap.Int64("cache", *i), zap.Error(err))
			}
			return true
		}
	}
	c.Mutex.Lock()
	if v, found := c.groupKeys[key]; found && v == *lastUpdated {
		c.Mutex.Unlock()
		return false
	} else {
		ctxLogger.Debug(ctx, "last updated does not match group", zap.Int64("group_key", v), zap.Int64("last_updated", *lastUpdated))
	}
	c.groupKeys[key] = *lastUpdated
	c.Mutex.Unlock()
	return true
}
