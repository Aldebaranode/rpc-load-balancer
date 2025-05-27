package types

import (
	"net/url"
	"sync"
	"time"
)

// RpcEndpoint holds the state and details of a single upstream RPC node.
type RpcEndpoint struct {
	URL              *url.URL
	BlockNumber      int64
	Latency          time.Duration
	IsRateLimited    bool
	RateLimitedUntil time.Time
	IsReachable      bool
	Mutex            sync.RWMutex
}

// EthBlockNumberRequest defines the JSON structure for the eth_blockNumber request.
type EthBlockNumberRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	ID      int    `json:"id"`
}

// EthBlockNumberResponse defines the JSON structure for the eth_blockNumber response.
type EthBlockNumberResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  string `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	ID int `json:"id"`
}
