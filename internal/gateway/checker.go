package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"net/http"
	"rpc-load-balancer/internal/types"
	"sort"
	"sync"
	"time"
)

// CheckEndpointStatus performs a health check.
func (gw *Gateway) CheckEndpointStatus(ep *types.RpcEndpoint) {
	ep.Mutex.Lock()
	defer ep.Mutex.Unlock()

	now := time.Now()
	if ep.IsRateLimited && now.Before(ep.RateLimitedUntil) {
		ep.IsReachable = false
		return
	}
	if ep.IsRateLimited && now.After(ep.RateLimitedUntil) {
		log.Printf("Retrying %s (Backoff Ended)", ep.URL.String())
		ep.IsRateLimited = false
	}

	startTime := time.Now()
	reqPayload := types.EthBlockNumberRequest{Jsonrpc: "2.0", Method: "eth_blockNumber", Params: []interface{}{}, ID: 1}
	payloadBytes, _ := json.Marshal(reqPayload)
	req, err := http.NewRequest("POST", ep.URL.String(), bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Error creating request for %s: %v", ep.URL.String(), err)
		ep.IsReachable = false
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := gw.client.Do(req) // gw.client already uses config timeout
	latency := time.Since(startTime)

	if err != nil {
		log.Printf("Error checking %s: %v", ep.URL.String(), err)
		ep.IsReachable = false
		return
	}
	defer resp.Body.Close()

	ep.Latency = latency

	if resp.StatusCode == http.StatusTooManyRequests {
		log.Printf("🚦 Rate limit detected for %s", ep.URL.String())
		ep.IsRateLimited = true
		ep.RateLimitedUntil = now.Add(gw.config.RateLimitBackoff) // Use config
		ep.IsReachable = false
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP Error %d from %s", resp.StatusCode, ep.URL.String())
		ep.IsReachable = false
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response from %s: %v", ep.URL.String(), err)
		ep.IsReachable = false
		return
	}

	var rpcResp types.EthBlockNumberResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		log.Printf("Error parsing JSON from %s: %v", ep.URL.String(), err)
		ep.IsReachable = false
		return
	}

	if rpcResp.Error != nil {
		log.Printf("RPC Error from %s: %s (%d)", ep.URL.String(), rpcResp.Error.Message, rpcResp.Error.Code)
		ep.IsReachable = false
		return
	}

	blockNumBig := new(big.Int)
	_, success := blockNumBig.SetString(rpcResp.Result, 0)
	if !success {
		log.Printf("Error parsing block number '%s' from %s", rpcResp.Result, ep.URL.String())
		ep.IsReachable = false
		return
	}

	ep.BlockNumber = blockNumBig.Int64()
	ep.IsReachable = true
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
	currentURL := gw.GetBestEndpoint().URL.String()
	bestURL := best.URL.String()
	bestBlock := best.BlockNumber
	bestLatency := best.Latency
	best.Mutex.RUnlock()

	if currentURL != bestURL {
		log.Printf("✅ New best endpoint: %s (Block: %d, Latency: %v)", bestURL, bestBlock, bestLatency)
		gw.setBestEndpoint(best)
	} else {
		log.Printf("👍 Best endpoint remains: %s (Block: %d, Latency: %v)", bestURL, bestBlock, bestLatency)
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
