package sidecar

import (
	"log"
	"net/http"
	"os"
	"time"
)

type MiddlewareChain struct {
	middlewares []Middleware
}

func NewChain(middlewares ...Middleware) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: middlewares,
	}
}

func (c *MiddlewareChain) Then(handler http.Handler) http.Handler {
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

func (c *MiddlewareChain) ThenFunc(handlerFunc http.HandlerFunc) http.Handler {
	return c.Then(handlerFunc)
}

func BuildDefaultChain(config Config) *MiddlewareChain {
	accessLogger := NewAccessLogger(LoggingConfig{
		Enabled:   config.Logging.Enabled,
		LogDir:    config.Logging.LogDir,
		AccessLog: true,
	})

	accessLogMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lrw, r)

			accessLogger.Log(AccessLogRecord{
				Timestamp:  time.Now(),
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: lrw.statusCode,
				Latency:    time.Since(start).String(),
				ClientIP:   r.RemoteAddr,
				UserAgent:  r.UserAgent(),
			})
		})
	}

	var middlewares []Middleware

	_ = log.New(os.Stdout, "[SIDECAR] ", log.LstdFlags)

	middlewares = append(middlewares, Recovery())

	if config.Auth.Enabled {
		auth := NewAuthenticator(config.Auth)
		middlewares = append(middlewares, Auth(auth))
	}

	if config.RateLimit.Enabled {
		limiter := NewRateLimiter(config.RateLimit)
		middlewares = append(middlewares, RateLimit(limiter))
	}

	if config.Circuit.Enabled {
		cb := NewCircuitBreaker(config.Circuit)
		middlewares = append(middlewares, CircuitBreakerMiddleware(cb))
	}

	middlewares = append(middlewares, accessLogMiddleware)

	return NewChain(middlewares...)
}