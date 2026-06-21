//go:build ignore

// Локальный нагрузочный тест: go run loadtest.go
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type authResp struct {
	Token string `json:"token"`
}

type result struct {
	status     int
	duration   time.Duration
	instanceID string
	cache      string
	err        error
}

func main() {
	baseURL := flag.String("url", "http://localhost:8090", "base URL (nginx or gateway task API)")
	userURL := flag.String("user", "http://localhost:8081", "user-service URL")
	username := flag.String("user-name", "loadtest_user", "test username")
	password := flag.String("password", "LoadTest123!", "test password")
	concurrency := flag.Int("c", 20, "concurrent workers")
	requests := flag.Int("n", 500, "total requests")
	warmup := flag.Int("warmup", 50, "warmup requests (cache priming)")
	useCookie := flag.Bool("cookie", false, "send JWT as cookie (for gateway HTML)")
	flag.Parse()

	cookieMode := *useCookie

	token := ensureToken(*userURL, *username, *password)
	fmt.Printf("JWT obtained (%d chars)\n", len(token))

	if *warmup > 0 {
		fmt.Printf("\n=== Warmup (%d requests) ===\n", *warmup)
		runLoad(*baseURL+"/tasks", token, *warmup, min(5, *concurrency), "warmup", cookieMode)
	}

	fmt.Printf("\n=== Load test: %d requests, %d workers ===\n", *requests, *concurrency)
	stats := runLoad(*baseURL+"/tasks", token, *requests, *concurrency, "load", cookieMode)

	fmt.Printf("\n=== Results ===\n")
	printStats(stats)
}

func ensureToken(userURL, username, password string) string {
	regBody, _ := json.Marshal(map[string]string{
		"username": username,
		"email":    username + "@loadtest.local",
		"password": password,
	})
	loginBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	resp, err := http.Post(userURL+"/api/register", "application/json", bytes.NewReader(regBody))
	if err != nil {
		fatalf("register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		resp.Body.Close()
		resp, err = http.Post(userURL+"/api/login", "application/json", bytes.NewReader(loginBody))
		if err != nil {
			fatalf("login: %v", err)
		}
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		fatalf("auth failed %s: %s", resp.Status, string(b))
	}
	var ar authResp
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		fatalf("decode token: %v", err)
	}
	if ar.Token == "" {
		fatalf("empty token")
	}
	return ar.Token
}

func runLoad(url, token string, total, workers int, label string, useCookie bool) []result {
	jobs := make(chan int, total)
	for i := 0; i < total; i++ {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	results := make([]result, 0, total)
	var mu sync.Mutex
	var completed int64

	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				r := doRequest(client, url, token, useCookie)
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
				n := atomic.AddInt64(&completed, 1)
				if n%max(1, int64(total/10)) == 0 || n == int64(total) {
					fmt.Printf("[%s] %d/%d done\n", label, n, total)
				}
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("Elapsed: %s, RPS: %.1f\n", elapsed.Round(time.Millisecond), float64(total)/elapsed.Seconds())
	return results
}

func doRequest(client *http.Client, url, token string, useCookie bool) result {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if useCookie {
		req.AddCookie(&http.Cookie{Name: "token", Value: token})
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	start := time.Now()
	resp, err := client.Do(req)
	r := result{duration: time.Since(start), err: err}
	if err != nil {
		return r
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	r.status = resp.StatusCode
	r.instanceID = resp.Header.Get("X-Instance-ID")
	r.cache = resp.Header.Get("X-Cache")
	return r
}

func printStats(results []result) {
	var ok, fail int
	durations := make([]time.Duration, 0, len(results))
	instances := map[string]int{}
	cacheHits, cacheMiss, cacheOther := 0, 0, 0

	for _, r := range results {
		if r.err != nil {
			fail++
			continue
		}
		if r.status >= 200 && r.status < 300 {
			ok++
			durations = append(durations, r.duration)
			if r.instanceID != "" {
				instances[r.instanceID]++
			}
			switch strings.ToUpper(r.cache) {
			case "HIT":
				cacheHits++
			case "MISS":
				cacheMiss++
			default:
				cacheOther++
			}
		} else {
			fail++
		}
	}

	fmt.Printf("Success: %d, Failed: %d\n", ok, fail)
	if len(durations) == 0 {
		return
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	fmt.Printf("Latency ms — min: %d, p50: %d, p95: %d, p99: %d, max: %d\n",
		durations[0].Milliseconds(),
		percentile(durations, 50).Milliseconds(),
		percentile(durations, 95).Milliseconds(),
		percentile(durations, 99).Milliseconds(),
		durations[len(durations)-1].Milliseconds(),
	)
	if len(instances) > 0 {
		fmt.Println("X-Instance-ID distribution:")
		for k, v := range instances {
			fmt.Printf("  %s: %d (%.1f%%)\n", k, v, 100*float64(v)/float64(ok))
		}
	}
	if cacheHits+cacheMiss+cacheOther > 0 {
		fmt.Printf("X-Cache — HIT: %d (%.1f%%), MISS: %d (%.1f%%), other: %d\n",
			cacheHits, 100*float64(cacheHits)/float64(ok),
			cacheMiss, 100*float64(cacheMiss)/float64(ok),
			cacheOther,
		)
	}
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := (len(sorted)-1)*p/100
	return sorted[idx]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
