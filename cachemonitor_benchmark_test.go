package ctx_cache

import (
	"context"
	"testing"
	"time"
)

func benchmarkCacheMonitor_AddGroupKeys(b *testing.B, monitor CacheMonitor) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	}
}

func benchmarkCacheMonitor_HasGroupKeyBeenUpdated(b *testing.B, monitor CacheMonitor) {
	ctx := context.Background()
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	for i := 0; i < b.N; i++ {
		monitor.HasGroupKeyBeenUpdated(ctx, "group1")
	}
}

func benchmarkCacheMonitor_GetGroupKeys(b *testing.B, monitor CacheMonitor) {
	ctx := context.Background()
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	for i := 0; i < b.N; i++ {
		monitor.GetGroupKeys(ctx, "group1")
	}
}

func benchmarkCacheMonitor_DeleteCache(b *testing.B, monitor CacheMonitor) {
	ctx := context.Background()
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	for i := 0; i < b.N; i++ {
		monitor.DeleteCache(ctx, "group1")
	}
}

func benchmarkCacheMonitor_UpdateCache(b *testing.B, monitor CacheMonitor) {
	ctx := context.Background()
	monitor.AddGroupKeys(ctx, "group1", "key1", "key2", "key3")
	for i := 0; i < b.N; i++ {
		monitor.UpdateCache(ctx, "group1", "key4")
	}
}

func BenchmarkNewMonitorCacheMonitor(b *testing.B) {
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
