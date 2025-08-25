package scriptbench

import (
	"testing"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func nativeSum() int {
	total := 0
	for i := 0; i < 1000; i++ {
		total += i
	}
	return total
}

func BenchmarkNativeSum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = nativeSum()
	}
}

func BenchmarkInterpretedSum(b *testing.B) {
	const src = `
package main

func Sum() int {
    total := 0
    for i := 0; i < 1000; i++ {
        total += i
    }
    return total
}
`
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	if _, err := i.Eval(src); err != nil {
		b.Fatalf("eval: %v", err)
	}
	v, err := i.Eval("main.Sum")
	if err != nil {
		b.Fatalf("lookup: %v", err)
	}
	sum := v.Interface().(func() int)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sum()
	}
}
