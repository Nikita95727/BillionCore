//go:build ignore

package main

import (
	"fmt"
	"runtime"
	"testing"
	"tether-bin-go/internal/models"
	"tether-bin-go/internal/store"
)

// scratch_benchmark.go — manual benchmark for BIN store write throughput.
// Run with: go run scratch_benchmark.go
func main() {
	sizes := []int{1000, 10000, 100000, 1000000}
	fmt.Println("Data Volume (Rules) | Avg Swap Speed (ns)")
	fmt.Println("-------------------|-----------------------")

	for _, size := range sizes {
		s := store.NewBinStore()

		// Build initial data set.
		rules := make(map[string]map[string]models.BinRule, size)
		for i := 0; i < size; i++ {
			bin := fmt.Sprintf("%06d", i)
			rules[bin] = map[string]models.BinRule{"US": {Action: "ENABLE"}}
		}
		s.Swap(rules, nil)

		// Benchmark atomic Swap speed.
		result := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s.Swap(rules, nil)
			}
		})

		fmt.Printf("%18d | %18.2f\n", size, float64(result.NsPerOp()))
		runtime.GC()
	}
}
