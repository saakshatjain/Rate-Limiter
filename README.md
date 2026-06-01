# ratelimiter

A production-grade HTTP rate limiting library for Go, built with clean architecture and interface-driven design. Supports pluggable algorithms and storage backends.

```bash
go get github.com/saakshatjain/ratelimiter
```

---

## Quick Start

```go
package main

import (
    "net/http"
    "time"

    "github.com/saakshatjain/ratelimiter"
    "github.com/saakshatjain/ratelimiter/middleware"
)

func main() {
    config := ratelimiter.NewDefaultConfig()
    config.Limit  = 100
    config.Window = time.Minute

    limiter := ratelimiter.New(config)
    m := middleware.New(limiter)

    mux := http.NewServeMux()
    mux.Handle("/api", m.Handler(yourHandler))

    http.ListenAndServe(":8080", mux)
}
```

---

## Features

- Token bucket algorithm with lazy refill — no background goroutines per user
- Per-IP rate limiting by default, customizable via `KeyFunc`
- Pluggable storage backends via `Store` interface (Memory included, Redis planned)
- Pluggable algorithms via `Algorithm` interface
- Automatic cleanup of stale entries to prevent memory leaks
- Custom block handler — return any response on 429
- Works as standard `net/http` middleware

---

## How It Works

Every incoming HTTP request passes through two layers:

```
Incoming Request
      │
      ▼
┌─────────────────────┐
│  Middleware          │  extracts client IP
│  middleware/http.go  │  calls limiter.Allow(ip)
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  RateLimiter        │  coordinates store + algorithm
│  limiter.go         │  fetches state, runs check, saves state
└────────┬────────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌────────┐ ┌─────────────────┐
│ Store  │ │ Algorithm        │
│        │ │                  │
│ Get()  │ │ Allow()          │
│ Set()  │ │ lazy token refill│
└────────┘ └─────────────────┘
    │
    ▼
map[ip]*ClientData
{ Tokens: 4.0, LastRequest: time.Now() }
```

---

## Architecture

### File Structure

```
ratelimiter/
├── result.go                  ← Result struct returned on every check
├── config.go                  ← Config struct + defaults
├── limiter.go                 ← Core RateLimiter, connects all components
├── store/
│   ├── interface.go           ← Store interface + ClientData struct
│   └── memory.go              ← In-memory implementation (sync.RWMutex)
├── algorithms/
│   ├── interface.go           ← Algorithm interface
│   └── token_bucket.go        ← Token bucket with lazy refill
├── middleware/
│   └── http.go                ← net/http middleware
└── examples/
    └── basic/main.go          ← Simple demo server
```

### Key Design Decisions

**Interface-driven design**

Both storage and algorithm are behind interfaces. Swap Redis for memory or sliding window for token bucket with a single config change — no other code changes needed.

```go
type Store interface {
    Get(key string) (*ClientData, error)
    Set(key string, data *ClientData, ttl time.Duration) error
    Delete(key string) error
    Cleanup(olderThan time.Duration) error
}

type Algorithm interface {
    Allow(data *store.ClientData, limit int, window time.Duration) (*store.ClientData, bool, int, time.Duration, time.Time)
}
```

**Lazy token refill**

Tokens are not refilled on a timer. They are calculated mathematically at the moment of each request based on elapsed time. No goroutine per user, no wasted CPU for idle clients.

```
elapsed    = now - lastRequest
refillRate = limit / window
tokensToAdd = elapsed × refillRate
tokens     = min(capacity, tokens + tokensToAdd)
```

**Separation of concerns**

```
middleware  → HTTP layer only (extract IP, write 429)
limiter     → orchestration only (connect store + algorithm)
algorithm   → math only (token calculation)
store       → persistence only (read/write client state)
```

**Thread safety**

The memory store uses `sync.RWMutex` — multiple goroutines can read simultaneously, writes are exclusive. This means concurrent requests from different IPs never block each other unnecessarily.

```go
// reads — parallel allowed
s.mu.RLock()
defer s.mu.RUnlock()

// writes — exclusive
s.mu.Lock()
defer s.mu.Unlock()
```

**Automatic memory cleanup**

A background goroutine runs every `CleanupInterval` and removes client entries not seen in `Window * 2`. Prevents unbounded memory growth from millions of unique IPs.

---

## Configuration

```go
type Config struct {
    Limit           int                                          // max requests allowed
    Window          time.Duration                               // per this time period
    SendHeaders     bool                                        // send X-RateLimit-* headers
    CleanupInterval time.Duration                               // how often to remove stale IPs
    KeyFunc         func(r *http.Request) string                // how to identify a client
    OnBlocked       func(w http.ResponseWriter, r *http.Request) // custom 429 response
}
```

### Examples

**Limit by API key instead of IP:**
```go
config.KeyFunc = func(r *http.Request) string {
    return r.Header.Get("X-API-Key")
}
```

**Limit by user ID (after auth middleware):**
```go
config.KeyFunc = func(r *http.Request) string {
    return r.Context().Value("userID").(string)
}
```

**Different limits per route:**
```go
config.KeyFunc = func(r *http.Request) string {
    return r.RemoteAddr + ":" + r.URL.Path
}
```

**Custom 429 response:**
```go
config.OnBlocked = func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusTooManyRequests)
    json.NewEncoder(w).Encode(map[string]string{
        "error":   "rate limit exceeded",
        "message": "please slow down",
    })
}
```

---

## Result

Every call to `Allow()` returns a `Result`:

```go
type Result struct {
    Allowed    bool          // true = let through, false = block
    Remaining  int           // requests left in current window
    RetryAfter time.Duration // how long to wait if blocked
    ResetAt    time.Time     // when quota fully resets
    Limit      int           // total limit for reference
}
```

---

## Testing

Run the included tests:

```bash
go test ./... -v
```

Load test with hey:

```bash
go install github.com/rakyll/hey@latest
hey -n 20 -c 1 http://localhost:8080/ping
```

Expected output with limit=5:
```
Status code distribution:
  [200] 5 responses   ← allowed
  [429] 15 responses  ← blocked
```

---

## Roadmap

### Redis Store (distributed rate limiting)

The current memory store works for a single server. For multiple servers behind a load balancer, each server has its own independent counter — a user could hit 5 requests on server A and 5 more on server B, bypassing the limit entirely.

```
Current (memory):                 Planned (Redis):
                                  
Server A  Server B                Server A  Server B
[map]     [map]                       ↘        ↙
independent ❌                       Redis Store
                                    shared ✅
```

Redis integration will implement the same `Store` interface:

```go
// store/redis.go (planned)
type RedisStore struct {
    client *redis.Client
}

func (s *RedisStore) Get(key string) (*ClientData, error) {
    // fetch from Redis, deserialize JSON → ClientData
}

func (s *RedisStore) Set(key string, data *ClientData, ttl time.Duration) error {
    // serialize ClientData → JSON
    // store with TTL — Redis auto-expires, no manual cleanup needed
}
```

Switching to Redis will require one config change:

```go
// today
config.StoreType = ratelimiter.Memory

// after Redis is added
config.StoreType = ratelimiter.Redis
config.RedisAddr = "localhost:6379"
```

No other code changes needed — the interface abstracts the difference.

### Sliding Window Algorithm

Add a global server-wide limit on top of the per-user token bucket. Protects against thundering herd scenarios where many well-behaved users spike simultaneously.

### Gin Middleware

```go
// middleware/gin.go (planned)
router.Use(m.GinMiddleware())
```

### Rate Limit Headers

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 43
X-RateLimit-Reset: 1715000000
Retry-After: 30
```

---

## License

MIT