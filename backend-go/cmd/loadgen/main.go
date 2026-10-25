// Command loadgen 是后端压测/容量基线用的小工具（仅 stdlib）：
// 固定并发持续打一个 HTTP 端点，输出 RPS、状态码/业务码分布与延迟分位数。
//
// 用法示例（dev 环境请对 APP_ENV=test 的 backend-test 实例压测，避免消耗
// 生产限流桶；localhost:8082 为 docker compose test profile 映射端口）：
//
//	go run ./cmd/loadgen -url http://localhost:8082/healthz -c 50 -d 10s
//	go run ./cmd/loadgen -url http://localhost:8082/public/v1/system/health \
//	  -method POST -body '{}' -c 50 -d 15s
//	go run ./cmd/loadgen -url http://localhost:8082/api/v1/users/account \
//	  -method POST -body '{}' -H 'Authorization: Bearer <token>' -c 50 -d 20s
//
// 输出中的 codes 为响应 envelope 里的业务 code（-1 表示响应体非法或非
// envelope）。压测机与目标同机会互相抢 CPU，结果用于相对比较而非生产容量
// 绝对值。
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var codeRe = regexp.MustCompile(`"code"\s*:\s*(\d+)`)

type sample struct {
	status int
	code   int
	lat    time.Duration
	err    string
}

func main() {
	url := flag.String("url", "", "request URL（必填）")
	method := flag.String("method", "GET", "HTTP method")
	body := flag.String("body", "", "request body（非空时设置 Content-Type: application/json）")
	concurrency := flag.Int("c", 50, "并发 worker 数")
	duration := flag.Duration("d", 20*time.Second, "压测时长")
	headers := flag.String("H", "", "额外请求头，多个用分号分隔，如 k:v;k:v")
	timeout := flag.Duration("timeout", 10*time.Second, "单请求超时")
	idemUnique := flag.Bool("idem-unique", false, "为每个请求生成唯一 Idempotency-Key（写压测需开启）")
	bodySeq := flag.Bool("body-seq", false, "把 body 中的 {{seq}} 替换为本次请求唯一序号（用于唯一昵称/内容）")
	tokenFile := flag.String("token-file", "", "每行一个 Bearer token；请求间轮询使用（写压测多用户）")
	flag.Parse()
	if *url == "" {
		fmt.Fprintln(os.Stderr, "missing -url")
		os.Exit(2)
	}
	extra := map[string]string{}
	for _, h := range strings.Split(*headers, ";") {
		if h == "" {
			continue
		}
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			extra[http.CanonicalHeaderKey(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
		}
	}
	var tokens []string
	if *tokenFile != "" {
		blob, err := os.ReadFile(*tokenFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read token file: %v\n", err)
			os.Exit(2)
		}
		for _, line := range strings.Split(string(blob), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				tokens = append(tokens, line)
			}
		}
		if len(tokens) == 0 {
			fmt.Fprintln(os.Stderr, "token file has no tokens")
			os.Exit(2)
		}
	}
	var seqAtomic int64
	tr := &http.Transport{
		MaxIdleConns:        *concurrency * 2,
		MaxIdleConnsPerHost: *concurrency * 2,
		MaxConnsPerHost:     *concurrency * 2,
		IdleConnTimeout:     60 * time.Second,
		DisableCompression:  true,
	}
	client := &http.Client{Transport: tr, Timeout: *timeout}
	var mu sync.Mutex
	results := make([]sample, 0, 1<<18)
	var wg sync.WaitGroup
	deadline := time.Now().Add(*duration)
	start := time.Now()
	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var local []sample
			for time.Now().Before(deadline) {
				seq := atomic.AddInt64(&seqAtomic, 1)
				reqBody := *body
				if *bodySeq {
					reqBody = strings.ReplaceAll(reqBody, "{{seq}}", strconv.FormatInt(seq, 10))
				}
				req, err := http.NewRequest(*method, *url, bytes.NewReader([]byte(reqBody)))
				if err != nil {
					local = append(local, sample{err: err.Error()})
					continue
				}
				if *body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				for k, v := range extra {
					req.Header.Set(k, v)
				}
				if *idemUnique {
					req.Header.Set("Idempotency-Key", fmt.Sprintf("lt-%032x", time.Now().UnixNano()^seq))
				}
				if len(tokens) > 0 {
					req.Header.Set("Authorization", "Bearer "+tokens[int(seq)%len(tokens)])
				}
				t0 := time.Now()
				resp, err := client.Do(req)
				lat := time.Since(t0)
				if err != nil {
					local = append(local, sample{lat: lat, err: err.Error()})
					continue
				}
				blob, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
				_ = resp.Body.Close()
				code := -1
				if m := codeRe.FindSubmatch(blob); m != nil {
					fmt.Sscanf(string(m[1]), "%d", &code)
				}
				local = append(local, sample{status: resp.StatusCode, code: code, lat: lat})
			}
			mu.Lock()
			results = append(results, local...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	lats := make([]time.Duration, 0, len(results))
	statuses := map[string]int{}
	codes := map[string]int{}
	errs := map[string]int{}
	for _, r := range results {
		if r.err != "" {
			errs[r.err]++
			continue
		}
		lats = append(lats, r.lat)
		statuses[fmt.Sprint(r.status)]++
		codes[fmt.Sprint(r.code)]++
	}
	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })
	pct := func(p float64) time.Duration {
		if len(lats) == 0 {
			return 0
		}
		return lats[int(float64(len(lats)-1)*p)]
	}
	fmt.Printf("url=%s method=%s c=%d dur=%.1fs reqs=%d rps=%.0f\n", *url, *method, *concurrency, elapsed.Seconds(), len(results), float64(len(results))/elapsed.Seconds())
	fmt.Printf("latency_ms p50=%.2f p90=%.2f p95=%.2f p99=%.2f max=%.2f\n", pct(0.50).Seconds()*1000, pct(0.90).Seconds()*1000, pct(0.95).Seconds()*1000, pct(0.99).Seconds()*1000, pct(1).Seconds()*1000)
	fmt.Printf("statuses=%v codes=%v errors=%v\n", statuses, codes, errs)
}
