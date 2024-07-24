package ctx_cache

//// Helper function to create a slice of strings for testing
//func generateStringSlice(size int) []string {
//	slice := make([]string, size)
//	for i := 0; i < size; i++ {
//		slice[i] = "string"
//	}
//	return slice
//}
//
//// Benchmark for strings.Join
//func BenchmarkStringsJoin(b *testing.B) {
//	data := generateStringSlice(1000)
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		_ = strings.Join(data, "")
//	}
//}
//
//// Benchmark for += in a loop
//func BenchmarkPlusEqual(b *testing.B) {
//	data := generateStringSlice(1000)
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		var result string
//		for _, str := range data {
//			result += str
//		}
//	}
//}
//
//// Benchmark for bytes.Buffer
//func BenchmarkBytesBuffer(b *testing.B) {
//	data := generateStringSlice(1000)
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		var buffer bytes.Buffer
//		for _, str := range data {
//			buffer.WriteString(str)
//		}
//		_ = buffer.String()
//	}
//}
//
//// Benchmark for strings.Builder
//func BenchmarkStringsBuilder(b *testing.B) {
//	data := generateStringSlice(1000)
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		var builder strings.Builder
//		for _, str := range data {
//			builder.WriteString(str)
//		}
//		_ = builder.String()
//	}
//}
