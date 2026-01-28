package benchmarks

import (
	"strings"
	"testing"
)

// BenchmarkStringConcat benchmarks string concatenation using the + operator
func BenchmarkStringConcat(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = "hello" + " " + "world"
	}
}

// BenchmarkStringBuilder benchmarks string concatenation using strings.Builder
func BenchmarkStringBuilder(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.WriteString("hello")
		sb.WriteString(" ")
		sb.WriteString("world")
		_ = sb.String()
	}
}

// BenchmarkStringJoin benchmarks string concatenation using strings.Join
func BenchmarkStringJoin(b *testing.B) {
	b.ReportAllocs()
	parts := []string{"hello", "world"}
	for i := 0; i < b.N; i++ {
		_ = strings.Join(parts, " ")
	}
}
