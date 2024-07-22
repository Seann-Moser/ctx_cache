package ctx_cache

import (
	"context"
	"github.com/patrickmn/go-cache"
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
	for i := 0; i < b.N; i++ {
		monitor.HasGroupKeyBeenUpdated(ctx, "group1")
	}
}

func benchmarkCacheMonitor_GetGroupKeys(b *testing.B, monitor CacheMonitor) {
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	for i := 0; i < b.N; i++ {
		monitor.GetGroupKeys(ctx, "group1")
	}
}

func benchmarkCacheMonitor_DeleteCache(b *testing.B, monitor CacheMonitor) {
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	for i := 0; i < b.N; i++ {
		monitor.DeleteCache(ctx, "group1")
	}
}

func benchmarkCacheMonitor_UpdateCache(b *testing.B, monitor CacheMonitor) {
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	for i := 0; i < b.N; i++ {
		monitor.UpdateCache(ctx, "group1", "key4")
	}
}

func BenchmarkNewMonitorCacheMonitor(b *testing.B) {
	ctx = ContextWithCache(ctx, NewGoCache(cache.New(5*time.Minute, time.Minute), time.Minute, ""))
	monitor := NewMonitor()
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
