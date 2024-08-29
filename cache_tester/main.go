package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"github.com/Seann-Moser/ctx_cache"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"math/rand/v2"
	"strconv"
	"time"
)

func main() {
	var watchers []*CacheKeyWatcher
	ctx := context.Background()
	testDuration := 5 * time.Minute
	workers := 10
	logger, err := ctxLogger.NewLogger(true, "debug", true)
	if err != nil {
		panic(err)
	}
	ctx = ctxLogger.ConfigureCtx(logger, ctx)
	//rdb := redis.NewClient(&redis.Options{
	//	Addr:     "localhost:6379", // Redis server address
	//	Password: "",               // no password set
	//	DB:       0,                // use default DB
	//})
	//pong, err := rdb.Ping().Result()
	//if err != nil {
	//	fmt.Println("Error connecting to Redis:", err)
	//	return
	//}

	//redisCache := ctx_cache.NewRedisCache(rdb, testDuration, "", true)
	//ctx = ctx_cache.ContextWithCache(ctx, redisCache)
	//fmt.Println("Connected to Redis:", pong)

	for i := 0; i < workers; i++ {
		watchers = append(watchers, &CacheKeyWatcher{
			Group:    "test",
			Key:      "test-key",
			Interval: time.Duration(randRange(10, 1500)) * time.Millisecond,
			Setter: func() string {
				h := sha1.New()
				h.Write([]byte(strconv.FormatInt(int64(time.Now().Nanosecond()), 10)))
				return hex.EncodeToString(h.Sum(nil))
			},
			reader: i%2 == 0,
		})
	}
	ctx, cancel := context.WithTimeout(ctx, testDuration)
	defer cancel()
	for _, w := range watchers {
		w.Start(ctx)
		w.StartWatcher(ctx)
	}
	<-ctx.Done()
	println("DONE")
}

func randRange(min, max int) int {
	return rand.IntN(max-min) + min
}

type CacheKeyWatcher struct {
	Group    string
	Key      string
	Interval time.Duration
	Setter   func() string
	reader   bool
}

func (c *CacheKeyWatcher) Updated(ctx context.Context) bool {
	_, err := ctx_cache.Get[string](ctx, c.Group, c.Key)
	if errors.Is(ctx_cache.ErrCacheUpdated, err) {
		if ctx_cache.GlobalCacheMonitor.HasGroupKeyBeenUpdated(ctx, c.Group) {
			ctxLogger.Info(ctx, "Updating Group Key")
		}

		return true
	}
	return false
}

func (c *CacheKeyWatcher) Start(ctx context.Context) {
	if c.reader {
		return
	}
	go func() {
		ticker := time.NewTicker(c.Interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := ctx_cache.Set[string](ctx, c.Group, c.Key, c.Setter())
				if err != nil {
					println(err)
				}
			}
		}

	}()
}

func (c *CacheKeyWatcher) StartWatcher(ctx context.Context) {
	if !c.reader {
		return
	}
	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.Updated(ctx)

			}
		}

	}()
}
