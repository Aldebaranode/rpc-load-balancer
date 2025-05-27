package gateway

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
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

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		ip := getRequestIP(r)
		// Log the request details including IP
		log.Printf("📥 [%s] Received request: %s %s", ip, r.Method, r.URL.String())

		lrw := NewLoggingResponseWriter(w)

		proxy.ServeHTTP(w, r)

		duration := time.Since(startTime)

		// Log the completion details including status and duration
		log.Printf("📤 [%s] <-- %s %s - Status %d (%v)", ip, r.Method, r.URL.String(), lrw.statusCode, duration)
	})
}

func getRequestIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // Fallback
	}
	return ip
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewLoggingResponseWriter creates a new loggingResponseWriter.
func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	// Default status code is 200 (OK) if WriteHeader is never called.
	return &loggingResponseWriter{w, http.StatusOK}
}

// WriteHeader captures the status code before calling the original WriteHeader.
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Write calls the original Write but ensures WriteHeader(200) is called
// if it hasn't been called yet (Go's default behavior).
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	// If WriteHeader has not been called, Write will call WriteHeader(http.StatusOK)
	// We don't need to explicitly capture it here as WriteHeader handles it.
	return lrw.ResponseWriter.Write(b)
}
