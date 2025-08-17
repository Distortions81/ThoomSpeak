package main

import "testing"

func BenchmarkHandleDrawState(b *testing.B) {
	msg := make([]byte, 23)
	for i := 0; i < b.N; i++ {
		handleDrawState(msg)
	}
}
