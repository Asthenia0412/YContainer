package sidecar

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type CircuitState int32

const (
	StateClosed   CircuitState = 0
	StateOpen     CircuitState = 1
	StateHalfOpen CircuitState = 2
)

type CircuitBreaker struct {
	state           CircuitState
	failureCount    int64
	successCount    int64
	totalRequests   int64
	lastStateChange time.Time
	config          CircuitConfig
	mu              sync.RWMutex
}

type CircuitConfig struct {
	Enabled      bool
	MaxRequests  int
	Timeout      time.Duration
	SleepWindow  time.Duration
	ErrorPercent float64
}

func NewCircuitBreaker(config CircuitConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:           StateClosed,
		lastStateChange: time.Now(),
		config:          config,
	}
}

func (cb *CircuitBreaker) getState() CircuitState {
	return CircuitState(atomic.LoadInt32((*int32)(&cb.state)))
}

func (cb *CircuitBreaker) setState(state CircuitState) {
	atomic.StoreInt32((*int32)(&cb.state), int32(state))
	cb.lastStateChange = time.Now()
}

func (cb *CircuitBreaker) Execute(req func() (*http.Response, error)) (*http.Response, error) {
	if !cb.config.Enabled {
		return req()
	}

	state := cb.getState()

	switch state {
	case StateOpen:
		if time.Since(cb.lastStateChange) > cb.config.SleepWindow {
			cb.setState(StateHalfOpen)
		} else {
			return nil, ErrCircuitOpen
		}
	case StateHalfOpen:
		atomic.AddInt64(&cb.totalRequests, 1)
		if atomic.LoadInt64(&cb.totalRequests) > int64(cb.config.MaxRequests) {
			return nil, ErrCircuitOpen
		}
	}

	resp, err := req()
	if err != nil {
		atomic.AddInt64(&cb.failureCount, 1)
		cb.checkThreshold()
		return nil, err
	}

	atomic.AddInt64(&cb.successCount, 1)

	if state == StateHalfOpen {
		cb.setState(StateClosed)
		atomic.StoreInt64(&cb.failureCount, 0)
		atomic.StoreInt64(&cb.totalRequests, 0)
	}

	return resp, nil
}

func (cb *CircuitBreaker) checkThreshold() {
	failures := atomic.LoadInt64(&cb.failureCount)
	successes := atomic.LoadInt64(&cb.successCount)
	total := failures + successes

	if total > 0 && float64(failures)/float64(total)*100 >= cb.config.ErrorPercent {
		cb.setState(StateOpen)
	}
}

var ErrCircuitOpen = &circuitError{"circuit breaker is open"}

type circuitError struct {
	msg string
}

func (e *circuitError) Error() string { return e.msg }

func CircuitBreakerMiddleware(cb *CircuitBreaker) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cb.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			state := cb.getState()
			switch state {
			case StateOpen:
				if time.Since(cb.lastStateChange) > cb.config.SleepWindow {
					cb.setState(StateHalfOpen)
				} else {
					http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
					return
				}
			case StateHalfOpen:
				atomic.AddInt64(&cb.totalRequests, 1)
				if atomic.LoadInt64(&cb.totalRequests) > int64(cb.config.MaxRequests) {
					http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
					return
				}
			}

			lrw := &circuitResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lrw, r)

			if lrw.statusCode >= 500 {
				atomic.AddInt64(&cb.failureCount, 1)
				cb.checkThreshold()
			} else {
				atomic.AddInt64(&cb.successCount, 1)
				if state == StateHalfOpen {
					cb.setState(StateClosed)
					atomic.StoreInt64(&cb.failureCount, 0)
					atomic.StoreInt64(&cb.totalRequests, 0)
				}
			}
		})
	}
}

type circuitResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *circuitResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}