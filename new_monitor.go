package ctx_cache

import (
	"context"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"sync"
	"time"
)

type CacheMonitorV2 struct {
	groupMutex *sync.RWMutex
	groups     map[string]uint8

	groupKeys     []GroupKeys
	workers       int
	cacheDuration time.Duration
}

type GroupKeys struct {
	LastUpdateTime time.Time `json:"last_update_time"`
	keys           map[string]struct{}
	mutex          *sync.RWMutex
}

func MonitorV2Flags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet("monitor", pflag.ExitOnError)
	fs.Int("monitor-workers", 10, "")
	fs.Duration("monitor-cache-duration", 10*time.Minute, "")
	return fs
}

func NewMonitorV2WithFlags() CacheMonitor {
	return &CacheMonitorV2{
		groupMutex:    &sync.RWMutex{},
		groups:        make(map[string]uint8),
		groupKeys:     make([]GroupKeys, 0),
		cacheDuration: viper.GetDuration("monitor-cache-duration"),
		workers:       viper.GetInt("monitor-workers"),
	}
}
func NewMonitorV2(duration time.Duration) CacheMonitor {
	return &CacheMonitorV2{
		groupMutex:    &sync.RWMutex{},
		groups:        make(map[string]uint8),
		groupKeys:     make([]GroupKeys, 0),
		cacheDuration: duration,
	}
}

func (c *CacheMonitorV2) AddGroupKeys(ctx context.Context, group string, newKeys ...string) error {
	index := c.SetGroupIndex(group)
	if index < 0 {
		return nil
	}
	c.groupKeys[index].mutex.Lock()
	defer c.groupKeys[index].mutex.Unlock()
	c.groupKeys[index].LastUpdateTime = time.Now()
	for _, key := range newKeys {
		c.groupKeys[index].keys[key] = struct{}{}
	}
	return SetWithExpiration[int64](ctx, c.cacheDuration, GroupPrefix, group, c.groupKeys[index].LastUpdateTime.Unix())
}

func (c *CacheMonitorV2) HasGroupKeyBeenUpdated(ctx context.Context, group string) bool {
	index := c.GetGroupIndex(group)
	if index < 0 {
		return false
	}
	g, err := Get[int64](ctx, GroupPrefix, group)
	if err != nil {
		return false
	}

	c.groupKeys[index].mutex.RLock()
	v := c.groupKeys[index].LastUpdateTime.Before(time.Unix(int64(*g), 0))
	c.groupKeys[index].mutex.RUnlock()
	return v
}

func (c *CacheMonitorV2) GetGroupKeys(ctx context.Context, group string) (map[string]struct{}, error) {
	index := c.GetGroupIndex(group)
	if index < 0 {
		return make(map[string]struct{}), nil
	}
	return c.groupKeys[index].keys, nil
}

func (c *CacheMonitorV2) DeleteCache(ctx context.Context, group string) error {
	index := c.GetGroupIndex(group)
	if index < 0 {
		return nil
	}
	if index > len(c.groupKeys) {
		return nil
	}
	c.groupKeys[index].mutex.Lock()
	c.groupKeys[index].keys = make(map[string]struct{})
	c.groupKeys[index].LastUpdateTime = time.UnixMicro(0)
	wg := sync.WaitGroup{}

	ch := make(chan string)
	go func() {
		for key, _ := range c.groupKeys[index].keys {
			ch <- key
		}
		close(ch)
	}()
	if c.workers <= 0 {
		c.workers = 1
	}
	wg.Add(c.workers)
	for i := 0; i < c.workers; i++ {
		go func() {
			for key := range ch {
				_ = DeleteKey(ctx, key)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	c.groupKeys[index].mutex.Unlock()
	return Delete[int64](ctx, GroupPrefix, group)
}

func (c *CacheMonitorV2) UpdateCache(ctx context.Context, group string, key string) error {
	index := c.SetGroupIndex(group)
	if index < 0 {
		return nil
	}
	c.groupKeys[index].mutex.Lock()
	c.groupKeys[index].LastUpdateTime = time.Now()
	c.groupKeys[index].keys[key] = struct{}{}
	c.groupKeys[index].mutex.Unlock()
	return SetWithExpiration[int64](ctx, c.cacheDuration, GroupPrefix, group, c.groupKeys[index].LastUpdateTime.Unix())
}

func (c *CacheMonitorV2) Close() {

}

func (c *CacheMonitorV2) Start(ctx context.Context) {

}

func (c *CacheMonitorV2) SetGroupIndex(group string) uint8 {
	i := c.GetGroupIndex(group)
	if i >= 0 {
		return uint8(i)
	}
	c.groupMutex.Lock()
	defer c.groupMutex.Unlock()
	c.groupKeys = append(c.groupKeys, GroupKeys{
		LastUpdateTime: time.Now(),
		keys:           map[string]struct{}{},
		mutex:          &sync.RWMutex{},
	})
	c.groups[group] = uint8(len(c.groupKeys) - 1)
	return c.groups[group]
}

func (c *CacheMonitorV2) GetGroupIndex(group string) int {
	c.groupMutex.RLock()
	i, found := c.groups[group]
	c.groupMutex.RUnlock()
	if found {
		return int(i)
	}
	return -1
}

func (c *CacheMonitorV2) Record(ctx context.Context, cmd CacheCmd, status Status) func(err error) {
	return func(err error) {}
}

// ConvertToBytes attempts to convert various primary types to a []byte representation
