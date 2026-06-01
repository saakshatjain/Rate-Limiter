package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/saakshatjain/ratelimiter"
	"github.com/saakshatjain/ratelimiter/middleware"
)

func main() {
	config := ratelimiter.NewDefaultConfig()
	config.Limit  = 5
	config.Window = time.Minute
	limiter := ratelimiter.New(config)
	pingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong!"))
	})
	m := middleware.New(limiter)

	mux := http.NewServeMux()
	mux.Handle("/ping", m.Handler(pingHandler))

	fmt.Println("Server running on :8080")
	fmt.Println("Try: curl http://localhost:8080/ping")
	fmt.Println("Rate limit: 5 requests per minute")
	http.ListenAndServe(":8080", mux)
}