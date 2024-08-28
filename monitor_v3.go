package ctx_cache

import (
	"context"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"strings"
	"sync"
	"time"
)

type CacheMonitorV3 struct {
	groupMutex *sync.RWMutex
	groups     map[string]*sync.Map

	workers       int
	cacheDuration time.Duration

	deleteGroupQueue    chan string
	addGroupQueue       chan map[string][]string
	localCacheDuaration time.Duration
	localCache          sync.Map
}

func MonitorV3Flags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet("monitorv3", pflag.ExitOnError)
	fs.Int("monitor-workers", 20, "")
	fs.Duration("monitor-cache-duration", 10*time.Minute, "")
	return fs
}

func NewMonitorV3WithFlags() CacheMonitor {
	return &CacheMonitorV3{
		groupMutex:    &sync.RWMutex{},
		cacheDuration: viper.GetDuration("monitor-cache-duration"),
		workers:       viper.GetInt("monitor-workers"),
	}
}
func NewMonitorV3(duration time.Duration) CacheMonitor {
	return &CacheMonitorV2{
		groupMutex:    &sync.RWMutex{},
		groups:        make(map[string]uint8),
		groupKeys:     make([]GroupKeys, 0),
		cacheDuration: duration,
	}
}

func (c *CacheMonitorV3) AddGroupKeys(ctx context.Context, group string, newKeys ...string) error {
	if strings.EqualFold(group, GroupPrefix) || group == "" {
		return nil
	}
	c.groupMutex.RLock()
	if _, found := c.groups[group]; !found {
		c.groupMutex.RUnlock()
		c.groupMutex.Lock()
		c.groups[group] = &sync.Map{}
		c.groupMutex.Unlock()
	}
	c.groupMutex.RUnlock()

	for key := range newKeys {
		c.groups[group].Store(key, struct{}{})

	}
	//output := map[string]struct{}{}
	//c.groups[group].Range(func(key, value any) bool {
	//	output[key.(string)] = struct{}{}
	//	return true
	//})
	//return SetWithExpiration[map[string]struct{}](ctx, 60*time.Minute, GroupPrefix, group, output)
	return nil
}

func (c *CacheMonitorV3) HasGroupKeyBeenUpdated(ctx context.Context, group string) bool {
	if strings.EqualFold(group, GroupPrefix) || group == "" {
		return false
	}
	return false
}

func (c *CacheMonitorV3) GetGroupKeys(ctx context.Context, group string) (map[string]struct{}, error) {
	c.groupMutex.RLock()
	g, found := c.groups[group]
	c.groupMutex.RUnlock()
	if !found {
		return map[string]struct{}{}, nil
	}
	output := map[string]struct{}{}
	c.groups[group].Range(func(key, value any) bool {
		output[key.(string)] = struct{}{}
		return true
	})
	return output, nil
}

func (c *CacheMonitorV3) DeleteCache(ctx context.Context, group string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.deleteGroupQueue <- group:
		return nil
	}
}

func (c *CacheMonitorV3) UpdateCache(ctx context.Context, group string, key string) error {
	if strings.EqualFold(group, GroupPrefix) || group == "" {
		return nil
	}
	c.addGroupQueue <- map[string][]string{group: {key}}
	return nil
}

func (c *CacheMonitorV3) Close() {

}

func (c *CacheMonitorV3) Start(ctx context.Context) {
	eg, ctx := errgroup.WithContext(ctx)
	for i := 0; i < c.workers; i++ {
		eg.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case groupKeys := <-c.addGroupQueue:
					for key, value := range groupKeys {
						_ = c.AddGroupKeys(ctx, key, value...)
					}
				case group := <-c.deleteGroupQueue:
					keys, err := c.GetGroupKeys(ctx, group)
					if err != nil {
						ctxLogger.Error(ctx, "failed getting group keys")
						continue
					}
					for key := range keys {
						err = DeleteKey(ctx, key)
						if err != nil {
							ctxLogger.Info(ctx, "failed getting deleting key", zap.String("group", group), zap.String("key", key))
							continue
						}
					}

				}
			}
		})
	}
	_ = eg.Wait()
}

func (c *CacheMonitorV3) Record(ctx context.Context, cmd CacheCmd, status Status) func(err error) {
	return func(err error) {

	}
}
