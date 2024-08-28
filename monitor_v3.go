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

type CacheMonitorImpl struct {
	groupMutex *sync.RWMutex
	groups     map[string]*sync.Map

	workers       int
	cacheDuration time.Duration

	localDeleteCacheQueue chan *groupStruct
	deleteGroupQueue      chan string
	addGroupQueue         chan map[string][]string
	localCacheDuaration   time.Duration
}
type groupStruct struct {
	Group   string
	Key     string
	expires time.Time
}

func MonitorV3Flags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet("monitor", pflag.ExitOnError)
	fs.Int("monitor-workers", 0, "")
	fs.Duration("monitor-cache-duration", 10*time.Minute, "")
	return fs
}

func NewMonitorWithFlags() CacheMonitor {
	return &CacheMonitorImpl{
		groupMutex:            &sync.RWMutex{},
		groups:                make(map[string]*sync.Map),
		workers:               viper.GetInt("monitor-workers"),
		cacheDuration:         viper.GetDuration("monitor-cache-duration"),
		localDeleteCacheQueue: make(chan *groupStruct),
		deleteGroupQueue:      make(chan string),
		addGroupQueue:         make(chan map[string][]string),
		localCacheDuaration:   0,
	}
}
func NewMonitor(duration time.Duration) CacheMonitor {
	return &CacheMonitorImpl{
		groupMutex:            &sync.RWMutex{},
		groups:                make(map[string]*sync.Map),
		workers:               10,
		cacheDuration:         duration,
		deleteGroupQueue:      make(chan string),
		addGroupQueue:         make(chan map[string][]string),
		localCacheDuaration:   duration,
		localDeleteCacheQueue: make(chan *groupStruct),
	}
}

func (c *CacheMonitorImpl) AddGroupKeys(ctx context.Context, group string, newKeys ...string) error {
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
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.localDeleteCacheQueue <- &groupStruct{Group: group, Key: newKeys[0], expires: time.Now().Add(c.cacheDuration)}:
		return nil
	}

	//output := map[string]struct{}{}
	//c.groups[group].Range(func(key, value any) bool {
	//	output[key.(string)] = struct{}{}
	//	return true
	//})
	//return SetWithExpiration[map[string]struct{}](ctx, 60*time.Minute, GroupPrefix, group, output)

}

func (c *CacheMonitorImpl) HasGroupKeyBeenUpdated(ctx context.Context, group string) bool {
	//if strings.EqualFold(group, GroupPrefix) || group == "" {
	//	return false
	//}
	return false
}

func (c *CacheMonitorImpl) GetGroupKeys(ctx context.Context, group string) (map[string]struct{}, error) {
	c.groupMutex.RLock()
	g, found := c.groups[group]
	c.groupMutex.RUnlock()
	if !found {
		return map[string]struct{}{}, nil
	}
	output := map[string]struct{}{}
	g.Range(func(key, value any) bool {
		output[key.(string)] = struct{}{}
		return true
	})
	return output, nil
}

func (c *CacheMonitorImpl) DeleteCache(ctx context.Context, group string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.deleteGroupQueue <- group:
		return nil
	}
}

func (c *CacheMonitorImpl) UpdateCache(ctx context.Context, group string, key string) error {
	if strings.EqualFold(group, GroupPrefix) || group == "" {
		return nil
	}
	return c.DeleteCache(ctx, group)
	//c.deleteGroupQueue<-
	//c.addGroupQueue <- map[string][]string{group: {key}}
	//return nil
}

func (c *CacheMonitorImpl) Close() {

}

func (c *CacheMonitorImpl) Start(ctx context.Context) {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case data := <-c.localDeleteCacheQueue:
				if time.Now().Before(data.expires) {
					time.Sleep(data.expires.Sub(time.Now()))
				}
				c.groupMutex.RLock()
				g, found := c.groups[data.Group]
				c.groupMutex.RUnlock()
				if found {
					g.Delete(data.Key)
				}
			}
		}
	})
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
					c.groupMutex.RLock()
					g, found := c.groups[group]
					c.groupMutex.RUnlock()
					if found {
						g.Range(func(key interface{}, value interface{}) bool {
							g.Delete(key)
							return true
						})
					}

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
