package gateway

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"rpc-load-balancer/internal/config"
	"rpc-load-balancer/internal/types"
	"sync"
)

// Gateway manages all endpoints, the selection process, and the HTTP client.
type Gateway struct {
	Endpoints   []*types.RpcEndpoint
	CurrentBest *types.RpcEndpoint
	client      *http.Client
	mutex       sync.RWMutex
	config      *config.Config
}

// NewGateway creates and initializes a new Gateway using the loaded configuration.
func NewGateway(cfg *config.Config) (*Gateway, error) {
	gw := &Gateway{
		client: &http.Client{
			Timeout: cfg.RequestTimeout, // Use timeout from config
		},
		config: cfg, // Store config reference
	}

	for _, endpointStr := range cfg.RpcEndpoints { // Use endpoints from config
		parsedURL, err := url.Parse(endpointStr)
		if err != nil {
			log.Printf("Warning: Skipping invalid endpoint URL %s: %v", endpointStr, err)
			continue
		}
		gw.Endpoints = append(gw.Endpoints, &types.RpcEndpoint{
			URL: parsedURL,
		})
	}

	if len(gw.Endpoints) == 0 {
		return nil, errors.New("no valid RPC endpoints provided in configuration")
	}

	gw.CurrentBest = gw.Endpoints[0]
	log.Printf("Gateway initialized with %d endpoints. Initial best: %s", len(gw.Endpoints), gw.CurrentBest.URL.String())
	return gw, nil
}

// GetBestEndpoint safely retrieves the current best endpoint.
func (gw *Gateway) GetBestEndpoint() *types.RpcEndpoint {
	gw.mutex.RLock()
	defer gw.mutex.RUnlock()
	return gw.CurrentBest
}

// setBestEndpoint safely sets the current best endpoint.
func (gw *Gateway) setBestEndpoint(endpoint *types.RpcEndpoint) {
	gw.mutex.Lock()
	defer gw.mutex.Unlock()
	gw.CurrentBest = endpoint
}
