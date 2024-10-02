package ctx_cache

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type GroupData struct {
	mu   sync.Mutex `json:"-"`
	keys map[string]struct{}
}

type CacheMonitorImpl struct {
	localCache *cache.Cache

	workers       int
	cacheDuration time.Duration

	deleteGroupQueue chan string
	addGroupQueue    chan map[string]string
	started          atomic.Bool
	useChans         bool
}

func MonitorV3Flags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet("monitor", pflag.ExitOnError)
	fs.Int("monitor-workers", 100, "")
	fs.Int("monitor-queue-size", 10000, "")
	fs.Duration("monitor-cache-duration", 10*time.Minute, "")
	fs.Bool("monitor-use-chan", false, "")
	return fs
}

func NewMonitorWithFlags() CacheMonitor {
	return &CacheMonitorImpl{
		workers:          viper.GetInt("monitor-workers"),
		cacheDuration:    viper.GetDuration("monitor-cache-duration"),
		deleteGroupQueue: make(chan string, viper.GetInt("monitor-queue-size")),
		addGroupQueue:    make(chan map[string]string, viper.GetInt("monitor-queue-size")),
		started:          atomic.Bool{},
		localCache:       cache.New(viper.GetDuration("monitor-cache-duration"), time.Minute),
		useChans:         viper.GetBool("monitor-use-chan"),
	}
}

func NewMonitor(duration time.Duration, useChans bool) CacheMonitor {
	return &CacheMonitorImpl{
		workers:          100,
		localCache:       cache.New(duration, time.Minute),
		cacheDuration:    duration,
		deleteGroupQueue: make(chan string, 100000),
		addGroupQueue:    make(chan map[string]string, 100000),
		started:          atomic.Bool{},
		useChans:         useChans,
	}
}

func (c *CacheMonitorImpl) AddGroupKeys(ctx context.Context, group string, newKeys ...string) error {
	if strings.EqualFold(group, GroupPrefix) || group == "" || (len(newKeys) == 1 && newKeys[0] == group) {
		return nil
	}

	dataInterface, found := c.localCache.Get(group)
	if !found {
		data := &GroupData{
			keys: make(map[string]struct{}, len(newKeys)),
		}
		data.mu.Lock()
		for _, newKey := range newKeys {
			data.keys[newKey] = struct{}{}
		}
		// Create a copy of keys under the lock
		copyKeys := make(map[string]struct{}, len(data.keys))
		for k := range data.keys {
			copyKeys[k] = struct{}{}
		}
		data.mu.Unlock()

		c.localCache.Set(group, data, cache.DefaultExpiration)
		_ = Set[map[string]struct{}](ctx, group, group, copyKeys)
		return nil
	}

	data := dataInterface.(*GroupData)
	data.mu.Lock()
	for _, newKey := range newKeys {
		data.keys[newKey] = struct{}{}
	}
	// Create a copy of keys under the lock
	copyKeys := make(map[string]struct{}, len(data.keys))
	for k := range data.keys {
		copyKeys[k] = struct{}{}
	}
	data.mu.Unlock()
	_ = Set[map[string]struct{}](ctx, group, group, copyKeys)
	return nil
}

func (c *CacheMonitorImpl) HasGroupKeyBeenUpdated(ctx context.Context, group string) bool {
	return false
}

func (c *CacheMonitorImpl) GetGroupKeys(ctx context.Context, group string) (map[string]struct{}, error) {
	if dataInterface, found := c.localCache.Get(group); found {
		data := dataInterface.(*GroupData)
		data.mu.Lock()
		defer data.mu.Unlock()
		keysCopy := make(map[string]struct{}, len(data.keys))
		for k := range data.keys {
			keysCopy[k] = struct{}{}
		}
		return keysCopy, nil
	}
	return map[string]struct{}{}, nil
}

func (c *CacheMonitorImpl) DeleteCache(ctx context.Context, group string) error {
	dataInterface, found := c.localCache.Get(group)
	if !found {
		return nil
	}
	if !c.started.Load() {
		return nil
	}
	data := dataInterface.(*GroupData)
	if !c.useChans {
		data.mu.Lock()
		// Create a copy of keys under the lock
		keysCopy := make(map[string]struct{}, len(data.keys))
		for key := range data.keys {
			keysCopy[key] = struct{}{}
		}
		// Clear the original keys
		data.keys = make(map[string]struct{})
		data.mu.Unlock()

		c.localCache.Set(group, data, cache.DefaultExpiration)
		for key := range keysCopy {
			err := DeleteKey(ctx, key)
			if err != nil {
				continue
			}
		}

		dataFromGlobalCache, _ := Get[map[string]struct{}](ctx, group, group)
		if dataFromGlobalCache != nil {
			for key := range *dataFromGlobalCache {
				err := DeleteKey(ctx, key)
				if err != nil {
					continue
				}
			}
		}
		return nil
	}

	select {
	case c.deleteGroupQueue <- group:
		return nil
	default:
		data.mu.Lock()
		// Create a copy of keys under the lock
		keysCopy := make(map[string]struct{}, len(data.keys))
		for key := range data.keys {
			keysCopy[key] = struct{}{}
		}
		// Clear the original keys
		data.keys = make(map[string]struct{})
		data.mu.Unlock()

		c.localCache.Set(group, data, cache.DefaultExpiration)
		for key := range keysCopy {
			err := DeleteKey(ctx, key)
			if err != nil {
				continue
			}
		}

		dataFromGlobalCache, _ := Get[map[string]struct{}](ctx, group, group)
		if dataFromGlobalCache != nil {
			for key := range *dataFromGlobalCache {
				err := DeleteKey(ctx, key)
				if err != nil {
					continue
				}
			}
		}
	}
	return nil
}

func (c *CacheMonitorImpl) UpdateCache(ctx context.Context, group string, key string) error {
	if strings.EqualFold(group, GroupPrefix) || group == "" || group == key {
		return nil
	}
	if !c.useChans {
		if err := c.AddGroupKeys(ctx, group, key); err != nil {
			return fmt.Errorf("failed adding to cache: %w", err)
		} else {
			return nil
		}
	}
	if !c.started.Load() {
		return nil
	}
	select {
	case c.addGroupQueue <- map[string]string{group: key}:
		return nil
	default:
		if err := c.AddGroupKeys(ctx, group, key); err != nil {
			return fmt.Errorf("failed adding to cache: %w", err)
		} else {
			return nil
		}
	}
}

func (c *CacheMonitorImpl) Close() {}

func (c *CacheMonitorImpl) Start(ctx context.Context) {
	if c.started.Load() {
		return
	}
	eg, ctx := errgroup.WithContext(ctx)
	c.started.Store(true)
	for i := 0; i < c.workers; i++ {
		eg.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case groupKeys := <-c.addGroupQueue:
					for key, value := range groupKeys {
						_ = c.AddGroupKeys(ctx, key, value)
					}
				case group := <-c.deleteGroupQueue:
					dataInterface, found := c.localCache.Get(group)
					if !found {
						continue
					}
					data := dataInterface.(*GroupData)
					data.mu.Lock()
					// Create a copy of keys under the lock
					keysCopy := make(map[string]struct{}, len(data.keys))
					for key := range data.keys {
						keysCopy[key] = struct{}{}
					}
					// Clear the original keys
					data.keys = make(map[string]struct{})
					data.mu.Unlock()
					c.localCache.Set(group, data, cache.DefaultExpiration)

					for key := range keysCopy {
						err := DeleteKey(ctx, key)
						if err != nil {
							ctxLogger.Info(ctx, "failed deleting key", zap.String("group", group), zap.String("key", key))
							continue
						}
					}

					dataFromGlobalCache, _ := Get[map[string]struct{}](ctx, group, group)
					if dataFromGlobalCache != nil {
						for key := range *dataFromGlobalCache {
							err := DeleteKey(ctx, key)
							if err != nil {
								continue
							}
						}
					}
				}
			}
		})
	}
	_ = eg.Wait()
}

func (c *CacheMonitorImpl) Record(ctx context.Context, cmd CacheCmd, status Status) func(err error) {
	return func(err error) {}
}
