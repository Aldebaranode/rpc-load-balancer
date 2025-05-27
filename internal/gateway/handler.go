package gateway

import (
	"log"
	"net/http"
	"net/http/httputil"
	"rpc-load-balancer/internal/metrics"
	"rpc-load-balancer/internal/utils"
	"strconv"
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
			endpointURL := best.URL.String()
			log.Printf("🚦 Rate limit detected during forward to %s", endpointURL)

			best.Mutex.Lock()
			best.IsRateLimited = true
			best.RateLimitedUntil = time.Now().Add(gw.config.RateLimitBackoff)
			best.Mutex.Unlock()

			metrics.RpcRateLimitsTotal.WithLabelValues(endpointURL, "proxy").Inc() // <-- Inc rate limit

			go gw.SelectBestEndpoint()
		}
		return nil
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("❌ Proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	proxyHandler := &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: modifyResponse,
		ErrorHandler:   errorHandler,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		ip := utils.GetRequestIP(r)
		lrw := utils.NewLoggingResponseWriter(w)

		// Get endpoint *before* proxying (best guess)
		currentEndpoint := gw.GetBestEndpoint().URL.String()

		log.Printf("📥 [%s] --> %s %s (to %s)", ip, r.Method, r.URL.String(), currentEndpoint)

		proxyHandler.ServeHTTP(lrw, r) // Use our proxy

		duration := time.Since(startTime)
		statusCodeStr := strconv.Itoa(lrw.StatusCode)

		// Update Prometheus Metrics
		metrics.HttpRequestDuration.WithLabelValues(r.Method, statusCodeStr, currentEndpoint).Observe(duration.Seconds())
		metrics.HttpRequestTotal.WithLabelValues(r.Method, statusCodeStr, currentEndpoint).Inc()

		log.Printf("📤 [%s] <-- %s %s - Status %d (%v)", ip, r.Method, r.URL.String(), lrw.StatusCode, duration)
	})
}
