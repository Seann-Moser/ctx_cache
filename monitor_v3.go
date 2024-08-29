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

type CacheMonitorImpl struct {
	groupMutex *sync.RWMutex
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
		groupMutex:       &sync.RWMutex{},
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
		groupMutex:       &sync.RWMutex{},
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
	if strings.EqualFold(group, GroupPrefix) || group == "" {
		return nil
	}
	//c.groupMutex.RLock()
	data, found := c.localCache.Get(group)
	//c.groupMutex.RUnlock()

	if !found {
		v := make(map[string]struct{}, len(newKeys))
		c.groupMutex.Lock()
		for _, newKey := range newKeys {
			v[newKey] = struct{}{}
		}
		c.groupMutex.Unlock()
		c.localCache.Set(group, v, cache.DefaultExpiration)

		return nil
	}
	c.groupMutex.Lock()
	d := data.(map[string]struct{})
	for _, newKey := range newKeys {
		d[newKey] = struct{}{}
	}
	c.localCache.Set(group, d, cache.DefaultExpiration)
	c.groupMutex.Unlock()
	return nil
}

func (c *CacheMonitorImpl) HasGroupKeyBeenUpdated(ctx context.Context, group string) bool {
	//if strings.EqualFold(group, GroupPrefix) || group == "" {
	//	return false
	//}
	return false
}

func (c *CacheMonitorImpl) GetGroupKeys(ctx context.Context, group string) (map[string]struct{}, error) {
	if data, found := c.localCache.Get(group); found {
		return data.(map[string]struct{}), nil
	}
	return map[string]struct{}{}, nil
}

func (c *CacheMonitorImpl) DeleteCache(ctx context.Context, group string) error {
	if v, f := c.localCache.Get(group); !f || len(v.(map[string]struct{})) == 0 {
		return nil
	}
	if !c.started.Load() {
		return nil
	}
	if !c.useChans {
		keys, err := c.GetGroupKeys(ctx, group)
		if err != nil {
			return err
		}
		c.groupMutex.Lock()
		for key := range keys {
			err = DeleteKey(ctx, key)
			if err != nil {
				continue
			}
		}
		c.localCache.Set(group, map[string]struct{}{}, cache.DefaultExpiration)
		c.groupMutex.Unlock()
		return nil
	}

	select {
	case c.deleteGroupQueue <- group:
		// Successfully added to the channel
		return nil
	default:
		keys, err := c.GetGroupKeys(ctx, group)
		if err != nil {
			return err
		}
		c.groupMutex.Lock()
		for key := range keys {
			err = DeleteKey(ctx, key)
			if err != nil {
				continue
			}
		}
		c.localCache.Set(group, map[string]struct{}{}, cache.DefaultExpiration)
		c.groupMutex.Unlock()
	}
	return nil
}

func (c *CacheMonitorImpl) UpdateCache(ctx context.Context, group string, key string) error {
	if strings.EqualFold(group, GroupPrefix) || group == "" {
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

func (c *CacheMonitorImpl) Close() {

}

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
					keys, err := c.GetGroupKeys(ctx, group)
					if err != nil {
						ctxLogger.Error(ctx, "failed getting group keys")
						continue
					}
					c.groupMutex.Lock()
					for key := range keys {
						err = DeleteKey(ctx, key)
						if err != nil {
							ctxLogger.Info(ctx, "failed getting deleting key", zap.String("group", group), zap.String("key", key))
							continue
						}
					}
					c.localCache.Set(group, map[string]struct{}{}, cache.DefaultExpiration)
					c.groupMutex.Unlock()

				}
			}
		})
	}
	_ = eg.Wait()
}

func (c *CacheMonitorImpl) Record(ctx context.Context, cmd CacheCmd, status Status) func(err error) {
	return func(err error) {

	}
}
