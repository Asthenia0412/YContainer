package sidecar

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Proxy struct {
	target       *url.URL
	reverseProxy *httputil.ReverseProxy
	middlewares  []Middleware
	config       Config
}

type Config struct {
	ProxyPort int
	AppPort   int
	RateLimit RateLimitConfig
	Circuit   CircuitConfig
	Auth      AuthConfig
	Logging   LoggingConfig
}

type Middleware func(http.Handler) http.Handler

func NewProxy(targetURL string, config Config) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("parse target URL: %w", err)
	}

	p := &Proxy{
		target:       target,
		reverseProxy: httputil.NewSingleHostReverseProxy(target),
		config:       config,
	}

	p.reverseProxy.ModifyResponse = p.modifyResponse
	p.reverseProxy.ErrorHandler = p.errorHandler

	return p, nil
}

func (p *Proxy) Handler() http.Handler {
	var handler http.Handler = p.reverseProxy

	for i := len(p.middlewares) - 1; i >= 0; i-- {
		handler = p.middlewares[i](handler)
	}

	return handler
}

func (p *Proxy) Use(m Middleware) {
	p.middlewares = append(p.middlewares, m)
}

func (p *Proxy) Start() error {
	addr := fmt.Sprintf(":%d", p.config.ProxyPort)
	log.Printf("Sidecar proxy listening on %s, forwarding to :%d", addr, p.config.AppPort)
	return http.ListenAndServe(addr, p.Handler())
}

func (p *Proxy) modifyResponse(resp *http.Response) error {
	resp.Header.Set("X-Proxy", "YContainer-Sidecar")
	return nil
}

func (p *Proxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
}

func Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("Recovered from panic: %v", rec)
					http.Error(w, "Internal error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func AccessLog(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lrw, r)
			logger.Printf("%s %s %d %s",
				r.Method, r.URL.Path, lrw.statusCode, time.Since(start))
		})
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}