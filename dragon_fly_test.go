package ctx_cache

import (
	"context"
	"errors"
	redis "github.com/redis/go-redis/v9"
	"strings"
	"testing"
	"time"
)

type CacheTest struct {
	Name  string
	Group string
	Key   string

	Value        string
	Expected     error
	TimeDuration time.Duration
	ShouldExpire bool
}

var tests = []CacheTest{
	{
		Name:         "BasicSetGet",
		Group:        "group1",
		Key:          "key1",
		Value:        "value1",
		Expected:     nil,
		TimeDuration: time.Second * 10,
		ShouldExpire: false,
	},
	{
		Name:         "CacheMiss",
		Group:        "group2",
		Key:          "key2",
		Value:        "",
		Expected:     ErrCacheMiss,
		TimeDuration: time.Second * 10,
		ShouldExpire: false,
	},
	{
		Name:         "CacheExpire",
		Group:        "group3",
		Key:          "key3",
		Value:        "value3",
		Expected:     nil,
		TimeDuration: time.Millisecond * 100, // Short TTL for expiration test
		ShouldExpire: true,
	},
}

func TestCacheDragonFly(t *testing.T) {
	r := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	if nil != r.Ping(ctx).Err() {
		return
	}
	cache := NewRedisCache(r, time.Second*10, "test", true)
	ctx = ContextWithCache(context.Background(), cache)
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Set value if provided
			if test.Value != "" {
				err := Set[string](ctx, test.Group, test.Key, test.Value)
				if !errors.Is(err, test.Expected) {
					t.Fatalf("Set[%q] error = %v, want %v", test.Group, err, test.Expected)
				}
			}

			// Get value and check result
			data, err := Get[string](ctx, test.Group, test.Key)
			if !errors.Is(err, test.Expected) {
				t.Fatalf("Get[%q] error = %v, want %v", test.Group, err, test.Expected)
			}

			// Check the value if there's no cache miss
			if err == nil && !strings.EqualFold(*data, test.Value) {
				t.Errorf("Get[%q] = %q, want %q", test.Group, *data, test.Value)
			}

			// Test expiration if applicable
			if test.ShouldExpire {
				time.Sleep(test.TimeDuration + time.Millisecond*50) // Wait for cache to expire
				_, err = Get[string](ctx, test.Group, test.Key)
				if !errors.Is(err, ErrCacheMiss) {
					t.Errorf("Expected cache miss after expiration, but got: %v", err)
				}
			}
		})
	}
}
