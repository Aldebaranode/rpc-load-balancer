package utils

import (
	"net"
	"net/http"
	"strings"
)

func GetRequestIP(r *http.Request) string {
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
	StatusCode int
}

// NewLoggingResponseWriter creates a new loggingResponseWriter.
func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	// Default status code is 200 (OK) if WriteHeader is never called.
	return &loggingResponseWriter{w, http.StatusOK}
}

// WriteHeader captures the status code before calling the original WriteHeader.
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.StatusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Write calls the original Write but ensures WriteHeader(200) is called
// if it hasn't been called yet (Go's default behavior).
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	// If WriteHeader has not been called, Write will call WriteHeader(http.StatusOK)
	// We don't need to explicitly capture it here as WriteHeader handles it.
	return lrw.ResponseWriter.Write(b)
}
