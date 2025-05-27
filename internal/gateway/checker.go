package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"net/http"
	"rpc-load-balancer/internal/metrics"
	"rpc-load-balancer/internal/types"
	"sort"
	"sync"
	"time"
)

// CheckEndpointStatus performs a health check.
func (gw *Gateway) CheckEndpointStatus(ep *types.RpcEndpoint) {
	ep.Mutex.Lock()
	defer ep.Mutex.Unlock()

	endpointURL := ep.URL.String() // Get URL for labels

	now := time.Now()
	if ep.IsRateLimited && now.Before(ep.RateLimitedUntil) {
		ep.IsReachable = false
		metrics.RpcEndpointIsActive.WithLabelValues(endpointURL).Set(0)
		return
	}
	if ep.IsRateLimited && now.After(ep.RateLimitedUntil) {
		log.Printf("Retrying %s (Backoff Ended)", endpointURL)
		ep.IsRateLimited = false
	}

	startTime := time.Now()
	reqPayload := types.EthBlockNumberRequest{Jsonrpc: "2.0", Method: "eth_blockNumber", Params: []interface{}{}, ID: 1}
	payloadBytes, _ := json.Marshal(reqPayload)
	req, err := http.NewRequest("POST", endpointURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Error creating request for %s: %v", endpointURL, err)
		ep.IsReachable = false
		metrics.RpcCheckErrorsTotal.WithLabelValues(endpointURL, "request_creation").Inc()
		metrics.RpcEndpointIsActive.WithLabelValues(endpointURL).Set(0)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := gw.client.Do(req)
	latency := time.Since(startTime)
	metrics.RpcCheckDuration.WithLabelValues(endpointURL).Observe(latency.Seconds()) // <-- Observe duration

	if err != nil {
		log.Printf("Error checking %s: %v", endpointURL, err)
		ep.IsReachable = false
		metrics.RpcCheckErrorsTotal.WithLabelValues(endpointURL, "http_do").Inc()
		metrics.RpcEndpointIsActive.WithLabelValues(endpointURL).Set(0)
		return
	}
	defer resp.Body.Close()

	ep.Latency = latency
	metrics.RpcEndpointLatency.WithLabelValues(endpointURL).Set(latency.Seconds()) // <-- Set latency gauge

	if resp.StatusCode == http.StatusTooManyRequests {
		log.Printf("🚦 Rate limit detected for %s", endpointURL)
		ep.IsRateLimited = true
		ep.RateLimitedUntil = now.Add(gw.config.RateLimitBackoff)
		ep.IsReachable = false
		metrics.RpcRateLimitsTotal.WithLabelValues(endpointURL, "check").Inc() // <-- Inc rate limit
		metrics.RpcEndpointIsActive.WithLabelValues(endpointURL).Set(0)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP Error %d from %s", resp.StatusCode, endpointURL)
		ep.IsReachable = false
		metrics.RpcCheckErrorsTotal.WithLabelValues(endpointURL, "http_status").Inc()
		metrics.RpcEndpointIsActive.WithLabelValues(endpointURL).Set(0)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response from %s: %v", endpointURL, err)
		ep.IsReachable = false
		metrics.RpcCheckErrorsTotal.WithLabelValues(endpointURL, "read_body").Inc()
		metrics.RpcEndpointIsActive.WithLabelValues(endpointURL).Set(0)
		return
	}

	var rpcResp types.EthBlockNumberResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		log.Printf("Error parsing JSON from %s: %v", endpointURL, err)
		ep.IsReachable = false
		metrics.RpcCheckErrorsTotal.WithLabelValues(endpointURL, "json_parse").Inc()
		metrics.RpcEndpointIsActive.WithLabelValues(endpointURL).Set(0)
		return
	}

	if rpcResp.Error != nil {
		log.Printf("RPC Error from %s: %s (%d)", endpointURL, rpcResp.Error.Message, rpcResp.Error.Code)
		ep.IsReachable = false
		metrics.RpcCheckErrorsTotal.WithLabelValues(endpointURL, "rpc_error").Inc()
		metrics.RpcEndpointIsActive.WithLabelValues(endpointURL).Set(0)
		return
	}

	blockNumBig := new(big.Int)
	_, success := blockNumBig.SetString(rpcResp.Result, 0)
	if !success {
		log.Printf("Error parsing block number '%s' from %s", rpcResp.Result, endpointURL)
		ep.IsReachable = false
		metrics.RpcCheckErrorsTotal.WithLabelValues(endpointURL, "block_parse").Inc()
		metrics.RpcEndpointIsActive.WithLabelValues(endpointURL).Set(0)
		return
	}

	ep.BlockNumber = blockNumBig.Int64()
	ep.IsReachable = true
	metrics.RpcEndpointBlockNumber.WithLabelValues(endpointURL).Set(float64(ep.BlockNumber)) // <-- Set block gauge
	metrics.RpcEndpointIsActive.WithLabelValues(endpointURL).Set(1)                          // <-- Set active gauge
}

// SelectBestEndpoint uses gw.config.BlockTolerance.
func (gw *Gateway) SelectBestEndpoint() {
	log.Println("\n🔍 Checking for the best RPC endpoint...")
	var wg sync.WaitGroup

	for _, ep := range gw.Endpoints {
		wg.Add(1)
		go func(endpoint *types.RpcEndpoint) {
			defer wg.Done()
			gw.CheckEndpointStatus(endpoint)
		}(ep)
	}
	wg.Wait()

	var candidates []*types.RpcEndpoint
	var highestBlock int64 = -1

	for _, ep := range gw.Endpoints {
		ep.Mutex.RLock()
		if ep.IsReachable && !ep.IsRateLimited {
			candidates = append(candidates, ep)
			if ep.BlockNumber > highestBlock {
				highestBlock = ep.BlockNumber
			}
		}
		ep.Mutex.RUnlock()
	}

	if len(candidates) == 0 {
		log.Println("⚠️ No reachable, non-rate-limited endpoints found. Keeping current best.")
		for _, ep := range gw.Endpoints {
			metrics.RpcEndpointIsCurrentBest.WithLabelValues(ep.URL.String()).Set(metrics.RpcEndpointCurrentBestNotActive)
			// Check verbose before logging
			if gw.config.Verbose {
				log.Printf("📊 METRIC: RpcEndpointIsCurrentBest{endpoint=\"%s\"} set to %v (No candidates)", ep.URL.String(), metrics.RpcEndpointCurrentBestNotActive)
			}
		}
		return
	}

	blockThreshold := highestBlock - gw.config.BlockTolerance // Use config
	log.Printf("📈 Highest block found: %d. Threshold: >= %d", highestBlock, blockThreshold)

	var finalCandidates []*types.RpcEndpoint
	for _, ep := range candidates {
		ep.Mutex.RLock()
		if ep.BlockNumber >= blockThreshold {
			finalCandidates = append(finalCandidates, ep)
		}
		ep.Mutex.RUnlock()
	}

	if len(finalCandidates) == 0 {
		log.Println("🟡 No endpoints within block tolerance. Considering all reachable.")
		finalCandidates = candidates
	}

	sort.Slice(finalCandidates, func(i, j int) bool {
		finalCandidates[i].Mutex.RLock()
		finalCandidates[j].Mutex.RLock()
		defer finalCandidates[i].Mutex.RUnlock()
		defer finalCandidates[j].Mutex.RUnlock()
		return finalCandidates[i].Latency < finalCandidates[j].Latency
	})

	best := finalCandidates[0]
	best.Mutex.RLock()
	currentBestURL := gw.GetBestEndpoint().URL.String()
	bestURL := best.URL.String()
	bestBlock := best.BlockNumber
	bestLatency := best.Latency
	best.Mutex.RUnlock()

	if currentBestURL != bestURL {
		log.Printf("✅ New best endpoint: %s (Block: %d, Latency: %v)", bestURL, bestBlock, bestLatency)
		gw.setBestEndpoint(best)
		// Update metrics: Set old best to 0, new best to 1
		metrics.RpcEndpointIsCurrentBest.WithLabelValues(currentBestURL).Set(metrics.RpcEndpointCurrentBestNotActive)
		if gw.config.Verbose { // <-- Check verbose
			log.Printf("📊 METRIC: RpcEndpointIsCurrentBest{endpoint=\"%s\"} set to %v", currentBestURL, metrics.RpcEndpointCurrentBestNotActive)
		}

		metrics.RpcEndpointIsCurrentBest.WithLabelValues(bestURL).Set(metrics.RpcEndpointCurrentBestActive)
		if gw.config.Verbose { // <-- Check verbose
			log.Printf("📊 METRIC: RpcEndpointIsCurrentBest{endpoint=\"%s\"} set to %v", bestURL, metrics.RpcEndpointCurrentBestActive)
		}
	} else {
		log.Printf("👍 Best endpoint remains: %s (Block: %d, Latency: %v)", bestURL, bestBlock, bestLatency)
		// Ensure it's set to 1
		metrics.RpcEndpointIsCurrentBest.WithLabelValues(bestURL).Set(metrics.RpcEndpointCurrentBestActive)
		if gw.config.Verbose { // <-- Check verbose
			log.Printf("📊 METRIC: RpcEndpointIsCurrentBest{endpoint=\"%s\"} set to %v (reaffirmed)", bestURL, metrics.RpcEndpointCurrentBestActive)
		}
	}

	// Ensure all *other* endpoints are set to 0
	for _, ep := range gw.Endpoints {
		epURL := ep.URL.String()
		if epURL != bestURL {
			metrics.RpcEndpointIsCurrentBest.WithLabelValues(epURL).Set(metrics.RpcEndpointCurrentBestNotActive)
			if gw.config.Verbose { // <-- Check verbose
				log.Printf("📊 METRIC: RpcEndpointIsCurrentBest{endpoint=\"%s\"} set to %v (not best)", epURL, metrics.RpcEndpointCurrentBestNotActive)
			}
		}
	}

}

// StartChecker uses gw.config.CheckInterval.
func (gw *Gateway) StartChecker(ctx context.Context) {
	gw.SelectBestEndpoint()
	ticker := time.NewTicker(gw.config.CheckInterval) // Use config

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				gw.SelectBestEndpoint()
			case <-ctx.Done():
				log.Println("Checker goroutine stopping.")
				return
			}
		}
	}()
	log.Printf("Periodic endpoint checker started (Interval: %v).", gw.config.CheckInterval)
}
