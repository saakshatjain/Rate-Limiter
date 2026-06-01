package middleware

import (
	"net/http"

	"github.com/saakshatjain/ratelimiter"
)

type RateLimiterMiddleware struct {
	limiter *ratelimiter.RateLimiter
}

func New(limiter *ratelimiter.RateLimiter) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{limiter: limiter}
}

func (m *RateLimiterMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        result, err := m.limiter.Allow(r.RemoteAddr)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        if !result.Allowed {
            w.WriteHeader(429)
            return
        }
        next.ServeHTTP(w, r)
    })
}