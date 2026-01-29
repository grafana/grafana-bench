package tests

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
