package ctx_cache

import (
	"context"
	"time"
)

var GlobalCacheMonitor CacheMonitor = NewMonitor(time.Minute, false)

const GroupPrefix = "[CTX_CACHE_GROUP]"

type CacheMonitor interface {
	AddGroupKeys(ctx context.Context, group string, newKeys ...string) error
	HasGroupKeyBeenUpdated(ctx context.Context, group string) bool
	GetGroupKeys(ctx context.Context, group string) (map[string]struct{}, error)
	DeleteCache(ctx context.Context, group string) error
	UpdateCache(ctx context.Context, group string, key string) error
	Close()
	Start(ctx context.Context)
	Record(ctx context.Context, cmd CacheCmd, status Status) func(err error)
}
