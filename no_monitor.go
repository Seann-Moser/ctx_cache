package ctx_cache

import (
	"context"
	"time"
)

type CacheMonitorNone struct {
}

func NewMonitorNoneWithFlags() CacheMonitor {
	return &CacheMonitorNone{}
}
func NewCacheMonitorNone(duration time.Duration) CacheMonitor {
	return &CacheMonitorNone{}
}

func (c *CacheMonitorNone) AddGroupKeys(ctx context.Context, group string, newKeys ...string) error {
	return nil
}

func (c *CacheMonitorNone) HasGroupKeyBeenUpdated(ctx context.Context, group string) bool {
	return false
}

func (c *CacheMonitorNone) GetGroupKeys(ctx context.Context, group string) (map[string]struct{}, error) {
	return nil, nil
}

func (c *CacheMonitorNone) DeleteCache(ctx context.Context, group string) error {
	return nil
}

func (c *CacheMonitorNone) UpdateCache(ctx context.Context, group string, key string) error {
	return nil
}

func (c *CacheMonitorNone) Close() {

}

func (c *CacheMonitorNone) Start(ctx context.Context) {

}

func (c *CacheMonitorNone) Record(ctx context.Context, cmd CacheCmd, status Status) func(err error) {
	return func(err error) {}
}

// ConvertToBytes attempts to convert various primary types to a []byte representation
