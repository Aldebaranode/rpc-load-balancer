package gateway

import (
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

// ProxyHandler creates the reverse proxy handler.
// It now uses gw.config.RateLimitBackoff when flagging.
func (gw *Gateway) ProxyHandler() http.Handler {

	director := func(req *http.Request) {
		best := gw.GetBestEndpoint()
		targetURL := best.URL

		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = targetURL.Path
		req.Host = targetURL.Host

		log.Printf("  -> Forwarding %s %s to %s", req.Method, req.URL.Path, targetURL.String())
	}

	modifyResponse := func(resp *http.Response) error {
		if resp.StatusCode == http.StatusTooManyRequests {
			best := gw.GetBestEndpoint()
			log.Printf("🚦 Rate limit detected during forward to %s", best.URL.String())

			best.Mutex.Lock()
			best.IsRateLimited = true
			best.RateLimitedUntil = time.Now().Add(gw.config.RateLimitBackoff) // Use config
			best.Mutex.Unlock()

			go gw.SelectBestEndpoint()
		}
		return nil
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("❌ Proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	proxy := &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: modifyResponse,
		ErrorHandler:   errorHandler,
	}
	return proxy
}
