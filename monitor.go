package ctx_cache

import (
	"context"
	"sync"
	"time"
)

var MonitorCache CacheMonitor = &TableCacheResetMock{}


func SetGlobalCacheMonitor(monitor CacheMonitor) {
	if MonitorCache.IsActive() {
		return
	}
	MonitorCache = monitor
}

type CacheMonitor interface {
	PublishUpdate(ctx context.Context, group string)
	CacheTable(group string, key string)
	Start(ctx context.Context)
	IsActive() bool
}


type TableCacheResetMock struct {
}

func (t TableCacheResetMock) IsActive() bool {
	return false
}

func (t TableCacheResetMock) PublishUpdate(ctx context.Context, tableName string) {

}

func (t TableCacheResetMock) CacheTable(tableName string, key string) {

}

func (t TableCacheResetMock) Start(ctx context.Context) {

}



type TableCacheReset struct {
	tableNameSignal chan string
	syncMutex       *sync.RWMutex
	TableCache      map[string]map[string]struct{}
	running         bool
}

func NewTableCacheReset() *TableCacheReset {
	return &TableCacheReset{
		tableNameSignal: make(chan string, 100),
		syncMutex:       &sync.RWMutex{},
		TableCache:      map[string]map[string]struct{}{},
		running:         false,
	}
}

func (r *TableCacheReset) PublishUpdate(ctx context.Context, tableName string) {
	if r.running {
		tick := time.NewTicker(5 * time.Second)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			return
		case r.tableNameSignal <- tableName:
			return
		}
	}
}

func (r *TableCacheReset) CacheTable(tableName string, key string) {
	r.syncMutex.Lock()
	if _, found := r.TableCache[tableName]; !found {
		r.TableCache[tableName] = map[string]struct{}{}
	}
	r.TableCache[tableName][key] = struct{}{}
	r.syncMutex.Unlock()
}

func (r *TableCacheReset) Start(ctx context.Context) {
	go func() {
		tick := time.NewTicker(5 * time.Minute)
		r.running = true
		defer func() { r.running = false }()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				go func() {
					cache := GetCacheFromContext(ctx)
					for table, cacheKeys := range r.TableCache {
						for key, _ := range cacheKeys {
							if _, err := cache.GetCache(ctx, key); err == ErrCacheMiss {
								r.syncMutex.Lock()
								delete(r.TableCache[table], key)
								r.syncMutex.Unlock()
							}
						}
					}
				}()
			case v, ok := <-r.tableNameSignal:
				if !ok {
					return
				}
				cache := GetCacheFromContext(ctx)
				r.syncMutex.Lock()
				for key, _ := range r.TableCache[v] {
					_ = cache.DeleteKey(ctx, key)
				}
				//todo add to cache so other instances can ref
				delete(r.TableCache[v], v)
				r.syncMutex.Unlock()
			}
		}
	}()

}

func (r *TableCacheReset) IsActive() bool {
	return r.running
}
