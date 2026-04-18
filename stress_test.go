package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	url := "http://188.166.93.241:8083/api/bin/lookup?bin=679835&country=ES"
	apiKey := "TETHER_ROCKET_2026_SECRET"
	concurrency := 50
	requestsPerRoutine := 1000

	fmt.Printf("🚀 Starting Brutal Stress Test: %d routines, %d requests each...\n", concurrency, requestsPerRoutine)

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{}
			for j := 0; j < requestsPerRoutine; j++ {
				req, _ := http.NewRequest("GET", url, nil)
				req.Header.Set("X-API-Key", apiKey)
				resp, err := client.Do(req)
				if err != nil {
					continue
				}
				resp.Body.Close()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)
	totalRequests := concurrency * requestsPerRoutine

	fmt.Printf("\n--- RESULTS ---\n")
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Total Time:     %v\n", duration)
	fmt.Printf("Avg Latency:    %v\n", duration/time.Duration(totalRequests))
	fmt.Printf("Requests/sec:   %.2f\n", float64(totalRequests)/duration.Seconds())
}
