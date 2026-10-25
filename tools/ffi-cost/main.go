package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

func main() {
	call, release, err := load()
	if err != nil {
		panic(err)
	}
	defer release()
	for iteration := 0; iteration < 100; iteration++ {
		call()
	}
	if os.Getenv("VIVA_FFI_HOLD") != "" {
		time.Sleep(30 * time.Second)
		return
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	started := time.Now()
	const iterations = 1000000
	for iteration := 0; iteration < iterations; iteration++ {
		call()
	}
	elapsed := time.Since(started)
	runtime.ReadMemStats(&after)
	fmt.Printf("ns_per_call=%.2f bytes_per_call=%.2f allocations_per_call=%.2f goroutines=%d\n", float64(elapsed.Nanoseconds())/iterations, float64(after.TotalAlloc-before.TotalAlloc)/iterations, float64(after.Mallocs-before.Mallocs)/iterations, runtime.NumGoroutine())
}
