package ctx_cache

import (
	"fmt"
	"testing"
)

// GetTypeSprintf returns the type of a generic value using fmt.Sprintf("%T", val)
func GetTypeSprintf[T any](val T) string {
	return fmt.Sprintf("%T", val)
}

// Benchmark for GetTypeReflect
func BenchmarkGetTypeReflect(b *testing.B) {
	b.Run("int", func(b *testing.B) {
		var val int = 42
		for i := 0; i < b.N; i++ {
			GetTypeReflect(val)
		}
	})
	b.Run("float64", func(b *testing.B) {
		var val float64 = 3.14
		for i := 0; i < b.N; i++ {
			GetTypeReflect(val)
		}
	})
	b.Run("string", func(b *testing.B) {
		var val string = "hello"
		for i := 0; i < b.N; i++ {
			GetTypeReflect(val)
		}
	})
	b.Run("bool", func(b *testing.B) {
		var val bool = true
		for i := 0; i < b.N; i++ {
			GetTypeReflect(val)
		}
	})
}

// Benchmark for GetTypeSprintf
func BenchmarkGetTypeSprintf(b *testing.B) {
	b.Run("int", func(b *testing.B) {
		var val int = 42
		for i := 0; i < b.N; i++ {
			GetTypeSprintf(val)
		}
	})
	b.Run("float64", func(b *testing.B) {
		var val float64 = 3.14
		for i := 0; i < b.N; i++ {
			GetTypeSprintf(val)
		}
	})
	b.Run("string", func(b *testing.B) {
		var val string = "hello"
		for i := 0; i < b.N; i++ {
			GetTypeSprintf(val)
		}
	})
	b.Run("bool", func(b *testing.B) {
		var val bool = true
		for i := 0; i < b.N; i++ {
			GetTypeSprintf(val)
		}
	})
}
