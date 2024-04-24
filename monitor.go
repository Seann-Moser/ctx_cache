package ctx_cache

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"sync"
	"time"
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
	groupKeys     map[string]time.Time
}

func NewMonitor() CacheMonitor {
	return &CacheMonitorImpl{
		cleanDuration: time.Minute,
		groupKeys:     make(map[string]time.Time),
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
	err = SetWithExpiration[int64](ctx, 60*time.Minute, GroupPrefix, groupKey, now.Unix())
	if err != nil {
		return err
	}
	ctxLogger.Debug(ctx, "setting cache", zap.String("group", group), zap.String("key", key))
	c.Mutex.Lock()
	c.groupKeys[groupKey] = now
	c.Mutex.Unlock()
	return nil
}

func (c *CacheMonitorImpl) DeleteCache(ctx context.Context, group string) error {
	keys, err := c.GetGroupKeys(ctx, group)
	if err != nil {
		return err
	}
	for k, _ := range keys {
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
	key := fmt.Sprintf("%s_%s_keys", GroupPrefix, group)
	keys, err := Get[map[string]struct{}](ctx, GroupPrefix, key)
	var foundKeys map[string]struct{}
	if err != nil {
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
	//todo check all caches to verify they are all the same value
	key := fmt.Sprintf("%s_%s_updated", GroupPrefix, group)
	lastUpdated, err := Get[int64](ctx, GroupPrefix, key)
	if err != nil {
		err = SetWithExpiration[int64](ctx, 60*time.Minute, GroupPrefix, key, time.Now().Unix())
		if err != nil {
			return true
		}
		return true
	}
	ctxLogger.Debug(ctx, "last updated", zap.Int64("lastUpdated", *lastUpdated))
	for _, c := range GetCacheFromContext(ctx).GetParentCaches() {
		i, err := GetFromCache[int64](ctx, c, GroupPrefix, key)
		if err != nil || *i != *lastUpdated {
			ctxLogger.Debug(ctx, "last updated does not match", zap.Int64("lastUpdated", *lastUpdated), zap.Int64("cache", i))
			return true
		}
	}
	c.Mutex.Lock()
	if v, found := c.groupKeys[key]; found && v.Equal(time.Unix(*lastUpdated, 0)) {
		c.Mutex.Unlock()
		ctxLogger.Debug(ctx, "last updated does not match", zap.Time("group_key", v), zap.Time("last_updated", time.Unix(*lastUpdated, 0)))
		return false
	}
	c.groupKeys[key] = time.Unix(*lastUpdated, 0)
	c.Mutex.Unlock()
	return true
}
