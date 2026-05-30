package ratelimiter

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultLimit           = 100
	DefaultWindow          = time.Minute
	DefaultSendHeaders     = true
	DefaultCleanupInterval = 5 * time.Minute
)

type Config struct {
	Limit           int
	Window          time.Duration
	SendHeaders     bool
	CleanupInterval time.Duration
	KeyFunc         func(r *http.Request) string
	OnBlocked       func(w http.ResponseWriter, r *http.Request)
}

func NewDefaultConfig() Config {
	return Config{
		Limit:           DefaultLimit,
		Window:          DefaultWindow,
		SendHeaders:     DefaultSendHeaders,
		CleanupInterval: DefaultCleanupInterval,

		KeyFunc: func(r *http.Request) string {
			ip := r.RemoteAddr
			if strings.Contains(ip, ":") {
				ip = strings.Split(ip, ":")[0]
			}
			return ip
		},

		OnBlocked: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests) // 429
			json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded",
			})
		},
	}
}