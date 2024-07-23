package ctx_cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
)

type cacheTestCase struct {
	Name           string
	Cache          Cache
	Key            string
	Group          string
	Value          string
	ExpectedOutput string
	ExpectedErr    error
}

func TestTieredCache(t *testing.T) {
	testCases := []cacheTestCase{
		{
			Name:           "go cache",
			Cache:          NewGoCache(cache.New(time.Minute, time.Minute), time.Minute, ""),
			Key:            "test_cache",
			Value:          "test",
			ExpectedOutput: "test",
			ExpectedErr:    nil,
		},
		{
			Name:           "go cache",
			Cache:          NewGoCache(cache.New(time.Minute, time.Minute), time.Minute, ""),
			Key:            "test_cache_fail",
			Value:          "",
			ExpectedOutput: "",
			ExpectedErr:    ErrCacheMiss,
		},
		{
			Name:           "go cache tiered",
			Cache:          NewTieredCache(nil, NewGoCache(cache.New(time.Minute, time.Minute), time.Minute, "")),
			Key:            "test_cache_fail",
			Value:          "",
			ExpectedOutput: "",
			ExpectedErr:    ErrCacheMiss,
		},
		{
			Name:  "go cache tiered",
			Cache: NewTieredCache(nil, NewGoCache(cache.New(time.Minute, time.Minute), time.Minute, "")),
			Key:   "test_cache",
			Value: "test",

			ExpectedOutput: "test",
		},
	}
	GlobalCacheMonitor = NewMonitor()
	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Value != "" {
				err := tc.Cache.SetCache(ctx, tc.Group, tc.Key, tc.Value)
				if err != nil {
					t.Errorf("failed setting cache:%s", err.Error())
					return
				}
			}
			value, err := tc.Cache.GetCache(ctx, tc.Group, tc.Key)
			if err != nil && !errors.Is(err, tc.ExpectedErr) {
				t.Errorf("failed getting cache:%s", err.Error())
				return
			}
			if tc.ExpectedErr != nil {
				return
			}
			if string(value) != tc.ExpectedOutput {
				t.Errorf("does not match expected output: %s != %s", tc.ExpectedOutput, string(value))
			}

			err = Set[string](ctx, tc.Group, tc.Key, tc.Value)
			if err != nil {
				t.Errorf("failed setting cache:%s", err.Error())
				return
			}

		})
	}

}

func TestGet(t *testing.T) {
	GlobalCacheMonitor = NewMonitorV2(time.Minute)
	c := NewGoCache(cache.New(5*time.Minute, time.Minute), time.Minute, "test")
	ctx = ContextWithCache(ctx, c)
	key := "test_key"
	group := "test_group"

	// Test case: Group key has been updated
	//
	_ = Set[int](ctx, group, key, 10)
	_ = SetWithExpiration[int64](ctx, time.Minute, GroupPrefix, group, time.Now().Add(time.Second*5).Unix())
	_, err := Get[int](ctx, group, key)
	if !errors.Is(err, ErrCacheUpdated) {
		t.Fatalf("expected ErrCacheUpdated, got %v", err)
	}
	_ = Set[int](ctx, group, key, 10)

	// Test case: Cache miss
	_, err = Get[int](ctx, group, key+"2")
	if err == nil || err.Error() != "cache missed" {
		t.Fatalf("expected cache miss, got %v", err)
	}

	expectedInt := 123
	_ = Set[int](ctx, group, key, expectedInt)
	result, err := Get[int](ctx, group, key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if *result != expectedInt {
		t.Fatalf("expected %v, got %v", expectedInt, *result)
	}
	// Test case: Struct retrieval
	expectedStruct := Wrapper[int]{Data: 456}
	_ = Set[Wrapper[int]](ctx, group, key, expectedStruct)
	resultStruct, err := Get[Wrapper[int]](ctx, group, key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resultStruct.Data != expectedStruct.Data {
		t.Fatalf("expected %v, got %v", expectedStruct.Data, *resultStruct)
	}
}
