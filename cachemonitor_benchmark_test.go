package ctx_cache

import (
	"context"
	"errors"
	"github.com/patrickmn/go-cache"
	"sync"
	"testing"
	"time"
)

var ctx = context.Background()

func benchmarkCacheMonitor_AddGroupKeys(b *testing.B, monitor CacheMonitor) {
	for i := 0; i < b.N; i++ {
		monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	}
}

func benchmarkCacheMonitor_HasGroupKeyBeenUpdated(b *testing.B, monitor CacheMonitor) {
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.HasGroupKeyBeenUpdated(ctx, "group1")
	}
}

func benchmarkCacheMonitor_GetGroupKeys(b *testing.B, monitor CacheMonitor) {
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.GetGroupKeys(ctx, "group1")
	}
}

func benchmarkCacheMonitor_DeleteCache(b *testing.B, monitor CacheMonitor) {
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.DeleteCache(ctx, "group1")
	}
}

func benchmarkCacheMonitor_UpdateCache(b *testing.B, monitor CacheMonitor) {
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.UpdateCache(ctx, "group1", "key4")
	}
}

//func BenchmarkNewMonitorCacheMonitor(b *testing.B) {
//	ctx = ContextWithCache(ctx, NewGoCache(cache.New(5*time.Minute, time.Minute), time.Minute, ""))
//	monitor := NewMonitor()
//	b.Run("AddGroupKeys", func(b *testing.B) {
//		benchmarkCacheMonitor_AddGroupKeys(b, monitor)
//	})
//	b.Run("HasGroupKeyBeenUpdated", func(b *testing.B) {
//		benchmarkCacheMonitor_HasGroupKeyBeenUpdated(b, monitor)
//	})
//	b.Run("GetGroupKeys", func(b *testing.B) {
//		benchmarkCacheMonitor_GetGroupKeys(b, monitor)
//	})
//	b.Run("DeleteCache", func(b *testing.B) {
//		benchmarkCacheMonitor_DeleteCache(b, monitor)
//	})
//	b.Run("UpdateCache", func(b *testing.B) {
//		benchmarkCacheMonitor_UpdateCache(b, monitor)
//	})
//}

func BenchmarkNewMonitorV2Monitor(b *testing.B) {
	ctx = ContextWithCache(ctx, NewGoCache(cache.New(5*time.Minute, time.Minute), time.Minute, ""))
	monitor := NewMonitorV2(5 * time.Minute)
	b.Run("AddGroupKeys", func(b *testing.B) {
		benchmarkCacheMonitor_AddGroupKeys(b, monitor)
	})
	b.Run("HasGroupKeyBeenUpdated", func(b *testing.B) {
		benchmarkCacheMonitor_HasGroupKeyBeenUpdated(b, monitor)
	})
	b.Run("GetGroupKeys", func(b *testing.B) {
		benchmarkCacheMonitor_GetGroupKeys(b, monitor)
	})
	b.Run("DeleteCache", func(b *testing.B) {
		benchmarkCacheMonitor_DeleteCache(b, monitor)
	})
	b.Run("UpdateCache", func(b *testing.B) {
		benchmarkCacheMonitor_UpdateCache(b, monitor)
	})
}

func BenchmarkCheckPrimaryType(b *testing.B) {
	b.Run("int", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CheckPrimaryType(42)
		}
	})
	b.Run("float64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CheckPrimaryType(3.14)
		}
	})
	b.Run("string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CheckPrimaryType("hello")
		}
	})
	b.Run("bool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CheckPrimaryType(true)
		}
	})
	b.Run("slice", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CheckPrimaryType([]int{1, 2, 3})
		}
	})
	b.Run("struct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CheckPrimaryType(struct{}{})
		}
	})
}

// Benchmark tests for ConvertBytesToType function
func BenchmarkConvertBytesToType(b *testing.B) {
	intData := []byte("123")
	floatData := []byte("3.14")
	boolData := []byte("true")
	stringData := []byte("hello")

	b.Run("int", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ConvertBytesToType[int](intData)
		}
	})

	b.Run("float64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ConvertBytesToType[float64](floatData)
		}
	})

	b.Run("bool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ConvertBytesToType[bool](boolData)
		}
	})

	b.Run("string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ConvertBytesToType[string](stringData)
		}
	})
}

func benchmarkGet(b *testing.B, group string, key string, cacheSize int) {
	c := NewGoCache(cache.New(5*time.Minute, time.Minute), time.Minute, "test")
	ctx = ContextWithCache(context.Background(), c)

	// Populate cache
	for i := 0; i < cacheSize; i++ {
		_ = Set[int](ctx, group, key+string(rune(i)), i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Get[int](ctx, group, key+string(rune(i%cacheSize)))
		if err != nil && err.Error() != "cache missed" && !errors.Is(err, ErrCacheUpdated) {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkGet_SmallCacheV2(b *testing.B) {
	GlobalCacheMonitor = NewMonitorV2(time.Minute)
	benchmarkGet(b, "test_group", "test_key", 100)
}

func BenchmarkGet_MediumCacheV2(b *testing.B) {
	GlobalCacheMonitor = NewMonitorV2(time.Minute)
	benchmarkGet(b, "test_group", "test_key", 1000)
}

func BenchmarkGet_LargeCacheV2(b *testing.B) {
	GlobalCacheMonitor = NewMonitorV2(time.Minute)
	benchmarkGet(b, "test_group", "test_key", 10000)
}

func BenchmarkGet_SmallCache(b *testing.B) {
	benchmarkGet(b, "test_group", "test_key", 100)
}

func BenchmarkGet_MediumCache(b *testing.B) {
	benchmarkGet(b, "test_group", "test_key", 1000)
}

func BenchmarkGet_LargeCache(b *testing.B) {
	benchmarkGet(b, "test_group", "test_key", 10000)
}

// BenchmarkSyncMap benchmarks sync.Map with concurrent access
func BenchmarkSyncMap(b *testing.B) {
	var m sync.Map

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Store("key", 42)
			m.Load("key")
			m.Delete("key")
		}
	})
}

// MutexMap is a map protected by a sync.Mutex
type MutexMap struct {
	mu sync.RWMutex
	m  map[string]int
}

func (mm *MutexMap) Store(key string, value int) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.m[key] = value
}

func (mm *MutexMap) Load(key string) (int, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	value, ok := mm.m[key]
	return value, ok
}

func (mm *MutexMap) Delete(key string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	delete(mm.m, key)
}

// BenchmarkMutexMap benchmarks map protected by sync.Mutex with concurrent access
func BenchmarkMutexMap(b *testing.B) {
	mm := &MutexMap{m: make(map[string]int)}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mm.Store("key", 42)
			mm.Load("key")
			mm.Delete("key")
		}
	})
}
