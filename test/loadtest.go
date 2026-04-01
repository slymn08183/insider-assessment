package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Event struct {
	EventName  string   `json:"event_name"`
	Channel    string   `json:"channel"`
	CampaignID string   `json:"campaign_id"`
	UserID     string   `json:"user_id"`
	Timestamp  int64    `json:"timestamp"`
	Tags       []string `json:"tags"`
}

func main() {
	total := flag.Int("n", 20000, "total requests")
	concurrency := flag.Int("c", 50, "concurrent workers")
	url := flag.String("url", "http://127.0.0.1:8080/events", "target URL")
	flag.Parse()

	fmt.Printf("Load test: %d requests, %d concurrent\n", *total, *concurrency)
	fmt.Printf("Target: %s\n\n", *url)

	var (
		success int64
		fail    int64
		dupes   int64
		wg      sync.WaitGroup
		mu      sync.Mutex
		start   = time.Now()
	)

	// Response surelerini topla
	latencies := make([]time.Duration, 0, *total)

	// Work channel
	work := make(chan int, *total)
	for i := 0; i < *total; i++ {
		work <- i
	}
	close(work)

	// Tek transport, tum worker'lar paylasir
	transport := &http.Transport{
		MaxIdleConns:        *concurrency,
		MaxIdleConnsPerHost: *concurrency,
		IdleConnTimeout:     30 * time.Second,
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	// Worker'lari baslat
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for id := range work {
				event := Event{
					EventName:  "product_view",
					Channel:    "web",
					CampaignID: fmt.Sprintf("cmp_%d", rand.Intn(100)),
					UserID:     fmt.Sprintf("user_%d", id),
					Timestamp:  time.Now().Unix() - int64(rand.Intn(86400)),
					Tags:       []string{"electronics", "test"},
				}

				body, _ := json.Marshal(event)

				reqStart := time.Now()
				resp, err := client.Post(*url, "application/json", bytes.NewReader(body))
				latency := time.Since(reqStart)

				if err != nil {
					atomic.AddInt64(&fail, 1)
					continue
				}

				respBody := make([]byte, 512)
				resp.Body.Read(respBody)
				resp.Body.Close()

				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()

				switch resp.StatusCode {
				case http.StatusAccepted:
					atomic.AddInt64(&success, 1)
				case http.StatusConflict:
					atomic.AddInt64(&dupes, 1)
				default:
					atomic.AddInt64(&fail, 1)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Latency hesapla
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	var avg time.Duration
	if len(latencies) > 0 {
		var total time.Duration
		for _, l := range latencies {
			total += l
		}
		avg = total / time.Duration(len(latencies))
	}

	fmt.Println("=== Results ===")
	fmt.Printf("Total:      %d\n", *total)
	fmt.Printf("Success:    %d\n", success)
	fmt.Printf("Duplicates: %d\n", dupes)
	fmt.Printf("Failed:     %d\n", fail)
	fmt.Printf("Duration:   %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("RPS:        %.0f req/sec\n", float64(*total)/elapsed.Seconds())

	if len(latencies) > 0 {
		p50 := latencies[len(latencies)*50/100]
		p95 := latencies[len(latencies)*95/100]
		p99 := latencies[len(latencies)*99/100]

		fmt.Println("\n=== Latency ===")
		fmt.Printf("Min:        %s\n", latencies[0].Round(time.Microsecond))
		fmt.Printf("Max:        %s\n", latencies[len(latencies)-1].Round(time.Microsecond))
		fmt.Printf("Avg:        %s\n", avg.Round(time.Microsecond))
		fmt.Printf("P50:        %s\n", p50.Round(time.Microsecond))
		fmt.Printf("P95:        %s\n", p95.Round(time.Microsecond))
		fmt.Printf("P99:        %s\n", p99.Round(time.Microsecond))
	}
}
